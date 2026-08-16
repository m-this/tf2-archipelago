/**
 * Talking to the bridge.
 *
 * Every call is asynchronous: a blocking HTTP call stalls the game frame for
 * everyone on the server (ADR 0002). Nothing here knows what Archipelago is.
 *
 * The plugin's copy of an objective is not durable, so it retries until the
 * bridge takes it. The bridge is where "never lose a check" is actually
 * guaranteed; this queue only has to survive the seconds in between.
 */

// PendingMax bounds the retry queue. A mission is at most nine objectives, so
// anything approaching this means the bridge has been gone a long time.
#define PendingMax 64

// RetryInterval is how often the queue is drained. Slow enough not to hammer a
// bridge that is down, quick enough that a wave clear lands before the next.
#define RetryInterval 5.0

enum
{
    Unlock_Class = 0,
    Unlock_Slot,
    Unlock_Mission,
};

enum struct Objective
{
    // id identifies the objective across a retry. The queue index cannot: an
    // erase shifts every index after it while requests are still in flight.
    int id;
    char kind[24];
    char popFile[64];
    int wave;
}

ConVar g_CvarBridgeURL;
ConVar g_CvarChat;

ArrayList g_Pending;
bool g_PollingChat;
int g_ChatSeq = -1;
int g_NextObjectiveID = 1;
bool g_Polling;
int g_GrantSeq;
char g_LastBridgeError[192];

void Bridge_Init()
{
    g_CvarBridgeURL = CreateConVar("tf2ap_bridge_url", "http://127.0.0.1:24680",
        "Base URL of the Archipelago bridge. Loopback only.");
    g_CvarChat = CreateConVar("tf2ap_chat", "1",
        "Show what the rest of the multiworld says in chat.", _, true, 0.0, true, 1.0);
    g_Pending = new ArrayList(sizeof(Objective));
    CreateTimer(RetryInterval, Timer_DrainPending, _, TIMER_REPEAT | TIMER_FLAG_NO_MAPCHANGE);
}

static HTTPRequest BridgeRequest(const char[] path)
{
    char base[192];
    g_CvarBridgeURL.GetString(base, sizeof(base));

    char url[256];
    FormatEx(url, sizeof(url), "%s%s", base, path);
    return new HTTPRequest(url);
}

int Bridge_PendingCount()
{
    return g_Pending.Length;
}

void Bridge_LastError(char[] buffer, int maxlength)
{
    strcopy(buffer, maxlength, g_LastBridgeError);
}

int Bridge_Sequence()
{
    return g_GrantSeq;
}

/**
 * Report an objective. Queued first, sent immediately after: if the send
 * fails the objective is already where the retry timer will find it.
 */
void Bridge_ReportObjective(const char[] kind, const char[] popFile, int wave)
{
    if (g_Pending.Length >= PendingMax)
    {
        AP_Error("the objective queue is full (%d), dropping %s %s wave %d",
            PendingMax, kind, popFile, wave);
        return;
    }

    Objective objective;
    objective.id = g_NextObjectiveID++;
    strcopy(objective.kind, sizeof(objective.kind), kind);
    strcopy(objective.popFile, sizeof(objective.popFile), popFile);
    objective.wave = wave;
    g_Pending.PushArray(objective);

    AP_Debug("queued %s %s wave %d", kind, popFile, wave);
    Bridge_SendObjective(objective);
}

static void Bridge_SendObjective(Objective objective)
{
    JSONObject body = new JSONObject();
    body.SetString("kind", objective.kind);
    body.SetString("popfile", objective.popFile);
    body.SetInt("wave", objective.wave);

    HTTPRequest request = BridgeRequest("/objective");
    request.Timeout = 10;
    request.Post(body, OnObjectivePosted, objective.id);
    delete body;
}

static int Bridge_FindPending(int id)
{
    for (int index = 0; index < g_Pending.Length; index++)
    {
        Objective objective;
        g_Pending.GetArray(index, objective);
        if (objective.id == id)
        {
            return index;
        }
    }
    return -1;
}

/**
 * The objective leaves the queue only once the bridge has it on disk. The
 * bridge is idempotent by location id, so a retry after a timeout that
 * actually landed costs nothing.
 */
static void OnObjectivePosted(HTTPResponse response, any value, const char[] error)
{
    int index = Bridge_FindPending(value);
    if (index == -1)
    {
        return;
    }
    Objective objective;
    g_Pending.GetArray(index, objective);

    if (!StrEqual(error, ""))
    {
        Bridge_NoteError("could not reach the bridge: %s", error);
        return;
    }
    if (response.Status == HTTPStatus_NoContent)
    {
        g_Pending.Erase(index);
        Bridge_ClearError();
        AP_Debug("bridge took %s %s wave %d", objective.kind, objective.popFile, objective.wave);
        return;
    }
    if (response.Status == HTTPStatus_BadRequest)
    {
        // The bridge and this plugin disagree about what exists. Retrying
        // cannot fix that, and keeping it would block the queue forever.
        g_Pending.Erase(index);
        AP_Error("the bridge rejected %s %s wave %d: it does not know that objective",
            objective.kind, objective.popFile, objective.wave);
        return;
    }
    Bridge_NoteError("the bridge answered %d for %s %s wave %d, will retry",
        response.Status, objective.kind, objective.popFile, objective.wave);
}

static Action Timer_DrainPending(Handle timer)
{
    if (g_Pending.Length == 0)
    {
        return Plugin_Continue;
    }
    AP_Debug("retrying %d queued objective(s)", g_Pending.Length);
    for (int index = 0; index < g_Pending.Length; index++)
    {
        Objective objective;
        g_Pending.GetArray(index, objective);
        Bridge_SendObjective(objective);
    }
    return Plugin_Continue;
}

/**
 * Ask for the whole unlock set. Called on plugin start and on every map
 * change, because the plugin remembers nothing across either.
 */
void Bridge_FetchUnlocks()
{
    HTTPRequest request = BridgeRequest("/unlocks");
    request.Timeout = 10;
    request.Get(OnUnlocks);
}

static void OnUnlocks(HTTPResponse response, any value, const char[] error)
{
    if (!StrEqual(error, ""))
    {
        Bridge_NoteError("could not fetch the unlock set: %s", error);
        return;
    }
    if (response.Status != HTTPStatus_OK)
    {
        Bridge_NoteError("the unlock set came back as %d", response.Status);
        return;
    }
    JSONObject unlocks = view_as<JSONObject>(response.Data);

    Unlocks_Clear();
    g_GrantSeq = unlocks.GetInt("seq");
    ApplyKeyArray(unlocks, "classes", Unlock_Class);
    ApplyKeyArray(unlocks, "slots", Unlock_Slot);
    ApplyKeyArray(unlocks, "missions", Unlock_Mission);
    g_HaveUnlocks = true;
    Bridge_ClearError();

    AP_Debug("unlock set at sequence %d applied", g_GrantSeq);
    Unlocks_ApplyToAll();
    Unlocks_WarnIfMissionLocked();
}

static void ApplyKeyArray(JSONObject unlocks, const char[] field, int kind)
{
    JSONArray keys = view_as<JSONArray>(unlocks.Get(field));
    if (keys == null)
    {
        return;
    }
    for (int index = 0; index < keys.Length; index++)
    {
        char key[64];
        keys.GetString(index, key, sizeof(key));
        switch (kind)
        {
            case Unlock_Class: Unlocks_GrantClass(key);
            case Unlock_Slot: Unlocks_GrantSlot(key);
            case Unlock_Mission: Unlocks_GrantMission(key);
        }
    }
    delete keys;
}

/**
 * Long-poll for grants. The bridge holds the request open, so this is one
 * connection that mostly sits idle, not a poll loop.
 */
void Bridge_PollGrants()
{
    if (g_Polling)
    {
        return;
    }
    char path[64];
    FormatEx(path, sizeof(path), "/grants?since=%d", g_GrantSeq);

    HTTPRequest request = BridgeRequest(path);
    // Longer than the bridge's own poll timeout, so the bridge is what ends
    // the wait and a timeout here always means something is wrong.
    request.Timeout = 60;
    g_Polling = true;
    request.Get(OnGrants);
}

static void OnGrants(HTTPResponse response, any value, const char[] error)
{
    g_Polling = false;

    if (!StrEqual(error, ""))
    {
        Bridge_NoteError("the grant poll failed: %s", error);
        CreateTimer(RetryInterval, Timer_PollAgain);
        return;
    }
    if (response.Status == HTTPStatus_NoContent)
    {
        Bridge_ClearError();
        Bridge_PollGrants();
        return;
    }
    if (response.Status != HTTPStatus_OK)
    {
        Bridge_NoteError("the grant poll answered %d", response.Status);
        CreateTimer(RetryInterval, Timer_PollAgain);
        return;
    }

    JSONObject payload = view_as<JSONObject>(response.Data);
    JSONArray grants = view_as<JSONArray>(payload.Get("grants"));
    if (grants != null)
    {
        for (int index = 0; index < grants.Length; index++)
        {
            JSONObject grant = view_as<JSONObject>(grants.Get(index));
            ApplyGrant(grant);
            delete grant;
        }
        delete grants;
    }
    Bridge_ClearError();
    Unlocks_ApplyToAll();
    Bridge_PollGrants();
}

static Action Timer_PollAgain(Handle timer)
{
    Bridge_PollGrants();
    return Plugin_Stop;
}

/**
 * One grant. The kinds are the bridge's half of the wire contract; an unknown
 * one is a bridge newer than this plugin, which is worth saying out loud but
 * not worth stopping for.
 */
static void ApplyGrant(JSONObject grant)
{
    int seq = grant.GetInt("seq");
    char kind[24], key[64], name[64];
    grant.GetString("kind", kind, sizeof(kind));
    if (!grant.GetString("key", key, sizeof(key)))
    {
        key[0] = '\0';
    }
    if (!grant.GetString("name", name, sizeof(name)))
    {
        strcopy(name, sizeof(name), key);
    }

    if (StrEqual(kind, "class"))
    {
        Unlocks_GrantClass(key);
        AP_Announce("Class unlocked: %s", name);
    }
    else if (StrEqual(kind, "weapon_slot"))
    {
        Unlocks_GrantSlot(key);
        AP_Announce("Weapon slot unlocked: %s", name);
    }
    else if (StrEqual(kind, "mission_ticket"))
    {
        Unlocks_GrantMission(key);
        AP_Announce("Mission unlocked: %s", name);
    }
    else if (StrEqual(kind, "credits"))
    {
        MvM_GrantCredits(grant.GetInt("amount"));
    }
    else
    {
        AP_Error("the bridge sent a grant of kind %s, which this plugin does not know", kind);
    }

    if (seq > g_GrantSeq)
    {
        g_GrantSeq = seq;
    }
}

/**
 * Bridge_NoteError keeps the last failure for sm_ap and says it once. The
 * retry loop would otherwise fill chat with the same line every five seconds.
 */
static void Bridge_NoteError(const char[] format, any ...)
{
    char message[192];
    VFormat(message, sizeof(message), format, 2);
    if (StrEqual(message, g_LastBridgeError))
    {
        AP_Debug("%s", message);
        return;
    }
    strcopy(g_LastBridgeError, sizeof(g_LastBridgeError), message);
    AP_Error("%s", message);
}

static void Bridge_ClearError()
{
    g_LastBridgeError[0] = '\0';
}


/**
 * The multiworld's chat.
 *
 * The bridge holds this request open the same way it does for grants. The
 * first poll asks with a negative sequence, which means "start from now": a
 * server joining an evening late must not replay hours of other people's
 * conversation into everyone's chat.
 */
void Bridge_PollMessages()
{
    if (g_PollingChat)
    {
        return;
    }
    char path[64];
    FormatEx(path, sizeof(path), "/messages?since=%d", g_ChatSeq);

    HTTPRequest request = BridgeRequest(path);
    request.Timeout = 60;
    g_PollingChat = true;
    request.Get(OnMessages);
}

static void OnMessages(HTTPResponse response, any value, const char[] error)
{
    g_PollingChat = false;

    if (!StrEqual(error, "") || response.Status != HTTPStatus_OK)
    {
        // Quiet on purpose: the objective and grant paths already say when the
        // bridge is unreachable, and chat is the least important of the three.
        AP_Debug("the message poll failed, retrying");
        CreateTimer(RetryInterval, Timer_PollMessagesAgain);
        return;
    }

    JSONObject payload = view_as<JSONObject>(response.Data);
    g_ChatSeq = payload.GetInt("seq");

    JSONArray messages = view_as<JSONArray>(payload.Get("messages"));
    if (messages != null)
    {
        bool show = g_CvarChat.BoolValue;
        for (int index = 0; index < messages.Length; index++)
        {
            JSONObject message = view_as<JSONObject>(messages.Get(index));
            char text[254];
            message.GetString("text", text, sizeof(text));
            if (show && text[0] != '\0')
            {
                PrintToChatAll("%s\x07AAAAAA%s", AP_TAG, text);
            }
            delete message;
        }
        delete messages;
    }
    Bridge_PollMessages();
}

static Action Timer_PollMessagesAgain(Handle timer)
{
    Bridge_PollMessages();
    return Plugin_Stop;
}

/**
 * Pass a line to the multiworld. Not queued: a message that lands ten minutes
 * after it was typed is worse than one that was refused, and the player who
 * typed it is standing right there to be told.
 */
void Bridge_Say(int client, const char[] text)
{
    JSONObject body = new JSONObject();
    body.SetString("text", text);

    HTTPRequest request = BridgeRequest("/say");
    request.Timeout = 10;
    request.Post(body, OnSaid, client > 0 ? GetClientUserId(client) : 0);
    delete body;
}

static void OnSaid(HTTPResponse response, any value, const char[] error)
{
    int client = value > 0 ? GetClientOfUserId(value) : 0;
    if (StrEqual(error, "") && response.Status == HTTPStatus_NoContent)
    {
        return;
    }
    if (client > 0)
    {
        AP_PrintToClient(client, "The multiworld did not take that: it is not connected.");
    }
    AP_Debug("say failed with status %d %s", response.Status, error);
}

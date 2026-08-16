/**
 * Talking to the bridge.
 *
 * Every call is asynchronous: a blocking HTTP call stalls the game frame for
 * everyone on the server (ADR 0002). Nothing here knows what Archipelago is.
 *
 * The queue below is not durable and does not have to be. The bridge is where
 * "never lose a check" is guaranteed; this only covers the seconds in between.
 */

// A mission is at most nine objectives, so nearing this means the bridge has been gone a long time.
#define PendingMax 64

// Slow enough not to hammer a bridge that is down, quick enough to land a wave before the next one.
#define RetryInterval 5.0

enum
{
    Unlock_Class = 0,
    Unlock_Slot,
    Unlock_Mission,
};

enum struct Objective
{
    // The queue index cannot identify a retry: an erase shifts every index after it mid-flight.
    int id;
    char kind[24];
    char popFile[64];
    int wave;
}

ConVar g_CvarBridgeURL;
ConVar g_CvarChat;

ArrayList g_Pending;
bool g_PollingChat;

// A negative sequence means "start from now": joining late must not replay hours of chat.
int g_ChatSeq = -1;

int g_NextObjectiveID = 1;
bool g_PollingGrants;
int g_GrantSeq;
char g_LastBridgeError[192];

void Bridge_Init()
{
    g_CvarBridgeURL = CreateConVar("tf2ap_bridge_url", "http://127.0.0.1:24680",
        "Base URL of the Archipelago bridge. Loopback only.");
    g_CvarChat = CreateConVar("tf2ap_chat", "1",
        "Write what the other players in the multiworld say to the chat.", _, true, 0.0, true, 1.0);
    g_Pending = new ArrayList(sizeof(Objective));
    // No TIMER_FLAG_NO_MAPCHANGE: it dies at the first map change, a queued check must survive one.
    CreateTimer(RetryInterval, Timer_DrainPending, _, TIMER_REPEAT);
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

// Queued before it is sent, so a failed send leaves it where the retry timer will find it.
void Bridge_ReportObjective(const char[] kind, const char[] popFile, int wave)
{
    if (g_Pending.Length >= PendingMax)
    {
        AP_Error("The objective queue is full at %d. The plugin dropped %s %s wave %d.",
            PendingMax, kind, popFile, wave);
        return;
    }

    Objective objective;
    objective.id = g_NextObjectiveID++;
    strcopy(objective.kind, sizeof(objective.kind), kind);
    strcopy(objective.popFile, sizeof(objective.popFile), popFile);
    objective.wave = wave;
    g_Pending.PushArray(objective);

    AP_Debug("The plugin queued %s %s wave %d.", kind, popFile, wave);
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

// Dequeued only once the bridge has it on disk; it is idempotent, so a needless retry is free.
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
        Bridge_NoteError("The plugin cannot reach the bridge. %s", error);
        return;
    }
    if (response.Status == HTTPStatus_NoContent)
    {
        g_Pending.Erase(index);
        Bridge_ClearError();
        AP_Debug("The bridge took %s %s wave %d.", objective.kind, objective.popFile, objective.wave);
        return;
    }
    if (response.Status == HTTPStatus_BadRequest)
    {
        // Retrying cannot fix a disagreement about what exists, and keeping it blocks the queue.
        g_Pending.Erase(index);
        AP_Error("The bridge does not know the objective %s %s wave %d. It refused the check.",
            objective.kind, objective.popFile, objective.wave);
        return;
    }
    Bridge_NoteError("The bridge answered %d for %s %s wave %d. The plugin tries again.",
        response.Status, objective.kind, objective.popFile, objective.wave);
}

static Action Timer_DrainPending(Handle timer)
{
    if (g_Pending.Length == 0)
    {
        return Plugin_Continue;
    }
    AP_Debug("The plugin sends %d queued objective(s) again.", g_Pending.Length);
    for (int index = 0; index < g_Pending.Length; index++)
    {
        Objective objective;
        g_Pending.GetArray(index, objective);
        Bridge_SendObjective(objective);
    }
    return Plugin_Continue;
}

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
        Bridge_NoteError("The plugin cannot get the unlock set. %s", error);
        return;
    }
    if (response.Status != HTTPStatus_OK)
    {
        Bridge_NoteError("The bridge answered %d for the unlock set.", response.Status);
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

    AP_Debug("The plugin applied the unlock set at sequence %d.", g_GrantSeq);
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

// The bridge holds the request open: one connection that mostly sits idle, not a poll loop.
void Bridge_PollGrants()
{
    if (g_PollingGrants)
    {
        return;
    }
    char path[64];
    FormatEx(path, sizeof(path), "/grants?since=%d", g_GrantSeq);

    HTTPRequest request = BridgeRequest(path);
    // Longer than the bridge's own poll timeout, so a timeout here always means something is wrong.
    request.Timeout = 60;
    g_PollingGrants = true;
    request.Get(OnGrants);
}

static void OnGrants(HTTPResponse response, any value, const char[] error)
{
    g_PollingGrants = false;

    if (!StrEqual(error, ""))
    {
        Bridge_NoteError("The grant request failed. %s", error);
        CreateTimer(RetryInterval, Timer_PollAgain);
        return;
    }
    if (response.Status != HTTPStatus_OK)
    {
        Bridge_NoteError("The bridge answered %d for the grants.", response.Status);
        CreateTimer(RetryInterval, Timer_PollAgain);
        return;
    }

    JSONObject payload = view_as<JSONObject>(response.Data);

    // A bridge behind this plugin means the run restarted: these grants belong to a different one.
    if (payload.GetInt("seq") < g_GrantSeq)
    {
        AP_Error("The run restarted. The plugin asks for the unlock set again.");
        g_GrantSeq = 0;
        Bridge_FetchUnlocks();
        CreateTimer(RetryInterval, Timer_PollAgain);
        return;
    }

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
        AP_Error("The bridge sent a grant of kind %s. This plugin does not know it.", kind);
    }

    if (seq > g_GrantSeq)
    {
        g_GrantSeq = seq;
    }
}

// Says a repeated failure once: the retry loop would otherwise fill chat every RetryInterval.
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
        // Quiet on purpose: the objective and grant paths already report an unreachable bridge.
        AP_Debug("The message request failed. The plugin tries again.");
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

// Never queued: a line landing ten minutes late is worse than one refused to the player's face.
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
        AP_PrintToClient(client, "The bridge has no connection to the multiworld. It refused your line.");
    }
    AP_Debug("The line failed with status %d. %s", response.Status, error);
}

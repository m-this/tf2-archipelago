/**
 * Talking to the bridge.
 *
 * Every call is asynchronous: a blocking HTTP call stalls the game frame for
 * everyone on the server (ADR 0002). Nothing here knows what Archipelago is.
 *
 * The queue below is not durable and does not have to be. The bridge is where
 * "never lose a check" is guaranteed; this only covers the seconds in between.
 * Every objective is written to the SourceMod log as it is queued, so an
 * admin can replay one with sm_ap_report if srcds dies holding it.
 */

// The plugin-facing contract. The bridge reports its own; a mismatch means one
// half of the compose file was updated and the other was not.
#define BridgeAPIVersion 1

// A mission is at most nine objectives, so nearing this means the bridge has been gone a long time.
#define PendingMax 64

// Slow enough not to hammer a bridge that is down, quick enough to land a wave before the next one.
#define RetryInterval 5.0

// A player may speak to the multiworld this often. The bridge has its own
// guard; this one exists to tell the player which of them it was.
#define SayCooldown 3.0

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

    // What the game says the mission's length is, zero when it would not say.
    // The wave counts in the tables come from the wiki and none of them has
    // been checked against a running server.
    int wavesTotal;
}

ConVar g_CvarBridgeURL;
ConVar g_CvarChat;

ArrayList g_Pending;
bool g_PollingChat;

// A negative sequence means "start from now": joining late must not replay hours of chat.
int g_ChatSeq = -1;

int g_NextObjectiveID = 1;
bool g_PollingGrants;
bool g_FetchingUnlocks;
int g_GrantSeq;

// How far the bridge has been told the plugin got. Effects at or below it are
// never sent again, which is what stops a reload paying the run's cash twice.
int g_AckedSeq;

char g_LastBridgeError[192];
float g_LastSaid[MAXPLAYERS + 1];

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

// Read once at load. A plugin and a bridge that disagree about the wire format
// fail in the middle of a wave otherwise, with nothing saying why.
void Bridge_CheckVersion()
{
    HTTPRequest request = BridgeRequest("/healthz");
    request.Timeout = 10;
    request.Get(OnVersionChecked);
}

static void OnVersionChecked(HTTPResponse response, any value, const char[] error)
{
    if (!StrEqual(error, "") || response.Status != HTTPStatus_OK)
    {
        Bridge_NoteError("The plugin cannot reach the bridge at all. Nothing will be recorded.");
        return;
    }
    JSONObject health = view_as<JSONObject>(response.Data);
    int version = health.GetInt("api_version");
    if (version != BridgeAPIVersion)
    {
        AP_Error("The bridge speaks API version %d and this plugin speaks %d. Update the one that is behind.",
            version, BridgeAPIVersion);
        return;
    }
    AP_Debug("The bridge answered, API version %d.", version);
}

// Queued before it is sent, so a failed send leaves it where the retry timer will find it.
void Bridge_ReportObjective(const char[] kind, const char[] popFile, int wave, int wavesTotal)
{
    if (g_Pending.Length >= PendingMax)
    {
        AP_Error("The objective queue is full at %d. The plugin dropped %s %s wave %d.",
            PendingMax, kind, popFile, wave);
        return;
    }

    // Logged whatever happens next, and only once it is really queued. The
    // queue lives in memory, so an srcds crash while the bridge is down loses
    // it, and this line is what lets an admin put it back by hand with
    // sm_ap_report.
    LogMessage("objective %s %s wave %d (mission length %d) queued for the bridge",
        kind, popFile, wave, wavesTotal);

    Objective objective;
    objective.id = g_NextObjectiveID++;
    strcopy(objective.kind, sizeof(objective.kind), kind);
    strcopy(objective.popFile, sizeof(objective.popFile), popFile);
    objective.wave = wave;
    objective.wavesTotal = wavesTotal;
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
    body.SetInt("waves_total", objective.wavesTotal);

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
        LogMessage("objective %s %s wave %d is on the bridge's disk",
            objective.kind, objective.popFile, objective.wave);
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

// One timer for everything that has to be said again: the queued objectives,
// the sequence the bridge has not confirmed yet, and the two request loops.
//
// The loops are the reason this is a watchdog rather than a retry. Each one
// keeps itself alive by starting the next request from its own callback, so a
// native error anywhere in a callback ends it for the rest of the map with no
// timer left to notice. Nothing in SourcePawn recovers a plugin from that
// except something outside the loop looking in.
static Action Timer_DrainPending(Handle timer)
{
    if (g_HaveUnlocks && !g_PollingGrants)
    {
        AP_Debug("The grant poll had stopped. The plugin starts it again.");
        Bridge_PollGrants();
    }
    if (!g_HaveUnlocks)
    {
        Bridge_FetchUnlocks();
    }
    if (!g_PollingChat)
    {
        Bridge_PollMessages();
    }
    if (g_GrantSeq > g_AckedSeq)
    {
        Bridge_SendAck();
    }
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

// Tells the bridge how far the plugin got, so an effect it already applied is
// not handed to it again. Classes and slots do not need it: granting one twice
// is the same as granting it once. Credits are not, and neither are the traps
// that come later.
static void Bridge_SendAck()
{
    JSONObject body = new JSONObject();
    body.SetInt("seq", g_GrantSeq);

    HTTPRequest request = BridgeRequest("/grants/ack");
    request.Timeout = 10;
    request.Post(body, OnAcked, g_GrantSeq);
    delete body;
}

static void OnAcked(HTTPResponse response, any value, const char[] error)
{
    if (StrEqual(error, "") && response.Status == HTTPStatus_BadRequest)
    {
        // The bridge holds fewer items than this sequence, so the run changed
        // underneath the plugin. Retrying the same number every five seconds
        // would never stop; asking what the run is now does.
        AP_Debug("The bridge refused the acknowledgement of sequence %d. The plugin resyncs.", value);
        g_GrantSeq = 0;
        g_AckedSeq = 0;
        Bridge_FetchUnlocks();
        return;
    }
    if (!StrEqual(error, "") || response.Status != HTTPStatus_NoContent)
    {
        AP_Debug("The bridge did not take the acknowledgement of sequence %d. The plugin tries again.", value);
        return;
    }
    // A run restart zeroes both cursors while an acknowledgement may still be
    // in flight. Letting that late answer set the cursor above where the plugin
    // now is stops it acknowledging anything for the rest of the run.
    if (value > g_GrantSeq)
    {
        AP_Debug("An acknowledgement of sequence %d arrived from a run that ended.", value);
        return;
    }
    if (value > g_AckedSeq)
    {
        g_AckedSeq = value;
    }
    AP_Debug("The bridge acknowledged sequence %d.", value);
}

// Guarded like the polls are. A map change, sm_ap_resync and the retry timer
// all ask for the unlock set, and without this each one starts its own retry
// chain that never ends: a bridge down over three map changes comes back to
// four fetches every five seconds, each one clearing and reapplying the set.
void Bridge_FetchUnlocks()
{
    if (g_FetchingUnlocks)
    {
        return;
    }
    HTTPRequest request = BridgeRequest("/unlocks");
    request.Timeout = 10;
    g_FetchingUnlocks = true;
    request.Get(OnUnlocks);
}

static void OnUnlocks(HTTPResponse response, any value, const char[] error)
{
    g_FetchingUnlocks = false;

    // Asked for again rather than moving on to the grants: the poll cursor
    // comes from the unlock set, and polling without one asks from zero.
    if (!StrEqual(error, ""))
    {
        Bridge_NoteError("The plugin cannot get the unlock set. %s", error);
        CreateTimer(RetryInterval, Timer_FetchUnlocksAgain);
        return;
    }
    if (response.Status != HTTPStatus_OK)
    {
        Bridge_NoteError("The bridge answered %d for the unlock set.", response.Status);
        CreateTimer(RetryInterval, Timer_FetchUnlocksAgain);
        return;
    }
    JSONObject payload = view_as<JSONObject>(response.Data);

    Unlocks_Clear();

    // The bridge answers with the sequence it has been acknowledged up to, not
    // with how far its item list reaches. Anything between the two is state the
    // plugin is about to be handed again, which costs nothing, plus every
    // effect it never received. Taking the item count here instead would step
    // over an unapplied cash bundle and lose it for good.
    g_GrantSeq = payload.GetInt("resume_from");
    ApplyUnlockSet(payload);
    g_HaveUnlocks = true;
    Bridge_ClearError();

    AP_Debug("The plugin applied the unlock set at sequence %d.", g_GrantSeq);
    Unlocks_ApplyToAll();
    Unlocks_WarnIfMissionLocked();

    // Only now. Polling for grants before the unlock set has landed means
    // polling from sequence zero, and every effect in the run arrives again.
    Bridge_PollGrants();
}

static Action Timer_FetchUnlocksAgain(Handle timer)
{
    Bridge_FetchUnlocks();
    return Plugin_Stop;
}

// The unlock set is keyed by grant kind rather than by a field per kind, so a
// kind this plugin has never heard of is skipped instead of failing the parse.
static void ApplyUnlockSet(JSONObject payload)
{
    JSONObject byKind = view_as<JSONObject>(payload.Get("unlocks"));
    if (byKind == null)
    {
        AP_Error("The bridge sent an unlock set this plugin cannot read.");
        return;
    }

    JSONObjectKeys kinds = byKind.Keys();
    // Wider than any kind the bridge can send. ReadKey truncates without
    // saying so, and reading a truncated key back out of the object is a
    // native error that takes the whole callback with it, which is the
    // opposite of skipping a kind this plugin does not know. HasKey is what
    // makes the skip real.
    char kind[64];
    while (kinds.ReadKey(kind, sizeof(kind)))
    {
        if (!byKind.HasKey(kind))
        {
            AP_Debug("The unlock set holds a kind this plugin cannot read: %s.", kind);
            continue;
        }
        JSONArray keys = view_as<JSONArray>(byKind.Get(kind));
        bool known = keys.Length == 0;
        for (int index = 0; index < keys.Length; index++)
        {
            char key[64];
            keys.GetString(index, key, sizeof(key));
            known = Bridge_ApplyKind(kind, key) || known;
        }
        if (!known)
        {
            AP_Debug("The unlock set holds a kind this plugin does not know: %s.", kind);
        }
        delete keys;
    }
    delete kinds;
    delete byKind;
}

// What a player sees a grant called. The wire kind is not for reading.
static char[] GrantLabel(const char[] kind)
{
    char label[24];
    if (StrEqual(kind, "class"))
    {
        strcopy(label, sizeof(label), "Class");
    }
    else if (StrEqual(kind, "weapon_slot"))
    {
        strcopy(label, sizeof(label), "Weapon slot");
    }
    else if (StrEqual(kind, "mission_ticket"))
    {
        strcopy(label, sizeof(label), "Mission");
    }
    else
    {
        strcopy(label, sizeof(label), "Item");
    }
    return label;
}

// The whole grant vocabulary, in one place. It is the wire format, so it lives
// here rather than beside the unlock set it writes into.
static bool Bridge_ApplyKind(const char[] kind, const char[] key)
{
    if (StrEqual(kind, "class"))
    {
        Unlocks_GrantClass(key);
        return true;
    }
    if (StrEqual(kind, "weapon_slot"))
    {
        Unlocks_GrantSlot(key);
        return true;
    }
    if (StrEqual(kind, "mission_ticket"))
    {
        Unlocks_GrantMission(key);
        return true;
    }
    return false;
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

static void Bridge_PollGrantsLater()
{
    CreateTimer(RetryInterval, Timer_PollAgain);
}

static void OnGrants(HTTPResponse response, any value, const char[] error)
{
    g_PollingGrants = false;

    if (!StrEqual(error, ""))
    {
        Bridge_NoteError("The grant request failed. %s", error);
        Bridge_PollGrantsLater();
        return;
    }
    if (response.Status != HTTPStatus_OK)
    {
        Bridge_NoteError("The bridge answered %d for the grants.", response.Status);
        Bridge_PollGrantsLater();
        return;
    }

    JSONObject payload = view_as<JSONObject>(response.Data);

    // A bridge behind this plugin means the run restarted: these grants belong to a different one.
    if (payload.GetInt("seq") < g_GrantSeq)
    {
        AP_Error("The run restarted. The plugin asks for the unlock set again.");
        g_GrantSeq = 0;
        g_AckedSeq = 0;
        Bridge_FetchUnlocks();
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
    if (g_GrantSeq > g_AckedSeq)
    {
        Bridge_SendAck();
    }
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

    // Applied once. The bridge holds an effect back once it is acknowledged,
    // but the acknowledgement is a round trip and this is what covers it.
    if (seq <= g_GrantSeq)
    {
        AP_Debug("The bridge re-sent sequence %d, which the plugin already applied.", seq);
        return;
    }

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

    if (StrEqual(kind, "credits"))
    {
        MvM_GrantCredits(grant.GetInt("amount"));
    }
    else if (Bridge_ApplyKind(kind, key))
    {
        AP_Announce("%s unlocked: %s", GrantLabel(kind), name);
    }
    else
    {
        AP_Error("The bridge sent a grant of kind %s. This plugin does not know it.", kind);
    }

    g_GrantSeq = seq;
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

void Bridge_ClearCooldown(int client)
{
    g_LastSaid[client] = 0.0;
}

// Never queued: a line landing ten minutes late is worse than one refused to the player's face.
//
// Which commands the multiworld will take is the bridge's rule, not this
// plugin's: one list, on the side that knows Archipelago. The cooldown is here
// because only this side knows which player typed it.
void Bridge_Say(int client, const char[] text)
{
    if (client > 0)
    {
        float now = GetEngineTime();
        if (now - g_LastSaid[client] < SayCooldown)
        {
            AP_PrintToClient(client, "Wait a moment before speaking to the multiworld again.");
            return;
        }
        g_LastSaid[client] = now;
    }

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
    if (client > 0 && IsClientInGame(client))
    {
        AP_PrintToClient(client, "%s", SayRefusal(response.Status));
    }
    AP_Debug("The line failed with status %d. %s", response.Status, error);
}

// The bridge refuses a line in three different ways and the player is standing
// there waiting to be told which.
static char[] SayRefusal(HTTPStatus status)
{
    char reason[128];
    switch (status)
    {
        case HTTPStatus_Forbidden:
        {
            strcopy(reason, sizeof(reason),
                "That multiworld command cannot be sent from the game. It cannot be undone.");
        }
        case HTTPStatus_TooManyRequests:
        {
            strcopy(reason, sizeof(reason), "Too much is going to the multiworld. Wait a moment.");
        }
        case HTTPStatus_RequestEntityTooLarge:
        {
            strcopy(reason, sizeof(reason), "That line is too long for the multiworld.");
        }
        default:
        {
            strcopy(reason, sizeof(reason),
                "The bridge has no connection to the multiworld. It refused your line.");
        }
    }
    return reason;
}

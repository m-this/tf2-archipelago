/**
 * Saying things.
 *
 * There is no client mod, so chat, the server console and the SourceMod log
 * are the whole diagnostic surface (ADR 0002). This plugin gets debugged on a
 * live server by reading chat: a failure nobody can see is blamed on the game.
 */

#define AP_TAG "\x07FFD700[AP]\x01 "
#define AP_TAG_ERROR "\x07FF4040[AP]\x01 "

// How long a line may be before the tag goes in front of it. The engine refuses
// a user message over 255 bytes and drops it without telling anyone, which is a
// failure that looks exactly like a plugin that never ran. Same number as the
// bridge's chat.LineMax, because both sides build a line for this chat.
#define ChatLineMax 200

ConVar g_CvarAnnounce;
ConVar g_CvarDebug;

void Log_Init()
{
    g_CvarAnnounce = CreateConVar("tf2ap_announce", "1",
        "Write the received items and the unlocks to the chat.", _, true, 0.0, true, 1.0);
    g_CvarDebug = CreateConVar("tf2ap_debug", "0",
        "Write every bridge call and every game event to the chat and the console.",
        _, true, 0.0, true, 1.0);
}

stock void AP_Announce(const char[] format, any ...)
{
    char message[ChatLineMax];
    VFormat(message, sizeof(message), format, 2);
    if (g_CvarAnnounce.BoolValue)
    {
        PrintToChatAll("%s%s", AP_TAG, message);
    }
    PrintToServer("[AP] %s", message);
}

// Reaches chat even when announcements are muted: players notice missing checks before anyone else.
stock void AP_Error(const char[] format, any ...)
{
    char message[ChatLineMax];
    VFormat(message, sizeof(message), format, 2);
    PrintToChatAll("%s%s", AP_TAG_ERROR, message);
    PrintToServer("[AP] error: %s", message);
    LogError("%s", message);
}

stock void AP_Debug(const char[] format, any ...)
{
    if (!g_CvarDebug.BoolValue)
    {
        return;
    }
    char message[ChatLineMax];
    VFormat(message, sizeof(message), format, 2);
    PrintToChatAll("%s\x07AAAAAA%s", AP_TAG, message);
    PrintToServer("[AP] debug: %s", message);
    LogMessage("%s", message);
}

// Answers the console as readily as a player. Client 0 is the server console,
// which is where an rcon command runs, and a player can leave while a request
// is in flight; PrintToChat throws on either and takes the whole callback with
// it.
stock void AP_PrintToClient(int client, const char[] format, any ...)
{
    char message[ChatLineMax];
    VFormat(message, sizeof(message), format, 3);
    if (client > 0 && client <= MaxClients && IsClientInGame(client))
    {
        PrintToChat(client, "%s%s", AP_TAG, message);
        return;
    }
    PrintToServer("[AP] %s", message);
}

/**
 * Saying things.
 *
 * There is no client mod, so chat, the server console and the SourceMod log
 * are the whole diagnostic surface (ADR 0002). This plugin will be debugged on
 * a live server by reading chat, so errors go there by default: a failure
 * nobody can see is a failure that gets blamed on the game.
 *
 * Three levels, and the difference matters:
 *   AP_Announce  what a player wants to know. Chat, unless muted.
 *   AP_Error     what has gone wrong. Chat and the log, always.
 *   AP_Debug     what an operator wants while hunting a bug. Chat, console and
 *                the log, only when tf2ap_debug is on.
 */

#define AP_TAG "\x07FFD700[AP]\x01 "
#define AP_TAG_ERROR "\x07FF4040[AP]\x01 "

ConVar g_CvarAnnounce;
ConVar g_CvarDebug;

void Log_Init()
{
    g_CvarAnnounce = CreateConVar("tf2ap_announce", "1",
        "Announce received items and unlocks in chat.", _, true, 0.0, true, 1.0);
    g_CvarDebug = CreateConVar("tf2ap_debug", "0",
        "Echo every bridge call and game event to chat and the console.",
        _, true, 0.0, true, 1.0);
}

stock void AP_Announce(const char[] format, any ...)
{
    char message[254];
    VFormat(message, sizeof(message), format, 2);
    if (g_CvarAnnounce.BoolValue)
    {
        PrintToChatAll("%s%s", AP_TAG, message);
    }
    PrintToServer("[AP] %s", message);
}

/**
 * Errors reach chat whether or not announcements are muted. If the bridge is
 * unreachable the players are the ones who will notice their checks are not
 * landing, and they should be told why rather than left guessing.
 */
stock void AP_Error(const char[] format, any ...)
{
    char message[254];
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
    char message[254];
    VFormat(message, sizeof(message), format, 2);
    PrintToChatAll("%s\x07AAAAAA%s", AP_TAG, message);
    PrintToServer("[AP] debug: %s", message);
    LogMessage("%s", message);
}

stock void AP_PrintToClient(int client, const char[] format, any ...)
{
    char message[254];
    VFormat(message, sizeof(message), format, 3);
    PrintToChat(client, "%s%s", AP_TAG, message);
}

#if defined _tf2_archipelago_downloads_included
    #endinput
#endif
#define _tf2_archipelago_downloads_included

void Downloads_OnMapStart()
{
    char map[PLATFORM_MAX_PATH];
    char manifest[PLATFORM_MAX_PATH];
    GetCurrentMap(map, sizeof(map));
    BuildPath(Path_SM, manifest, sizeof(manifest),
        "data/tf2_archipelago/downloads/%s.txt", map);

    File file = OpenFile(manifest, "r");
    if (file == null)
    {
        return;
    }

    char path[PLATFORM_MAX_PATH];
    int added;
    while (file.ReadLine(path, sizeof(path)))
    {
        TrimString(path);
        if (path[0] == '\0' || !FileExists(path, true))
        {
            continue;
        }
        AddFileToDownloadsTable(path);
        added++;
    }
    delete file;
    AP_Debug("Advertised %d community HUD asset(s) for %s.", added, map);
}

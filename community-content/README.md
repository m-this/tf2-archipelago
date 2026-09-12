# Community maps and missions

## The downloaded Potato archives and `tf2ap.exe`

This build directly recognizes these official full Potato packs by filename:

```text
archive-assets.zip
mlarchive-assets.zip
```

In `tf2ap.exe`, open **Settings → Missions**, choose the folder that will hold
the ZIPs, and tick **Potato Archive** and/or **Moonlight Archive**. Press
**Download Selected Community Assets** to fetch only the checked packs. This
button is the only operation that downloads community content. **Start** never
downloads a community file: it validates and installs selected local ZIPs, and
reports a missing or invalid ZIP instead. Existing valid downloads are reused.
A cancelled download leaves only a disposable `.partial` file.

Community mission and start-mission rows are not populated merely because a
pack checkbox is selected. They appear only after the matching ZIP exists and
validates. Maps without a bot `.nav` then appear in red as unavailable and
cannot be added to a seed.

### Use asset ZIPs you already have

Keep the original full-with-maps ZIP intact; do not extract or rename it. Put
either or both supported ZIPs directly in the folder selected in the launcher:

```text
archive-assets.zip
mlarchive-assets.zip
```

Then press **Use Local Community Assets**, choose that folder, and let the
launcher validate the files. It checks the matching pack boxes and refreshes
the mission list without contacting the network. On the terminal interface,
type the folder into **Asset pack folder** first and activate the same action.

The ZIP may have Potato's normal `tf/download/...` root or a direct `tf/...`
root. The launcher reads it in place and installs its contents beneath the
dedicated server's `tf/` directory; it does not move, rewrite, or delete the
source archive.

The recognized downloads are:

- `https://dlarchive.potato.tf/archive-assets.zip`
- `https://dlml.potato.tf/mlarchive-assets.zip`

The `-no-maps.zip` alternatives are deliberately not used: they omit the
BSP/NAV files required by this catalog.

The mission table has explicit **Source** and **Mods** columns (`Valve`,
`Potato Archive`, or `Moonlight Archive` for the source), and every
start-mission choice has the same source prefix. The two archives contain 201
missions whose map BSP is also present: 86 run on a stock server, 99 declare
SigMod (`sigsegv-mvm`), and 16 are listed but locked because the archive has no
bot navigation mesh for their map. This is 116 more missions and 25 more maps
than the original curated catalog. Another 363 historical population files in
the archives are not registered because neither archive contains their map;
installing a population file without its BSP cannot make that mission playable.

The catalog enables missions that use stock server features and have the
required map, population, and navigation files. Entries missing bot navigation
remain visible as unavailable and cannot be drawn by a seed. The launcher
strips the archive's `tf/download/` prefix so assets arrive under the correct
TF2 game directory on Windows and Linux.

On Linux, turn on **SigMod** and press **Download / set up selected server
mods**. The launcher downloads the release pinned in `deploy/env/versions.env`,
verifies its package checksum, installs it with SourceMod, and records hashes
for its critical installed files. **Start** performs the same setup
automatically. A SigMod mission cannot be ticked or chosen as the start until
that inspection passes; its Compatibility cell says whether to turn SigMod on
or run setup. SigMod has no supported Windows server build, so those missions
remain visible and locked in the Windows launcher.

## Build a new Windows launcher

Run the build from WSL/Linux at the repository root. Building the standalone
Windows launcher does not require Docker:

```sh
make community-check COMMUNITY_CONTENT="archive-assets.zip mlarchive-assets.zip"
make export
make plugin
make launcher
```

The usable launcher is `dist/tf2ap.exe`. Do not substitute a plain
`go build`: the release target embeds the compiled SourceMod plugin, apworld,
ripext, and defender bots and injects their pinned versions.

Build the native WSL/Linux launcher from the same catalog with:

```sh
make launcher-linux
```

That produces `dist/tf2ap-linux-amd64`. Both binaries recognize the same ZIP
names and show the same 201 community missions with their compatibility and
required server mod.

## Start a server with a Potato map

1. Run `dist\tf2ap.exe`.
2. Open **Settings → Missions**.
3. Choose the folder for the supported ZIPs and tick both archives for the full
    set. Press **Download Selected Community Assets**, or press **Use Local
    Community Assets** if the ZIPs are already in the selected folder.
4. Choose any available community mission as the start mission.
5. Tick the community missions wanted in the pool and untick unwanted Valve
   missions. Save.
6. For local play, enable **Test mode**; otherwise press **Generate seed**,
   upload the generated archive to Archipelago, and enter the new room.
7. Press **Start**. The first run installs TF2 and extracts the selected local
   asset pack; it performs no community download.
8. In the server console or RCON, run `sm_ap_status`. In game, `!mission`
   lists the run and `!mission <number>` switches missions once their tickets
   are unlocked.

Generate a new seed after changing the registered mission pool; an existing
room cannot acquire the new stable mission IDs.

This directory is a content-pack overlay. Put the pack's normal `tf/` tree
under `community-content/tf/`; do not flatten it:

```text
community-content/
└── tf/
    ├── maps/mvm_underground_rc3.bsp
    ├── maps/mvm_underground_rc3.nav
    └── scripts/population/mvm_underground_rc3_welcometomymine.pop
```

The server copies this tree into TF2's game directory on startup. Joining
clients receive the active map through Source's direct downloader; the managed
server configuration raises its stock 16 MB limit to the 64 MB engine cap.
Population files and `.nav` files are server-side.

Prefer a map whose client assets are packed into its BSP. If a content pack
ships loose materials, models, particles, or sounds, it also needs its own
download manifest/SourceMod downloader; merely putting loose client assets in
this directory does not make TF2 send them.

## Register the content

Run the archive importer when updating the official packs:

```sh
make community-catalog
```

It adds missions only when one of the supplied archives also contains their
map, detects gameplay-relevant SigMod syntax, and preserves every existing ID
and hand-written name. Review the result in `gamedata/community.json`. IDs are
permanent Archipelago identities: start at 100, never reuse an ID, and keep the
manifest with every server and apworld that can load the resulting seed.

```json
{
  "format_version": 1,
  "maps": [
    {"id": 101, "name": "mvm_underground_rc3"}
  ],
  "missions": [
    {
      "id": 101,
      "pop_file": "mvm_underground_rc3_welcometomymine",
      "name": "WelcomeToMyMine",
      "map_id": 101,
      "difficulty": "intermediate",
      "waves": 12,
      "has_tank": false,
      "has_giant": true
    }
  ]
}
```

The example uses a real Potato community mission: the TF2 Wiki identifies
[WelcomeToMyMine](https://wiki.teamfortress.com/wiki/WelcomeToMyMine_%28custom_mission%29)
as a 12-wave intermediate mission on Underground and gives its exact popfile
name. Replace every value with the metadata from the pack you actually install;
`examples/community.json` is documentation, not live content.

Validate and regenerate the apworld after every manifest edit:

```sh
make community-check
make export
```

`community-check` fails early if a registered BSP, navigation mesh, or
population file is absent. `make export` bakes the new stable IDs into the apworld. All
registered community missions then participate in the existing difficulty,
mission-count, start-mission, and exclusion options alongside Valve missions.
Use `MVM_EXCLUDED_MISSIONS` if a particular custom mission should not be drawn.

In the standalone launcher, press **Check Run Selection** after choosing the
pool. It reports the eligible missions, their cataloged checks, and the unlocks
that must fit. The same check runs automatically before saving, writing
`tf2.yaml`, generating a seed, and starting Test mode. This guarantees the
catalog is logically large enough; runtime certification of a community
mission's declared waves, tank, and giant remains a separate compatibility
level.

## Rebuild and relaunch

Community content changes require an SRCDS restart. Manifest changes also
change the apworld and therefore require a newly generated seed:

```sh
make down
make community-check
make export
make seed
make build
make up
make logs
```

Upload the new file from `seed/` to Archipelago and put that room's host/port
in `.env` before `make up`. Do not reuse an old room after changing the
manifest: its seed does not know the new mission IDs. In `TF2AP_TEST_MODE=1`,
there is no seed or room, so `make seed` can be skipped.

For a content-file-only update that leaves `gamedata/community.json` unchanged,
restart just the game service so the overlay is recopied:

```sh
docker compose --project-directory . \
  --env-file deploy/env/versions.env --env-file .env \
  -f deploy/compose.yml restart srcds
```

## What Potato-style content works

- Vanilla `.bsp`, `.nav`, `.pop`, packed VScript, and normal TF2 assets work
  with the layout above. Selectable missions use Valve's upgrade table.
- A mission that needs a server mod names it in `requires`, with a key from
  `gamedata/server_mods.go`. Today that is `sigsegv-mvm` (SigMod). The
  Docker image stages the pinned release from `deploy/env/versions.env` and
  installs it when `SRCDS_MODS` names it; `make seed` passes the same list as
  the apworld's `server_mods` option. The native Linux launcher downloads and
  verifies the same pin when SigMod is selected. Both paths let the seed draw
  those missions only for a server configured to load the mod. The Windows
  launcher lists them locked because upstream ships no Windows build.
- A mission that needs a mod the catalog does not know is not accepted by
  `community.json`. Add the mod to the catalog and pin it first.

# launcher

Go. The all-in-one Windows exe. It embeds the compiled plugin, the ripext
Windows build and the MvM defender bots. It installs SteamCMD, the TF2
dedicated server, Metamod:Source and SourceMod. It then runs the bridge
in-process next to the `srcds.exe` subprocess.

One exe, no Docker, no clone. The primary way to run a Mann vs Archipelago
server on Windows.

## Build

```sh
make launcher
```

Cross-compiles `tf2ap.exe` into `dist/`. The target stages the bots, fetches
the ripext Windows zip into the embed dir, copies the compiled plugin, and
injects the pinned versions from `deploy/env/versions.env` with `-ldflags`.

The Missions tab can download missing official Potato/Moonlight full asset
ZIPs into a player-selected cache folder and install them into SRCDS. It labels Valve and Potato missions
separately in both the pool table and start-mission menu. See the
[community-content guide](../community-content/README.md) for the verified
archive layout, custom-upgrade findings, build commands, and RafMod boundary.
The [archive snapshot hashes](../docs/en/community-archives.md) document the
exact ZIPs the launcher accepts and how to review a mismatch.

## Layout

| Package | Holds |
| --- | --- |
| `cmd/tf2ap` | Entrypoint: flags, the guided flow, subcommands |
| `internal/assets` | The embedded files and the injected version strings |
| `internal/settings` | The saved configuration, the environment overlay, the player YAML |
| `internal/installer` | SteamCMD, TF2 server, Metamod, SourceMod, ripext, plugin, bots |
| `internal/srcdsconfig` | Renders `server.cfg`, `admins_simple.ini`, `tf2_archipelago.cfg` |
| `internal/runtime` | The `srcds.exe` subprocess and the in-process bridge, interleaved |
| `internal/webapi` | The launcher as the browser talks to it: state, Connect services, `/ws` |
| `internal/spa` | The Angular build, embedded and served |
| `internal/browser` | Opening the player's browser, per desktop |
| `internal/tray` | The icon in the Windows notification area: open the page, Quit |
| `web/` | The Angular app. Not part of the Go module: see `web/go.mod` |
| `internal/generate` | Drives the Archipelago app's generator: installs the apworld, writes the player file, runs it |
| `internal/debugbundle` | The zip a play-tester sends: logs, settings without passwords, player file |
| `../fakeroom` | The multiworld of one that test mode serves, shared with the bridge |
| `internal/rcon` | Source RCON client, shared by the command box and `cmd/rcon`, which `make rcon` runs |
| `internal/runshape` | The run's choices, counted from `gamedata` |
| `internal/ui` | Console prompts, for `-configure` and the first-run question |

## Configuration

The no-args run asks one question, the room address, and only on a first run.
Everything else carries a default, and `settings.NewRconPassword` invents the
one secret rather than asking for it. `-configure` opens the full list.

Three layers, in order. The defaults, then `%APPDATA%\tf2ap\config.json`, then
the environment. `settings.ApplyEnv` reads the names `deploy/.env.example`
already uses, so a compose operator's file works here unchanged. An environment
value is never written back: an override for one run must not become the saved
answer.

Each variable is one line of `settings.EnvVars`: its field, how it is spelled,
and the Compose services that read it. `ApplyEnv`, the `.env` the Compose admin
page writes, and the tests against `compose.yml` and `.env.example` all come
from that line.

## One face, and it is a browser

`tf2ap` with no arguments serves the interface on 127.0.0.1 on a port the
operating system picks, prints the address, and opens a browser on it. That is
the same program and the same screen on Windows and on Linux.

Loopback only, one player, no authentication and no CSRF token: the four
passwords never cross a network. What the mux does check is that the request is
for the address the listener bound. Comparing `Origin` with `Host` is not enough
on its own, because a DNS-rebinding attacker controls both, so `Host` is pinned
to the authority the listener actually took.

| Flag | What it does |
| --- | --- |
| *(none)* | Serve on a free loopback port, open a browser |
| `-addr host:port` | Serve here instead |
| `-no-browser` | Print the address, open nothing |
| `-console` | The log and nothing over it, for Docker and for a machine with no desktop |
| `-configure` | The console prompts, then exit |

Closing the browser tab leaves the server running. On Windows the launcher is
an icon in the notification area while it runs: a click opens the page again,
and the right-click menu has Quit. Quit in the interface stops it too, and so
does Ctrl-C in a console.

Opening the browser is `internal/browser`, and WSL is the awkward one:
`xdg-open` there either does nothing or opens a browser inside the distribution
that the player cannot see, so the Windows browser is asked instead through
`wslview` or `cmd.exe /c start`. Windows reaches the listener on 127.0.0.1
through WSL's own localhost forwarding, so the address needs no rewriting. A
desktop with no opener is normal over SSH and on a minimal install, so a failure
prints the address and carries on rather than stopping the launcher.

## The contract

`proto/tf2ap/launcher/v1` is what the two halves agree on. `form.proto` mirrors
`form.Model` field for field, so the Angular renderer never needs a second
source for what a row is: adding a setting is still one `Spec`.

Neither generated tree is committed. `make proto` writes Go into
`launcher/internal/gen` and TypeScript into `launcher/web/src/gen`, and every Go
target depends on it, so `make test` and `make lint` generate first.
`TestProtoKindsMatchFormKinds` fails in both directions when a `Kind` is added
to one side only.

Requests go over Connect. The live state is a WebSocket at `/ws`, because only
one side ever speaks: the launcher pushes and the browser listens. The first
frame is the whole state with its log; after it a line arrives on its own and
the whole state arrives without the log whenever anything else moved. Bounded
on both sides: twenty thousand lines kept, a hundred and twenty-eight events
queued, and a browser that falls further behind is sent the whole state again
rather than left drawing one that has moved on.

A tab left open costs nothing. Hidden, the socket closes and the launcher drops
the listener; visible again, it opens and its first frame is the whole state.

## Running the interface without a game server

`launcher/cmd/fakelauncher` answers the same contract with nothing behind it:
the real handlers, the real WebSocket and a real `form.Model`, with no game
server, no bridge, no Steam install and no multiworld. It is what the browser
tests drive, and it is useful by hand:

```sh
make web-build
go run ./launcher/cmd/fakelauncher -addr 127.0.0.1:8471
```

It is never shipped: no release target builds it and the launcher does not
import it.

```sh
make web-check   # eslint, prettier, the component tests, the 500 kB budget
make web-e2e     # Playwright against the fake launcher
```

## How it fits

The launcher imports `bridge/` and runs it in-process. The `srcds.exe`
subprocess shares the machine's loopback with the bridge, so the plugin reaches
it at `127.0.0.1:24680` exactly as it does under `network_mode: service:srcds`
in compose.

Seed generation stays with the Archipelago app: the generator is Python, and a
bundled Python breaks the one-exe promise. What the launcher does is drive that
app. Generate seed finds it where the installer put it and installs the
embedded apworld into `custom_worlds`. It writes the player file into a folder
of its own, runs `ArchipelagoGenerate.exe`, and opens the folder the archive
landed in. The player uploads that archive to archipelago.gg.

## The embeds

`internal/assets/embedded/` holds:

- `tf2_archipelago.smx` (gitignored, copied by `make launcher-assets`)
- `sm-ripext-windows.zip` (gitignored, fetched by `make launcher-assets`)
- `defender-bots-windows.zip` (gitignored, built by `make bots`, `.so` stripped)
- `tf2_mvm.apworld` (gitignored, built without Docker by `make apworld-package`)
- `tf2_archipelago.cfg` (committed)

`server.cfg.tmpl` lives in `internal/provision` instead. That package decides
server.cfg, the srcds command line and the mods loaded for both the launcher
and the Docker image, which calls it as `tf2ap-provision`.

The first three are build artefacts, and a hand `go build` needs placeholders
in their place. Such a build also leaves the version strings empty, so
`assets.RequireVersions()` stops the installer rather than let it guess a
version.

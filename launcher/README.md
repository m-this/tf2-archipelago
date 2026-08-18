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

## Layout

| Package | Holds |
| --- | --- |
| `cmd/tf2ap` | Entrypoint: flags, the guided flow, subcommands |
| `internal/assets` | The embedded files and the injected version strings |
| `internal/settings` | The saved configuration, the environment overlay, the player YAML |
| `internal/installer` | SteamCMD, TF2 server, Metamod, SourceMod, ripext, plugin, bots |
| `internal/srcdsconfig` | Renders `server.cfg`, `admins_simple.ini`, `tf2_archipelago.cfg` |
| `internal/runtime` | The `srcds.exe` subprocess and the in-process bridge, interleaved |
| `internal/ui` | Interactive prompts with saved-config defaults |

## Configuration

Three layers, in order. The defaults, then `%APPDATA%\tf2ap\config.json`, then
the environment. `settings.ApplyEnv` reads the names `deploy/.env.example`
already uses, so a compose operator's file works here unchanged. An environment
value is never written back: an override for one run must not become the saved
answer.

## How it fits

The launcher imports `bridge/` and runs it in-process. The `srcds.exe`
subprocess shares the machine's loopback with the bridge, so the plugin reaches
it at `127.0.0.1:24680` exactly as it does under `network_mode: service:srcds`
in compose.

Seed generation stays out. The Archipelago generator is Python, and a bundled
Python breaks the one-exe promise. The launcher writes the player YAML from the
saved run shape: on demand with `-yaml`, and once into the install root on the
first start. The player drops that file into the Archipelago app and generates
there.

## The embeds

`internal/assets/embedded/` holds:

- `tf2_archipelago.smx` (gitignored, copied by `make launcher-assets`)
- `sm-ripext-windows.zip` (gitignored, fetched by `make launcher-assets`)
- `defender-bots-windows.zip` (gitignored, built by `make bots`, `.so` stripped)
- `tf2_archipelago.cfg` (committed)
- `server.cfg.tmpl` (committed)

The first three are build artefacts, and a hand `go build` needs placeholders
in their place. Such a build also leaves the version strings empty, so
`assets.RequireVersions()` stops the installer rather than let it guess a
version.

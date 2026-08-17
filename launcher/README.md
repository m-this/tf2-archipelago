# launcher

Go. The all-in-one Windows exe. Embeds the compiled plugin and the ripext
Windows build, installs SteamCMD, the TF2 dedicated server and SourceMod, then
runs the bridge in-process alongside the `srcds.exe` subprocess.

One exe, no Docker, no clone. The primary way to run a Mann vs Archipelago
server on Windows.

## Build

```sh
make launcher
```

Cross-compiles `tf2ap.exe` into `dist/`. The target builds the plugin first (on
Linux; on macOS it uses a placeholder), fetches the ripext Windows zip into the
embed dir, and injects the pinned versions from `deploy/env/versions.env` via
`-ldflags`.

## Layout

| Package | Holds |
| --- | --- |
| `cmd/tf2ap` | Entrypoint: flags, the guided flow, subcommands |
| `internal/assets` | Embedded files: the plugin, ripext, config templates, version strings |
| `internal/settings` | The saved configuration, in `%APPDATA%\tf2ap\config.json` |
| `internal/installer` | SteamCMD, TF2 server, SourceMod, ripext and plugin install |
| `internal/srcdsconfig` | Renders `server.cfg`, `admins_simple.ini`, `tf2_archipelago.cfg` |
| `internal/runtime` | The `srcds.exe` subprocess and the in-process bridge, interleaved |
| `internal/ui` | Interactive prompts with saved-config defaults |

## How it fits

The launcher imports the bridge package (`bridge/`) and runs it in-process. The
`srcds.exe` subprocess shares the Windows machine's loopback with the bridge, so
the plugin reaches it at `127.0.0.1:24680` exactly as it does inside the
`network_mode: service:srcds` compose arrangement.

Seed generation is out of scope. The Archipelago generator is Python, and
bundling Python would break the "one easy exe" promise. The user installs the
official Archipelago app once, drops `tf2_mvm.apworld` into its
`custom_worlds/`, and generates there. The launcher only owns the TF2 server
side.

## The embeds

`internal/assets/embedded/` holds:

- `tf2_archipelago.smx` (gitignored, copied by `make launcher-assets`)
- `sm-ripext-windows.zip` (gitignored, fetched by `make launcher-assets`)
- `tf2_archipelago.cfg` (committed)
- `server.cfg.tmpl` (committed)

The first two are build artefacts: the Makefile populates them before `go
build`, and `.gitignore` keeps them out of the tree. A hand `go build` works
with placeholders, but `assets.RequireVersions()` stops the installer when the
version strings are empty (they are injected via `-ldflags` from
`deploy/env/versions.env`).

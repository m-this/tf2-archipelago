Every release attaches the same files. `gh release create --generate-notes`
appends the commit list below this.

## Run the stack without a clone

Download `compose.yaml` and `.env.example` into an empty directory:

```sh
mkdir mann-vs-archipelago && cd mann-vs-archipelago
base=https://github.com/m-this/tf2-archipelago/releases/latest/download
curl -fsSLO "$base/compose.yaml"
curl -fsSL -o .env "$base/.env.example"
```

Set `SRCDS_RCONPW` in `.env`, then make a session and start:

```sh
docker compose --profile seed run --rm seed   # writes ./seed
docker compose up -d
```

Upload the file from `seed/` at [archipelago.gg/uploads](https://archipelago.gg/uploads),
create a room, and write its port into `AP_PORT`.

The images come from `ghcr.io/m-this/tf2-archipelago`. `compose.yaml` pins them
to this release; set `TF2AP_VERSION` in `.env` to run another one.

To host the session on this machine instead of on archipelago.gg, set
`COMPOSE_PROFILES=selfhost`, `AP_HOST=archipelago`, `AP_PORT=38281` and
`AP_TLS=false`. Then `docker compose up -d` needs no seed and no room.

## The other files

| File | What it is |
| --- | --- |
| `tf2_mvm.apworld` | The Archipelago world. Drop it in `custom_worlds/` of an Archipelago 0.6.7 install to generate a seed without Docker. |
| `tf2_archipelago.smx` | The compiled SourceMod plugin, for an existing srcds that the compose file does not run. |
| `meta.json`, `items.json`, `missions.json` | The item and location tables `gamedata/` exports. They are already inside the apworld; they are here to read. |

[The book](https://m-this.github.io/tf2-archipelago/) covers the rest.

<p align="center">
  <a href="https://github.com/m-this/tf2-archipelago/releases/latest/download/tf2ap.exe">
    <img alt="Download tf2ap.exe for Windows" src="https://img.shields.io/badge/Download-tf2ap.exe%20for%20Windows-2ea44f?style=for-the-badge&logo=windows&logoColor=white">
  </a>
</p>

## Windows

Download `tf2ap.exe` above and run it. One file: no Docker, no clone, no
compiler. It opens a window, asks for the address of your Archipelago room,
and installs everything the server needs. The bots that fill your team are in
it too.

The
[Windows guide](https://m-this.github.io/tf2-archipelago/en/setup/install-windows.html)
takes you from the download to the first wave.

## Docker

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

Upload the file from `seed/` at
[archipelago.gg/uploads](https://archipelago.gg/uploads), open a room, and
write its port into `AP_PORT`.

The images come from `ghcr.io/m-this/tf2-archipelago`. `compose.yaml` pins
them to this release. Set `TF2AP_VERSION` in `.env` to run another one.

To host the session on this machine, set `COMPOSE_PROFILES=selfhost`,
`AP_HOST=archipelago`, `AP_PORT=38281` and `AP_TLS=false`. Then
`docker compose up -d` needs no seed and no room.

## The other files

| File | What it is |
| --- | --- |
| `tf2_mvm.apworld` | The Archipelago world, for the official app. The launcher installs it for you. |
| `tf2_archipelago.smx` | The compiled plugin, for a server that neither the launcher nor the compose file runs. |
| `tf2-defender-bots.zip` | The bots that fill the RED team, for that same server. Unzip it into `tf/`. |
| `meta.json`, `items.json`, `missions.json` | The item and location tables. They are inside the apworld; they are here to read. |

[The book](https://m-this.github.io/tf2-archipelago/) covers the rest.

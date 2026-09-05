# deploy

Compose stack. Two services by default, and two more that run on demand.

| File | Service | Notes |
| --- | --- | --- |
| `compose.yml` | `srcds` | TF2 dedicated server, SourceMod plus `ripext` plus our plugin. |
| `compose.yml` | `bridge` | Go, from `bridge/`. |
| `compose.yml` | `archipelago` | The Archipelago server, unmodified, with our apworld baked in. Profile `selfhost` only. |
| `compose.yml` | `fastdl` | Caddy serving only downloadable TF2 asset directories. |
| `compose.yml` | `tailscale-fastdl` | Official Tailscale Funnel sidecar. Profile `tailscale-fastdl` only. |
| `compose.seed.yml` | `seed` | The same image, run once to generate a seed into `./seed`. |
| `compose.release.yml` | — | An overlay, not a stack. Names a `ghcr.io` image for each service above. |

The stack plays a room that already exists, and needs only its address and a
slot. The multiworld runs on archipelago.gg by default: `make seed` writes the file,
the operator uploads it there and opens a room, and the bridge dials that room.
`COMPOSE_PROFILES=selfhost` in `.env` hosts it here instead.

Generation lives in a compose file of its own because it comes first. The stack
refuses to load without the address of a room, and there is no room before a
seed exists.

## The released compose file

A release attaches one flat `compose.yaml` for an operator with no clone.
`make compose-release` renders it. `docker compose config` merges the three
files above, and `--no-interpolate` keeps every `${VAR}` for the operator's
`.env`. An awk drops the `build:` blocks, which point at a repository that is
not there. Generation comes back as the `seed` profile, since a release ships
one file rather than two.

Each service and each environment variable lives in one file. The Caddy and
Tailscale configurations are inline so this release artifact needs no sibling
configuration files. Add a service to
`compose.yml` and it reaches the released file at the next tag;
`compose.release.yml` only has to name its image.

## The port exception

House rule is that every `ports:` entry binds to `127.0.0.1`, because dev
happens on a remote box reached over an SSH forward. This stack breaks that
rule exactly once:

- `srcds` needs **27015/udp** reachable from the internet, or nobody can join.
  There is no way around it. A game client connects directly.
- **Everything else stays on loopback.** The bridge's HTTP API, the Archipelago
  server's web UI and its game port, and RCON above all. RCON is never exposed,
  not even on loopback outside the compose network.

Consequence worth being explicit about: this puts inbound game traffic on
whatever host runs it. `srcds` is a large C++ process parsing untrusted input
from anyone who knows the address. Run it unprivileged, in its own compose
network, with no volume it does not need. If the target host also runs
something that matters, that is a decision to make deliberately rather than
by default.

## Startup order

The bridge depends on the Archipelago server, and the plugin depends on the
bridge. But none of them may hard-fail on the others being absent:

- The bridge starts, queues, and connects when the AP server appears.
- The plugin starts, and tolerates the bridge being down (ADR 0002 requires
  this anyway, since the bridge can be restarted mid-session).

There is no `depends_on` on the Archipelago server at all, because usually it
is not a service in this file. That costs nothing: boot order was never what
made this correct. Anything that only works because the containers happened to
start in the right order is a bug.

## What is mounted where

- The built `.apworld` into the Archipelago image's custom worlds directory, at
  build time rather than as a mount.
- `./seed` on the host into `/ap/output` of the `seed` service. The generated
  archives have to leave the container: they go to archipelago.gg.
- The compiled `.smx` plugin into the `srcds` container's SourceMod plugins
  directory.
- The bridge's durable state onto a named volume. This holds the check queue
  and the unlock set, and losing it loses the run.

## Ansible

Per the house deploy convention, a `deploy/ansible/` recipe goes here once the
stack actually runs. Not before: an Ansible role for a compose file that has
never come up is fiction.

# deploy

Compose stack. Three services.

Nothing exists yet.

| Service | Image | Notes |
| --- | --- | --- |
| `archipelago` | upstream, pinned | The Archipelago server, unmodified. Our apworld is mounted in, not baked in. |
| `srcds` | TF2 dedicated server | SourceMod plus `ripext` plus our plugin. Candidate base: `cm2network/tf2`, to verify. |
| `bridge` | built here | Go, from `bridge/`. |

Neither the Archipelago image nor the TF2 image is chosen yet. Both need
verifying against the same criteria: pinned tag, no `latest`, non-root, and
sane behaviour on restart.

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

So `depends_on` is a convenience for boot ordering, not a correctness
mechanism. Anything that only works because the containers happened to start in
the right order is a bug.

## What is mounted where

- The built `.apworld` (or `apworld/tf2_mvm/` directly) into the Archipelago
  container's custom worlds directory.
- The compiled `.smx` plugin into the `srcds` container's SourceMod plugins
  directory.
- The bridge's durable state onto a named volume. This holds the check queue
  and the unlock set, and losing it loses the run.

## Ansible

Per the house deploy convention, a `deploy/ansible/` recipe goes here once the
stack actually runs. Not before: an Ansible role for a compose file that has
never come up is fiction.

# TODO

Ordered by what unblocks what. Nothing here is started.

## Blocking unknowns

These gate design, not implementation. Answer them before writing the tables,
because each one can change what an item or a location is.

- [ ] **Shop check injection.** Can a SourceMod plugin add an arbitrary
      purchasable entry to the MvM upgrade station UI, or only hook purchases
      of existing upgrades? Blocks the whole `shop_checks` location group.
      Needs a live server and an afternoon.
- [ ] **Allied bot upgrade sharing.** Do RED `tf_bot`s still work in MvM, and
      can they inherit the player's purchased upgrades? Blocks `Allied Mercs`
      and `Merc Loadouts`. The thread raised it and nobody knew.
- [ ] **Wave counts per mission.** Parse the `.pop` files in `gamedata/` at
      build time, or hardcode the Valve missions and refuse community ones in
      v1? Blocks the location count, which blocks everything.
- [ ] **Community mission identity.** Potato.tf and Moonlight.tf missions have
      no stable global id. Pick a naming scheme that survives a rename or a
      pack update, probably `map_name/pop_file_basename`. Blocks id assignment,
      which is append-only and therefore unfixable later.
- [ ] **DeathLink semantics.** Individual death is noise in MvM. Wave failure,
      or death outside the respawn grace window? Blocks nothing, but decide
      before the option ships.

## Data

- [ ] Port the tables from ALPHAMARIOX's `worlds/tf2/Items.py` to Go structs.
      556 lines, 14 dicts. See `docs/prior-art.md`.
- [ ] Port the `Group(IntFlag)` enum to a Go bitmask with the same member
      names.
- [ ] Pick the game name string, once. The fork disagrees with itself.
- [ ] Id assignment plus the stability test (uniqueness, and no committed id
      ever changes value).
- [ ] The JSON exporter, and the CI check that the committed export matches.

## Bridge

- [ ] Archipelago websocket client. `Connect`, `LocationChecks`,
      `ReceivedItems`, `StatusUpdate`, `Bounced`, `Say`. Both `ws://` and
      `wss://`.
- [ ] Reconnect with backoff, and replay of the queue on reconnect.
- [ ] Durable check queue on disk. Write, then 200, then send.
- [ ] Received-item dedup. AP replays the full list on every reconnect.
- [ ] Unlock-set persistence, and the resync endpoint the plugin calls after a
      reload.
- [ ] Long-poll endpoint for grants.

## apworld

- [ ] `Options.py`, adapted by hand from the fork. Add Roseburst's missing
      options: Mission Order, Goal, Tour Size, Allied Mercs, Merc Loadouts,
      Giants and Bosses, and the whole Check Options block.
- [ ] `__init__.py`: read the export, build items and locations, build the
      region graph.
- [ ] Access rules. The sphere 0 guarantee first: at least one class with at
      least one usable weapon, or wave 1 is unwinnable.
- [ ] The three goals: Final Boss, Missionsanity, Australium Hunt.
- [ ] Setup guide and game info page.

## Plugin

- [ ] Objective detection: wave, mission, tank, giant, money bonus.
- [ ] Grant application: weapon slots, weapons, classes, upgrade gating,
      canteens.
- [ ] Resync from the bridge on load and on map change.
- [ ] Allied bot spawning and template assignment.
- [ ] Traps.
- [ ] Player-facing output through chat, HUD text and annotations.

## Deploy

- [ ] Pick and pin the Archipelago server image.
- [ ] Pick and pin the TF2 dedicated server image. Verify `cm2network/tf2`.
- [ ] Compose file. 27015/udp public, everything else loopback, RCON never.
- [ ] Bridge Dockerfile.
- [ ] `deploy/ansible/`, once the stack actually comes up.

## Community

- [ ] Ask in the Discord thread for a link to Snolid Ice's MvM Manual. It was
      mentioned twice and never linked, and it likely has usable item and
      location naming.
- [ ] Tell Damonj17 and Roseburst this exists. The design is theirs.

## Repo

- [ ] `.forgejo/workflows/ci.yml`, once there is code to check. Mirror the
      `simple-webapp` kit gate: gofumpt, vet, golangci-lint, go fix, build,
      race tests, govulncheck, plus the export freshness check.
- [ ] Makefile, same time.
- [ ] Decide whether this repo goes public. It is private today, and the design
      came from a public Discord thread.

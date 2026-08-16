# TODO

`docs/spec.md` says what we are building and why. This file says what to build,
in what order, with the exact contracts. Where a fact here was verified against
a real artefact, it says so; where it is still a guess, it is marked
**UNVERIFIED**.

Goal: `docker compose up` gives a playable MvM Archipelago server, configured
by one `.env`.

## Verified facts

Checked 2026-08-13 on moon18.

| Thing | Value | How it was checked |
| --- | --- | --- |
| Archipelago | `0.6.7`, released 2026-04-01 | GitHub releases API |
| Python for AP | `>=3.11.9, <3.14` | `docs/running from source.md` |
| apworld drop dir | `custom_worlds/` when running from source | `worlds/__init__.py:20` |
| Generate | `python Generate.py`, reads YAMLs from `Players/`, writes a `.zip` to `output/` | AP docs |
| Host | `python MultiServer.py <archive.zip>` | AP docs |
| srcds image | `cm2network/tf2:sourcemod`, 326 MB, runs as `steam` | `docker image inspect` |
| SourceMod / Metamod | 1.12 / 1.12, preinstalled in that tag | image env |
| Game download | ~14 GB, done, in volume `tf2-archipelago_tf2game` | `du -sh` |
| ripext | `1.3.2`, released 2025-07-20, `sm-ripext-1.3.2-linux.zip` | GitHub releases API |
| MvM maps on disk | 7 `.bsp` | `ls tf/maps` |
| Pop files | inside `tf2_misc_dir.vpk`, not loose on disk | `find` returned 0, `strings` found them |

Checked 2026-08-16 against an AP 0.6.7 checkout, while building milestone 2.

| Thing | Value | How it was checked |
| --- | --- | --- |
| `custom_worlds/` | takes `.apworld` **zips only**. A plain folder there is found and then fails to import: only zips get a meta-path finder (`worlds/__init__.py:182`). A folder has to sit in `worlds/`. | tried both |
| Packaging | `python Launcher.py "Build APWorlds" -- "<game>"` writes `build/apworlds/tf2_mvm.apworld` and stamps the manifest | ran it |
| `pkgutil.get_data` | reads `data/*.json` out of the zip, so the data files survive packaging | generated from the built `.apworld` |
| Headless generation | needs `SKIP_REQUIREMENTS_UPDATE=1`, or `ModuleUpdate` blocks on `input()` for every world's optional deps | `Generate.py` hung otherwise |
| `pkg_resources` | `ModuleUpdate` imports it, so setuptools must be `<81`. 84 has dropped it. | pinned 80.9.0 |
| `websockets` | must be the pinned `13.1`. `MultiServer.py` uses `socket.open`, gone in 14+, and every client connection dies with `AttributeError`. | hosting a seed with 17.0.1 |

The `.pop` files being inside the VPK matters: nothing can read them from the
host without a VPK extractor. Wave counts therefore come from the table below,
not from parsing, and open question 4 in `spec.md` is answered: **hardcode the
Valve missions, refuse community missions in v1.**

## The mission table

Pop file names extracted from `tf2_misc_dir.vpk`. Wave counts and difficulties
from the official TF2 wiki. 29 missions, which matches the wiki's own count.

| Pop file | Mission | Map | Difficulty | Waves |
| --- | --- | --- | --- | --- |
| `mvm_decoy` | Doe's Drill | mvm_decoy | Normal | 8 |
| `mvm_decoy_intermediate` | Doe's Doom | mvm_decoy | Intermediate | 7 |
| `mvm_decoy_intermediate2` | Day of Wreckening | mvm_decoy | Intermediate | 6 |
| `mvm_decoy_advanced` | Disk Deletion | mvm_decoy | Advanced | 8 |
| `mvm_decoy_advanced2` | Data Demolition | mvm_decoy | Advanced | 6 |
| `mvm_decoy_advanced3` | Disintegration | mvm_decoy | Advanced | 6 |
| `mvm_decoy_expert1` | Desperation | mvm_decoy | Expert | 7 |
| `mvm_coaltown` | Crash Course | mvm_coaltown | Normal | 6 |
| `mvm_coaltown_intermediate` | Cave-in | mvm_coaltown | Intermediate | 6 |
| `mvm_coaltown_intermediate2` | Quarry | mvm_coaltown | Intermediate | 6 |
| `mvm_coaltown_advanced` | Ctrl+Alt+Destruction | mvm_coaltown | Advanced | 7 |
| `mvm_coaltown_advanced2` | CPU Slaughter | mvm_coaltown | Advanced | 6 |
| `mvm_coaltown_expert1` | Cataclysm | mvm_coaltown | Expert | 7 |
| `mvm_mannworks` | Mann-euvers | mvm_mannworks | Normal | 7 |
| `mvm_mannworks_intermediate` | Mean Machines | mvm_mannworks | Intermediate | 6 |
| `mvm_mannworks_intermediate2` | Mannhunt | mvm_mannworks | Intermediate | 6 |
| `mvm_mannworks_advanced` | Machine Massacre | mvm_mannworks | Advanced | 7 |
| `mvm_mannworks_ironman` | Mech Mutilation | mvm_mannworks | Advanced | 3 |
| `mvm_mannworks_expert1` | Mannslaughter | mvm_mannworks | Expert | 5 |
| `mvm_bigrock` | Benign Infiltration | mvm_bigrock | Normal | 6 |
| `mvm_bigrock_advanced1` | Broken Parts | mvm_bigrock | Advanced | 7 |
| `mvm_bigrock_advanced2` | Bone Shaker | mvm_bigrock | Advanced | 8 |
| `mvm_mannhattan` | Big Apple Barricade | mvm_mannhattan | Intermediate | 6 |
| `mvm_mannhattan_advanced1` | Empire Escalation | mvm_mannhattan | Advanced | 6 |
| `mvm_mannhattan_advanced2` | Metro Malice | mvm_mannhattan | Advanced | 6 |
| `mvm_rottenburg` | Village Vanguard | mvm_rottenburg | Intermediate | 7 |
| `mvm_rottenburg_advanced1` | Hamlet Hostility | mvm_rottenburg | Advanced | 7 |
| `mvm_rottenburg_advanced2` | Bavarian Botbash | mvm_rottenburg | Advanced | 7 |
| `mvm_ghost_town_666` | Caliginous Caper | mvm_ghost_town | Haunted | 1 |

Total: 181 waves, which is the upper bound on the wave-clear location group.
With the 29 mission clears that is 210 checks. An earlier revision of this
table said 176; the rows sum to 181 and the export is built from the rows.

The last tier is Valve's `haunted`, from the `mvm_maps` block of
`tf/scripts/items/items_game.txt`. The wiki calls it Nightmare.

- [ ] **UNVERIFIED: the pop-file-to-mission mapping within a difficulty tier.**
      The pop file names and the mission names are both certain, and the
      pairing is certain wherever a tier has one entry. It is a guess for
      `mvm_decoy_intermediate` vs `_intermediate2` (Doe's Doom / Day of
      Wreckening), `mvm_coaltown_intermediate` vs `_intermediate2` (Cave-in /
      Quarry), `mvm_mannworks_intermediate` vs `_intermediate2` (Mean Machines
      / Mannhunt), and the three `_advanced*` groups. Resolve by reading
      `resource/tf_english.txt` in the VPK, which keys mission display names by
      pop file name. A wrong pairing gives the player the wrong mission name in
      chat, nothing worse, but fix it before the first seed is played.

## Milestone 1: gamedata — done

Go package, no dependencies. See ADR 0001 for the id rules.

- [x] `Mission{ID, PopFile, Name, Map, Difficulty, Waves}`, the 29 rows above.
- [x] `Map{ID, Name}`, the 7 maps.
- [x] `Class{ID, Key, Name}`, the 9 classes. `Key` is what crosses the wire to
      the plugin, so nothing here depends on TF2's internal class numbering.
- [x] `WeaponSlot{ID, Key, Name}`: Primary, Secondary, Melee. The table order
      is the order the progressive item hands them out, so it is as frozen as
      an id.
- [x] `Difficulty` enum: normal, intermediate, advanced, expert, haunted.
- [x] The game name string, exactly once: `"Team Fortress 2 Mann vs Machine"`.
- [x] AP base id offset, one constant: `7_442_000_000`.
- [x] Data format version constant.
- [x] Location id derivation: `base + mission.ID*100 + wave` for wave clears,
      `base + mission.ID*100 + 99` for mission clears. Items sit at
      `base + 1_000_000` and up so the two spaces cannot meet.
- [x] `LocationByObjective(kind, popfile, wave)`, the whole southbound
      translation. The bridge keeps no id table of its own.
- [x] JSON exporter writing `apworld/tf2_mvm/data/{missions,items,meta}.json`,
      behind `go generate ./gamedata`.
- [x] Test: ids unique across the whole space.
- [x] Test: no id in `testdata/ids-frozen.json` has moved. New ids are recorded
      with `go test ./gamedata/ -freeze`, which never rewrites an existing one.
- [x] Test: `1 <= waves <= 98` for every mission, and the last wave of a
      mission never reaches its own clear slot.
- [x] Test: the committed export matches what the Go source produces.

Deliberately **not** in v1: weapons (210 lines in the fork's table), upgrade
lines, canteens, robot templates, traps. Weapon *slots* are enough to make a
progression, and the full weapon table is the single biggest chunk of data
entry in the project. Add it in v2.

The item pool the export declares: 29 mission tickets, 9 class items, one
`Progressive Weapon Slot` in 3 copies, and `Cash Bundle` filler worth 200
credits. 40 named items against 210 locations, so filler carries the rest.

## Milestone 2: apworld — done

Python, `apworld/tf2_mvm/`.

- [x] `__init__.py`: `World` subclass, `game`, `item_name_to_id`,
      `location_name_to_id` built from the JSON.
- [x] `create_regions`: `Menu` region, one region per mission, an `Entrance`
      per mission gated on that mission's ticket.
- [x] Locations: `"<Mission> Wave <N>"` per wave, `"<Mission> Complete"` per
      mission.
- [x] Items: one `Mission Ticket: <Mission>` per mission (progression),
      `Progressive Weapon Slot` x3 (progression), one `Class: <Name>` per class
      (progression), `Cash Bundle` filler to pad the pool.
- [x] Access rule: mission M's locations need M's ticket and a number of
      classes and weapon slots that climbs with M's tier (`rules.py`). A flat
      "one of each" rule put every mission in sphere 1 and made the run a list
      of tickets; the tier ladder is what gives a seed spheres.
- [x] **Sphere 0 guarantee**: the easiest mission drawn is the starting one,
      and its ticket plus exactly what its tier asks for are precollected. AP's
      own `test_empty_state_can_reach_something` guards it on every option set
      in `test/`.
- [x] Goal: `final_boss` = the hardest mission drawn is reachable;
      `missionsanity` = a share of the missions drawn are.
- [x] `options.py`, hand-written: `mission_count` (Range 1-29),
      `difficulty_pool` (Choice), `goal` (Choice), `missionsanity_percentage`
      (Range), `death_link` (DeathLink). `difficulty_pool`'s tiers are checked
      against the export at import, not trusted.
- [x] `fill_slot_data`: the mission pop files, the goal, the goal mission, the
      missionsanity target, death link, and the data format version.
- [x] `docs/setup_en.md` and `docs/en_Team Fortress 2 Mann vs Machine.md`.
      The game-info file name has to be `<lang>_<game name>.md`, spaces and all.
- [x] Refuse to load a `data/` whose format version is unknown, per file.
- [x] `archipelago.json` manifest and an `.apignore` that keeps `test/` out of
      the built artifact.
- [x] `test/`, six option sets. AP's `WorldTestBase` contributes fill,
      all-state reachability and sphere 0 to each.

**Acceptance, done 2026-08-16 against AP 0.6.7**: seeds generate for
normal/1, normal/29, intermediate/8, expert/4, advanced/29 and both goals; the
spoiler log for the default YAML shows five spheres ending at the goal; AP's
own `test/general` suite passes for this world; the packaged `.apworld`
generates from `custom_worlds/`.

Known and deliberate: `difficulty_pool: haunted` cannot generate, because the
one haunted mission is 2 checks against 4 unlock items. It raises an
`OptionError` naming the option instead of producing a broken seed, and it
starts working the day a second haunted mission exists.

## Milestone 3: bridge — done

Go. See ADR 0002 for the invariants; they are not repeated here. The API and
the environment contract are in `bridge/README.md`.

### Northbound, Archipelago websocket

- [x] Connect, then send `Connect` with `game`, `name`, `password`, `uuid`,
      `version`, `items_handling`, `tags`.
- [x] Handle `RoomInfo`, `Connected`, `ConnectionRefused`, `ReceivedItems`,
      `PrintJSON`, `Bounced`.
- [x] Send `LocationChecks` with the location ids. The whole set goes every
      time: Archipelago ignores repeats, and that makes a reconnect mid-wave a
      non-event.
- [x] Send `StatusUpdate` = `CLIENT_GOAL` when the goal is met, once, recorded
      on disk so a reconnect does not announce the win twice. Both goals are
      read off the checks: a mission is cleared exactly when its clear location
      is.
- [x] Reconnect with exponential backoff, capped at 30s. Both `ws://` and
      `wss://`. A refusal (`InvalidSlot`, wrong password) or slot data from
      another data format version stops the bridge instead of hammering the
      server forever.
- [x] Dedup received items by index, including the index-0 full resend and a
      `Sync` when the lists diverge.
- [x] Bind to the seed name. A different seed drops the previous run rather
      than replaying its ids into a world where they mean something else.

### Southbound, HTTP on 127.0.0.1

- [x] Durable queue on disk. Write, fsync, rename, then `204`, then upstream.
- [x] Idempotent by location id, because the plugin retries on timeout.
- [x] Unlock set persisted, survives a bridge restart. It is derived from the
      received item list rather than stored beside it, so the two cannot
      disagree.
- [x] Config from env only. No config file.

**Acceptance, done 2026-08-16** against Archipelago 0.6.7 hosting a seed
generated by our apworld:

- The bridge connects and appears in the server log: `TF2 (Team #1) playing
  Team Fortress 2 Mann vs Machine has joined`.
- It reads the slot data and applies the starting inventory: `/unlocks`
  returned two classes, one slot and one mission ticket, which is exactly what
  the intermediate starting tier asks for.
- `curl -X POST /objective` landed a check: `TF2 sent Mission Ticket: Village
  Vanguard to TF2 (Mannhunt Wave 1)`. The same POST repeated produced nothing
  the second time.
- The ticket came back as a received item and appeared in `/grants` and
  `/unlocks` without anything being asked twice.
- Clearing the goal mission produced `TF2 (Team #1) has completed their goal.`

Two bugs the tests caught and one the live run did not have to:

- Learning the seed name wiped checks taken before Archipelago was ever
  reachable, which is precisely the case the durable queue exists for.
- The pump took its wake-up channel after reporting rather than before, so a
  check recorded during a report could sit unsent until something else
  happened.

Not implemented, deliberately: DeathLink. The seed can ask for it and the
bridge logs that it will not honour it, because what a death means in MvM is
still open (`docs/spec.md`, question 5). It does not claim the tag, so it does
not promise other players something it will not deliver.

## Milestone 4: plugin — written, never run

SourcePawn, SourceMod 1.12, `ripext` 1.3.2. Every bridge call asynchronous.

- [x] Detect wave clear: `mvm_wave_complete`, with the wave number carried from
      `mvm_begin_wave`. **Both UNVERIFIED.** Hooked with `HookEventEx`, so a
      missing event degrades instead of failing the load, and a one-second
      timer watching `m_nMannVsMachineWaveCount` takes over if the event is not
      there.
- [x] Detect mission clear: `mvm_mission_complete`, plus the last wave of the
      mission reporting it too. Both are idempotent at the bridge, so between
      them one will fire.
- [x] Read the current pop file: `m_iszMvMPopfileName`, path and extension
      trimmed, falling back to the map name. **UNVERIFIED.**
- [x] `POST /objective` on each, with retry. The queue keys objectives by id,
      not by index, because an erase shifts every index after it while requests
      are in flight.
- [x] `GET /unlocks` on plugin load and on `OnMapStart`, then apply.
- [x] Long-poll `/grants`, apply as they arrive, re-poll immediately.
- [x] Enforce weapon slots on every inventory application, which covers
      spawning, resupply lockers and the upgrade station.
- [x] Enforce classes at `joinclass`, and move a player off a locked class on
      spawn. Enforcement is a no-op until the bridge has answered once: a
      server where nobody can hold a weapon because the bridge hiccuped is
      worse than a wave played with too much kit.
- [x] Announce received items in chat, with errors going to chat as well.
      `sm_ap` prints the whole state, `sm_ap_report` sends an objective by
      hand, `tf2ap_debug 1` echoes everything.
- [x] Compile with `spcomp`: `plugin/build.sh` fetches the pinned toolchain and
      compiles with warnings as errors. Clean as of 2026-08-16.
- [ ] Run it. **Nothing here has been executed.** No Team Fortress 2 server was
      available, so every game event and entity property above is a compile-time
      guess. The first live session should be `tf2ap_debug 1`, and `sm_ap` will
      say which of the three events actually exist.
- [ ] Dockerise the compile for CI (milestone 6).

## Milestone 5: compose

- [ ] `deploy/Dockerfile.archipelago`: pinned AP 0.6.7 source, Python 3.13,
      `ModuleUpdate.py`, our apworld into `custom_worlds/`. It has to be built
      into an `.apworld` first; a bind-mounted folder there does not load.
- [ ] Entrypoint: if `output/` has no `.zip`, run `Generate.py` from the YAML
      rendered out of env, then `MultiServer.py` on the result.
- [ ] `deploy/Dockerfile.srcds`: `FROM cm2network/tf2:sourcemod`, add ripext
      and the compiled plugin.
- [ ] `deploy/Dockerfile.bridge`: Go build, distroless.
- [ ] `deploy/compose.yml`, three services, `.env` for everything.
- [ ] Reuse the existing volume `tf2-archipelago_tf2game` so the 14 GB download
      is not repeated. Check whether compose accepts a pre-existing volume with
      no compose labels; if not, declare it `external: true`.

### Env contract

| Variable | Default | What it does |
| --- | --- | --- |
| `AP_SLOT_NAME` | `tf2` | The multiworld slot name. One slot for the whole server. |
| `AP_PASSWORD` | empty | Multiworld password. |
| `AP_PORT` | `38281` | AP server port, loopback. |
| `MVM_MISSION_COUNT` | `8` | Missions in the run. |
| `MVM_DIFFICULTY` | `intermediate` | Lowest tier in the pool. |
| `MVM_GOAL` | `final_boss` | `final_boss` or `missionsanity`. |
| `SRCDS_HOSTNAME` | | Passed through to the image. |
| `SRCDS_RCONPW` | | Required, no default. Never exposed. |
| `SRCDS_PW` | | Server join password. |
| `SRCDS_PORT` | `27015` | **The only public port.** |
| `SRCDS_MAXPLAYERS` | `6` | MvM RED team size. |

The bridge adds `AP_HOST`, `AP_TLS`, `BRIDGE_LISTEN`, `BRIDGE_STATE` and
`BRIDGE_POLL_TIMEOUT`; see `bridge/README.md` for their defaults.

- [ ] `27015/udp` public, everything else `127.0.0.1`. RCON never exposed.
- [ ] `.env.example` committed, `.env` gitignored.

## Milestone 6: verify and document

- [ ] Seed generates.
- [ ] AP server hosts it and accepts a connection.
- [ ] Bridge completes the handshake and appears in the AP server log.
- [ ] `curl POST /objective` lands a check visible to AP.
- [ ] srcds boots into an MvM map with the plugin loaded.
- [ ] **Cannot be verified without a human and a TF2 client**: that a wave
      clear in-game actually fires the objective, and that a granted weapon
      slot is actually enforced. Say so in the docs rather than implying it
      was tested.
- [ ] `docs/archipelago-101.md`: what a multiworld, a slot, a check, an item
      and a seed are, for someone who has never played one.
- [ ] `docs/running.md`: `.env`, `docker compose up`, joining, what to expect.
- [ ] `.forgejo/workflows/ci.yml`: gofumpt, vet, golangci-lint, go fix, build,
      race tests, govulncheck, export freshness, `spcomp`, and a generation
      smoke test.
- [ ] Makefile.

## Carried over from spec.md

Still open, still not blocking v1 because each one is behind an option that
ships off:

- [ ] Shop check injection: can a plugin add a purchasable entry to the upgrade
      station UI? Blocks the `shop_checks` group entirely.
- [ ] Allied bot upgrade sharing: do RED bots still work in MvM, and can they
      inherit purchased upgrades? Blocks `Allied Mercs`.
- [ ] DeathLink semantics: individual death is noise in MvM. Wave failure, or
      death outside the respawn grace window?

## Community

- [ ] Ask the Discord thread for Snolid Ice's MvM Manual. Mentioned twice,
      never linked.
- [ ] Tell Damonj17 and Roseburst this exists. The design is theirs.

## Repo

- [ ] Decide whether this goes public. It is private today, and the design came
      from a public thread. Note that the Forgejo instance forces new repos
      private, so this needs a deliberate flip in the UI.

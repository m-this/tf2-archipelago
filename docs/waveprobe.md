# Unattended MvM wave sweep

`deploy/run-waveprobe.sh` starts disposable Docker servers and runs every
catalog mission with and without Bot Surge. It skips any future missions marked `no_nav` (the current catalog has none),
which cannot run in the game. The live Compose project is never restarted.

The local wrapper and daily checker enable all playable reverse missions with
`-reverse-objectives`: 122 reverse wave/mode cases and 2,628 total cases in the
current catalog. Reverse failures count against the percentage. See
[reverse coverage](reverse-waveprobe.md) for the objective witnesses and limits.
Set `WAVEPROBE_REVERSE_OBJECTIVES=0` to use the earlier bomb/carrier subset, or
also set `WAVEPROBE_REVERSE_BOMBS=0` for a standard-only control.

Each standard test starts the real wave, attempts to defeat each observed BLU
bot or tank 15–25 game seconds after it first appears, and requires the game's
`mvm_wave_complete` event to pass.
Spawn events are counted when they happen, including enemies that die between
the probe's timer ticks.
Before wave setup, the headless runner sets and reads back
`tf_allow_server_hibernation 0`. Fake clients do not wake the server like a
human connection, so leaving hibernation enabled can prevent SigMod's robot
pool from being allocated. This matches the fixture in SigMod's own
[Linux](https://github.com/m-this/sigsegv-mvm-win/blob/a83ba95177dd2091a366ed2c9edafca869b22b1e/.github/workflows/linuxbed.yml#L145)
and [Windows](https://github.com/m-this/sigsegv-mvm-win/blob/a83ba95177dd2091a366ed2c9edafca869b22b1e/.github/workflows/windows.yml#L384)
test beds. It leaves the production robot limiter and invader cap in place.
This headless fixture does not test idle hibernation or human join/wake behavior;
those need a separate real-client check.

The fake RED player uses Medic for missions tagged `medic_only`, including
Remedic, Engineer for Robotfactory, and Scout otherwise. A case is one mission wave with Bot Surge either
off or on; the two modes have separate results and scores.
The disposable probe prevents damage to RED participants. This keeps
survival objectives and friendly mission NPCs alive while it measures BLU
spawn clearance and wave completion.
It uses `tf_bot_flag_kill_on_touch` so an unguarded hatch does not turn a
population test into an automatic loss. A robot with a boss health bar drops the
bomb as it takes it instead, so a boss whose phases change at health
thresholds reaches them. A wave whose robot count stands still for 300 game
seconds gets the capture a player would make on any enabled capture area, as
trespasser's landing zone needs. Robots on the players' team are killed like
any other unless the wave lists their icon as support, and those too when SigMod's
wave dump shows a wave spawn the wave counts waiting for them to die. A robot tagged `timer` or
`bot_timer` is a wave clock and is left alive. The spawn delays use a fixed seed and
can be changed for a repeat run.

Run this from a checkout with both community packs extracted under
`community-content/tf/` (see the [community content guide](../community-content/README.md)).
Point the probe at the image used by the running Compose server:

```sh
export WAVEPROBE_SRCDS_IMAGE="$(docker inspect --format '{{.Config.Image}}' tf2-archipelago-srcds-1)"
```

Prepare a disposable base volume once. This copies the installed game files
from the live volume without writing to it:

```sh
docker volume create tf2-archipelago-waveprobe_tf2game_waveprobe
docker run --rm \
  -v tf2-archipelago_tf2game:/src:ro \
  -v tf2-archipelago-waveprobe_tf2game_waveprobe:/dst \
  debian:bookworm-slim sh -c 'cp -a /src/. /dst/'
```

Then run the sweep on a machine with enough RAM and disk for the requested
number of game copies:

```sh
export WAVEPROBE_RCONPW="$(openssl rand -hex 24)"
WAVEPROBE_SHARDS=6 WAVEPROBE_SPEED=50 bash deploy/run-waveprobe.sh
```

`WAVEPROBE_SHARDS`, `WAVEPROBE_SPEED` (1–50; default 50),
`WAVEPROBE_FIRST_PASS_TIMEOUT` (default `180s`), and `WAVEPROBE_TIMEOUT`
(default `900s`) control runtime. Every nonpass from the first pass is retried
automatically with the full 900 real-second limit before the final report is
written. At 20×, 900 real seconds can represent up to five game hours; the
deadline never shrinks to 45 real seconds. Map and wave load have a separate
90-second deadline.
`WAVEPROBE_RUN_DIR`
chooses where the JSONL files and `REPORT.md` go; otherwise a timestamped
directory is made in `docs/audits/`. `SUMMARY.json` carries comparable counts
and the stability score (`passed / tested`). `REPORT.md` puts that score and
verified coverage (`passed / eligible`) first, then separates Bot Surge modes
and lists every issue with a spawn/kill/alive/progress timeline. “Wave failed”
means the game emitted a loss, “Wave 0” means the population manager never
initialized a wave, and “No enemies spawned” means the probe saw no BLU bot or
tank in a standard wave. A wave still active at the wall limit is “Wave timed
out”; that result includes enemy counts and final HUD progress.
HUD progress uses the game's non-support enemy count relative to its highest
observed value. Some custom missions complete with that counter below 100%,
so the game's completion event remains the pass criterion.

A pass is one completion witness at the chosen timing and seed. It does not
prove every authored bot appeared, or that the wave will complete under every
player strategy. Re-run timeouts at a lower speed or another seed before
attributing them to the mission or Bot Surge.

Villa's Recalled to Life mission needs its earlier rooms initialized before
the wave-5 hunt. A targeted `-start-wave 5 -end-wave 5` run therefore plays
waves 1–4 first. If one of those setup waves fails, wave 5 is inconclusive;
the runner does not jump past it and report a misleading wave-5 timeout.

The standard sweep runs the second pass and writes `REPORT.md` and `SUMMARY.json`
itself, including when a shard exits with a failure. To resume an interrupted
sweep, run `bash deploy/resume-waveprobe.sh <run-directory>`. It starts the
recorded disposable containers, screens unverified cases at the first-pass
limit, retries remaining nonpasses at the full 900-second limit, writes both
reports, and stops the containers. Existing passes are preserved. The screening
rows go to `screen-<id>.jsonl`; full retries go to `retest-<id>.jsonl`. A wave
that remains active at the full limit needs inspection; the limit alone does
not prove a deadlock.

For manual partial retries, `deploy/waveprobe-retest.py` accepts the run
directory, runner binary, comma-separated worker IDs, and an optional fourth
argument listing source shard IDs. Set `WAVEPROBE_PHASE=screen` for the short
screening pass; omit it for full retries.

Each sweep writes its unique Compose project names to `projects.txt` in the run
folder. The runner stops those containers on completion or interruption, but
keeps their copied game volumes for retests. After retesting, remove every
shard container and volume from that run with:

```sh
while IFS= read -r project; do
  WAVEPROBE_RCONPW=cleanup docker compose -p "$project" -f deploy/compose.waveprobe.yml down -v
done < docs/audits/waveprobe-YYYYMMDD-HHMMSS/projects.txt
```

Keep the pristine `tf2-archipelago-waveprobe_tf2game_waveprobe` source volume
until you no longer need new sweeps. It is never used as a shard volume.

## Local concurrency and shared image layers

`make waveprobe-plan` prints the available CPU, RAM and Docker disk budget.
`make waveprobe-local` prepares a local image, then runs a resource-bounded
number of isolated workers at 50×. Set `WAVEPROBE_SRCDS_IMAGE` to the production
image to test and `WAVEPROBE_SOURCE_VOLUME` to a pristine test game volume;
the source is read-only. On first preparation, provide the existing full
archives with `--community-dir /path/to/community-content` (or
`WAVEPROBE_COMMUNITY_DIR`). Both archives are validated and installed once into
the image by the launcher's own installer. Their catalog fingerprint invalidates
the cache when the required content changes. The archive location is remembered
locally for later rebuilds. The current Ryzen 9950X3D/61 GiB machine's resource
plan selects twelve workers, each limited to two CPUs and 3 GiB RAM, leaving
four physical cores, eight GiB RAM and a disk reserve for the host.

Each worker shares immutable game image layers and writes into its private
container layer. No writable game directory is shared between workers. This
avoids a full game copy per shard. The first preparation needs space for both
the copy and image commit plus a 30 GiB reserve; later runs reuse the image
only while its base image and live TF2 version still match. After preparation,
the default command reuses the recorded base image; pass `--image` to select a
new production build when updating the mod. Every worker still
runs the production entrypoint's normal SteamCMD update. The prepared image
is a warm wave fixture; fresh-install startup remains the separate daily CI
check, and preparation logs remain in the run directory.

The direct command allows a lower worker count or a forced image rebuild:

```sh
python3 deploy/local-waveprobe.py --plan
python3 deploy/local-waveprobe.py --workers 8
# Maximum-throughput option on the measured 16-core machine:
python3 deploy/local-waveprobe.py --workers 16
python3 deploy/local-waveprobe.py --community-dir /path/to/community-content --rebuild
```

Whole missions are assigned by the recorded `partition.json`, balancing the
number of planned wave/mode cases instead of hashing mission names. First
passes and retries use the same manifest. Old run directories without a
manifest retain their original CRC32 partition on resume.

The [measured concurrency comparison](https://github.com/kelly-cs/tf2-archipelago/blob/e7ecb0f/docs/audits/shard-tuning-20261024/REPORT.md)
passed 798/798 Decoy wave/mode cases. Twelve workers achieved 157 cases/minute;
sixteen achieved 176–177 cases/minute in two seeds. Twelve remains the default
with host headroom; `--workers 16` is the measured maximum-throughput option
on that machine. These measurements do not certify the full community catalog.

The sweep stops its own workers and keeps them for resuming. After finishing,
remove only the containers named in that run's `projects.txt`; their writable
layers are then freed. The prepared image stays cached for the next sweep.

## Defender bot runs

`deploy/run-botprobe.sh` runs the same probe on one server with the defender
bots on, and records how each RED bot moved: when it left its spawn room in
each life, its longest time standing still once the wave runs and where, how
close it came to the hatch, and its teleports. The bots can die in this mode,
so every respawn is another spawn exit. The mod's own spawn and wedge rescue
lines are copied from the SourceMod log and matched to each wave.

It uses an existing game volume in place rather than copying one, and caps the
server at `WAVEPROBE_CPUS` (default 1) and `WAVEPROBE_MEMORY` (default 1536m):

```sh
export WAVEPROBE_RCONPW="$(openssl rand -hex 24)"
WAVEPROBE_GAME_VOLUME=tf2-archipelago_tf2game \
WAVEPROBE_SRCDS_IMAGE=tf2ap-srcds:v117 \
WAVEPROBE_ONE_PER_MAP=1 \
WAVEPROBE_BOTS_SMX=/path/to/tf2_defenderbots.smx \
    bash deploy/run-botprobe.sh
```

`WAVEPROBE_BOTS_SMX` defaults to the pinned build from `make bots`. Point it at
a build from a tf2-mvm-bots-go branch to measure that branch. Only maps the
volume holds are run, and only community missions whose popfile it holds.
`WAVEPROBE_MAPS` restricts the run to a comma-separated list, `WAVEPROBE_WAVES`
(default 1, 0 for all) is how many waves of each mission to play,
`WAVEPROBE_MODE` is `normal` (default), `surge` or `both`, and
`WAVEPROBE_ONE_PER_MAP` plays the first installed mission of each map.

`WAVEPROBE_SHARDS` (default 1) splits the missions over that many servers, all
on the same game volume, with RCON on `WAVEPROBE_RCON_PORT` and the ports after
it. Each server takes `WAVEPROBE_CPUS` and `WAVEPROBE_MEMORY`, so size it to
the host. Every server start runs SteamCMD over the volume, so a server is
started only once the one before it answers. The shards share one SourceMod
log, so a sharded report counts the mod's rescues but does not match them to
missions.

`REPORT.md` lists the bots that never left spawn, took over 20 seconds to, or
stood idle for 30 seconds, and the rescues per wave. Idle means standing still
and not attacking while a robot or a tank is within 1500 units. Standing still
alone is not idle here. The probe kills each robot 15 to 25 game seconds after
it appears, so most never reach the front, and a bot that holds its post there
has nothing to shoot. `still_max_seconds` keeps the raw stillness. Two runs
compare map by map with:

```sh
python3 deploy/botprobe-compare.py docs/audits/<run-a> docs/audits/<run-b>
```

## Release gate

`make bots-gate` is the run before a release: every wave of every mission the
game volume holds, with and without Bot Surge, with the bots from `make bots`.
`REPORT.md` opens with the verdict, and the run exits 1 when it fails. It fails
on:

- a planned wave with no defender record, because the server crashed, hung or
  never played it;
- a bot that never left its spawn room in more than 20 game seconds, or whose
  last life has spent more than 20 seconds there;
- a bot other than an engineer or a sniper idle for 30 game seconds, as defined
  above.

A lost or timed-out wave does not fail it by itself: the probe's kills decide
those, not the bots. The sweep above is what judges wave completion.

It takes hours and a 14 GB game, so it runs by hand on a Linux Docker host, not
on a GitHub runner:

```sh
make build    # the image under test, tf2-archipelago-srcds:latest
WAVEPROBE_GAME_VOLUME=tf2-archipelago_tf2game WAVEPROBE_SHARDS=4 make bots-gate
```

At speed 4 one server played about two waves a minute on Decoy, loads
included. Both modes of every Valve mission are about 360 waves, three hours
on one server; with both community packs it is about 2,300, so shard as far
as the host allows.
`WAVEPROBE_MAPS` narrows a rerun to the maps that failed.

A server that cannot keep up with `WAVEPROBE_SPEED` times out waves that pass
on a quiet host. Compare arms run under the same load, and rerun a map whose
game seconds fall well short of its wall seconds times the speed.

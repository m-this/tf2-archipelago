# Unattended MvM wave sweep

`deploy/run-waveprobe.sh` starts disposable Docker servers and runs every
catalog mission with and without Bot Surge. It skips missions marked `no_nav`,
which cannot run in the game. The live Compose project is never restarted.

The report marks Reverse MvM missions separately. They force players to BLU
and finish through BLU's objective; defeating RED robots alone cannot prove
their waves can complete. A separate objective simulator is needed for those
missions.

Each standard test starts the real wave, attempts to defeat each observed BLU
bot or tank 15–25 game seconds after it first appears, and requires the game's
`mvm_wave_complete` event to pass.
Spawn events are counted when they happen, including enemies that die between
the probe's timer ticks.
The fake RED player uses Medic for missions tagged `medic_only`, including
Remedic, and Scout otherwise. A case is one mission wave with Bot Surge either
off or on; the two modes have separate results and scores.
The disposable probe prevents damage to RED participants. This keeps
survival objectives and friendly mission NPCs alive while it measures BLU
spawn clearance and wave completion.
It uses `tf_bot_flag_kill_on_touch` so an unguarded hatch does not turn a
population test into an automatic loss. The spawn delays use a fixed seed and
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
WAVEPROBE_SHARDS=6 WAVEPROBE_SPEED=20 bash deploy/run-waveprobe.sh
```

`WAVEPROBE_SHARDS`, `WAVEPROBE_SPEED` (1–20),
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
(default 1) is how many waves of each mission to play, and `WAVEPROBE_ONE_PER_MAP`
plays the first installed mission of each map.

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

A server that cannot keep up with `WAVEPROBE_SPEED` times out waves that pass
on a quiet host. Compare arms run under the same load, and rerun a map whose
game seconds fall well short of its wall seconds times the speed.

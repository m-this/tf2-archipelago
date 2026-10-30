# Daily stability

The [daily report](https://github.com/m-this/tf2-archipelago/releases/tag/stability)
measures the repository’s `main` branch on native **Ubuntu 24.04**, **Windows**, and
the **production Docker image** built from that commit. It runs daily at 05:17
UTC. Start it manually through Actions → Daily stability → Run workflow,
selecting `main`. Use **scope: full** for the real checker. **scope: smoke**
checks startup and Decoy Expert in both modes on one runner per environment;
it publishes an explicitly scoped Actions report and never updates the badges.
**scope: reverse-smoke** runs every Breaking Walls wave in both modes on one
runner per environment, also without publishing badges.
The README widgets link to the report. They show passed / planned percentages; the report carries the measurement date.
Results older than 36 hours are stale, including when scheduling stops. A green badge
from a previous day is not evidence about a subsequent TF2 update.

Each of twelve shards per environment gets a fresh disposable install of TF2 and both community
packs, using the launcher's installer and pinned mod dependencies. Startup
requires RCON, SourceMod, the Archipelago plugin, the wave probe, and no failed
plugin or extension loads. The fixture explicitly sets
`SRCDS_MOD_LOADING=always` for its entire mission catalog, matching the native
bed: without a seeded mission pool, the production image's normal "when needed"
policy would start stock Decoy without SigMod. The limiter guard rejects that
configuration instead of scoring an unmodded sweep. The Docker check runs the production entrypoint,
including its own SteamCMD update; the launcher-prepared tree supplies the
community packs. It never mounts a live server's files.

Startup is checked before the wave runner applies its headless wake-up fixture
(`tf_allow_server_hibernation 0`). The sweep keeps `sig_mvm_robot_limit` enabled
and records its readback before and after testing; a disabled or unreadable
limiter prevents a numeric score. The invader cap, population files, mod code
and mod settings are unchanged. This fixture models an awake server with a human
connected; it does not validate idle hibernation or real-client wake-up.
Fatal errors and segmentation faults in the captured engine logs prevent a
passing verdict, including when `srcds_run` restarts the engine and later checks succeed.

SteamCMD installs the current public TF2 server, without a game version pin.
The measurement compares `tf/steam.inf`'s `ServerVersion` with Valve's live
`ISteamApps/UpToDateCheck` response before setup, at readiness, after the sweep,
and again when publishing the combined report. A failed update cannot silently
test an old server. If Valve updates during a run, the report marks affected
environments unverified. `TF2_SERVER_VERSION` is recorded by the normal build,
but does not select which TF2 build this job installs.
The engine's own startup version must also match at readiness; the report
includes the dependency pins and both the running and installed version.

## What the score means

The [existing wave probe](waveprobe.md) supplies the plan and the detailed
reports: every eligible mission on every catalog map, every wave, Bot Surge
off and on, at requested 50× speed and seed 1. It screens at 180 real seconds
per wave, then retries every nonpass at 900 real seconds. The game's completion
event is the pass criterion. First-pass records survive retries. Nonpasses
carry spawn/kill/alive counts, progress and timeline/debug evidence.

**Score = passed cases / all eligible planned cases × 100.** Missing results
count against it. Every environment with valid counts displays a percentage,
even when startup, freshness, limiter or crash checks fail. Those checks appear
in Result, keep badges red, and fail the workflow. An absent or invalid plan
shows `n/a`; it cannot become a passing result.

The dashboard and detailed wave reports use the same denominator. The compact
release page links to a full report and `issues.csv`, listing environment, map,
mission, wave, mode, shard, stage and error for every failed or missing case.
Earlier failures recovered by retries remain in a separate expandable table.
Crashes include the last observed map/mission/wave when captured; that context
does not establish which code caused the crash. Raw logs and timelines remain
in the evidence archive. Release notes stay small even when many cases fail.

The daily and local sweeps enable `-reverse-objectives`: all 122 catalog
reverse wave/mode cases count in the score, with zero reverse exclusions.
The probe observes native bomb capture, control-point capture, authored boss
deaths, building repair, escort stages, and tank delivery as appropriate to the
mission. Missing objective evidence is a probe error and counts against the
percentage. No winning relays, capture outputs, building health or ownership
are forced. See [reverse coverage](reverse-waveprobe.md) for exact witnesses and
fixture limitations.
The catalog audit recovered 16 missions (174 wave/mode cases) whose navigation
was embedded in their BSPs. All 2,628 catalog cases now enter the plan; no current
mission is excluded for navigation. Archive installation receipts include the
catalog identity so existing installs receive those recovered popfiles. The score
measures wave completion under this probe's strategy. It does not measure
human client joins, multiworld connectivity, performance, or every bot shopping
and combat path. The separate defender-bot release gate remains useful.

The [local speed comparison](https://github.com/kelly-cs/tf2-archipelago/blob/e7ecb0f/docs/audits/speedprobe-wake-20261023/REPORT.md) tested
Decoy Expert with the wake-up fix and production robot limiter enabled. 50×
passed five full runs across four seeds, averaging 48.4 seconds versus 89.4 at
20×. Higher tested settings intermittently lost the last wave's completion
during automatic map changes. This is one-map evidence; the daily sweep now uses 50× to validate that candidate across the full catalog and environments.

## Hosted runner setup

No self-hosted runners are required. Twelve balanced mission shards run for each
of Ubuntu 24.04, Docker on Ubuntu 24.04, and native Windows Server 2022, using
standard GitHub-hosted runners. At most twelve measurement jobs run together.
A deterministic manifest assigns whole missions by planned wave/mode count,
longest first, to the least-loaded shard. The current catalog has 2,628 eligible
cases (including all 122 reverse cases); twelve shards receive 218–220 each. Both Bot Surge modes stay together.
The workflow's Run dialog can select 6, 12, 18 or 20 shards per environment and
12, 18 or 20 concurrent jobs. The daily defaults are twelve shards per environment and twelve concurrent jobs;
standard [Free accounts allow twenty concurrent hosted jobs](https://docs.github.com/en/actions/reference/limits#job-concurrency-limits-for-github-hosted-runners) across workflows.
GitHub supplies a separate VM to each matrix job; these are not multiple game
servers competing for a single runner. More shards add fresh-install overhead,
so shards and concurrency are separate controls. Twelve concurrent workers leave
eight slots on a Free account for the existing CI and release builds.
The build uses production CI's Debian container and cache action, matching its
gzip cache format as well as the exact dependency key. GitHub includes compression
in a cache's version; a zstd-capable Ubuntu VM misses the gzip entry. The build then
checks the pinned build stamps and recompiles the plugins. If a pinned compiler download has been pruned, a cold build resolves that exact
SourceMod build number in the official branch history, checks its product version,
and builds its pinned SourcePawn submodule with the pinned AMBuild tool. It records
the source commits alongside the compiler; it never substitutes the latest compiler.
Python's packaging module replaces the removed pkg_resources version parser during
bootstrap, and old compiler warnings remain visible without becoming build errors.
The production Docker image is built once and distributed to all Docker shards.
It reuses the nightly/release BuildKit layers, checks base-image updates, and
builds cold when no cache exists. Cache export failures do not fail a valid build;
the report downloads measurement evidence and the canonical plan separately.
The build emits the complete canonical plan; aggregation checks every returned
shard against its assigned partition and reconstructs coverage from that full
plan. A missing or failed shard cannot shrink the denominator or become green.

Before installing TF2, jobs reclaim optional SDK space only until the game
volume has its required space plus a 5 GiB reserve, and
install Linux’s 32-bit runtime libraries. Linux runners prefer Ubuntu's existing
HTTPS archive over the slow Azure HTTP mirror, retaining both fallback mirrors.
Downloaded `.deb` files are cached per environment with a daily key; APT still
checks current repository versions and hashes. Complete downloads from a timed-out
attempt are retained, and cache failures do not block the checks. This cache holds
OS packages, not TF2: every game install/update and current-build check still runs. Docker hosts install only SteamCMD’s
compatibility libraries; the production image supplies the engine libraries. Host preparation is bounded at 30 minutes; game-wave
and retry budgets are unchanged. Cleanup refuses to run on a personal
or self-hosted machine. Disk checks require 35 GiB free for native installs and
45 GiB for Docker. A runner that cannot meet that check produces an explicit
host-setup error; it never silently drops community maps to fit.
Unexpected native server exits are recorded as crashes, including Windows fatal
status codes such as 0xc0000409. The checker restarts only its disposable server
and lets the probe reconnect. Before every mission load, the controller reapplies
and verifies speed, cheat enablement, headless wake-up and reverse support;
mission configuration then selects the appropriate team and bomb handling.
Steam metadata/login timeouts get at most three installation attempts with
staggered backoff inside the original two-hour setup budget. Each attempt keeps
its logs; bad pins, missing libraries and game failures are not network retries.
Reports show the captured installer cause, and explicitly say when no waves ran.
Docker's entrypoint can hit the same Steam timeout after the launcher installer
succeeds. Before readiness only, those specific metadata/login timeouts get up to
three logged startup attempts inside the original 15-minute readiness window.
Failed mod loads and engine crashes are not startup network retries. Crashes in
any retry log still fail the measurement.

First-screen results, retries, logs and the crash verdict are preserved. More than three exits in one minute stops recovery. Passing later cases
never clears a crash; this allows coverage to continue without disguising a
production failure or spending hours retrying a dead server.

Windows uses the roomiest available volume, so a small workspace drive does
not waste space reclaimed on the system drive.

GitHub-hosted jobs have a [six-hour execution limit](https://docs.github.com/en/actions/reference/limits).
Each measurement job is capped at 350 minutes, with a 4.5-hour test/install
budget. Timeouts preserve partial results and remain failures. Twelve shards
are the default capacity choice. Adjust shard count and concurrency through the
dispatch inputs after inspecting a complete run.
Standard hosted runner availability and account-wide Actions limits still apply.

Every report identifies the exact tested source commit.
The report job publishes the full report to a dedicated prerelease, leaving
`releases/latest` alone. Its three badge JSON endpoints are also published in
one commit to `stability-badges`, served through `raw.githubusercontent.com`.
Shields blocks `github.com` release URLs as custom endpoints; raw-content URLs
are supported. The badge branch contains generated JSON only.

Actions defaults remain read-only; only the report job receives `contents: write`
to update the report assets and badge branch. No extra secrets or Steam account
are needed. Each full run is archived first under `stability-<run ID>-<attempt>`;
its summary links to that attempt's report, CSV and evidence, so previous runs
remain inspectable after the latest dashboard changes. These archive assets are
never overwritten. The rolling `stability` prerelease stays mutable for README
links. Actions artifacts are also retained for 30 days.

## Evidence and local use

The [full local validation report](audits/full-reverse-20261024/REPORT.md) records
the complete playable sweep and final reverse-objective validation. It includes
first-screen and extended-retry percentages, crash findings, and the recovered
missions with embedded navigation.

The prerelease contains `REPORT.md`, `stability.json`, three Shields endpoint
JSON files, and `stability-evidence.zip`. The ZIP contains each environment's
full report, raw plan and screening/retest JSONL, startup/final plugin and
extension lists, server logs, SourceMod logs and TF2 manifest/version files.
The Actions run has the same evidence retained for 30 days, so previous daily
measurements remain available even when the latest assets are replaced.

Check out the repository’s `main` first. Build the beds with `BOTS_EXTENSIONS_DIR=/path/to/extensions make stability-bed`.
Then, from the repository root, on the environment being measured:

```sh
export WAVEPROBE_RCONPW="$(openssl rand -hex 24)"
python3 deploy/stability-run.py --environment ubuntu24 --bed dist/linuxbed \
  --root /tmp/tf2ap-stability-fresh --output /tmp/stability-result
```

Use `windows` and `dist/winbed` on Windows. For Docker, build the production
image with the pins as the workflow does, then use `--environment docker`
with `--image tf2ap-stability:current`. The root must not already exist. The
output directory must be empty, to keep previous passes out of a new run. The
script stops its own server after reporting; local runs leave their disposable
files for inspection. The workflow removes only its recorded temporary root
after uploading results. External cancellation or loss of the runner may
require removing that run's temporary files manually.

To combine downloaded environment artifacts placed in `evidence/<environment>/part-<shard>`:

```sh
python3 deploy/stability-report.py evidence dashboard \
  --commit <tested-commit> --run-url <actions-run-url> --check-live \
  --shards 12 --plan <canonical-plan.jsonl> --partition <partition.json>
```

Missing environment artifacts still produce an explicit failing report and
badges. Build/setup failures therefore replace the previous dashboard instead
of leaving yesterday's green result looking like today's measurement.

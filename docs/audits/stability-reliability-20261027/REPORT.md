# Setup reliability — 2026-10-27 UTC

The successful game sweep (40211297573) and the later all-zero sweep
(40312880412) used the same SteamCMD client 1792617903 and TF2 build 11138802.
A git comparison between their source commits shows no installer, winbed or
version-pin changes. The successful logs have no app-info timeouts; the failed
logs have repeated metadata/login timeouts. The logs establish changed external
request availability, not whether Valve or the runner network caused it.

[Hosted smoke 40433350366](https://github.com/kelly-cs/tf2-archipelago/actions/runs/40433350366)
now passes startup and all **14/14** Decoy Expert cases on both Windows and Docker,
with the latest build, limiter and crash checks verified. Both installs succeeded
on their first harness attempt. This demonstrates recovered hosted connectivity;
it does not prove retries caused that recovery. Native Ubuntu remained in its
package-install phase when these two measurements completed. Exact provenance
and timings are in `hosted-smoke.json`.

The original design amplified dependency failures: each shard fetched OS packages
again, and Docker's entrypoint performed a second Steam update with no startup
retry. Native Ubuntu's Azure mirror could trickle downloads long enough to hit
the 20-minute deadline without triggering the per-request idle timeout.

Linux setup now prefers the existing Ubuntu HTTPS archive while retaining Azure
and security fallback mirrors. The source list and archive signing keys are
unchanged. The mirror-list format comes from GitHub's official
[runner configuration](https://github.com/actions/runner-images/blob/main/images/ubuntu/scripts/build/configure-apt-sources.sh).
Complete package downloads are cached across shards and daily runs; current APT
metadata, package versions and hashes are still checked. Cache restore/save is
optional, and even a failed APT attempt can preserve complete package downloads.
No TF2 game content or stale game build is substituted. The image job also
reuses the production nightly/release BuildKit cache, checks base-image updates,
and allows cache export failure without failing an otherwise valid image build.

Docker startup now retries only observed Steam metadata/login failures before
readiness, keeping every attempt and the original readiness deadline. Engine
crashes and failed mod loads remain failures; fatal errors in later startup logs
are scanned and retain their last observed case context. The child-process
integration test exercises both a failed install and a failed Docker startup,
then verifies startup, current build, game results, final health and limiter.

Validation: 72 Python tests, workflow actionlint, Python syntax and whitespace
checks pass. Regression tests cover preserved partial package caches, cache path
confinement, mirror priorities and crashes after a Docker startup retry. The follow-up [hosted smoke 40435744059](https://github.com/kelly-cs/tf2-archipelago/actions/runs/40435744059)
is green on all three environments: **42/42** cases. Startup, latest build
11138802, final health, crash and production limiter checks all pass. Ubuntu
host setup completed in **52 seconds**, Docker in **21 seconds**, versus the
previous native Ubuntu 20-minute package timeout. Both Linux package caches
saved successfully (90,219,514 bytes Ubuntu; 1,015,202 bytes Docker), ready for
later shards/runs. Exact timing and source provenance are recorded in
`verified-smoke.json`. This smoke verifies one standard mission per environment,
not a full-catalog stability score.

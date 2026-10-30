# Setup failure investigation — 2026-10-26

[Run 40312880412](https://github.com/kelly-cs/tf2-archipelago/actions/runs/40312880412)
built successfully but tested no waves. Its 0% values are zero passed / 2,628
planned, not 2,628 observed wave losses per environment.

| Environment | Steam setup failed | Host package setup timed out | Waves tested |
| --- | ---: | ---: | ---: |
| Ubuntu 24.04 | 11 | 1 | 0 |
| Windows | 12 | 0 | 0 |
| Docker | 6 | 6 | 0 |

All 29 Steam-related setup failures contain metadata/login timeouts: 28 game
installs and one Docker entrypoint update after installation succeeded. Most show
`Info request for AppId 7 returned error Timeout` followed by `Failed to load
platform info`; one Windows runner timed out requesting AppId 232250, and one
Docker runner also timed out waiting for user info. SteamCMD exited 5 or 8.
The missing tf2_waveprobe.smx path is secondary: winbed tried to finish probe
installation even though installing the game had failed.

The seven other failures hit the bounded 20-minute apt install. One Ubuntu
runner was still fetching i386 packages, including the 35.6 MB libllvm20 package,
when terminated. Docker was unnecessarily requesting the native engine's SDL/
graphics dependency tree on the host as well as in its production image.

A separate fresh official SteamCMD bootstrap, using the same client version
1792617903, successfully logged in anonymously and fetched app 232250 metadata
locally. No game files were downloaded. This establishes local reachability at
the time of the check, not that GitHub-hosted connectivity has recovered or that
a Valve-wide outage was the cause.

Changes: preserve and report the installer cause instead of a generic command
exit; label runs with zero tested cases as setup failures; retry only observed
Steam network timeouts up to three times with staggered backoff within the
existing setup deadline; retain every attempt's logs; stop winbed immediately
after failed game installation; install only SteamCMD compatibility libraries
on Docker hosts while keeping all native Ubuntu engine dependencies.

Validation: 68 Python tests pass, including a real child-process first-install
failure followed by recovery, persistent service failure, deadline exhaustion,
non-retryable dependency errors and extraction of the captured Linux/Windows
log formats. Full Go race/shuffle tests, Go lint, Windows winbed cross-build,
workflow lint and whitespace checks pass. No game failure was reclassified as
a pass, no stale game install was substituted, and the robot limiter is unchanged.

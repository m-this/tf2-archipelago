# Stability investigation — 2026-10-26

Evidence: [full sweep 40211297573](https://github.com/kelly-cs/tf2-archipelago/actions/runs/40211297573),
TF2 ServerVersion 11138802. Measurement time and tested commit are retained in
`outcomes.json`; reformatting this evidence does not represent a new game run.

| Environment | Passed / planned | Percentage | Final failed cases | Missing |
| --- | ---: | ---: | ---: | ---: |
| Ubuntu 24.04 | 2574 / 2628 | 97.95% | 54 | 0 |
| Windows | 1547 / 2628 | 58.87% | 952 | 129 |
| Docker | 2350 / 2628 | 89.42% | 58 | 220 |

Docker shard 4 exhausted its 20-minute host package-install deadline before
starting the game. Its 220 assigned cases remain in the denominator. The other
eleven shards started. This is a runner setup failure, not evidence that every
Docker startup failed.

Ubuntu and Docker both recorded engine segfaults in shards 1 and 11. Last
observed contexts include Skeleclipse Vicious Delicious wave 5 and transitions
after Bigrock Bionic Breach / Skeleclipse Spooky Sprockets. These are context,
not established causes. The captured logs do not resolve the underlying fault.
Other Linux nonpasses include authored loss events, timeouts, rejected map
transitions and missing reverse completion witnesses; all appear in the CSV.

Windows recorded 49 unexpected wrapper exits, including fatal 0xc0000409 and
SigMod access-violation diagnostics (one at server.dll+0x605b6a). It also has
657 final probe errors, 138 load blocks and 86 timeouts. The underlying native
crashes are unresolved. A separate harness defect amplified them: after the
first restart in shard 1, logs contain 224 wave-jump commands but zero commands
restoring host_timescale 50, sv_cheats, reverse support or wake-up. Recorded
180-second timeouts then contain roughly 180 game seconds, rather than 50×
acceleration. The controller only prepared the server once per process.

The controller now reapplies and verifies its fixture before every mission load,
including recovery loads, before mission-specific team/bomb settings. This fixes
lost test configuration after a restart; it does not remove crashes or claim to
fix the game. A controlled local restart trial is recorded separately below.

Reporting now always computes passed / planned when counts are valid; health
checks independently retain failing verdicts and red badges. The compact release
summary links full per-environment case tables and an untruncated CSV. Missing
cases and failures recovered on retry remain visible. Crash table rows name the
last observed case, with an explicit context label rather than causal attribution.

The controlled local trial played all 26 cases of Decoy Expert and reverse
Breaking Walls at 50×, both Bot Surge modes, with an intentional server restart
after the first passing record. The interruption remains a failed case; the
other **25/26** passed, including all twelve reverse cases. Total controller
time was **285.97 seconds**. Post-restart command logs and final RCON evidence
verify 50×, wake-up, cheat enablement, reverse support and the enabled production
robot limiter. The isolated server was stopped afterward. This is a recovery
experiment, not a replacement production stability score or native Windows proof.

Validation: 60 Python reporting/harness tests, the full Go race/shuffle suite,
Go lint (zero issues), workflow actionlint, whitespace checks and Windows
controller cross-compilation pass.

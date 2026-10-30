# CI recovery validation — 2026-10-25

The Docker rows in run 40144992430 did not test any waves: their independent
image builds requested a pruned SourceMod 1.12.0-git7164 archive (HTTP 404).
A cold source recovery resolved that exact official SourceMod revision and its
SourcePawn submodule, built the compiler, and compiled all four production bot
plugins. `compiler-source.json` records the resolved identities.

The executable `.code` sections of all four plugins match those built with the
previously cached official compiler. Full SMX files are not byte-identical:
compiled date/time metadata and the defender plugin's dynamic-memory estimate
in the data header differ. `bytecode-equivalence.json` records the comparison;
this is compiler validation, not another wave sweep or a replacement score.

All 58 deploy Python tests passed, including real child-process exit/restart,
crash-loop bounds, Windows fatal-status classification, canonical plan coverage,
and merged-history build-number resolution. Shell/Python syntax checks,
workflow actionlint and git whitespace checks also passed.

Windows `winbed: exit status 0xc0000409` is a genuine server failure in the hosted
run. The old checker missed that signature and kept retrying a dead server.
The new supervisor records unexpected exits before restarting its own disposable
server, preserves the crash verdict, and stops repeated crash loops. It does not
claim to fix the underlying game crash.

The complete production Docker image also built successfully from a cold bot
dependency state with the workflow's pinned build arguments. Its identity is
recorded in `production-image.txt`. An isolated Linux server used those extracted
production assets over this task's existing game layers; SteamCMD confirmed the
latest public game was installed (ServerVersion 11138802). Installed plugin
hashes matched the new staged assets. Wave 1 of Decoy Expert and reverse Breaking
Walls passed in both Bot Surge modes at 50×: **4/4 cases**, 73.69 seconds including
startup (5.38 seconds standard, 23.67 seconds reverse). Final health and the
production robot limiter passed. This targeted smoke is not full-catalog or
native Windows validation. Raw server logs remain local; curated evidence is
committed here. Only this task's isolated server was stopped afterward.

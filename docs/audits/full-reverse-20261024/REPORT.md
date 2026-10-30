# Complete catalog stability validation — 2026-10-24

The production plan now includes **all 2,628 catalog wave/mode cases**, including **122 reverse cases across twelve missions**, with Bot Surge off and on. Every catalog case has been attempted locally. There are **zero unsupported reverse cases and zero navigation exclusions** in the current full plan. Coverage means attempted and reported; it does not mean every case passes.

The original full sweep recorded an engine segmentation fault. Its verdict is **failed**, regardless of the numerical completion percentage. Native Ubuntu and Windows runtime results come from the hosted sweep; a local Linux Docker run and Windows cross-build do not prove those environments.

## Engine results

| Cohort | Passed / planned | Completion | Tested |
| --- | ---: | ---: | ---: |
| Original 2,454 cases: first screen | 2393 / 2454 | 97.51% | 2454 |
| Original 2,454 cases: after retries | 2419 / 2454 | 98.57% | 2454 |
| Final probe: 112 reverse cases, first screen | 92 / 112 | 82.14% | 112 |
| Final probe: 112 reverse cases, after retries | 93 / 112 | 83.04% | 112 |
| Recovered 174 cases: first screen | 161 / 174 | 92.53% | 174 |
| Recovered 174 cases: after retries, initial Arctic profile | 162 / 174 | 93.10% | 174 |
| Corrected Arctic profile at 20× | 10 / 10 | 100.00% | 10 |
| Corrected Arctic profile at 50× | 10 / 10 | 100.00% | 10 |

All **122 reverse cases** were exercised with the final frozen native probe across the 112-case reverse proof and corrected 50× Arctic proof: **103/122 (84.43%)** passed after retries. The [coverage index](coverage.json) checks that the original 2,454 cases and recovered 174 cases are disjoint and exactly cover the current canonical plan.

The original sweep used twelve isolated Docker servers at requested 50×, seed 1, 180-second screens and 900-second retries. All 61 nonpasses received an extended retry. Its first screen took about 1 hour 11 minutes; same-map timeout retries dominated the tail. The final 112-case reverse validation used three fresh workers and took 26 minutes 14 seconds including retries. The recovered-mission cohort used five fresh workers and took 28.1 minutes including retries. All used actual TF2 ServerVersion **11138802**, corroborated against Valve after testing, with the production robot limiter enabled.

The original full sweep started while reverse handling was being refined. Later changes affected reverse event ordering, wrench input and cleanup witnesses, preserving standard-wave behavior and fixtures. The cohorts are shown separately rather than advertising a mixture as a uniform final-build full-catalog score. The final reverse proof and corrected Arctic trials use the same immutable native probe. Arctic's controller selects native point objectives on waves 1–3 and 5, and native combat/tank victory on wave 4. Its independent report requires all five enemy tank deaths, observed enemy defeats, native BLU victory and wave completion; a bomb capture cannot substitute. The 20× and 50× verified trials each passed all ten cases. The original Arctic failures remain in the recovered cohort's evidence. Complete Linux/Windows production beds generate the exact 2,628-case plan and the probe's executable code matches the frozen native build; only compiler timestamps and debug paths differ. See [build validation](build-validation.json) and the [cohort provenance](cohort-provenance.json). The first run records its original base commit and frozen binaries; it is not presented as a rebuild of the final source commit.

Per-case evidence: [original full sweep](full-cases.csv), [final reverse run](final-reverse-cases.csv), [recovered missions](restored-cases.csv), and the corrected Arctic profile at [20×](arctic-20-cases.csv) and [50×](arctic-50-cases.csv). First screens and after-retry JSON summaries preserve failures. The [full-sweep crash scan](engine-crash-scan.json), [final reverse crash scan](reverse-engine-crash-scan.json), [recovered-cohort crash scan](restored-engine-crash-scan.json), and [full worker provenance](full-worker-provenance.json) retain crash and build/limiter findings. Raw plans, first-pass/retry JSONL, debug snapshots, worker logs and plugin hashes remain in the original workspace audit directory; private RCON credentials and raw startup logs are not committed.

## Navigation and cached installs

The audit found that all four maps previously classified `no_nav` actually ship navigation in their BSP pak lumps: Bogland uses `maps/embed.nav`; Cyberia, Decompose and Mansion use their map-named `.nav` files. [Valve's native loader](https://github.com/ValveSoftware/source-sdk-2013/blob/master/src/game/server/nav_file.cpp) checks both locations. The catalog generator now follows that behavior, retains mission IDs and names, and recomputes the correct SigMod requirement. This restores [16 missions and 174 cases](restored-missions.json), including Cyberia's ten reverse cases. The [embedded-file inventory](embedded-navigation.json) records the actual shipped assets. No maps or navigation meshes were generated, copied out, renamed or modified to make them pass.

An initial direct supplemental trial bypassed the normal local runner's community-spec cache guard and used an image populated from the old catalog, so the recovered populations were absent. Those load-failure logs remain under `restored-nav-final/INVALID-FIXTURE.txt` in the original workspace and are not game stability measurements. That also exposed an installer receipt bug: unchanged ZIP/mod stamps could skip newly eligible catalog files. Receipts now include a catalog identity; a real ZIP extraction regression proves that a catalog change restores a previously excluded population. Both validated packs were re-extracted through the corrected production installer into a new immutable image before the recovered cohort ran.

## Final mission transitions

The first corrected Arctic trials passed all ten game cases at each speed but their immediate final RCON readback collided with natural map rotation. Their logs showed no engine crash, and their first results/postcheck errors are retained locally. The production harness now waits for healthy mods plus completed map configs, bounded by its final readiness deadline, before collecting final versions and limiter evidence. Persistent unhealthy state still fails; earlier crash logs still make the verdict red. Both speed trials were repeated on fresh workers and completed the final health, version and plugin-hash checks. Final installed/running versions are also compared with the live build, so a mid-run replacement cannot be reported as a current-build result.

## Remaining failures and fixture limits

Failures stay in the percentage. Encounter records native losses; Shogun control-point routes fail under the probe; Robotfactory repair wave 1 does not create its expected repair buildings in this fixture, and some route cases fail. Galois's combat finale can complete without the required observed BLU victory, which remains a probe error. Cyberia's initial generic bomb fixture recorded losses/missing witnesses because its authored objectives are points, tanks and a timed escape. The corrected native profile passes all ten cases at both tested speeds, without changing those authored stages. These observations alone do not prove that a human cannot complete the mission.

Separate [component controls](component-controls-summary.json) verified native wrench repair at 20× and 50× and ordered five-target repair plus authored completion at 50×. Their synthetic buildings are **not mission passes** and are excluded from every score and case CSV. Clean-load real repair-mission checks still failed, including a 1× control. No winning relays, capture outputs, building health, robot limits or mission timers were altered to convert mission failures into passes.

Teleporting the fake player into native objective volumes tests authored touch/capture rules, not human walking routes, survival, multiworld connectivity or every strategy. Reverse reports independently require the mission-specific native completion witness. Missing evidence counts against the percentage.

## Build and harness validation

- Full Go race/shuffle suite; 54 Python harness/report/retry tests; 213 apworld tests and apworld lint.
- Go lint with zero issues, pinned formatting, Go fix gate, actionlint, shell syntax and diff checks.
- BSP pak name/bounds/corruption and catalog-identity regressions; ZIP cache restoration regression; catalog-dependent launcher availability tests.
- Complete Linux/Windows production stability-bed builds; canonical plan and native/packaged probe equivalence checks.
- Staged/installed probe and AP hashes, native engine versions and production robot-limiter readbacks; post-run freshness evidence in [full](full-game-build.json), [reverse](reverse-game-build.json), and [recovered](restored-game-build.json) build records.

The defender-bot pin requires SourceMod 1.12.0.7164, whose official archive returned 404 during validation. Local builds used the verified exact-version compiler already present on this machine. Hosted builds restore production CI's exact dependency-cache key, verify pin stamps and rebuild; no alternative compiler pin is substituted. A cold cache still requires the pinned upstream artifact.

## Hosted build cache format

The [first new full hosted attempt](https://github.com/kelly-cs/tf2-archipelago/actions/runs/40143779285) stopped before measurements: the visible bots cache key matched, but GitHub's hidden cache version includes compression. Production CI created a gzip entry in its Debian container; the stability build's Ubuntu VM searched for a zstd entry and then reached the missing pinned compiler download. The build now uses the same Debian tools and cache action as production CI. The [cache-format validation](cache-format-validation.json) reproduces the stored version hash exactly for gzip and confirms the zstd version differs. The complete Linux/Windows beds are also rebuilt locally in that Debian environment. Compiler, mod and game pins are unchanged; a cold cache still requires the original pinned compiler artifact.

The [cache-compatible attempt](https://github.com/kelly-cs/tf2-archipelago/actions/runs/40144449522) confirmed the exact cache hit and compiled all pinned defender plugins. It then exposed the container ownership check during Go VCS stamping. The workflow now trusts the exact physical checkout path, as existing CI already does. A local root/container fixture reproduces the refusal before that step and verifies `-buildvcs=true` embeds the actual revision afterwards; VCS stamping remains enabled.

The build tools explicitly include Python 3 for partition generation. The canonical-plan, partition and matrix-output commands were run locally in that exact Debian tool environment: all 2,628 cases match, with six 220-case and six 218-case shards. The setup-only attempt without Python was superseded before measurements.

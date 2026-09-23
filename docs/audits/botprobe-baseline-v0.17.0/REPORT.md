# Defender bot run

```
speed=4
waves=1
mission=all
timeout=240s
bots_smx=/home/mathis/projects/tf2-archipelago/deploy/bots/build/package/addons/sourcemod/plugins/tf2_defenderbots.smx
bots_sha256=a33f109caad4af9ebe0c956aeaf2677da7158b6f199fad9a13ff9e6489cf7465
image=tf2ap-srcds:v117
started_utc=2026-09-23T11:36:48Z
source_commit=ed261386ef51cf6a77b51dd981ea51ee2c1d56bf
```

48 waves, 156 bot records. Never left spawn: 0. Bots with a fault: 86. Median time to leave spawn: 5.5s.

Rescues in the mod's log during the waves: 209 spawn recovery, 23 stuck, 5 wedge teleport.

| Mission | Rescues | Spawn recoveries by bot |
| --- | --- | --- |
| mvm_akure_rc1a_adv_gold_and_grit | 22 | RageQuit 6, The Administrator 4, GutsAndGlory! 4, LOS LOS LOS 4, CrySomeMore 3, Grim Bloody Fable 1 |
| mvm_area_52_rc3_adv_australian_christmas | 12 | Aperture Science Prototype XR7 3, SomeDude 3, LOS LOS LOS 3, CryBaby 1, GutsAndGlory! 1, Poopy Joe 1 |
| mvm_bigrock | 6 | Hat-Wearing MAN 2, The G-Man 2, CreditToTeam 1, It's Filthy in There! 1 |
| mvm_bloodlust_b6_adv_reanimation | 6 | GutsAndGlory! 3, BoomerBile 1, SomeDude 1 |
| mvm_bronx_rc2_adv_point_of_impact | 1 | The Freeman 1 |
| mvm_coaltown | 5 | 0xDEADBEEF 3, Mann Co. 1, Tiny Baby Man 1 |
| mvm_condemned_b3_adv_unholy_undead | 27 | Still Alive 6, ZAWMBEEZ 5, Headful of Eyeballs 5, LOS LOS LOS 4, AimBot 4, ThatGuy 3 |
| mvm_deathpour_rc1_exp_blackout | 12 | Hostage 5, Numnutz 4, Hat-Wearing MAN 1 |
| mvm_frostwynd_rc1_adv_fiefdom_fiasco | 14 | Numnutz 5, AimBot 4, The Combine 2, THEM 2, GLaDOS 1 |
| mvm_ghost_town_666 | 7 | Chell 3, Still Alive 2, Me 1, Delicious Cake 1 |
| mvm_memorial_b1_exp_necropolis | 13 | Poopy Joe 1, Crowbar 1, RageQuit 1 |
| mvm_nightsky_rc4d_adv_nightsky_nightmare | 12 | Archimedes! 3, LOS LOS LOS 3, RageQuit 3, Divide by Zero 1, I LIVE! 1, CrySomeMore 1 |
| mvm_oilrig_rc5d_adv_666_oilpocalypse | 51 | The G-Man 11, It's Filthy in There! 10, CreditToTeam 7, Hat-Wearing MAN 4, Black Mesa 3, HI THERE 2 |
| mvm_oxidize_rc3_adv_666_corroding_cadavers | 3 | H@XX0RZ 2, BeepBeepBoop 1 |
| mvm_oxidize_rr18_adv_grass_run | 1 |  |
| mvm_radar_b10_adv_rocky_ravage | 1 | It's Filthy in There! 1 |
| mvm_robotfactory_b30_adv_frantic_flood | 5 | Nom Nom Nom 2, SomeDude 1, Soulless 1, Me 1 |
| mvm_rottenburg | 7 | Chucklenuts 2, Glorified Toaster with Legs 2, trigger_hurt 2, THEM 1 |
| mvm_seabed_b6_adv_ocean_commotion | 3 | Kill Me 2, A Professional With Standards 1 |
| mvm_skeleclipse_b7a_adv_vicious_delicious | 29 | CrySomeMore 6, Archimedes! 6, Target Practice 5, Chucklenuts 5, LOS LOS LOS 4, RageQuit 3 |

| Mission | Wave | Outcome | Bot | Class | Fault |
| --- | --- | --- | --- | --- | --- |
| mvm_akure_rc1a_adv_gold_and_grit | 1 | passed | CrySomeMore | demoman | teleported 1x |
| mvm_akure_rc1a_adv_gold_and_grit | 1 | passed | RageQuit | spy | teleported 6x |
| mvm_akure_rc1a_adv_gold_and_grit | 1 | passed | Grim Bloody Fable | sniper | teleported 1x |
| mvm_akure_rc1a_adv_gold_and_grit | 1 | passed | GutsAndGlory! | pyro | teleported 3x |
| mvm_akure_rc1a_adv_gold_and_grit | 1 | passed | LOS LOS LOS | heavy | teleported 3x |
| mvm_akure_rc1a_adv_gold_and_grit | 1 | passed | The Administrator | pyro | teleported 3x |
| mvm_area_52_rc3_adv_australian_christmas | 1 | passed | SomeDude | demoman | teleported 1x |
| mvm_area_52_rc3_adv_australian_christmas | 1 | passed | GutsAndGlory! | heavy | still 119s at -936,1963,-414; teleported 1x |
| mvm_area_52_rc3_adv_australian_christmas | 1 | passed | CryBaby | scout | teleported 1x |
| mvm_area_52_rc3_adv_australian_christmas | 1 | passed | LOS LOS LOS | demoman | teleported 2x |
| mvm_area_52_rc3_adv_australian_christmas | 1 | passed | Aperture Science Prototype XR7 | spy | still 45s at -936,1963,-414; teleported 1x |
| mvm_area_52_rc3_adv_australian_christmas | 1 | passed | Poopy Joe | sniper | teleported 1x |
| mvm_bigrock | 1 | passed | The G-Man | demoman | a spawn exit took 20s over 1 lives; teleported 2x |
| mvm_bigrock | 1 | passed | Hat-Wearing MAN | pyro | teleported 1x |
| mvm_bigrock | 1 | passed | HI THERE | soldier | still 31s at -927,4250,134 |
| mvm_bigrock | 1 | passed | It's Filthy in There! | scout | teleported 1x |
| mvm_bloodlust_b6_adv_reanimation | 1 | wave timed out | BoomerBile | pyro | teleported 1x |
| mvm_bloodlust_b6_adv_reanimation | 1 | wave timed out | SomeDude | demoman | teleported 1x |
| mvm_bloodlust_b6_adv_reanimation | 1 | wave timed out | GutsAndGlory! | demoman | teleported 3x |
| mvm_bronx_rc2_adv_point_of_impact | 1 | passed | The Freeman | heavy | teleported 1x |
| mvm_coaltown | 1 | passed | Tiny Baby Man | heavy | teleported 1x |
| mvm_coaltown | 1 | passed | Mann Co. | demoman | teleported 1x |
| mvm_coaltown | 1 | passed | 0xDEADBEEF | scout | teleported 1x |
| mvm_condemned_b3_adv_unholy_undead | 1 | passed | ThatGuy | scout | teleported 1x |
| mvm_condemned_b3_adv_unholy_undead | 1 | passed | LOS LOS LOS | pyro | teleported 4x |
| mvm_condemned_b3_adv_unholy_undead | 1 | passed | Headful of Eyeballs | sniper | teleported 3x |
| mvm_condemned_b3_adv_unholy_undead | 1 | passed | Still Alive | scout | teleported 2x |
| mvm_condemned_b3_adv_unholy_undead | 1 | passed | ZAWMBEEZ | spy | teleported 4x |
| mvm_condemned_b3_adv_unholy_undead | 1 | passed | AimBot | demoman | teleported 3x |
| mvm_deathpour_rc1_exp_blackout | 1 | passed | Numnutz | medic | teleported 4x |
| mvm_deathpour_rc1_exp_blackout | 1 | passed | Hat-Wearing MAN | spy | still 76s at 545,-4775,-157; teleported 1x |
| mvm_deathpour_rc1_exp_blackout | 1 | passed | Hostage | medic | teleported 5x |
| mvm_frostwynd_rc1_adv_fiefdom_fiasco | 1 | passed | The Combine | pyro | teleported 1x |
| mvm_frostwynd_rc1_adv_fiefdom_fiasco | 1 | passed | AimBot | heavy | teleported 4x |
| mvm_frostwynd_rc1_adv_fiefdom_fiasco | 1 | passed | GLaDOS | pyro | teleported 1x |
| mvm_frostwynd_rc1_adv_fiefdom_fiasco | 1 | passed | THEM | demoman | teleported 2x |
| mvm_frostwynd_rc1_adv_fiefdom_fiasco | 1 | passed | Numnutz | soldier | teleported 5x |
| mvm_ghost_town_666 | 1 | wave timed out | Still Alive | heavy | teleported 1x |
| mvm_ghost_town_666 | 1 | wave timed out | The Freeman | demoman | a spawn exit took 24s over 2 lives; still 36s at -41,1492,282 |
| mvm_ghost_town_666 | 1 | wave timed out | Me | pyro | still 36s at -383,2309,239; teleported 1x |
| mvm_ghost_town_666 | 1 | wave timed out | Chell | scout | still 34s at 47,2087,192; teleported 1x |
| mvm_hideout_b3_adv_advanced | 1 | passed | BoomerBile | scout | still 35s at -3329,2017,513 |
| mvm_hideout_b3_adv_advanced | 1 | passed | AimBot | soldier | still 38s at -3757,1939,512 |
| mvm_hideout_b3_adv_advanced | 1 | passed | The Freeman | pyro | still 32s at -4455,1129,576 |
| mvm_memorial_b1_exp_necropolis | 1 | passed | Crowbar | engineer | a spawn exit took 49s over 1 lives; teleported 1x |
| mvm_memorial_b1_exp_necropolis | 1 | passed | Poopy Joe | soldier | teleported 1x |
| mvm_memorial_b1_exp_necropolis | 1 | passed | RageQuit | engineer | teleported 1x |
| mvm_nightsky_rc4d_adv_nightsky_nightmare | 1 | passed | Archimedes! | pyro | teleported 1x |
| mvm_nightsky_rc4d_adv_nightsky_nightmare | 1 | passed | LOS LOS LOS | demoman | teleported 1x |
| mvm_nightsky_rc4d_adv_nightsky_nightmare | 1 | passed | CrySomeMore | engineer | teleported 1x |
| mvm_nightsky_rc4d_adv_nightsky_nightmare | 1 | passed | RageQuit | soldier | teleported 1x |
| mvm_nightsky_rc4d_adv_nightsky_nightmare | 1 | passed | Divide by Zero | spy | still 49s at -4696,-2115,148; teleported 1x |
| mvm_nightsky_rc4d_adv_nightsky_nightmare | 1 | passed | I LIVE! | engineer | teleported 1x |
| mvm_oilrig_rc5d_adv_666_oilpocalypse | 1 | wave timed out | CreditToTeam | demoman | teleported 4x |
| mvm_oilrig_rc5d_adv_666_oilpocalypse | 1 | wave timed out | Black Mesa | engineer | teleported 1x |
| mvm_oilrig_rc5d_adv_666_oilpocalypse | 1 | wave timed out | The G-Man | soldier | teleported 7x |
| mvm_oilrig_rc5d_adv_666_oilpocalypse | 1 | wave timed out | Hat-Wearing MAN | spy | still 58s at -1359,-4247,1024; teleported 3x |
| mvm_oilrig_rc5d_adv_666_oilpocalypse | 1 | wave timed out | HI THERE | engineer | teleported 2x |
| mvm_oilrig_rc5d_adv_666_oilpocalypse | 1 | wave timed out | It's Filthy in There! | soldier | teleported 7x |
| mvm_oxidize_rc3_adv_666_corroding_cadavers | 1 | wave timed out | H@XX0RZ | medic | teleported 2x |
| mvm_oxidize_rc3_adv_666_corroding_cadavers | 1 | wave timed out | BeepBeepBoop | medic | teleported 1x |
| mvm_oxidize_rr18_adv_grass_run | 1 | passed | The G-Man | soldier | a spawn exit took 21s over 4 lives |
| mvm_radar_b10_adv_rocky_ravage | 1 | passed | CreditToTeam | demoman | a spawn exit took 23s over 2 lives |
| mvm_radar_b10_adv_rocky_ravage | 1 | passed | Hat-Wearing MAN | spy | still 38s at 2995,867,-333 |
| mvm_radar_b10_adv_rocky_ravage | 1 | passed | It's Filthy in There! | soldier | teleported 1x |
| mvm_redstone_ridge_rc5_adv_armored_apparatus | 1 | passed | GLaDOS | pyro | a spawn exit took 22s over 3 lives |
| mvm_redstone_ridge_rc5_adv_armored_apparatus | 1 | passed | Divide by Zero | soldier | a spawn exit took 25s over 3 lives |
| mvm_redstone_ridge_rc5_adv_armored_apparatus | 1 | passed | Ribs Grow Back | spy | still 30s at -428,3696,-151; teleported 1x |
| mvm_robotfactory_b30_adv_frantic_flood | 1 | passed | Soulless | heavy | teleported 1x |
| mvm_robotfactory_b30_adv_frantic_flood | 1 | passed | SomeDude | medic | teleported 1x |
| mvm_robotfactory_b30_adv_frantic_flood | 1 | passed | Nom Nom Nom | demoman | teleported 1x |
| mvm_rottenburg | 1 | passed | trigger_hurt | medic | teleported 2x |
| mvm_rottenburg | 1 | passed | THEM | heavy | teleported 1x |
| mvm_rottenburg | 1 | passed | Glorified Toaster with Legs | demoman | teleported 1x |
| mvm_rottenburg | 1 | passed | Chucklenuts | scout | teleported 1x |
| mvm_seabed_b6_adv_ocean_commotion | 1 | wave timed out | Kill Me | pyro | teleported 2x |
| mvm_seabed_b6_adv_ocean_commotion | 1 | wave timed out | A Professional With Standards | demoman | still 48s at 334,-1488,96; teleported 1x |
| mvm_sharp_rc9_adv_sudden_equinox | 1 | passed | DeadHead | spy | still 51s at -72,852,83 |
| mvm_sharp_rc9_adv_sudden_equinox | 1 | passed | Divide by Zero | heavy | still 32s at 8,535,96 |
| mvm_sharp_rc9_adv_sudden_equinox | 1 | passed | AimBot | spy | still 38s at -413,415,92 |
| mvm_skeleclipse_b7a_adv_vicious_delicious | 1 | passed | Target Practice | heavy | teleported 4x |
| mvm_skeleclipse_b7a_adv_vicious_delicious | 1 | passed | Archimedes! | spy | teleported 4x |
| mvm_skeleclipse_b7a_adv_vicious_delicious | 1 | passed | Chucklenuts | heavy | teleported 4x |
| mvm_skeleclipse_b7a_adv_vicious_delicious | 1 | passed | LOS LOS LOS | sniper | teleported 2x |
| mvm_skeleclipse_b7a_adv_vicious_delicious | 1 | passed | CrySomeMore | soldier | teleported 3x |
| mvm_skeleclipse_b7a_adv_vicious_delicious | 1 | passed | RageQuit | sniper | teleported 2x |

Waves without a defender record:

- mvm_autumnull_rc2_adv_dynamic_disaster wave 0: load blocked population file mvm_autumnull_rc2_adv_dynamic_disaster was rejected: Could not find a valid population file matching: mvm_autumnull_rc2_adv_dynamic_disaster.
- mvm_chateau_rc3_adv_remedic wave 1: wave failed game or probe failed wave 1: wave_failed
- mvm_decoy wave 0: load blocked timed out after 1m30s (last status {State: Reason: Map: Pop: Max:0 GameWave:0 Expected:0 Observed:0 Bots:0 Tanks:0 BotSpawns:0 TankSpawns:0 Attempts:0 Alive:0 Remaining:0 Initial:0 DefClass:0 DefTeam:0 PlayerTeam:0 EnemyTeam:0 Elapsed:0 Progress:0}): cannot read the reply: read tcp 127.0.0.1:46602->127.0.0.1:27045: i/o timeout
- mvm_downpour_rc3a_adv_666_last_stand_of_dead_men wave 0: load blocked population file mvm_downpour_rc3a_adv_666_last_stand_of_dead_men was rejected: Could not find a valid population file matching: mvm_downpour_rc3a_adv_666_last_stand_of_dead_men.
- mvm_heatrock_rc6a_adv_brain_taker wave 0: load blocked population file mvm_heatrock_rc6a_adv_brain_taker was rejected: Could not find a valid population file matching: mvm_heatrock_rc6a_adv_brain_taker.
- mvm_kelly_rc1b_adv_awakening wave 0: load blocked population file mvm_kelly_rc1b_adv_awakening was rejected: Could not find a valid population file matching: mvm_kelly_rc1b_adv_awakening.
- mvm_legerdemain_a6e_adv_midnight_patrol wave 0: load blocked changelevel mvm_legerdemain_a6e from mvm_kelly_rc1b/mvm_kelly_rc1b_adv_homestead_happenings: reply "", command error <nil>, final status error cannot read the reply: read tcp 127.0.0.1:39256->127.0.0.1:27045: i/o timeout: timed out after 1m30s (last status {State: Reason: Map: Pop: Max:0 GameWave:0 Expected:0 Observed:0 Bots:0 Tanks:0 BotSpawns:0 TankSpawns:0 Attempts:0 Alive:0 Remaining:0 Initial:0 DefClass:0 DefTeam:0 PlayerTeam:0 EnemyTeam:0 Elapsed:0 Progress:0}): cannot read the reply: read tcp 127.0.0.1:42416->127.0.0.1:27045: i/o timeout
- mvm_lotus_b6_adv_ledmotif wave 0: load blocked cannot read the reply: read tcp 127.0.0.1:39814->127.0.0.1:27045: i/o timeout
- mvm_mannhattan wave 0: load blocked cannot read the reply: read tcp 127.0.0.1:57406->127.0.0.1:27045: i/o timeout
- mvm_mannworks wave 0: load blocked cannot read the reply: read tcp 127.0.0.1:52976->127.0.0.1:27045: i/o timeout
- mvm_marsbase_rc5_adv_secret_struggle wave 0: load blocked changelevel mvm_marsbase_rc5 from mvm_legerdemain_a6e/mvm_legerdemain_a6e_adv_midnight_patrol: reply "", command error <nil>, final status error cannot read the reply: read tcp 127.0.0.1:56660->127.0.0.1:27045: i/o timeout: timed out after 1m30s (last status {State: Reason: Map: Pop: Max:0 GameWave:0 Expected:0 Observed:0 Bots:0 Tanks:0 BotSpawns:0 TankSpawns:0 Attempts:0 Alive:0 Remaining:0 Initial:0 DefClass:0 DefTeam:0 PlayerTeam:0 EnemyTeam:0 Elapsed:0 Progress:0}): cannot read the reply: read tcp 127.0.0.1:48888->127.0.0.1:27045: i/o timeout
- mvm_null_b9c_adv_baneful_harvest wave 0: load blocked population file mvm_null_b9c_adv_baneful_harvest was rejected: Could not find a valid population file matching: mvm_null_b9c_adv_baneful_harvest.
- mvm_sludge_b6_adv_claire_de_lune wave 0: load blocked changelevel mvm_sludge_b6 from mvm_skeleclipse_b7a/mvm_skeleclipse_b7a_adv_vicious_delicious: reply "", command error <nil>, final status error cannot read the reply: read tcp 127.0.0.1:42032->127.0.0.1:27045: read: connection reset by peer: timed out after 1m30s (last status {State: Reason: Map: Pop: Max:0 GameWave:0 Expected:0 Observed:0 Bots:0 Tanks:0 BotSpawns:0 TankSpawns:0 Attempts:0 Alive:0 Remaining:0 Initial:0 DefClass:0 DefTeam:0 PlayerTeam:0 EnemyTeam:0 Elapsed:0 Progress:0}): cannot read the reply: read tcp 127.0.0.1:38896->127.0.0.1:27045: i/o timeout
- mvm_smog_b3a_adv_respiration_restriction wave 0: load blocked cannot read the reply: read tcp 127.0.0.1:42040->127.0.0.1:27045: read: connection reset by peer
- mvm_snowpine_rc4_fix1_adv_permafrost_panic wave 0: load blocked cannot read the reply: read tcp 127.0.0.1:42046->127.0.0.1:27045: read: connection reset by peer
- mvm_spybase_rc8_exp_waters_of_a_robot_regime wave 0: load blocked cannot read the reply: read tcp 127.0.0.1:42050->127.0.0.1:27045: read: connection reset by peer
- mvm_teien_rc6_adv_onsen_onslaught wave 0: load blocked cannot read the reply: EOF
- mvm_terrorlict_final1c5_adv_accursed_aggrievocation wave 0: load blocked cannot read the reply: read tcp 127.0.0.1:42070->127.0.0.1:27045: read: connection reset by peer
- mvm_transmission_rc7a_adv_signal_shutdown wave 0: load blocked cannot read the reply: read tcp 127.0.0.1:42076->127.0.0.1:27045: read: connection reset by peer
- mvm_traumatic_b2a_adv_terrorstorm wave 0: load blocked cannot read the reply: read tcp 127.0.0.1:42090->127.0.0.1:27045: read: connection reset by peer
- mvm_villa_b13f_adv_forgotten wave 0: load blocked cannot read the reply: read tcp 127.0.0.1:42096->127.0.0.1:27045: read: connection reset by peer
- mvm_yiresa_rc5a_adv_mesa_malware wave 0: load blocked cannot read the reply: read tcp 127.0.0.1:42100->127.0.0.1:27045: read: connection reset by peer

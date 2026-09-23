# Defender bot run

```
speed=4
waves=1
mission=all
timeout=240s
bots_smx=/tmp/mvm-hatchfix/package/addons/sourcemod/plugins/tf2_defenderbots.smx
bots_sha256=7decfbb2a43a357ee3832f106734d2967379d58f71b6bda4726ed82e785feef2
image=tf2ap-srcds:v117
started_utc=2026-09-23T13:38:53Z
source_commit=ed261386ef51cf6a77b51dd981ea51ee2c1d56bf
```

47 waves, 264 bot records. Never left spawn: 1. Bots with a fault: 121. Median time to leave spawn: 5.7s.

Rescues in the mod's log during the waves: 261 spawn recovery, 35 stuck, 5 wedge teleport.

| Mission | Rescues | Spawn recoveries by bot |
| --- | --- | --- |
| mvm_akure_rc1a_adv_gold_and_grit | 15 | RageQuit 6, Still Alive 5, Divide by Zero 2, One-Man Cheeseburger Apocalypse 1, Kaboom! 1 |
| mvm_area_52_rc3_adv_australian_christmas | 11 | The Freeman 3, C++ 2, Humans Are Weak 2, Someone Else 2, CreditToTeam 1, Mann Co. 1 |
| mvm_bigrock | 3 | It's Filthy in There! 1, CreditToTeam 1, Hat-Wearing MAN 1 |
| mvm_bloodmoon_b8_adv_sed_de_sangre | 1 |  |
| mvm_bronx_rc2_adv_point_of_impact | 3 | H@XX0RZ 2, Crowbar 1 |
| mvm_coaltown | 3 | Mann Co. 1, Tiny Baby Man 1, 0xDEADBEEF 1 |
| mvm_condemned_b3_adv_unholy_undead | 13 | GLaDOS 3, HI THERE 2, AmNot 2, Pow! 1, Saxton Hale 1, Hostage 1 |
| mvm_deathpour_rc1_exp_blackout | 1 | Hat-Wearing MAN 1 |
| mvm_decoy | 5 | BeepBeepBoop 2, The Freeman 1, Black Mesa 1, GutsAndGlory! 1 |
| mvm_frostwynd_rc1_adv_fiefdom_fiasco | 6 | Hostage 2, A Professional With Standards 1, Aperture Science Prototype XR7 1, Humans Are Weak 1, Black Mesa 1 |
| mvm_ghost_town_666 | 4 | IvanTheSpaceBiker 2, CEDA 1, BoomerBile 1 |
| mvm_heatrock_rc6a_adv_heated_argument | 1 | Ribs Grow Back 1 |
| mvm_kelly_rc1b_adv_homestead_happenings | 94 | Black Mesa 47, HI THERE 42, CreditToTeam 2, The G-Man 1 |
| mvm_mannhattan | 14 | C++ 1 |
| mvm_marsbase_rc5_adv_secret_struggle | 6 | AimBot 1 |
| mvm_memorial_b1_exp_necropolis | 12 | MindlessElectrons 1, SomeDude 1, Tiny Baby Man 1 |
| mvm_nightsky_rc4d_adv_nightsky_nightmare | 13 | Archimedes! 3, Divide by Zero 3, LOS LOS LOS 3, RageQuit 2, I LIVE! 1, CrySomeMore 1 |
| mvm_null_b9c_adv_void_voyage | 1 |  |
| mvm_oilrig_rc5d_adv_666_oilpocalypse | 47 | The G-Man 14, It's Filthy in There! 13, CreditToTeam 8, Hat-Wearing MAN 7, Black Mesa 1, HI THERE 1 |
| mvm_oxidize_rc3_adv_666_corroding_cadavers | 7 | Kill Me 4, C++ 2, MoreGun 1 |
| mvm_oxidize_rr18_adv_grass_run | 1 | CreditToTeam 1 |
| mvm_radar_b10_adv_rocky_ravage | 1 | H@XX0RZ 1 |
| mvm_robotfactory_b30_adv_frantic_flood | 4 | Maggot 2, Nobody 1, LOS LOS LOS 1 |
| mvm_rottenburg | 4 | The Freeman 1, GutsAndGlory! 1, Black Mesa 1 |
| mvm_seabed_b6_adv_ocean_commotion | 2 | Humans Are Weak 1 |
| mvm_sharp_rc9_adv_sudden_equinox | 1 |  |
| mvm_smog_b3a_adv_respiration_restriction | 2 | Grim Bloody Fable 2 |
| mvm_snowpine_rc4_fix1_adv_permafrost_panic | 3 | H@XX0RZ 1, MindlessElectrons 1, Ribs Grow Back 1 |
| mvm_spybase_rc8_exp_waters_of_a_robot_regime | 4 | Hat-Wearing MAN 1, It's Filthy in There! 1, The G-Man 1, CreditToTeam 1 |
| mvm_teien_rc6_adv_onsen_onslaught | 6 | WITCH 1, Poopy Joe 1, AimBot 1, Divide by Zero 1, Mann Co. 1, trigger_hurt 1 |
| mvm_terrorlict_final1c5_adv_accursed_aggrievocation | 1 | CRITRAWKETS 1 |
| mvm_transmission_rc7a_adv_signal_shutdown | 6 | I LIVE! 1, Herr Doktor 1, Poopy Joe 1, Mann Co. 1, Delicious Cake 1, The Freeman 1 |
| mvm_villa_b13f_adv_forgotten | 6 | Grim Bloody Fable 1, I LIVE! 1, Hat-Wearing MAN 1, Kill Me 1, Delicious Cake 1, IvanTheSpaceBiker 1 |

| Mission | Wave | Outcome | Bot | Class | Fault |
| --- | --- | --- | --- | --- | --- |
| mvm_akure_rc1a_adv_gold_and_grit | 1 | passed | Still Alive | demoman | teleported 5x |
| mvm_akure_rc1a_adv_gold_and_grit | 1 | passed | RageQuit | spy | teleported 6x |
| mvm_akure_rc1a_adv_gold_and_grit | 1 | passed | One-Man Cheeseburger Apocalypse | pyro | teleported 1x |
| mvm_akure_rc1a_adv_gold_and_grit | 1 | passed | Kaboom! | heavy | a spawn exit took 32s over 1 lives; teleported 1x |
| mvm_akure_rc1a_adv_gold_and_grit | 1 | passed | Divide by Zero | pyro | teleported 2x |
| mvm_area_52_rc3_adv_australian_christmas | 1 | passed | The Freeman | demoman | teleported 3x |
| mvm_area_52_rc3_adv_australian_christmas | 1 | passed | CreditToTeam | heavy | teleported 1x |
| mvm_area_52_rc3_adv_australian_christmas | 1 | passed | Humans Are Weak | scout | teleported 1x |
| mvm_area_52_rc3_adv_australian_christmas | 1 | passed | Someone Else | demoman | still 120s at -929,1963,-414; teleported 1x |
| mvm_area_52_rc3_adv_australian_christmas | 1 | passed | C++ | spy | still 35s at -896,1963,-414; teleported 1x |
| mvm_area_52_rc3_adv_australian_christmas | 1 | passed | Mann Co. | sniper | teleported 1x |
| mvm_bigrock | 1 | passed | CreditToTeam | medic | teleported 1x |
| mvm_bigrock | 1 | passed | Hat-Wearing MAN | pyro | teleported 1x |
| mvm_bigrock | 1 | passed | It's Filthy in There! | scout | teleported 1x |
| mvm_bloodlust_b6_adv_reanimation | 1 | wave timed out | SomeDude | heavy | still 45s at -4483,2883,-238 |
| mvm_bronx_rc2_adv_point_of_impact | 1 | passed | H@XX0RZ | demoman | teleported 2x |
| mvm_bronx_rc2_adv_point_of_impact | 1 | passed | Crowbar | pyro | teleported 1x |
| mvm_coaltown | 1 | passed | Tiny Baby Man | heavy | teleported 1x |
| mvm_coaltown | 1 | passed | Mann Co. | demoman | teleported 1x |
| mvm_coaltown | 1 | passed | 0xDEADBEEF | scout | teleported 1x |
| mvm_condemned_b3_adv_unholy_undead | 1 | passed | Hostage | heavy | teleported 1x |
| mvm_condemned_b3_adv_unholy_undead | 1 | passed | Pow! | sniper | teleported 1x |
| mvm_condemned_b3_adv_unholy_undead | 1 | passed | Saxton Hale | engineer | teleported 1x |
| mvm_condemned_b3_adv_unholy_undead | 1 | passed | GLaDOS | spy | teleported 3x |
| mvm_condemned_b3_adv_unholy_undead | 1 | passed | AmNot | demoman | teleported 2x |
| mvm_condemned_b3_adv_unholy_undead | 1 | passed | HI THERE | pyro | teleported 2x |
| mvm_decoy | 1 | passed | GutsAndGlory! | medic | teleported 1x |
| mvm_decoy | 1 | passed | BeepBeepBoop | heavy | teleported 2x |
| mvm_decoy | 1 | passed | The Freeman | demoman | teleported 1x |
| mvm_decoy | 1 | passed | Black Mesa | scout | teleported 1x |
| mvm_frostwynd_rc1_adv_fiefdom_fiasco | 1 | passed | Aperture Science Prototype XR7 | pyro | teleported 1x |
| mvm_frostwynd_rc1_adv_fiefdom_fiasco | 1 | passed | A Professional With Standards | heavy | teleported 1x |
| mvm_frostwynd_rc1_adv_fiefdom_fiasco | 1 | passed | Black Mesa | pyro | teleported 1x |
| mvm_frostwynd_rc1_adv_fiefdom_fiasco | 1 | passed | Humans Are Weak | demoman | teleported 1x |
| mvm_frostwynd_rc1_adv_fiefdom_fiasco | 1 | passed | Hostage | soldier | teleported 2x |
| mvm_ghost_town_666 | 1 | passed | Freakin' Unbelievable | medic | still 40s at 55,2952,192 |
| mvm_ghost_town_666 | 1 | passed | BoomerBile | heavy | still 34s at 97,2983,192; teleported 1x |
| mvm_ghost_town_666 | 1 | passed | CEDA | demoman | teleported 1x |
| mvm_ghost_town_666 | 1 | passed | CrySomeMore | pyro | still 39s at -577,2359,291 |
| mvm_ghost_town_666 | 1 | passed | IvanTheSpaceBiker | scout | still 42s at 352,1403,368; teleported 2x |
| mvm_heatrock_rc6a_adv_heated_argument | 1 | passed | Ribs Grow Back | soldier | a spawn exit took 24s over 1 lives; teleported 1x |
| mvm_hideout_b3_adv_advanced | 1 | passed | I LIVE! | medic | still 31s at -3436,1914,516 |
| mvm_hideout_b3_adv_advanced | 1 | passed | DeadHead | demoman | still 33s at -3775,1657,513 |
| mvm_kelly_rc1b_adv_homestead_happenings | 1 | passed | CreditToTeam | demoman | teleported 2x |
| mvm_kelly_rc1b_adv_homestead_happenings | 1 | passed | Black Mesa | engineer | teleported 31x |
| mvm_kelly_rc1b_adv_homestead_happenings | 1 | passed | The G-Man | soldier | teleported 1x |
| mvm_kelly_rc1b_adv_homestead_happenings | 1 | passed | Hat-Wearing MAN | spy | a spawn exit took 22s over 5 lives; still 30s at -206,2376,-90 |
| mvm_kelly_rc1b_adv_homestead_happenings | 1 | passed | HI THERE | engineer | teleported 42x |
| mvm_kelly_rc1b_adv_homestead_happenings | 1 | passed | It's Filthy in There! | soldier | a spawn exit took 48s over 8 lives |
| mvm_mannhattan | 1 | passed | C++ | demoman | teleported 1x |
| mvm_mannworks | 1 | passed | Still Alive | medic | still 32s at 1274,2212,130 |
| mvm_mannworks | 1 | passed | Freakin' Unbelievable | demoman | a spawn exit took 38s over 1 lives |
| mvm_mannworks | 1 | passed | One-Man Cheeseburger Apocalypse | heavy | still 58s at 1294,2184,132 |
| mvm_mannworks | 1 | passed | Kaboom! | heavy | a spawn exit took 38s over 1 lives |
| mvm_mannworks | 1 | passed | Divide by Zero | scout | a spawn exit took 37s over 3 lives |
| mvm_marsbase_rc5_adv_secret_struggle | 1 | wave timed out | Screamin' Eagles | medic | still 34s at 5011,3125,678 |
| mvm_marsbase_rc5_adv_secret_struggle | 1 | wave timed out | Big Mean Muther Hubbard | heavy | still 30s at 4992,3084,682 |
| mvm_marsbase_rc5_adv_secret_struggle | 1 | wave timed out | Still Alive | pyro | still 68s at 5022,3053,687 |
| mvm_marsbase_rc5_adv_secret_struggle | 1 | wave timed out | AimBot | soldier | teleported 1x |
| mvm_memorial_b1_exp_necropolis | 1 | passed | SomeDude | demoman | teleported 1x |
| mvm_memorial_b1_exp_necropolis | 1 | passed | MindlessElectrons | engineer | teleported 1x |
| mvm_memorial_b1_exp_necropolis | 1 | passed | Tiny Baby Man | engineer | a spawn exit took 47s over 1 lives; teleported 1x |
| mvm_nightsky_rc4d_adv_nightsky_nightmare | 1 | passed | Archimedes! | pyro | teleported 2x |
| mvm_nightsky_rc4d_adv_nightsky_nightmare | 1 | passed | LOS LOS LOS | demoman | teleported 2x |
| mvm_nightsky_rc4d_adv_nightsky_nightmare | 1 | passed | CrySomeMore | engineer | teleported 1x |
| mvm_nightsky_rc4d_adv_nightsky_nightmare | 1 | passed | RageQuit | soldier | teleported 2x |
| mvm_nightsky_rc4d_adv_nightsky_nightmare | 1 | passed | Divide by Zero | spy | teleported 3x |
| mvm_nightsky_rc4d_adv_nightsky_nightmare | 1 | passed | I LIVE! | engineer | teleported 1x |
| mvm_oilrig_rc5d_adv_666_oilpocalypse | 1 | wave timed out | CreditToTeam | demoman | still 36s at -1090,-5052,1024; teleported 6x |
| mvm_oilrig_rc5d_adv_666_oilpocalypse | 1 | wave timed out | Black Mesa | engineer | teleported 1x |
| mvm_oilrig_rc5d_adv_666_oilpocalypse | 1 | wave timed out | The G-Man | soldier | teleported 10x |
| mvm_oilrig_rc5d_adv_666_oilpocalypse | 1 | wave timed out | Hat-Wearing MAN | spy | teleported 4x |
| mvm_oilrig_rc5d_adv_666_oilpocalypse | 1 | wave timed out | HI THERE | engineer | teleported 1x |
| mvm_oilrig_rc5d_adv_666_oilpocalypse | 1 | wave timed out | It's Filthy in There! | soldier | teleported 8x |
| mvm_oxidize_rc3_adv_666_corroding_cadavers | 1 | wave timed out | MoreGun | soldier | teleported 1x |
| mvm_oxidize_rc3_adv_666_corroding_cadavers | 1 | wave timed out | Maggot | heavy | still 39s at -481,-123,0 |
| mvm_oxidize_rc3_adv_666_corroding_cadavers | 1 | wave timed out | C++ | medic | still 38s at -587,-181,0; teleported 2x |
| mvm_oxidize_rc3_adv_666_corroding_cadavers | 1 | wave timed out | AmNot | medic | still 32s at -1750,3409,-119 |
| mvm_oxidize_rc3_adv_666_corroding_cadavers | 1 | wave timed out | Kill Me | medic | a spawn exit took 24s over 3 lives; teleported 4x |
| mvm_oxidize_rc3_adv_666_corroding_cadavers | 1 | wave timed out | Screamin' Eagles | soldier | still 32s at -1790,3400,-102 |
| mvm_oxidize_rr18_adv_grass_run | 1 | passed | CreditToTeam | demoman | teleported 1x |
| mvm_oxidize_rr18_adv_grass_run | 1 | passed | The G-Man | soldier | a spawn exit took 21s over 4 lives |
| mvm_radar_b10_adv_rocky_ravage | 1 | passed | MindlessElectrons | demoman | a spawn exit took 23s over 3 lives |
| mvm_radar_b10_adv_rocky_ravage | 1 | passed | H@XX0RZ | spy | teleported 1x |
| mvm_redstone_ridge_rc5_adv_armored_apparatus | 1 | passed | LOS LOS LOS | demoman | a spawn exit took 38s over 3 lives |
| mvm_robotfactory_b30_adv_frantic_flood | 1 | passed | Maggot | heavy | teleported 1x |
| mvm_robotfactory_b30_adv_frantic_flood | 1 | passed | LOS LOS LOS | demoman | teleported 1x |
| mvm_robotfactory_b30_adv_frantic_flood | 1 | passed | Nobody | soldier | teleported 1x |
| mvm_rottenburg | 1 | passed | GutsAndGlory! | medic | teleported 1x |
| mvm_rottenburg | 1 | passed | BeepBeepBoop | heavy | a spawn exit took 35s over 1 lives |
| mvm_rottenburg | 1 | passed | The Freeman | demoman | teleported 1x |
| mvm_rottenburg | 1 | passed | Black Mesa | scout | teleported 1x |
| mvm_seabed_b6_adv_ocean_commotion | 1 | wave timed out | Humans Are Weak | heavy | teleported 1x |
| mvm_smog_b3a_adv_respiration_restriction | 1 | passed | Grim Bloody Fable | spy | teleported 2x |
| mvm_snowpine_rc4_fix1_adv_permafrost_panic | 1 | no enemies spawned | MindlessElectrons | demoman | still 227s at -139,2951,147; teleported 1x |
| mvm_snowpine_rc4_fix1_adv_permafrost_panic | 1 | no enemies spawned | trigger_hurt | soldier | never left spawn; in spawn for 239s at the end |
| mvm_snowpine_rc4_fix1_adv_permafrost_panic | 1 | no enemies spawned | H@XX0RZ | spy | still 227s at -183,3073,147; teleported 1x |
| mvm_snowpine_rc4_fix1_adv_permafrost_panic | 1 | no enemies spawned | Ribs Grow Back | soldier | still 226s at -131,3168,147; teleported 1x |
| mvm_spybase_rc8_exp_waters_of_a_robot_regime | 1 | wave timed out | CreditToTeam | demoman | teleported 1x |
| mvm_spybase_rc8_exp_waters_of_a_robot_regime | 1 | wave timed out | The G-Man | soldier | still 51s at -4727,5549,512; teleported 1x |
| mvm_spybase_rc8_exp_waters_of_a_robot_regime | 1 | wave timed out | Hat-Wearing MAN | spy | still 81s at -4906,5440,512; teleported 1x |
| mvm_spybase_rc8_exp_waters_of_a_robot_regime | 1 | wave timed out | It's Filthy in There! | soldier | teleported 1x |
| mvm_teien_rc6_adv_onsen_onslaught | 1 | passed | Poopy Joe | heavy | teleported 1x |
| mvm_teien_rc6_adv_onsen_onslaught | 1 | passed | WITCH | spy | teleported 1x |
| mvm_teien_rc6_adv_onsen_onslaught | 1 | passed | AimBot | heavy | teleported 1x |
| mvm_teien_rc6_adv_onsen_onslaught | 1 | passed | trigger_hurt | sniper | teleported 1x |
| mvm_teien_rc6_adv_onsen_onslaught | 1 | passed | Mann Co. | soldier | still 48s at 4288,-5526,180; teleported 1x |
| mvm_teien_rc6_adv_onsen_onslaught | 1 | passed | Divide by Zero | sniper | teleported 1x |
| mvm_terrorlict_final1c5_adv_accursed_aggrievocation | 1 | passed | CRITRAWKETS | soldier | teleported 1x |
| mvm_transmission_rc7a_adv_signal_shutdown | 1 | wave timed out | I LIVE! | soldier | teleported 1x |
| mvm_transmission_rc7a_adv_signal_shutdown | 1 | wave timed out | Mann Co. | demoman | teleported 1x |
| mvm_transmission_rc7a_adv_signal_shutdown | 1 | wave timed out | Poopy Joe | scout | teleported 1x |
| mvm_transmission_rc7a_adv_signal_shutdown | 1 | wave timed out | The Freeman | engineer | teleported 1x |
| mvm_transmission_rc7a_adv_signal_shutdown | 1 | wave timed out | Delicious Cake | heavy | teleported 1x |
| mvm_transmission_rc7a_adv_signal_shutdown | 1 | wave timed out | Herr Doktor | medic | teleported 1x |
| mvm_villa_b13f_adv_forgotten | 1 | passed | Kill Me | medic | still 143s at -9079,8402,528; teleported 1x |
| mvm_villa_b13f_adv_forgotten | 1 | passed | Hat-Wearing MAN | medic | still 145s at -9068,8260,528; teleported 1x |
| mvm_villa_b13f_adv_forgotten | 1 | passed | IvanTheSpaceBiker | medic | still 144s at -8880,8262,528; teleported 1x |
| mvm_villa_b13f_adv_forgotten | 1 | passed | Delicious Cake | medic | still 144s at -9165,8248,528; teleported 1x |
| mvm_villa_b13f_adv_forgotten | 1 | passed | I LIVE! | medic | still 144s at -8975,8170,528; teleported 1x |
| mvm_villa_b13f_adv_forgotten | 1 | passed | Grim Bloody Fable | medic | still 145s at -8967,8370,528; teleported 1x |

Waves without a defender record:

- mvm_chateau_rc3_adv_remedic wave 1: wave failed game or probe failed wave 1: wave_failed
- mvm_shadows_b3_adv_bauernhof wave 0: changelevel failure changelevel did not reach mvm_shadows_b3: before mvm_seabed_b6/mvm_seabed_b6_adv_ocean_commotion, after mvm_decoy/mvm_decoy_adv_grutesque_getaway (idle); reply ""; command error ""; timed out after 1m30s (last status {State:idle Reason:none Map:mvm_decoy Pop:mvm_decoy_adv_grutesque_getaway Max:0 GameWave:0 Expected:0 Observed:0 Bots:0 Tanks:0 BotSpawns:0 TankSpawns:0 Attempts:0 Alive:0 Remaining:0 Initial:0 DefClass:0 DefTeam:0 PlayerTeam:2 EnemyTeam:3 Elapsed:1 Progress:-1})
- mvm_yiresa_rc5a_int_black_east wave 0: load blocked changelevel mvm_yiresa_rc5a from mvm_villa_b13f/mvm_villa_b13f_adv_forgotten: reply "", command error <nil>, final status error cannot read the reply: read tcp 127.0.0.1:53942->127.0.0.1:27045: i/o timeout: timed out after 1m30s (last status {State: Reason: Map: Pop: Max:0 GameWave:0 Expected:0 Observed:0 Bots:0 Tanks:0 BotSpawns:0 TankSpawns:0 Attempts:0 Alive:0 Remaining:0 Initial:0 DefClass:0 DefTeam:0 PlayerTeam:0 EnemyTeam:0 Elapsed:0 Progress:0}): cannot read the reply: read tcp 127.0.0.1:45458->127.0.0.1:27045: i/o timeout

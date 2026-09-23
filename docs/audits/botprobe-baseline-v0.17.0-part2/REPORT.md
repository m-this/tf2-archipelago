# Defender bot run

```
speed=4
waves=1
mission=all
timeout=240s
bots_smx=/home/mathis/projects/tf2-archipelago/deploy/bots/build/package/addons/sourcemod/plugins/tf2_defenderbots.smx
bots_sha256=a33f109caad4af9ebe0c956aeaf2677da7158b6f199fad9a13ff9e6489cf7465
image=tf2ap-srcds:v117
started_utc=2026-09-23T13:02:52Z
source_commit=ed261386ef51cf6a77b51dd981ea51ee2c1d56bf
```

21 waves, 126 bot records. Never left spawn: 2. Bots with a fault: 49. Median time to leave spawn: 5.7s.

Rescues in the mod's log during the waves: 174 spawn recovery, 21 stuck, 4 wedge teleport.

| Mission | Rescues | Spawn recoveries by bot |
| --- | --- | --- |
| mvm_decoy | 5 | C++ 2, trigger_hurt 2, Numnutz 1 |
| mvm_heatrock_rc6a_adv_heated_argument | 3 | MindlessElectrons 3 |
| mvm_kelly_rc1b_adv_homestead_happenings | 112 | Totally Not A Bot 52, Soulless 46, MindlessElectrons 11, trigger_hurt 1 |
| mvm_mannhattan | 19 |  |
| mvm_mannworks | 4 | The Combine 2 |
| mvm_marsbase_rc5_adv_secret_struggle | 3 | The Freeman 2, Kaboom! 1 |
| mvm_smog_b3a_adv_respiration_restriction | 5 | ThatGuy 2, Poopy Joe 1, I LIVE! 1, Saxton Hale 1 |
| mvm_snowpine_rc4_fix1_adv_permafrost_panic | 2 | Hat-Wearing MAN 1, CreditToTeam 1 |
| mvm_spybase_rc8_exp_waters_of_a_robot_regime | 12 | H@XX0RZ 5, Ribs Grow Back 3, MindlessElectrons 3, trigger_hurt 1 |
| mvm_teien_rc6_adv_onsen_onslaught | 12 | Divide by Zero 3, Archimedes! 2, RageQuit 2, LOS LOS LOS 2, I LIVE! 2, CrySomeMore 1 |
| mvm_terrorlict_final1c5_adv_accursed_aggrievocation | 1 | CRITRAWKETS 1 |
| mvm_transmission_rc7a_adv_signal_shutdown | 11 | Poopy Joe 3, Cannon Fodder 3, Kill Me 2, Tiny Baby Man 1, Humans Are Weak 1, Force of Nature 1 |
| mvm_traumatic_b2a_adv_terrorstorm | 2 |  |
| mvm_villa_b13f_adv_forgotten | 8 | Archimedes! 3, Saxton Hale 1, Still Alive 1, Dog 1, Big Mean Muther Hubbard 1, Hostage 1 |

| Mission | Wave | Outcome | Bot | Class | Fault |
| --- | --- | --- | --- | --- | --- |
| mvm_decoy | 1 | passed | Numnutz | heavy | teleported 1x |
| mvm_decoy | 1 | passed | trigger_hurt | demoman | a spawn exit took 30s over 1 lives; teleported 1x |
| mvm_decoy | 1 | passed | C++ | scout | a spawn exit took 21s over 1 lives; teleported 1x |
| mvm_heatrock_rc6a_adv_heated_argument | 1 | passed | MindlessElectrons | demoman | teleported 1x |
| mvm_heatrock_rc6a_adv_heated_argument | 1 | passed | Ribs Grow Back | soldier | a spawn exit took 22s over 1 lives |
| mvm_kelly_rc1b_adv_homestead_happenings | 1 | passed | MindlessElectrons | demoman | still 37s at 599,1631,96; teleported 3x |
| mvm_kelly_rc1b_adv_homestead_happenings | 1 | passed | Soulless | engineer | teleported 46x |
| mvm_kelly_rc1b_adv_homestead_happenings | 1 | passed | trigger_hurt | soldier | teleported 1x |
| mvm_kelly_rc1b_adv_homestead_happenings | 1 | passed | H@XX0RZ | spy | a spawn exit took 22s over 6 lives; still 54s at -62,2376,-95 |
| mvm_kelly_rc1b_adv_homestead_happenings | 1 | passed | Totally Not A Bot | engineer | teleported 34x |
| mvm_kelly_rc1b_adv_homestead_happenings | 1 | passed | Ribs Grow Back | soldier | a spawn exit took 70s over 8 lives |
| mvm_mannworks | 1 | passed | Divide by Zero | demoman | a spawn exit took 39s over 2 lives |
| mvm_mannworks | 1 | passed | The Combine | heavy | still 44s at 1274,2181,132; teleported 1x |
| mvm_mannworks | 1 | passed | Me | heavy | a spawn exit took 38s over 2 lives |
| mvm_mannworks | 1 | passed | Pow! | scout | a spawn exit took 36s over 3 lives |
| mvm_marsbase_rc5_adv_secret_struggle | 1 | wave timed out | Kaboom! | heavy | teleported 1x |
| mvm_marsbase_rc5_adv_secret_struggle | 1 | wave timed out | Crowbar | medic | still 35s at 5012,3082,682 |
| mvm_marsbase_rc5_adv_secret_struggle | 1 | wave timed out | The Freeman | demoman | teleported 2x |
| mvm_smog_b3a_adv_respiration_restriction | 1 | passed | Poopy Joe | demoman | teleported 1x |
| mvm_smog_b3a_adv_respiration_restriction | 1 | passed | I LIVE! | heavy | teleported 1x |
| mvm_smog_b3a_adv_respiration_restriction | 1 | passed | ThatGuy | spy | teleported 2x |
| mvm_snowpine_rc4_fix1_adv_permafrost_panic | 1 | no enemies spawned | CreditToTeam | demoman | still 935s at -367,2925,147; teleported 1x |
| mvm_snowpine_rc4_fix1_adv_permafrost_panic | 1 | no enemies spawned | The G-Man | soldier | never left spawn; in spawn for 948s at the end |
| mvm_snowpine_rc4_fix1_adv_permafrost_panic | 1 | no enemies spawned | Hat-Wearing MAN | spy | still 935s at -258,3055,147; teleported 1x |
| mvm_snowpine_rc4_fix1_adv_permafrost_panic | 1 | no enemies spawned | It's Filthy in There! | soldier | never left spawn; in spawn for 948s at the end |
| mvm_spybase_rc8_exp_waters_of_a_robot_regime | 1 | passed | MindlessElectrons | demoman | teleported 2x |
| mvm_spybase_rc8_exp_waters_of_a_robot_regime | 1 | passed | trigger_hurt | soldier | teleported 1x |
| mvm_spybase_rc8_exp_waters_of_a_robot_regime | 1 | passed | H@XX0RZ | spy | teleported 1x |
| mvm_spybase_rc8_exp_waters_of_a_robot_regime | 1 | passed | Ribs Grow Back | soldier | teleported 1x |
| mvm_teien_rc6_adv_onsen_onslaught | 1 | passed | Archimedes! | pyro | teleported 2x |
| mvm_teien_rc6_adv_onsen_onslaught | 1 | passed | LOS LOS LOS | demoman | teleported 1x |
| mvm_teien_rc6_adv_onsen_onslaught | 1 | passed | CrySomeMore | engineer | teleported 1x |
| mvm_teien_rc6_adv_onsen_onslaught | 1 | passed | RageQuit | soldier | teleported 2x |
| mvm_teien_rc6_adv_onsen_onslaught | 1 | passed | Divide by Zero | spy | teleported 2x |
| mvm_teien_rc6_adv_onsen_onslaught | 1 | passed | I LIVE! | engineer | teleported 2x |
| mvm_terrorlict_final1c5_adv_accursed_aggrievocation | 1 | passed | AimBot | demoman | a spawn exit took 20s over 5 lives |
| mvm_terrorlict_final1c5_adv_accursed_aggrievocation | 1 | passed | CRITRAWKETS | soldier | a spawn exit took 22s over 4 lives |
| mvm_transmission_rc7a_adv_signal_shutdown | 1 | passed | Poopy Joe | sniper | teleported 2x |
| mvm_transmission_rc7a_adv_signal_shutdown | 1 | passed | Humans Are Weak | medic | teleported 1x |
| mvm_transmission_rc7a_adv_signal_shutdown | 1 | passed | Tiny Baby Man | pyro | teleported 1x |
| mvm_transmission_rc7a_adv_signal_shutdown | 1 | passed | Force of Nature | heavy | teleported 1x |
| mvm_transmission_rc7a_adv_signal_shutdown | 1 | passed | Cannon Fodder | sniper | teleported 1x |
| mvm_transmission_rc7a_adv_signal_shutdown | 1 | passed | Kill Me | heavy | teleported 1x |
| mvm_villa_b13f_adv_forgotten | 1 | passed | Dog | medic | still 122s at -9037,8313,528; teleported 1x |
| mvm_villa_b13f_adv_forgotten | 1 | passed | Archimedes! | medic | still 115s at -8903,8440,528; teleported 1x |
| mvm_villa_b13f_adv_forgotten | 1 | passed | Big Mean Muther Hubbard | medic | still 121s at -9136,8358,528; teleported 1x |
| mvm_villa_b13f_adv_forgotten | 1 | passed | Hostage | medic | still 115s at -9023,8515,512; teleported 1x |
| mvm_villa_b13f_adv_forgotten | 1 | passed | Still Alive | medic | still 120s at -9038,8221,528; teleported 1x |
| mvm_villa_b13f_adv_forgotten | 1 | passed | Saxton Hale | medic | still 122s at -8953,8178,528; teleported 1x |

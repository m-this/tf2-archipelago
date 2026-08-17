# Tests

Ce qui a encore besoin d'un vrai test, et ce qu'il faut pour chacun : un
serveur Archipelago, un second jeu dans le multiworld, un joueur, ou plus
d'un.

- **Une vague se termine.** Il faut : la stack lancée, un joueur, une vague
  en cours. Confirme que `mvm_wave_complete` se déclenche et que le plugin
  rapporte la check.
- **Le numéro de vague se lit correctement.** Il faut : la stack lancée,
  un joueur, une vague en cours. Confirme que `mvm_begin_wave` donne le
  bon numéro au plugin.
- **Une mission se termine.** Il faut : la stack lancée, un joueur, toutes
  les vagues d'une mission réussies. Confirme que `mvm_mission_complete`
  se déclenche, et que le repli de dernière vague n'est pas nécessaire.
- **Le compteur de vagues se lit correctement.** Il faut : la stack
  lancée, un joueur, une vague en cours. Confirme que
  `m_nMannVsMachineWaveCount` correspond à la vague sur laquelle le jeu se
  trouve réellement.
- **Les crédits sont payés.** Il faut : la stack lancée, un joueur, un item
  Cash Bundle reçu en cours de vague. Confirme que `m_nCurrency` se met à
  jour pour le joueur sur le serveur.
- **Les crédits atteignent chaque joueur, pas un seul.** Il faut : la
  stack lancée, deux joueurs ou plus, un item Cash Bundle reçu. Confirme
  que la boucle de paiement paie toute l'équipe.
- **Un emplacement d'arme verrouillé reste vide.** Il faut : la stack
  lancée, un joueur, une visite au casier de ravitaillement ou à
  l'upgrade station avec un emplacement verrouillé. Confirme que le
  plugin retire l'arme au lieu d'avertir et de la laisser.
- **Une classe verrouillée déplace le joueur.** Il faut : la stack
  lancée, un joueur sur une classe verrouillée, un respawn. Confirme que
  le changement se produit au prochain spawn, pas en milieu de vague.
- **La correspondance des noms de mission.** Il faut : la stack lancée,
  aucun serveur Archipelago requis — du MvM vanilla suffit. Chargez chaque
  popfile ambigu à la main (`sm_ap_mission <popfile>`) et lisez le nom de
  la mission en jeu via `sm_ap_status`. Confirme que le nom d'affichage de
  `gamedata/missions.go` correspond au fichier, pour les deux Decoy
  intermediate, les deux Coal Town intermediate, les deux Mannworks
  intermediate, et les trois groupes advanced.
- **Un second jeu dans le même multiworld.** Il faut : un serveur
  Archipelago avec une seed contenant TF2 et au moins un autre jeu, les
  deux connectés. Confirme qu'un item de l'autre jeu atteint TF2, et
  qu'une check TF2 atteint l'autre jeu.
- **Un redémarrage en cours de mission.** Il faut : la stack lancée, un
  joueur, une vague en cours, `make down && make up` lancé en plein
  milieu. Confirme que la file du bridge et l'ensemble des déblocages du
  plugin survivent à un redémarrage sans perdre ni répéter une check.

## Comment déclencher une vague seul

Plusieurs tests ci-dessus ont besoin d'une vague réussie, et MvM ne
démarre pas une vague tant qu'un joueur humain ne s'est pas déclaré prêt —
un bot ne peut pas le faire. Un seul joueur suffit :

```
rcon sv_cheats 1
rcon mp_disable_respawn_times 1
god
rcon tf_bot_kill all
rcon tf_mvm_tank_kill
```

`god` se tape dans votre propre console, pas via rcon : rcon tourne comme
le serveur, et le serveur n'est pas un joueur. `tf_bot_kill all` tue ce
qui est vivant maintenant ; la mission continue d'envoyer le reste de la
vague, donc relancez-la toutes les quelques secondes jusqu'à ce que la
vague se termine. Cela réussit la vague comme le jeu le fait, ce qui est le
but : `mvm_wave_complete` se déclenche pour de vrai.

`tf_mvm_force_victory` et `tf_mvm_jump_to_wave` existent aussi, et ne
valent pas la peine ici — ils sautent l'événement en cours de test.

Chargez d'abord une mission que la partie tient réellement, avec
`rcon sm_ap_mission`. Sur une mission hors de la partie, le plugin le dit
et compte quand même les checks, ce qui est le comportement correct et une
chose déroutante à tester par erreur.

## Ce qu'il faut surveiller, dans l'ordre

1. `[AP] Wave 1 cleared.` dans le chat, quand l'équipe réussit la vague 1.
2. Un emplacement d'arme verrouillé reste vide après une visite au casier
   de ravitaillement.
3. Une classe verrouillée déplace le joueur au prochain spawn, pas en
   milieu de vague.
4. Les crédits d'un Cash Bundle arrivent vraiment.
5. La fin de mission se déclenche sur la dernière vague, pas une vague
   trop tôt. Vérifiez `wave_drift` sur la page de santé du bridge. Voir
   [Dépannage](troubleshooting.md).

Activez d'abord le mode bavard, pour que chaque appel au bridge et chaque
événement de jeu arrive dans le chat :

```
rcon_password votre-SRCDS_RCONPW
rcon tf2ap_debug 1
rcon sm_ap_status
```

Une réponse saine de `sm_ap_status` ressemble à ceci :

```
[AP] version 0.1.0, mvm yes, mission mvm_decoy_intermediate, wave 0 of 7
[AP] events: begin_wave yes, wave_complete yes, mission_complete yes
[AP] unlocks held at sequence 5, 0 objective(s) waiting to be sent
[AP] classes: soldier, heavy
[AP] slots: primary
```

`mission` doit être la mission, pas la carte. `wave 0 of 7` doit
correspondre à la vraie longueur de la mission. `unlocks held` doit dire
held, pas NOT FETCHED.

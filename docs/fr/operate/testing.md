# Tests

`make integration` couvre la génération de seed, le serveur Archipelago et
le bridge, à chaque exécution.

Le plugin a tourné sur un serveur réel le 2026-08-17, via rcon, sans
client de jeu. Les trois événements MvM existent. La mission et sa
longueur se lisent correctement, et une check a atteint le multiworld.

Quatre choses restent à confirmer :

- une vague réussie
- des crédits qui arrivent
- un emplacement d'arme appliqué
- une classe déplacée

Cette page donne la marche à suivre pour confirmer chacune. Elle dit
aussi quoi surveiller pendant le test.

## 1. Connectez-vous à votre serveur

Lancez la stack, puis rejoignez avec un client TF2 standard, comme un
joueur. Voir [Invitez vos amis](../setup/invite-your-friends.md) pour la
commande de connexion et le mot de passe.

```sh
make up
```

```
connect votre.adresse.serveur:27015
```

## 2. Obtenez les droits admin

Deux choses distinctes comptent comme admin ici. Vous voudrez
probablement les deux.

**Rcon**, pour les commandes serveur (`sv_cheats`, `tf_bot_kill`, et les
tests ci-dessous) :

```sh
# dans .env
SRCDS_RCONPW=votre-mot-de-passe
```

```
rcon_password votre-SRCDS_RCONPW
rcon sm_ap_status
```

Si `sm_ap_status` répond, rcon fonctionne.

**Commandes de chat** (`!mission`, `sm_ap_mission` dans le chat), pour la
liste `SRCDS_ADMIN_STEAMIDS` :

```sh
# dans .env
SRCDS_ADMIN_STEAMIDS=STEAM_0:1:XXXXXXX
```

Séparez plusieurs valeurs par des virgules. Redémarrez la stack après le
changement (`make down && make up`). Vérifiez depuis le chat du jeu :

```
!ap
```

Un joueur non admin n'obtient rien de `!ap`. Un admin, si.

## 3. Activez le mode bavard

Faites ceci avant chaque test ci-dessous. Cela envoie chaque appel au
bridge et chaque événement de jeu dans le chat. Surveillez le chat
pendant chaque test.

```
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

## 4. Chargez une mission que la partie tient réellement

```
rcon sm_ap_mission <popfile>
```

Prenez un popfile dans `/missions`, ou dans le champ `mission` de
`sm_ap_status`. Il doit appartenir au pool de cette partie. Un popfile
hors du pool charge quand même. Le plugin le signale, et compte les
checks tout de même. Ne testez pas contre lui par erreur.

## 5. Réussissez une vague seul

Le serveur remplit RED avec des bots qui jouent, donc un testeur seul
forme une équipe complète. Voir
[Les bots de votre équipe](../play/defender-bots.md).

MvM ne démarre pas une vague tant qu'un joueur humain ne s'est pas
déclaré prêt. Déclarez-vous prêt, puis lancez ce bloc de commandes. Il
termine la vague sans la jouer :

```
rcon sv_cheats 1
rcon mp_disable_respawn_times 1
god
rcon tf_bot_kill all
rcon tf_mvm_tank_kill
```

`god` se tape dans votre propre console, pas via rcon. Rcon tourne comme
le serveur, et le serveur n'est pas un joueur. `tf_bot_kill all` tue ce
qui est vivant maintenant, bots défenseurs compris. Le mod les remplace
dans la seconde. La mission continue d'envoyer le reste de la
vague.

Relancez la commande toutes les quelques secondes, jusqu'à la fin de la
vague. Cela réussit la vague comme le jeu le fait. `mvm_wave_complete` se
déclenche donc pour de vrai, pas par un raccourci.

`tf_mvm_force_victory` et `tf_mvm_jump_to_wave` existent aussi. Ne les
utilisez pas ici : ils sautent l'événement à tester.

**Confirme :** `mvm_wave_complete` se déclenche, et le plugin rapporte la
check. `mvm_begin_wave` a donné le bon numéro de vague au plugin, en
amont. Comparez le `wave N of M` de `sm_ap_status` avant et après.
Surveillez dans le chat :

```
[AP] Wave 1 cleared.
```

## 6. Réussissez une mission entière

Répétez l'étape 5 pour chaque vague de la mission chargée.

**Confirme :** `mvm_mission_complete` se déclenche sur la dernière vague,
ni une vague trop tôt ni trop tard. Ça confirme aussi que le repli de
dernière vague reste inutilisé. Vérifiez `wave_drift` sur la page de
santé du bridge. Voir [Dépannage](troubleshooting.md) pour confirmer que
le nombre de vagues du jeu correspond à `gamedata`.

## 7. Confirmez le paiement des crédits

Il faut un item Cash Bundle reçu en cours de vague. N'importe quel item
convient. Si la file est vide, regardez la ligne « objective(s) waiting »
de `sm_ap_status`. Vous pouvez aussi en envoyer un depuis un autre jeu
connecté.

**Confirme :** `m_nCurrency` se met à jour pour le joueur sur le serveur.
Avec deux joueurs ou plus, confirmez que la boucle de paiement paie toute
l'équipe, pas un seul joueur.

## 8. Confirmez qu'un emplacement d'arme verrouillé reste vide

Il faut un emplacement verrouillé. Visitez le casier de ravitaillement,
ou l'upgrade station.

**Confirme :** le plugin retire l'arme, au lieu d'avertir et de la
laisser en place. Si le joueur tenait cette arme, il doit se retrouver
avec autre chose en main. Vérifiez ce point.

## 9. Confirmez qu'une classe verrouillée déplace le joueur

Rejoignez une classe verrouillée, puis respawnez.

**Confirme :** le changement se produit au prochain spawn, pas en milieu
de vague. Forcer un respawn en milieu de vague coûte le reste de la vague
pour rien.

## 10. Confirmez la correspondance des noms de mission

Aucun serveur Archipelago n'est nécessaire. Du MvM vanilla suffit.
Chargez chaque popfile ambigu à la main. Lisez le nom de la mission en
jeu via `sm_ap_status` :

```
rcon sm_ap_mission <popfile>
rcon sm_ap_status
```

**Confirme :** le nom d'affichage de `gamedata/missions.go` correspond au
fichier. Vérifiez les deux Decoy intermediate, les deux Coal Town
intermediate, les deux Mannworks intermediate, et les trois groupes
advanced.

## 11. Confirmez un second jeu dans le même multiworld

Il faut un serveur Archipelago avec une seed qui contient TF2 et au
moins un autre jeu. Les deux doivent être connectés.

**Confirme :** un item de l'autre jeu atteint TF2, et une check TF2
atteint l'autre jeu.

## 12. Confirmez un redémarrage en cours de mission

Avec une vague en cours :

```sh
make down && make up
```

**Confirme :** la file du bridge et l'ensemble des déblocages du plugin
survivent au redémarrage. Aucune check ne se perd, aucune ne se répète.

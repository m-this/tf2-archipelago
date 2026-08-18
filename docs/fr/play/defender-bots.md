# Les bots de votre équipe

Team Fortress 2 équilibre chaque vague de Mann vs Machine pour six joueurs dans
l'équipe RED. À deux, les robots passent. La vague 1 d'une mission Advanced ne
se gagne pas à deux joueurs, et la partie s'arrête là.

Le serveur remplit les places vides pour vous. Rien à installer, rien à taper.

## Ce qu'ils font

Le serveur remplit RED jusqu'à six joueurs au début de la vague, et la garde
pleine ensuite. Un bot qui meurt revient en une seconde. Les bots choisissent
leur classe, se battent, et dépensent leurs crédits à la station entre les
vagues. Ils se déclarent prêts aussi, donc la vague démarre quand *vous*
appuyez sur F4.

Ils ne sont pas humains. Les Engineers construisent trop près des robots, les
Spies se font repérer tard, et un bot ne fera jamais le coup malin de votre
ami. Ils sont assez bons pour rendre une vague gagnable, et c'est leur rôle.

## Les baisser, ou les couper

Deux réglages dans `.env` :

| Variable | Défaut | Effet |
| --- | --- | --- |
| `SRCDS_BOTS` | `1` | `0` les garde hors du terrain jusqu'à un `sm_addbots` d'un admin |
| `SRCDS_BOT_TEAM_SIZE` | `6` | Le nombre de joueurs dans RED, humains compris |

Baissez `SRCDS_BOT_TEAM_SIZE` pour une partie plus dure : à `4`, trois amis
reçoivent un bot. Mettez `SRCDS_BOTS=0` quand vous êtes six et que les places
vous reviennent.

Les deux prennent effet au chargement de la carte suivante. `make restart` est
la façon sûre.

Sur Windows, le lanceur pose les mêmes questions. Voir
[Installer sur Windows](../setup/install-windows.md).

## Qui les a écrits

[OfficerSpy/TF2-MvM-Defender-TFBots][mod], GPL-3.0, et cinq dépendances :
CBaseNPC, Actions, TF2Attributes, TF Econ Data et TF2Utils. Le serveur les
compile depuis la source, avec deux correctifs à nous dans `deploy/patches/`.

Le comportement des bots est celui du mod. Un bot qui rentre dans un mur se
signale à ce dépôt, pas à celui-ci.

[mod]: https://github.com/OfficerSpy/TF2-MvM-Defender-TFBots

## Sur un serveur qui n'est pas cette image

Chaque version publie `tf2-defender-bots.zip`. Il porte tout l'ensemble :
plugins, extensions pour Linux et Windows, gamedata et les repères de
navigation par carte. Le zip part de `addons/`, donc une seule décompression
dans le dossier du jeu (`tf/`) suffit.

Réglez ensuite les trois convars dans `server.cfg` :

```
sm_redbots_manager_mode 2
sm_redbots_manager_defender_team_size 6
sm_redbots_manager_min_players -1
```

`mode 2` fait apparaître les bots au début de la vague. `min_players -1` compte
plus qu'il n'y paraît : la barrière du mod compte RED *avant* la vague, où un
joueur seul n'a pas encore de bots. Laissez-la active, et elle bloque le F4 qui
les fait apparaître.

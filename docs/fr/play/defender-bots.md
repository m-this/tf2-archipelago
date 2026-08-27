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

Ils ne sont pas humains. Les Spies se font repérer tard, et un bot ne fera
jamais le coup malin de votre ami. Ils sont assez bons pour rendre une vague
gagnable, et c'est leur rôle.

Un bot cède sa place quand un ami arrive. RED tient six joueurs. Quand
l'équipe est pleine de bots et qu'un joueur se connecte, un bot part. Le
joueur prend le siège. Le mod remplit l'équipe à nouveau au début de la vague
suivante.

Les bots portent les noms de bots du jeu, ceux d'un serveur Valve.

## Les baisser, ou les couper

Les réglages dans `.env` :

| Variable | Défaut | Effet |
| --- | --- | --- |
| `SRCDS_BOTS` | `1` | `0` les garde hors du terrain jusqu'à un `sm_addbots` d'un admin |
| `SRCDS_BOT_TEAM_SIZE` | `6` | Le nombre de joueurs dans RED, humains compris |
| `SRCDS_BOT_CLASS_BLACKLIST` | vide | Les classes que les bots ne jouent jamais, séparées par des virgules : `sniper,spy` |
| `SRCDS_BOT_TEAM_COMP` | vide | Les classes dont les bots remplissent RED, dans l'ordre. Voir ci-dessous. |
| `TF2AP_BOT_UPGRADES_CHAT` | `0` | `1` écrit dans le chat ce que les bots achètent à la station d'améliorations |
| `SRCDS_BOT_HATS` | `1` | Un chapeau au hasard sur chaque bot |
| `SRCDS_BOT_HAT_EFFECTS` | `0` | `1` pose un effet unusual au hasard sur ce chapeau |

Baissez `SRCDS_BOT_TEAM_SIZE` pour une partie plus dure : à `4`, trois amis
reçoivent un bot. Mettez `SRCDS_BOTS=0` quand vous êtes six et que les places
vous reviennent.

Changez l'un de ces réglages sur des relevés, pas sur un souvenir. Valve règle
chaque vague pour six défenseurs, et les bots existent pour qu'une équipe plus
petite gagne. Personne n'a mesuré à quel point. `wave_failures` dans
`/healthz` nomme chaque vague perdue d'une soirée, les pires d'abord, et
`tf2ap_wave_lost_total` trace la même chose. Jouez une mission, lisez quelle
vague vous a arrêtés, puis changez un chiffre. Voir
[Dépannage](../operate/troubleshooting.md).

Les bots sont de mauvais Snipers et de mauvais Spies.
`SRCDS_BOT_CLASS_BLACKLIST=sniper,spy` les garde sur les classes qu'ils jouent
bien. Les noms de classe sont ceux du mod : `scout`, `soldier`, `pyro`,
`demoman`, `heavyweapons`, `engineer`, `medic`, `sniper`, `spy`.

Une liste noire interdit des classes. Elle ne dit pas ce qu'est l'équipe. Un
tirage dans le reste a donné trois Spies et deux Scouts sur une mission
Advanced. Une autre équipe n'avait pas d'Engineer et a perdu deux fois la
vague 1 de Quarry.

`SRCDS_BOT_TEAM_COMP=engineer,medic,heavyweapons,soldier,demoman` nomme
l'équipe à la place. L'ordre est celui dans lequel les places se remplissent.
Mettez donc en premier les classes dont vous ne pouvez pas vous passer. Les
humains prennent les places avant les bots, et les dernières entrées servent
rarement. Les noms de classe sont ceux du mod, comme pour la liste noire.

Une équipe nommée ici l'emporte sur la liste noire. Une liste plus courte que
les places libres laisse le reste au mod.

## Construire un équipement

Les préréglages couvrent les builds courants. Pour faire le vôtre, ouvrez la
page **Loadouts** des réglages, choisissez une classe, choisissez une arme par
emplacement, tapez un nom et appuyez sur Save.

L'équipement apparaît alors en bas de chaque menu d'armes de cette classe, sur
la page Team et sur la page Classes. Un équipement appartient à une seule
classe, parce qu'un Medic ne peut pas tenir un Gunslinger.

Si vous en supprimez un, toute place qui le nomme encore joue en stock.
L'équipe n'est pas perdue.

## Adapter une mission à une équipe incomplète

Valve règle chaque vague pour six défenseurs. La page **Balancing** des réglages
tient les deux façons d'aider une équipe qui n'en a pas six.

Les buffs d'armes rendent l'équipe plus forte. Ils sont actifs par défaut.

Les trois échelles robots rendent les robots plus faibles : dégâts, vie et
vitesse. Chacune est ce que les robots gardent avec un seul joueur sur RED, et
elle remonte à 100 à six. Une équipe complète joue donc toujours la mission
telle que Valve l'a écrite. Les trois partent à 100, ce qui ne change rien.

Seuls les dégâts robots ont une mesure derrière eux. À 70 ils ne font rien. À 50
ils commencent à infléchir une mission : une réussite sur huit tentatives. La
vague en question, le build d'origine l'a perdue vingt-quatre fois sur
vingt-quatre. La vie et la vitesse n'ont aucune mesure.

La vitesse robots fait plus que baisser la difficulté. Un robot plus lent
allonge la vague et laisse plus d'argent sur le terrain. Elle change aussi le
rythme d'une mission.

## Changer l'équipe en pleine mission

Une composition choisie pour la vague 1 est la mauvaise composition pour la
vague 5. Jusqu'ici, le seul recours était de relancer la mission.

Le lanceur a un onglet **Bot Switcher**, entre la liste des missions et le
journal. Il montre ce que chaque place joue et ce qu'elle porte. Pour changer
l'équipe, ouvrez la page Bots des réglages, réglez les places, puis appuyez sur
Appliquer.

Le mod remplace seulement les places dont la classe a changé, et laisse les
autres tranquilles. La vague continue, et les bots gardent l'argent qu'ils ont
gagné.

Dans le jeu, `!ap bots` ouvre la même équipe en menu. Choisissez une place,
puis choisissez une classe. Il faut le même droit d'admin que pour changer de
mission.

Un changement fait pendant une vague prend effet à la pause suivante. Un bot
retiré pendant une vague laisse tomber ses constructions, et son remplaçant
repart du spawn pendant que les robots avancent la bombe.

Le jeu ne permet plus d'inspecter les améliorations d'un coéquipier. Avec
`TF2AP_BOT_UPGRADES_CHAT=1`, le chat dit ce que chaque bot achète, une ligne
par achat. C'est désactivé par défaut parce qu'un bot achète beaucoup.

Les trois derniers ne changent que l'apparence. Un bot tire au sort une fois un
chapeau que sa classe peut porter et le garde jusqu'à la fin de la mission :
c'est le chapeau qui permet de distinguer un Heavy d'un autre. Il ne retire au
sort que s'il change de classe. Les effets sont désactivés par défaut parce que
six particules unusual restent à l'écran toute la vague. Aucun des deux ne
touche à la façon dont un bot joue, à ce qu'il achète ou à ce qu'il porte.

Les peintures d'armes ont existé et n'existent plus : elles peignaient les
entités d'arme que la station d'améliorations remplace, et le serveur mourait
dès que deux engineers finissaient leurs achats.

Tous prennent effet au chargement de la carte suivante. `make restart` est la
façon sûre.

Sur Windows, le lanceur a un onglet **Bots** pour les mêmes réglages. Six
menus, un par place, nomment l'équipe dans l'ordre. Un équipement prédéfini
par classe dit avec quelles armes un bot de cette classe apparaît. Les armes
de base sont le défaut. Les trois cases au bas de cet onglet sont l'apparence.
Voir
[Installer sur Windows](../setup/install-windows.md).

Un bot tient ses distances selon ce qu'il porte, pas selon sa classe. Un Brass
Beast se rapproche, parce qu'il ne peut plus se replacer une fois lancé ; un
Tomislav tient une ligne. Un fusil à pompe avance au lieu de tirer à portée de
minigun.

Un bot sort aussi une arme qui a encore des munitions, au lieu de marcher sur
un robot avec une arme vide. C'est ce que faisait un Heavy quand son minigun
était à sec.

Un Engineer s'installe près de la trappe, et non devant la porte de spawn des
robots. Une sentinelle posée là encaisse une vague entière sans équipe autour,
et ce que l'équipe y gagne c'est un Engineer qui reconstruit pendant tout le
reste de la vague. `sm_redbots_manager_engineer_nest_depth` dit jusqu'où il
peut monter sur le chemin de la bombe, en fraction de ce chemin : `0.4` par
défaut, `1.0` l'ancienne porte de spawn. Une fraction, parce que le chemin
fait quelques milliers d'unités sur Decoy et plusieurs fois ça sur
Rottenburg.

À l'upgrade station, un bot achète d'abord des dégâts, et il les achète pour
l'arme qu'il a en main. Avant, il achetait au hasard : c'est ainsi qu'un Heavy
finissait avec de la hauteur de saut et un minigun de base. Quelques armes
décident elles-mêmes : un Kritzkrieg achète du taux d'übercharge, un Rescue
Ranger achète du métal. Un Engineer achète la sentinelle, parce que ses dégâts
sont là, et un Medic achète du soin plutôt qu'un pistolet à seringues. Les
résistances viennent en dernier : un bot réapparaît à chaque vague.

## Qui les a écrits

[OfficerSpy/TF2-MvM-Defender-TFBots][mod], GPL-3.0, et cinq dépendances :
CBaseNPC, Actions, TF2Attributes, TF Econ Data et TF2Utils. Le serveur les
compile depuis la source. TF2Attributes reçoit un correctif à nous depuis
`deploy/patches/`, dont le README dit pourquoi.

Le mod lui-même vient de notre fork, [m-this/tf2-mvm-bots][fork]. Sa branche
`main` est un tag amont plus nos changements, et `DEFENDERBOTS_VERSION` nomme
un tag de cette branche.

Le comportement des bots est celui du mod. Un bot qui rentre dans un mur se
signale au dépôt d'OfficerSpy, pas à celui-ci. La liste noire de classes et le
fichier d'équipement du serveur sont à nous, sur le fork.

[mod]: https://github.com/OfficerSpy/TF2-MvM-Defender-TFBots
[fork]: https://github.com/m-this/tf2-mvm-bots

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

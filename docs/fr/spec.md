# TF2 MvM Archipelago : le cahier des charges

Ce document est la lecture consolidée du fil de discussion d'origine, en
anglais dans [`discord-mvm-thread.md`](./discord-mvm-thread.md), plus les
décisions d'architecture dans [`adr/`](./adr/). Là où le fil et ce fichier
sont en désaccord, ce fichier gagne, et nous signalons le désaccord.

## Périmètre

Mann vs Machine uniquement. Pas le TF2 compétitif, pas le casual, pas le
payload.

MvM est le seul mode TF2 qui a déjà une vraie forme de progression. Une
mission est une liste ordonnée de vagues. Les vagues se réussissent ou se
ratent. Une boutique vend des améliorations persistantes. Un tour est une
liste ordonnée de missions.

Cela se projette sur les régions et les locations d'Archipelago sans rien
inventer. Le TF2 classique n'a rien de tout cela. C'est pourquoi le fil est
allé directement à MvM, et pourquoi nous le suivons.

Déploiement visé : un `srcds` auto-hébergé dans un conteneur, un serveur
Archipelago dans un conteneur, un processus bridge. Les amis se
connectent avec un client TF2 standard et n'installent rien.

### Non-objectifs

- Aucun mod côté client. Les clients vanilla doivent pouvoir rejoindre
  (ADR 0002).
- Aucun support pour les serveurs officiels Valve Mann Up ou Boot Camp. Ils
  ne peuvent pas faire tourner de plugins, donc il n'y a nulle part où
  l'intégration peut vivre.
- Aucune tentative de soumettre ceci en amont à
  `ArchipelagoMW/Archipelago` dans la v1. À reconsidérer une fois que ça
  génère réellement et que quelqu'un termine une partie de bout en bout.
- Aucune exécution automatique des extensions serveur propres à Potato.tf ou
  Moonlight.tf. Les cartes communautaires, fichiers de population, VScript et
  tables d'améliorations standard sont pris en charge par un manifeste
  versionné ; une mission SigMod exige toujours l'installation séparée de
  cette dépendance.

## Architecture

Trois processus, une seule source de vérité partagée.

```
  gamedata/ (Go)  ──génère──>  apworld/tf2_mvm/data/*.json
        │                                 │
        │ compilé dans                    │ lu à la génération
        v                                 v
    bridge (Go)  <──websocket──>  serveur Archipelago (archipelago.gg)
        ^
        │ HTTP + JSON sur 127.0.0.1
        v
  plugin SourceMod  (dans le conteneur srcds)
```

**`gamedata/` (Go)** possède chaque fait MvM : les cartes, les missions,
les vagues par mission, les paliers de difficulté, la liste d'armes. Il
possède aussi les noms d'amélioration, les types de canteen, les
templates de robot, et assigne un id à chacun. Il se compile dans le
bridge et exporte du JSON pour l'apworld. Une table, deux consommateurs,
aucune dérive. ADR 0001.

**`apworld/tf2_mvm/` (Python)** reste volontairement mince. Il lit le JSON
exporté. Il construit les items et les locations à partir de lui. Il
déclare les régions et les règles d'accès, et expose les options YAML.
Aucune connaissance de MvM codée en dur en Python au-delà des règles
elles-mêmes.

**`bridge/` (Go)** est le client Archipelago. Il tient la session
websocket. Il gère la reconnexion et le rejeu, déduplique les items reçus,
et garde en mémoire ce qu'il a déjà appliqué. Il expose aussi une petite
API HTTP locale au plugin. ADR 0002.

**`plugin/` (SourcePawn)** est la seule chose qui voit le jeu. Il détecte
les objectifs et les rapporte, et applique les déblocages et les pièges.
Il n'a aucune connaissance d'Archipelago : il parle en termes MvM
(`wave_cleared`, `grant_weapon_slot`) et le bridge fait la traduction.

## Modèle de slot

C'est la question porteuse, parce que les tables d'items et de locations
dépendent de la réponse.

**Décision : un seul slot pour tout le serveur.** La progression est
collective. Chaque joueur de RED partage les mêmes armes débloquées, les
mêmes améliorations et les mêmes missions.

Justification : MvM est coopératif, et équilibré autour d'une équipe
coordonnée de six. Un slot par joueur laisse un ami sans arme principale,
pendant qu'un autre n'a pas d'arme de mêlée. La vague 6 d'une mission
Advanced ne se soucie pas de votre randomizer. Cela multiplie aussi le
nombre de checks par le nombre de joueurs, et rend la seed injouable si
quelqu'un part en cours de route.

Conséquence : le slot AP appartient au serveur, pas à un compte Steam. Le
bridge tient une seule connexion. Qui que ce soit sur le serveur joue ce
slot.

Des slots par joueur restent possibles plus tard comme option, mais pas en
v1. Le schéma d'id de gamedata ne doit pas supposer un slot unique pour
toujours.

## Locations (checks)

Chaque check doit être quelque chose que le plugin SourceMod peut
observer avec certitude, et une seule fois. Les succès MvM sont
délibérément exclus. Ils tiennent au compte Steam, donc un vétéran qui
les possède déjà ne déclenche jamais `achievement_earned`. Pour lui, la
seed devient imperdable ; pour un compte neuf, elle reste gagnable. C'est
un échec silencieux, pire qu'une fonctionnalité manquante.

| Groupe de locations | Déclencheur | Nombre | Option |
| --- | --- | --- | --- |
| Fin de vague | Vague N de la mission M terminée | vagues par mission, additionnées | `wave_checks` |
| Fin de mission | Dernière vague de la mission M terminée | une par mission du pool | `mission_checks` |
| Fin de tour | Chaque mission d'un tour terminée | une par tour | `tour_checks`, Campaign uniquement |
| Bonus d'argent | Vague terminée avec une note A+ | une par vague | `money_checks` |
| Achat en boutique | Une check achetée à l'upgrade station pour 100 à 400 $, puis remise | configurable | `shop_checks` |
| Kill de tank / boss | Tank détruit, Giant ou robot boss tué | par mission, plafonné | `boss_checks` |

Fin de vague est l'ossature, et le seul groupe activé par défaut. Tout le
reste est optionnel. La longueur d'une partie doit rester réglable, et le
nombre de vagues à lui seul donne déjà environ 6 à 8 checks par mission.

Les checks de boutique sont les plus intéressantes et les plus risquées.
Les deux variantes de Roseburst :

- **Remise à la vague** : achetez la check en boutique, réussissez la
  vague, recevez-la.
- **Remise à la mission** : achetez-la, réussissez toute la mission,
  recevez-la.

Les deux demandent au plugin d'injecter une entrée achetable dans
l'interface de l'upgrade station. Nous livrons `shop_checks` désactivé par
défaut, et il le reste tant que ce n'est pas confirmé sur un serveur réel.

L'idée de Damonj17 est amusante : « les améliorations comptent comme des
checks, comme les boutiques dans d'autres jeux ». Ouvrir la station
déverse alors un mur d'indices. Elle a sa place comme option, pas comme
défaut.

## Items

| Groupe d'items | Effet | Classification |
| --- | --- | --- |
| Emplacement d'arme | Débloque Principal, Secondaire ou Mêlée | progression |
| Arme | Débloque une arme précise pour une classe | progression |
| Paquet d'amélioration | Débloque une ligne d'amélioration sur chaque arme qui l'a | progression |
| Classe | Débloque une classe de mercenaire | progression |
| Canteen | Débloque un type de canteen | useful |
| Ticket de mission | Débloque une mission ou un tour | progression |
| Template de robot | Débloque un équipement de bot allié | useful |
| Crédits | Une liasse d'argent au début de la prochaine vague | filler |
| Piège | Voir ci-dessous | trap |

Deux façons mutuellement exclusives de verrouiller les améliorations,
proposées par Roseburst :

- **Weapon Unlocks** : une check par arme, et débloquer une arme donne
  toutes les améliorations dessus.
- **Upgrade Packages** : une check par ligne d'amélioration, partagée
  entre les armes. Obtenir « Primary Damage » donne des dégâts sur le
  Minigun, le Sydney Sleeper et le Flamethrower à la fois.

Les Packages produisent un pool d'items plus petit et plus intéressant.
Weapon Unlocks en produit un bien plus gros. Nous livrons les deux ;
`upgrades_shuffle` choisit.

L'état de départ doit être jouable. L'instinct d'adeleine64DS est le bon
réflexe : « on pourrait aussi exiger d'avoir la classe/l'arme avant de
pouvoir acheter l'amélioration. » Le générateur doit garantir au moins une
classe utilisable, avec au moins une arme utilisable, en sphère 0. Sinon
la vague 1 est imperdable, et la seed est morte-née.

## Bots alliés

Valve calibre les vagues MvM pour six joueurs. Une partie en solo sans
aide n'est pas un randomizer, c'est un mur.

L'option `Allied Mercs` de Roseburst (Off / Fill 6 / Fill 10 / Scavenge)
est la réponse. Elle vaut mieux que le multiplicateur de dégâts de
Damonj17. Un multiplicateur compense les dégâts manquants, mais pas un
Medic manquant, un Engineer manquant, ou un Scout manquant pour ramasser
l'argent. Les bots occupent au moins les rôles.

**Ce qui est livré :** le serveur embarque notre fork du mod MvM Defender
TFBots d'OfficerSpy. Il remplit RED au début de la vague. Ces bots choisissent
leur classe, se battent, achètent leurs propres améliorations et se déclarent
prêts. Partager les améliorations du joueur était le plan tant que les bots
RED d'origine semblaient la seule option. Des bots qui achètent les leurs
valent mieux, et ce sont eux qui tournent. Voir
[Les bots de votre équipe](play/defender-bots.md).

Les équipements de bot sont eux-mêmes des items sous `Robot Templates` ou
`Single Templates`, ce qui transforme « qui est dans mon équipe » en une
partie de la progression. Gardez cela. C'est l'idée la plus originale du
fil.

## Objectifs

- **Final Boss** : une mission du palier le plus difficile disponible est
  marquée, et la réussir gagne la partie.
- **Missionsanity** : réussir X % des missions du pool.
- **Australium Hunt** : le multiworld mélange N items Australium de
  rebut, et il en faut un pourcentage. C'est le seul objectif qui fait
  dépendre la partie des mondes des autres joueurs. C'est le principe
  même d'un multiworld, donc ce n'est pas un après-coup.

## Pièges et DeathLink

Une mort dans MvM est bon marché. Vous respawnez après un court délai, et
la vague continue. Une vague ne se termine que lorsque les robots déposent
la bombe. DeathLink sur une mort individuelle n'est donc que du bruit. Ici
une mort est une vague perdue, dans les deux sens :

- L'équipe perd une vague : le plugin écoute l'événement `mvm_wave_failed`
  du jeu. Le bridge envoie un DeathLink avec le nom de la mission et le
  numéro de la vague comme cause.
- Un DeathLink arrive : le plugin tue tout RED, bots compris. Il n'envoie
  aucun événement de vague perdue. C'est la trappe sans défense qui perd la
  vague, et c'est le jeu qui en décide. Une vague perdue pendant la fenêtre
  d'écho n'est pas renvoyée. Une mort ne peut donc pas faire l'aller-retour
  entre deux joueurs DeathLink.

La seed décide. Un slot avec `death_link` désactivé ne réclame jamais le
tag, n'entend aucune mort, et ignore ce que le plugin rapporte.

Pièges venant du fil, tous côté plugin :

- Canteen ou amélioration forcée mauvaise (Return to Spawn, Heavy Rage)
- Sentry Buster, Engineer, Sniper ou Spy généré
- Déclencheurs d'événement de carte (la barrière de Rottenburg, les points
  de capture de Mannhattan)
- Jarate sur toute l'équipe
- Bots alliés étourdis
- Un Giant ou un boss supplémentaire

Les pièges qui peuvent rendre une vague mathématiquement
imperdable-devenue-gagnable sont acceptables. Les pièges qui peuvent
corrompre l'état de la partie ne le sont pas : rien ne doit jamais retirer
définitivement un item reçu.

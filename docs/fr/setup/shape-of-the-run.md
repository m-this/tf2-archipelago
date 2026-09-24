# Les options de la partie

Les options de la partie sont les options Archipelago de votre seed. Elles
décident de la longueur de la partie, de sa difficulté, de ce qui la termine,
et de ce que les checks paient.

Chaque option a trois noms, pour les trois endroits où vous pouvez la régler :

- le libellé dans les **Settings** du lanceur,
- la clé dans le fichier joueur YAML, pour l'application Archipelago,
- la variable dans `.env`, pour Docker.

**Réglez-les avant de générer la seed.** La seed les garde. Un changement fait
plus tard ne fait rien tant que vous ne
[démarrez pas une nouvelle partie](../operate/start-a-new-run.md).

Le bouton **Check Run Selection** du lanceur est sur la page **Missions**. Il
vous dit si les missions choisies contiennent assez de checks pour les objets
de la partie. Le générateur Archipelago refait cette vérification et a le
dernier mot.

## La partie

Page du lanceur : **Player options**.

| Lanceur | YAML | `.env` | Défaut | Ce que l'option décide |
| --- | --- | --- | --- | --- |
| Easiest tier | `difficulty_pool` | `MVM_DIFFICULTY` | `intermediate` | Le palier le plus facile où la partie pioche. Les paliers plus durs sont toujours inclus. |
| Missions used | `mission_count` | `MVM_MISSION_COUNT` | `8` | Combien de missions la partie pioche, jusqu'à la taille de la réserve. |
| Goal | `goal` | `MVM_GOAL` | `final_boss` | Ce qui termine la partie. |
| Missionsanity share | `missionsanity_percentage` | `MVM_MISSIONSANITY_PERCENTAGE` | `80` | La part des missions que l'objectif Missionsanity demande. |
| Death Link | `death_link` | `MVM_DEATH_LINK` | désactivé | Partager les morts avec le multiworld. |
| Weapon slots per class | `class_weapon_slots` | `MVM_CLASS_WEAPON_SLOTS` | `off` | Chaque classe gagne ses propres emplacements. |
| Mission modifiers | `mission_modifiers` | `MVM_MISSION_MODIFIERS` | désactivé | Donner à chaque mission un jeu fixe de modificateurs. |
| Minimum modifiers | `minimum_mission_modifiers` | `MVM_MINIMUM_MISSION_MODIFIERS` | `1` | Le moins de modificateurs par mission. |
| Maximum modifiers | `maximum_mission_modifiers` | `MVM_MAXIMUM_MISSION_MODIFIERS` | `2` | Le plus de modificateurs par mission, jusqu'à 3. |

### Easiest tier

La partie pioche dans ce palier et dans tous les paliers au-dessus.

| Valeur | Missions dans la réserve | L'équipe commence avec |
| --- | --- | --- |
| `normal` | les 29 missions de Valve | 1 classe, 1 emplacement d'arme |
| `intermediate` | 25 | 2 classes, 1 emplacement d'arme |
| `advanced` | 17 | 3 classes, 2 emplacements d'arme |
| `expert` | 4 | 4 classes, 3 emplacements d'arme |
| `haunted` | 1, Caliginous Caper | 5 classes, 3 emplacements d'arme |

L'équipement de départ suit le palier de la mission la plus facile que la
partie a piochée.

Ne choisissez pas `haunted` pour une vraie partie. C'est une seule mission de
666 robots, et elle contient trop peu de checks pour les objets d'une partie.
Le lanceur ne la propose pas.

### Missions used

La plupart des missions de Valve ont six ou sept vagues. Huit missions font
environ 50 vagues, soit une soirée pour une équipe qui connaît le mode.

Si vous demandez plus de missions que la réserve n'en contient, vous avez
toute la réserve. Le générateur pioche plus de missions que demandé quand la
partie a besoin de plus de checks, et il le dit dans son journal.

### Goal

- `final_boss` marque la mission la plus dure que la partie a piochée.
  Réussissez-la pour gagner.
- `missionsanity` demande une part des missions, dans n'importe quel ordre.
  **Missionsanity share** est cette part, arrondie vers le haut. La valeur par
  défaut de 80 sur huit missions donne sept missions.

L'objectif Final Boss ignore la part Missionsanity.

### Death Link

Désactivé par défaut. Avec Death Link activé :

- une vague perdue tue tous les autres joueurs DeathLink du multiworld ;
- une mort d'un autre joueur DeathLink tue tout le monde sur RED, bots
  compris.

Attendez-vous à une partie plus dure. Une vague peut se perdre à cause de
l'erreur de quelqu'un d'autre.

### Weapon slots per class

- `off` : un objet `Progressive Weapon Slot` pour tout le monde, trois
  exemplaires. La valeur par défaut.
- `progressive` : chaque classe a son propre objet, deux exemplaires. Son
  premier emplacement vient avec la classe. Chaque classe ouvre ses
  emplacements dans son propre ordre.
- `any_order` : chaque emplacement de chaque classe est un objet nommé,
  trouvé dans n'importe quel ordre.

Les deux autres valeurs mettent dix-huit objets dans la réserve au lieu de
trois. Une partie courte n'a pas toujours les checks pour les contenir. La
génération le dit.

### Mission modifiers

Avec cette option, la seed donne à chaque mission une combinaison fixe de
modificateurs. Un modificateur change la carte, les robots, les joueurs ou
les vagues. Une mission garde la même combinaison quand vous la rejouez.
**Minimum** et **Maximum** bornent le nombre de modificateurs par mission.

## Plus de checks

Page du lanceur : **Player options**. Tout est désactivé par défaut.

| Lanceur | YAML | `.env` | Ce que l'option ajoute |
| --- | --- | --- | --- |
| Australium Medal on clear | `medal_on_clear` | `MVM_MEDAL_ON_CLEAR` | Une médaille à vous sur chaque mission réussie. L'objectif compte alors les médailles. Coûte un check par mission au multiworld. |
| Victory caches | `victory_caches` | `MVM_VICTORY_CACHES` | Une mission réussie paie des checks selon son palier : 1 en normal, 2 en intermediate, 3 en advanced, 5 en expert et haunted. |
| Milestone checks | `milestone_checks` | `MVM_MILESTONE_CHECKS` | Quinze checks pour des totaux sur toute la partie : robots, géants et tanks détruits, dans n'importe quelle mission. |
| Giantsanity | `giantsanity` | `MVM_GIANTSANITY` | Un check pour chaque géant de chaque vague. Empire Escalation en contient 82 à elle seule. |
| Tanksanity | `tanksanity` | `MVM_TANKSANITY` | Un check pour chaque tank de chaque vague. |

Utilisez ces options quand la partie manque de checks, ou quand votre équipe
veut plus de choses à trouver. Les caches de victoire ajoutent des checks sans
ajouter de missions.

Australium Medal on clear corrige un problème. Quand un autre joueur termine
et libère ses objets, vous pouvez recevoir une mission réussie avant de
l'avoir jouée. Avec une médaille sur chaque mission réussie, l'objectif
compte les médailles que vous avez, et personne ne peut vous en donner une.

## Les missions

Page du lanceur : **Missions**.

| Lanceur | YAML | `.env` | Défaut | Ce que l'option décide |
| --- | --- | --- | --- | --- |
| La liste des missions, avec une case par mission | `excluded_missions` | `MVM_EXCLUDED_MISSIONS` | aucune exclue | Les missions que la partie ne pioche jamais. |
| Community missions | `community_missions` | `MVM_COMMUNITY_MISSIONS` | activé | Si la partie pioche des missions communautaires. |
| Start mission | `start_mission` | `MVM_START_MISSION` | `random` | La mission où la partie commence. |
| Start class | `start_class` | `MVM_START_CLASS` | `random` | Une des classes avec lesquelles la partie commence. |
| La liste des mods serveur | `server_mods` | `SRCDS_MODS` | aucun | Les mods que le serveur charge. La partie ne pioche une mission qui demande un mod que si le mod est activé. |

### Missions exclues

Décochez une mission dans le lanceur, ou nommez-la dans le YAML ou `.env` par
son nom de fichier. `mvm_ghost_town_666` écarte Caliginous Caper, une seule
vague de 666 robots qui prend une heure à elle seule.

### Mission et classe de départ

`random` laisse les deux à la seed. La partie commence alors sur la mission la
plus facile qu'elle a piochée, avec des classes au hasard.

Nommez une mission, et la partie la pioche toujours et commence là. Nommez une
classe, et la partie commence toujours avec elle. Le palier de la mission de
départ décide toujours combien de classes la partie a au début.

Ne nommez pas la mission la plus dure de la partie comme départ avec
l'objectif Final Boss. La réussir gagne sur-le-champ, donc la génération
s'arrête.

### Missions communautaires

Les missions communautaires viennent de paquets de contenu que vous
téléchargez dans le lanceur, sur la page **Missions**. Une fois un paquet sur
le disque, ses missions apparaissent dans la liste. Voir le
[guide du contenu communautaire](https://github.com/m-this/tf2-archipelago/blob/main/community-content/README.md),
en anglais.

Certaines missions communautaires demandent un mod serveur. Le seul mod pour
l'instant est SigMod (`sigsegv-mvm`). Sur la page **Missions**, chaque mod a
trois réponses :

| Réponse | Ce que le serveur fait |
| --- | --- |
| **off** | Ne charge jamais le mod. Ses missions quittent la réserve. |
| **only when a mission needs it** | Le charge tant que la réserve contient une mission qui le nomme. La valeur par défaut. |
| **always, on every mission** | Le charge sur chaque carte. |

Le lanceur natif installe le mod quand il est nécessaire. L'image Docker le
contient déjà : enregistrez le choix et recréez les conteneurs pour l'appliquer.
Le désactiver laisse les fichiers en place, donc le réactiver ne coûte aucun
téléchargement.

Sur Windows la ligne affiche **SigMod (beta)**. Linux et Docker utilisent
la version publiée par le mod lui-même, que des joueurs font tourner tous les
jours. Windows n'a pas cette version : il télécharge le portage de ce projet.
Ce portage a fait planter le serveur d'un joueur. Laissez-le sur **only when a
mission needs it** sauf si vous le testez. Pour jouer aujourd'hui les missions qui le demandent,
lancez le serveur sur Linux, dans Docker, ou sous WSL.

Dans `.env` la réponse est `SRCDS_MOD_LOADING`, et Docker la lit comme on ou
off : l'image ne peut pas savoir quelles missions la réserve contient.

## Récompenses

Page du lanceur : **Rewards**. Ces options décident de ce qui remplit les
checks qui restent après les classes, les emplacements et les tickets.

| Lanceur | YAML | `.env` | Défaut | Ce que l'option décide |
| --- | --- | --- | --- | --- |
| Mission tickets | `mission_ticket_importance` | `MVM_MISSION_TICKET_IMPORTANCE` | `progression` | Si les tickets verrouillent les missions. |
| Class unlocks | `class_unlock_importance` | `MVM_CLASS_UNLOCK_IMPORTANCE` | `progression` | Si les classes comptent pour les exigences des paliers. |
| Weapon slots | `weapon_slot_importance` | `MVM_WEAPON_SLOT_IMPORTANCE` | `progression` | Si les emplacements comptent pour les exigences des paliers. |
| Weapon buffs | `weapon_buff_importance` | `MVM_WEAPON_BUFF_IMPORTANCE` | `useful` | Si les paliers durs demandent quelques bonus. |
| Cash rewards | `cash_rewards` | `MVM_CASH_REWARDS` | désactivé | Laisser les checks libres payer des crédits. |
| Cartes de bots déblocables | `bot_cards` | `MVM_BOT_CARDS` | désactivé | Placer chaque carte distincte restante sur un check libre si la seed a assez de place ; elles ne bloquent jamais la progression. |
| Mode des cartes de départ | `starting_bot_card_mode` | `MVM_STARTING_BOT_CARD_MODE` | `draw_random` | Cartes aléatoires ou une carte avec équipement de base par classe. |
| Cartes aléatoires de départ | `starting_bot_cards` | `MVM_STARTING_BOT_CARDS` | `0` | Cartes distinctes au départ ; ignoré en mode une par classe. |
| Buff share | `weapon_buff_percentage` | `MVM_WEAPON_BUFF_PERCENTAGE` | `75` | Avec les crédits activés, la part des checks libres qui paient un bonus. |
| Buff stack chance | `weapon_buff_stack_chance` | `MVM_WEAPON_BUFF_STACK_CHANCE` | `25` | La chance qu'un bonus ajoute un niveau à un bonus déjà dans la seed. |
| Traps (%) | `trap_percentage` | `MVM_TRAP_PERCENTAGE` | `1` | La part des checks libres qui contiennent un piège. |
| Grappling Hook | `server_settings` | `MVM_SERVER_SETTINGS` | désactivé | Mettre le grappin dans la réserve. |

La rareté et la forme sont tirées séparément : Commune / Élite / Légendaire
à 50 % / 40 % / 10 %, et Humain / robot RED / Géant à 50 % / 40 % / 10 %.

### Importance

`progression` veut dire que le générateur peut mettre l'objet derrière un
check dont vous avez besoin. `useful` veut dire que l'objet ne verrouille
jamais rien.

- Avec les tickets sur `useful`, chaque mission de la partie est ouverte dès
  le début.
- Avec les classes ou les emplacements sur `useful`, les paliers ne demandent
  rien.
- Avec les bonus sur `progression`, les paliers durs demandent quelques
  bonus.

### Crédits et bonus

Avec **Cash rewards** désactivé, chaque check libre paie un bonus d'arme. Avec
l'option activée, **Buff share** décide du partage, et le reste paie des
crédits.

Un `Cash Bundle` paie 200 crédits à chaque joueur de RED. Il est payé à la
station d'amélioration, entre les vagues, donc une vague perdue ne peut pas le
reprendre.

Un bonus d'arme est un avantage permanent sur une famille d'armes. Les bonus
numériques gagnent un niveau à chaque fois. Les bonus tout ou rien ne se
répètent jamais.

### Pièges

Un piège est un objet avec un mauvais effet. Un autre joueur le trouve, et
votre équipe le paie. Le seul piège pour l'instant est `Trap: Team Jarate` :
dix secondes de Jarate pour tout le monde sur RED, pendant la vague suivante.
Un piège ne reprend jamais un déblocage.

## Équilibrage

Page du lanceur : **Balancing**. C'est un réglage du serveur, pas une option
de la seed. Il s'applique au prochain chargement de carte.

| Lanceur | `.env` | Défaut | Ce que l'option décide |
| --- | --- | --- | --- |
| Robot health (%) | `SRCDS_BLU_HEALTH_PCT` | `100` | Un multiplicateur sur la santé de chaque robot, de 10 à 1000. |

Baissez-le pour une équipe réduite. Voir
[Les bots de votre équipe](../play/defender-bots.md#une-équipe-réduite).

## La room

Page du lanceur : **Archipelago room**.

| Lanceur | `.env` | Défaut | Ce que l'option décide |
| --- | --- | --- | --- |
| Room address | `AP_ROOM`, ou `AP_HOST` et `AP_PORT` | aucun | La ligne de la page de la room, du type `archipelago.gg:12345`. |
| Room password | `AP_PASSWORD` | vide | Le mot de passe de la room, si vous en avez mis un. |
| Slot name | `AP_SLOT_NAME` | `tf2` | Le nom de votre serveur dans la session. Il doit être le même que le `name` du fichier joueur. |
| Test mode | `TF2AP_TEST_MODE` | désactivé | Jouer sans room. Le lanceur simule un multiworld d'un seul joueur. |
| | `AP_TLS` | `true` | Docker seulement. `true` pour une room sur `archipelago.gg`, `false` pour une room hébergée dans la pile. |

## Le serveur de jeu

Page du lanceur : **Game server**.

| Lanceur | `.env` | Défaut | Ce que l'option décide |
| --- | --- | --- | --- |
| Server name | `SRCDS_HOSTNAME` | `Mann vs Archipelago` | Le nom dans le navigateur de serveurs. |
| Server password | `SRCDS_PW` | vide | Le mot de passe que les joueurs tapent pour rejoindre. Vide laisse entrer toute personne qui a l'adresse. |
| Game port | `SRCDS_PORT` | `27015` | Le port où les joueurs se connectent, UDP et TCP. |
| Join address | `TF2AP_JOIN_HOST` | cette machine | L'adresse affichée sur la ligne Join. |
| Admins by Steam id | `SRCDS_ADMIN_STEAMIDS` | vide | Qui peut changer de mission et de bots depuis le chat. Voir [Commandes de chat](../play/chat-commands.md#pour-ladmin). |
| | `SRCDS_RCONPW` | aucun | Docker seulement. Le mot de passe de la console distante. Le lanceur en choisit un pour vous. |
| | `SRCDS_START_MISSION` | vide | La mission que le serveur charge au démarrage. Le lanceur la règle depuis **Start mission**. |
| | `TF2AP_NEXT_MISSION_DELAY` | `30` | Les secondes entre une mission réussie et la mission suivante. |
| | `SRCDS_MAXPLAYERS` | `32` | Ne le baissez pas. TF2 refuse d'héberger MvM avec moins de places. |

La page **Networking** décide qui atteint le serveur. Voir
[Inviter vos amis](invite-your-friends.md).
[Les bots de votre équipe](../play/defender-bots.md) décrit les pages **Bots**
et **Loadouts**.

Suite : [Créer la session](create-the-session.md).

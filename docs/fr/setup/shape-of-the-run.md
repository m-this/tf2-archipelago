# La forme de la partie

Ces valeurs vivent dans `.env`. Elles décident de la longueur d'une
soirée, de sa difficulté, et de ce qui la termine.

**Réglez-les avant de fabriquer la session.** `make seed` les lit une fois, et
la session les garde. Un changement fait plus tard ne fait rien tant que
vous ne [démarrez pas une nouvelle partie](../operate/start-a-new-run.md).

## La partie

| Variable | Défaut | Plage | Ce qu'elle décide |
| --- | --- | --- | --- |
| `MVM_MISSION_COUNT` | `8` | 1 à 29 | Combien de missions la partie tire |
| `MVM_DIFFICULTY` | `intermediate` | `normal`, `intermediate`, `advanced`, `expert` | Le palier le plus facile que la partie peut tirer |
| `MVM_GOAL` | `final_boss` | `final_boss`, `missionsanity` | Ce qui termine la partie |
| `MVM_MISSIONSANITY_PERCENTAGE` | `80` | 10 à 100 | Quelle part de la partie l'objectif Missionsanity demande |
| `MVM_DEATH_LINK` | `false` | `true`, `false` | Une vague perdue tue les joueurs liés, et leurs morts anéantissent l'équipe. Voir ci-dessous. |

### `MVM_MISSION_COUNT`

La plupart des missions Valve contiennent six ou sept vagues, donc huit
missions font environ 50 vagues. C'est une soirée pour une équipe qui
connaît le mode.

Demander plus de missions que le pool n'en contient vous donne le pool
entier. Le générateur tire aussi plus de missions que demandé quand la
partie a besoin de plus de places pour ses déblocages. Il écrit une ligne
dans le log quand cela arrive.

### `MVM_DIFFICULTY`

C'est le palier le plus facile que la partie peut tirer. La partie tire ce
palier et tous ceux au-dessus. `intermediate` tire intermediate, advanced
et expert, soit 25 des 29 missions. `expert` tire les trois missions expert
et l'unique mission haunted, soit quatre.

Le palier de la mission la plus facile tirée fixe aussi ce avec quoi
l'équipe commence. Une partie normal commence avec une classe et un
emplacement d'arme. Une partie expert commence avec quatre classes et les
trois emplacements d'arme, parce qu'une mission expert ne se joue pas avec
moins.

**Ne réglez pas `haunted`.** Ce palier ne contient qu'une mission,
Caliginous Caper, et cette mission ne contient qu'une vague. Deux checks,
ce n'est pas assez de place pour les items dont une partie a besoin, donc
la génération s'arrête avec une erreur qui nomme l'option. Le conteneur
redémarre ensuite et affiche la même erreur à nouveau. Ce palier
fonctionnera le jour où Valve livrera une deuxième mission haunted.

### `MVM_GOAL`

`final_boss` marque la mission la plus difficile que la partie a tirée.
Réussir cette mission termine la partie. Cela donne une soirée avec un
dernier combat clair.

`missionsanity` demande une part des missions tirées, dans n'importe quel
ordre. `MVM_MISSIONSANITY_PERCENTAGE` est cette part, arrondie au
supérieur. Le défaut de `80` sur une partie à huit missions veut dire sept
missions dans n'importe quel ordre.

L'objectif `final_boss` ignore `MVM_MISSIONSANITY_PERCENTAGE`.

### `MVM_DEATH_LINK`

DeathLink est une convention où une mort dans un jeu tue tous les autres
joueurs qui l'ont activée. Dans MvM une mort individuelle est normale
plutôt que notable, donc ici une mort est une vague perdue par l'équipe.

Avec `true`, une vague perdue tue tous les joueurs liés du multiworld, et
l'une de leurs morts tue tout votre équipe, bots compris, ce qui fait
perdre la vague en cours. Cette perte-là n'est pas renvoyée. Attendez-vous
à une partie nettement plus dure : une vague peut se perdre sur l'erreur
de quelqu'un d'autre.

## La session

| Variable | Défaut | Ce qu'elle décide |
| --- | --- | --- |
| `AP_HOST` | `archipelago.gg` | Où tourne la session |
| `AP_PORT` | aucun | Le port de la room |
| `AP_TLS` | `true` | `wss://` plutôt que `ws://`. Une room sur `archipelago.gg` répond en `wss://`. |
| `AP_SLOT_NAME` | `tf2` | Le nom de votre serveur dans la session |
| `AP_PASSWORD` | vide | Le mot de passe de la room |

`AP_HOST`, `AP_PORT` et `AP_TLS` viennent de la page de la room. Voir
[Créer la session](create-the-session.md), qui couvre aussi les trois valeurs
qui hébergent la session sur votre propre machine.

N'importe qui atteint une room dont il connaît l'adresse. Mettez un mot de
passe sur la room et répétez-le dans `AP_PASSWORD`, sauf si l'adresse reste
entre amis.

`AP_SLOT_NAME` est le nom que la session donne à votre serveur dans son log et
dans le chat qui arrive à vos joueurs. Changez-le si vous voulez que les lignes
se lisent mieux.

Jouer avec quelqu'un dans un autre jeu demande une session qui contient plus
d'un slot. La génération lit un fichier d'options, écrit par
`deploy/archipelago-entrypoint.sh`, et un deuxième joueur demande un deuxième
fichier à côté. La room accueille ce joueur sans ouvrir de port.

## Le serveur de jeu

| Variable | Défaut | Ce qu'elle décide |
| --- | --- | --- |
| `SRCDS_HOSTNAME` | `Mann vs Archipelago` | Le nom du serveur dans le navigateur |
| `SRCDS_ADMIN_STEAMIDS` | vide | Qui peut utiliser `!mission` et les commandes `sm_ap_` dans le chat. Des identifiants Steam, séparés par des virgules. Soit le format à 17 chiffres d'une URL de profil, soit le `STEAM_0:1:...` de SourceMod. |
| `SRCDS_MAXPLAYERS` | `32` | Emplacements du serveur. Team Fortress 2 refuse d'héberger MvM avec moins, et plafonne RED à six lui-même. Ne le baissez pas. |
| `SRCDS_STARTMAP` | `mvm_decoy` | La carte sur laquelle le serveur démarre |

`SRCDS_STARTMAP` accepte n'importe quelle carte `mvm_`. La partie ne
choisit pas la carte pour vous. Changez-la entre les missions depuis la
console distante, ou modifiez `.env` et redémarrez la stack.

`SRCDS_PORT`, `SRCDS_PW`, `SRCDS_RCONPW` et `SRCDS_TOKEN` sont couverts
dans [Inviter vos amis](invite-your-friends.md).

## Où vont ces valeurs

`make seed` les écrit dans un fichier de configuration et génère la session à
partir de celui-ci. Ensuite, le fichier appartient au passé : c'est la session
que la partie suit.

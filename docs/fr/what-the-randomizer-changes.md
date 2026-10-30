# Ce que le randomizer change

Tout ce qui est sur cette page se passe sur le serveur. Les joueurs
n'installent rien. Pour les mots, voir
[Les mots d'Archipelago, pour les joueurs MvM](archipelago-for-mvm-players.md).

## Les classes commencent verrouillées

- Chacun des neuf mercenaires est un objet.
- Une partie commence avec un à quatre d'entre eux. Le palier de la mission la
  plus facile décide combien. Voir [L'équipement de départ par palier](#léquipement-de-départ-par-palier).
- Choisir une classe verrouillée dans le menu des classes ne fait rien, et le
  chat dit pourquoi.
- Un joueur déjà sur une classe qui se verrouille continue à jouer jusqu'à sa
  prochaine apparition. Le serveur ne force jamais une réapparition.

## Les emplacements d'arme commencent verrouillés

- Il y a trois emplacements d'arme : primaire, secondaire, mêlée.
- Par défaut, un seul objet, `Progressive Weapon Slot`, les ouvre un par un.
  La réserve en contient trois exemplaires.
- Un emplacement verrouillé est vide. Le serveur retire l'arme à l'apparition,
  à l'armoire de réapprovisionnement et à la station d'amélioration. Vous
  n'avez jamais les mains vides : le serveur vous passe une arme que vous
  avez encore.
- L'arme que vous mettez dans un emplacement ouvert est votre choix. La
  partie ne tire pas les armes au sort. Scattergun ou Force-A-Nature, les
  deux marchent une fois le primaire ouvert.

Le premier emplacement qui s'ouvre est celui dont la classe a le plus besoin.
L'ordre n'est donc pas le même pour toutes les classes :

| Classe | Premier | Deuxième | Troisième |
| --- | --- | --- | --- |
| Scout, Soldier, Pyro, Demoman, Heavy, Sniper | Primaire | Secondaire | Mêlée |
| Medic | Secondaire (Medigun) | Primaire | Mêlée |
| Engineer | Mêlée (Wrench) | Primaire | Secondaire |
| Spy | Mêlée (Knife) | Secondaire (Sapper) | Primaire |

L'[option de la partie](setup/shape-of-the-run.md#weapon-slots-per-class)
**Weapon slots per class** donne à chaque classe ses propres objets
d'emplacement. Cela fait dix-huit objets au lieu de trois, donc il faut une
partie plus longue.

## Les missions sont derrière des tickets

- Chaque mission a son propre objet `Mission Ticket`.
- La partie commence avec une mission ouverte. Les tickets ouvrent les autres.
- Le plugin ne refuse pas une carte. Si le serveur lance une mission que la
  partie n'a pas débloquée, le chat le dit et les vagues comptent quand même.
- La logique est par mission, pas par vague. Un ticket met toute la mission
  en logique d'un coup.

Le générateur demande aussi quelques classes et emplacements avant de
considérer une mission comme faisable. Ces nombres sont bas exprès. Une vague
dure reste possible.

### L'équipement de départ par palier

| Palier de la mission | Classes | Emplacements d'arme |
| --- | --- | --- |
| Normal | 1 | 1 |
| Intermediate | 2 | 1 |
| Advanced | 3 | 2 |
| Expert | 4 | 3 |
| Haunted | 5 | 3 |

La partie commence avec l'équipement que sa mission la plus facile demande.

## Une vague réussie est un check

Chaque mission donne ces checks :

- un par vague que l'équipe réussit,
- un quand l'équipe réussit la mission,
- un pour le premier tank que l'équipe détruit dans cette mission,
- un pour le premier géant que l'équipe tue dans cette mission.

Les missions de Mannhattan n'ont pas de tank, donc pas de check de tank. Une
vague perdue ne donne rien et ne coûte rien. L'équipe la rejoue, comme en MvM
normal.

Les [options de la partie](setup/shape-of-the-run.md#plus-de-checks) peuvent
ajouter des checks : caches de victoire, jalons, un check par géant et par
tank.

## Les objets que vous recevez

| Objet | Ce qu'il fait |
| --- | --- |
| `Class: Scout` et les huit autres | Ouvre cette classe pour tout le monde |
| `Progressive Weapon Slot` | Ouvre l'emplacement d'arme suivant pour tout le monde |
| `Mission Ticket: ...` | Met cette mission dans la partie |
| Bonus d'arme | Un bonus permanent sur une famille d'armes : plus de dégâts, plus de projectiles, rechargement plus rapide, etc. La station d'amélioration affiche la liste des bonus de votre équipement. |
| `Cash Bundle` | 200 crédits pour chaque joueur de RED, payés à la prochaine visite de la station d'amélioration |
| `Trap: Team Jarate` | Dix secondes de Jarate pour toute l'équipe, pendant la vague suivante |
| `Trap: Team Mad Milk`, `On Fire`, `Slow Down`, `Bleed`, `Marked For Death`, `Gas` | Pareil, avec cet effet à la place. Seulement dans une partie qui les demande. |
| `Grappling Hook` | Active le grappin de Mannpower pour tout le monde jusqu'à la fin de la partie. Désactivé par défaut. |

Les bonus, les crédits et les pièges remplissent les checks qui restent après
les classes, les emplacements et les tickets. Les
[options de la partie](setup/shape-of-the-run.md#récompenses) décident du
mélange.

Les classes, les emplacements et les tickets sont permanents. Le serveur les
applique de nouveau après tout redémarrage. Les crédits sont payés une fois,
et ils appartiennent à la mission comme les autres crédits.

## DeathLink

DeathLink est désactivé sauf si la seed le demande. Avec DeathLink activé :

- votre équipe perd une vague, et tous les autres joueurs DeathLink meurent ;
- un autre joueur DeathLink meurt, et tout le monde sur RED meurt, bots
  compris.

Le plugin ne fait que tuer. Le jeu décide si la vague est perdue, comme
toujours. Une vague perdue à cause d'une mort reçue n'est pas renvoyée.

## Ce qui ne change pas

- La station d'amélioration, les crédits, les gourdes, la composition des
  vagues et les robots sont ceux du MvM d'origine.
- Les armes elles-mêmes ne sont pas tirées au sort. Les améliorations non
  plus.
- La partie appartient au serveur. Rien ne va sur le compte Steam de qui que
  ce soit.

Suite : [Prérequis](setup/requirements.md).

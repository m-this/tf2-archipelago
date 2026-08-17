# Archipelago pour les joueurs MvM

Archipelago est un randomizer qui couvre plusieurs jeux à la fois. Cette
page définit les sept mots que le reste du livre utilise. Chaque mot a un
exemple MvM.

Vous n'avez pas besoin de cette page pour jouer. Lisez-la si vous voulez
savoir pourquoi votre Scout n'a pas de scattergun.

## Multiworld

Un multiworld est une seule session randomisée qui couvre plusieurs
joueurs et plusieurs jeux à la fois. Les déblocages de chaque jeu circulent
dans toute la session, pas seulement dans un jeu.

Un multiworld peut contenir votre serveur Team Fortress 2, un jeu Zelda et
un jeu Doom. Le fusil que votre équipe a besoin peut se trouver derrière
une porte dans le jeu Zelda. L'épée dont le joueur Zelda a besoin peut se
trouver derrière la vague 4 de Coal Town. Personne ne termine seul.

Un multiworld qui ne contient que ce serveur Team Fortress 2 est une
configuration normale. Le `.env` par défaut produit exactement cela.

## Slot

Un slot est un participant à un multiworld. Un slot a un nom et joue un
seul jeu.

**Dans ce projet, le slot est le serveur, pas un joueur.** Les six joueurs
de RED partagent un seul slot et un seul ensemble de déblocages. C'est
voulu. Valve équilibre MvM pour une équipe de six. Un slot par joueur
laisserait un joueur sans arme principale et un autre sans arme de mêlée,
dans un mode où l'équipe doit tenir une trappe ensemble.

Le mot « slot » désigne aussi un emplacement d'arme dans MvM. Ce livre
écrit toujours « emplacement d'arme » pour le sens MvM.

## Seed

Une seed est un multiworld généré. Le générateur décide une fois pour
toutes où se trouve chaque déblocage, et rien ne bouge ensuite.

Votre stack génère une seed au premier démarrage et l'héberge jusqu'à ce
que vous la supprimiez. La même vague tient le même déblocage toute la
soirée, et la soirée suivante aussi. Une nouvelle seed est une nouvelle
partie, pas un second essai. Voir
[Démarrer une nouvelle partie](operate/start-a-new-run.md).

## Item

Un item est un déblocage que le multiworld distribue. Dans ce projet, les
items sont :

| Item | Ce qu'il fait |
| --- | --- |
| `Class: Scout` et les huit autres classes | Rend ce mercenaire jouable |
| `Progressive Weapon Slot` | Ouvre l'emplacement principal, puis secondaire, puis mêlée |
| `Mission Ticket: Crash Course` et un par mission | Marque cette mission comme faisant partie de la partie |
| `Cash Bundle` | Paie 200 crédits à chaque joueur du serveur |

Un item peut appartenir à n'importe quel slot du multiworld. Réussir une
vague sur votre serveur peut donner une épée au joueur Zelda. Leurs items
arrivent dans votre chat quand ils trouvent les vôtres.

## Check et location

Une location est un endroit qui contient un item. Une check est l'action
d'atteindre cet endroit et de trouver ce qu'il contient.

Dans ce projet, une location est un objectif MvM :

- Chaque vague que l'équipe réussit est une location. « Doe's Doom Wave 3 »
  en est une.
- Chaque mission que l'équipe réussit est une location. « Doe's Doom
  Complete » en est une.

Les 29 missions Valve contiennent 181 vagues. Avec les 29 fins de mission,
cela fait 210 locations dans la table. Une partie utilise les missions
qu'elle a tirées, pas toutes.

Quand votre équipe réussit la vague 3 de Doe's Doom, le serveur fait la
check de cette location. Le multiworld apprend que l'item de cet endroit
est trouvé, et l'envoie à son propriétaire.

## Progression item

Un progression item est un item que la logique du générateur peut exiger
pour atteindre une autre location. Le générateur garantit que la partie
est finissable dans un certain ordre en n'utilisant que ceux-là.

Les emplacements d'arme, les items de classe et les tickets de mission sont
des progression items ici. Une mission advanced demande trois classes et
deux emplacements d'arme avant que le générateur ne place quoi que ce soit
d'important derrière. Une mission difficile en fin de chaîne se trouve
derrière beaucoup d'entre eux.

Se tromper sur cette classification produit une partie infinissable, ou une
partie sans aucun ordre. C'est pourquoi la liste d'items ci-dessus est
courte et fixe.

## Filler

Un filler item est un item sans place dans la logique. C'est ce qui
remplit le pool quand il y a plus de locations que de progression items.

`Cash Bundle` est le filler ici. La partie a 40 items nommés contre jusqu'à
210 locations, donc la plupart des vagues contiennent soit du filler, soit
un item appartenant à un autre jeu du multiworld.

## Ce que ça change au clavier

Vous commencez avec une partie d'un équipement. Une ou deux classes, un
emplacement d'arme, une mission. Chaque vague que vous réussissez trouve
quelque chose.

Une classe grisée, ou une arme qui n'est pas entre vos mains, est un item
que la partie n'a pas encore trouvé. C'est le jeu, pas un défaut.

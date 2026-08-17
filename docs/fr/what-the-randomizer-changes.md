# Ce que le randomizer change

Cette page suppose le vocabulaire de
[Archipelago pour les joueurs MvM](archipelago-for-mvm-players.md).

Tout ce qui suit se passe sur le serveur. Les joueurs n'installent rien.

## Les classes commencent verrouillées

Les neuf mercenaires sont neuf items séparés. Une partie commence avec un à
quatre d'entre eux, selon la difficulté de la mission la plus facile de la
partie.

| Palier de la mission de départ | Classes au départ | Emplacements d'arme au départ |
| --- | --- | --- |
| Normal | 1 | 1 |
| Intermediate | 2 | 1 |
| Advanced | 3 | 2 |
| Expert | 4 | 3 |

Un joueur qui choisit une classe verrouillée dans le menu de classe reçoit
une ligne dans le chat et reste où il est. Un joueur déjà sur une classe
que la partie n'a pas débloquée continue de jouer la vague et change de
classe au prochain spawn. Le plugin ne force jamais un respawn : dans MvM,
un respawn gratuit rend la vie que mourir vous a coûtée.

## Les emplacements d'arme commencent verrouillés

Il y a trois emplacements d'arme : principal, secondaire, mêlée. Un item,
`Progressive Weapon Slot`, les ouvre dans cet ordre. Il y en a trois
exemplaires dans le pool.

Un emplacement d'arme verrouillé est vide. Le serveur retire l'arme à
chaque fois que le jeu vous la donne : au spawn, au casier de
ravitaillement, et à l'upgrade station. Si l'arme qui disparaît est celle
entre vos mains, le serveur en met une autre à la place, pour que vous ne
restiez jamais les mains vides.

Les armes individuelles ne sont pas randomisées. Le Scattergun et le
Force-A-Nature sont le même item pour cette partie : un emplacement d'arme
principale.

## Les missions se trouvent derrière des tickets

Chacune des 29 missions Valve a son propre item `Mission Ticket`. Les
tickets sont ce que le générateur utilise pour décider l'ordre de la
partie.

Le plugin ne refuse pas une carte. Si le serveur fait tourner une mission
dont la partie n'a pas trouvé le ticket, le chat le dit et les vagues
comptent quand même comme des checks. La rotation des cartes vous
appartient, pas au randomizer.

## Une vague réussie est une check

Chaque vague que l'équipe réussit rapporte une check. Chaque mission que
l'équipe réussit en rapporte une de plus.

Une équipe anéantie rejoue la vague, comme dans MvM normal. Une vague ratée
ne rapporte rien. Il n'y a pas de pénalité au-delà de la vague elle-même.

## Les crédits peuvent arriver comme items

`Cash Bundle` paie 200 crédits. Chaque joueur de RED reçoit les 200
complets, donc un bundle qui arrive avec six joueurs sur le serveur, c'est
1200 crédits d'améliorations.

Un bundle qui arrive quand personne n'est sur le serveur ne paie personne.
Les crédits sont le seul item de ce projet qui arrive une fois et c'est
terminé. Les classes, les emplacements d'arme et les tickets sont des faits
qui restent vrais, donc le serveur peut les réappliquer après n'importe
quel redémarrage.

## Tout le monde partage les déblocages

Il y a un seul ensemble de déblocages pour tout le serveur. Un emplacement
d'arme que la partie trouve s'ouvre pour les six joueurs au même moment.
Une classe que la partie n'a pas trouvée est verrouillée pour les six.

Cela veut aussi dire qu'un joueur qui rejoint en milieu de soirée reçoit
l'ensemble actuel tout de suite. Il n'y a rien de propre à un joueur à
reporter.

## Personne n'installe rien

Vos amis se connectent avec un client Team Fortress 2 standard. Aucun mod,
aucun launcher, aucune liaison de compte. Tout ce qu'ils voient arrive
comme du texte de chat venant du serveur.

La partie appartient au serveur. Elle n'est stockée sur le compte Steam de
personne.

## Ce qui ne change pas

L'upgrade station, les crédits lâchés par les robots, les canteens, la
structure des vagues et les robots eux-mêmes sont du MvM standard. Cette
version randomise les classes, les emplacements d'arme, les missions et
l'argent. Les armes individuelles, les lignes d'amélioration, les canteens
et les pièges n'en font pas partie.

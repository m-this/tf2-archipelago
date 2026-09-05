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
`Progressive Weapon Slot`, les ouvre un par un. Il y en a trois exemplaires
dans le pool.

L'emplacement qu'un exemplaire ouvre dépend de la classe que vous jouez. Six
classes reçoivent le principal, puis le secondaire, puis la mêlée. Trois non,
parce que leur premier emplacement est ce qui définit la classe :

| Classe | Premier | Deuxième | Troisième |
| --- | --- | --- | --- |
| Medic | Secondaire, le Medigun | Principal | Mêlée |
| Engineer | Mêlée, la clé et les PDA | Principal | Secondaire |
| Spy | Mêlée, le couteau | Secondaire, le sapeur | Principal |

Ce que la partie tient est un nombre, pas un emplacement. Une partie avec un
exemplaire ouvre le Medigun pour un Medic et le Scattergun pour un Scout, au
même moment.

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

### La logique est par mission, pas par vague

Une mission est une seule région et toutes ses checks partagent la même porte :
les vagues, le tank, le géant et la réussite. Un ticket met donc la mission
entière en logique d'un coup. Aucune graine ne met la vague 3 en logique et la
vague 6 hors logique. La réponse à « quelle part d'une vague dois-je faire » est
donc : la mission entière.

Un ticket ne suffit pas toujours. Chaque palier demande aussi un nombre de
classes et d'emplacements d'arme avant que ses missions comptent pour battables :

| Palier | Classes | Emplacements |
| --- | --- | --- |
| Normal | 1 | 1 |
| Intermediate | 2 | 1 |
| Advanced | 3 | 2 |
| Expert | 4 | 3 |
| Haunted | 5 | 3 |

Les deux comptes restent toujours atteignables : le pool de chaque graine tient
toutes les classes et tous les emplacements. Ils restent volontairement sous ce
qu'une vraie équipe demande. La logique tranche ce qui est possible, et une
vague seulement difficile reste possible.

## Une vague réussie est une check

Chaque vague que l'équipe réussit rapporte une check. Chaque mission que
l'équipe réussit en rapporte une de plus. Le premier tank que l'équipe détruit
dans une mission en rapporte encore une, et le premier géant aussi. Les deux
comptent une seule fois par mission, quel que soit leur nombre.

Les trois missions de Mannhattan tournent sur des portes et n'ont pas de tank.
Elles n'ont donc pas de check de tank. Toutes les missions ont un géant, donc
toutes ont cette check-là. Une check que personne ne peut atteindre est une
partie que personne ne peut finir.

Une vague perdue ne rapporte rien, et le randomizer n'ajoute aucune pénalité.
L'équipe rejoue la vague, comme dans MvM normal.

Le jeu décide quand une vague est perdue, et ce projet ne change pas cette
règle. Les robots doivent porter la bombe jusqu'à la trappe. Un
anéantissement de l'équipe ne perd pas la vague à lui seul : le jeu fait
respawn l'équipe, et la vague continue.

## Le DeathLink tue l'équipe, et rien de plus

Le DeathLink reste désactivé sauf si la seed le demande. Activé, une mort
traverse le multiworld dans les deux sens.

Sortant : l'équipe perd une vague, et le bridge envoie cette perte comme une
mort. Le plugin écoute l'événement `mvm_wave_failed` du jeu. Les autres
joueurs reçoivent donc ce qui a terminé la vague sur votre écran.

Entrant : une mort arrive, et le plugin tue tout RED, bots compris. Il
n'envoie aucun événement de vague perdue. Personne ne tient la trappe jusqu'au
respawn de l'équipe. La vague est donc en général perdue, mais c'est le jeu
qui en décide, comme toujours.

Une vague perdue sur une mort entrante n'est pas renvoyée. Une mort ne peut
donc pas faire l'aller-retour entre deux joueurs liés.

## Les crédits peuvent arriver comme items

`Cash Bundle` paie 200 crédits. Chaque joueur de RED reçoit les 200
complets, donc un bundle qui arrive avec six joueurs sur le serveur, c'est
1200 crédits d'améliorations.

Un bundle paie entre les vagues, à l'upgrade station, et pas avant. Une vague
que l'équipe perd la ramène là où la vague a commencé, et l'argent versé dans
cette vague repart avec elle. Attendre l'upgrade station ne coûte rien, parce
que c'est là que l'argent se dépense.

Un bundle qui arrive en pleine vague attend donc la fin de celle-ci. Un bundle
qui arrive quand personne n'est sur le serveur attend quelqu'un. Le chat dit
combien attend. Rien n'est perdu dans les deux cas.

Les crédits restent le seul item de ce projet qui arrive une fois et c'est
terminé. Les classes, les emplacements d'arme et les tickets sont des faits
qui restent vrais. Le serveur peut donc les réappliquer après n'importe quel
redémarrage.

Un bundle est payé une fois. L'argent appartient ensuite à la mission comme
n'importe quels crédits, et la fin d'une mission l'efface, comme dans MvM
normal.

## Les pièges arrivent des autres joueurs

Un piège est un item à effet négatif, et il appartient au multiworld comme
n'importe quel autre item. Quelqu'un dans un autre monde ouvre un coffre, et
votre équipe le paie.

`trap_percentage` décide de la part de l'espace libre de la partie qui en
contient un. Il vaut zéro par défaut, donc une seed qui n'en demande pas n'en
reçoit aucun.

Il y a un piège pour l'instant. `Trap: Team Jarate` arrose tout RED, bots
compris : dix secondes à prendre 35 % de dégâts en plus et à n'infliger aucun
crit.

Un piège se déclenche pendant une vague. Celui qui arrive entre deux vagues
attend la suivante, et le chat le dit. Du Jarate sur une équipe plantée devant
l'upgrade station n'est pas un piège.

Un piège peut coûter une vague à l'équipe. Aucun piège ne reprend un déblocage
que la partie a déjà trouvé.

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
l'argent. Les armes individuelles, les lignes d'amélioration et les canteens
n'en font pas partie.

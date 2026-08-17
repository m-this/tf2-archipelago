# ADR 0001 — Go possède les données de jeu MvM ; l'apworld est un lecteur Python mince

- **Statut** : Accepté
- **Date** : 2026-08-13
- **Décideurs** : le propriétaire du projet
- **Lié à** : `docs/spec.md`, `docs/prior-art.md`, ADR 0002

## Contexte

Une intégration de jeu Archipelago a besoin de deux choses qui dépendent
toutes deux des mêmes faits sur le jeu :

1. Un **apworld**, qui tourne à l'intérieur du processus générateur
   Archipelago et doit être en Python. Il déclare le pool d'items, la
   liste de locations, les régions, les règles d'accès et les options
   YAML.
2. Un **client d'exécution**, qui tient la session multiworld et traduit
   entre les événements du jeu et les messages Archipelago.

Les deux ont besoin des mêmes tables : chaque carte MvM, chaque mission
avec son nombre de vagues et son palier de difficulté, chaque arme et à
quelle classe et quel emplacement elle appartient, chaque ligne
d'amélioration, chaque canteen, chaque template de robot allié, et un id
numérique stable pour chacun d'eux.

La convention du projet est que tout ce qui est sur mesure est écrit en
Go. L'apworld en Python n'est pas négociable : le générateur Archipelago
l'importe et l'appelle, donc il tourne dans ce processus ou nulle part.

Reste la question de savoir où vivent les tables. Le fork
d'`ALPHAMARIOX` les met en Python (`worlds/tf2/Items.py`, 556 lignes de
dicts) et laisse les fonctions d'assignation d'id comme des stubs vides,
ce qui est précisément la partie difficile.

## Décision

**`gamedata/` est un paquet Go et c'est le seul endroit où les faits MvM
sont écrits.** Il se compile dans le bridge, et il a un exporteur qui
écrit du JSON dans `apworld/tf2_mvm/data/`. L'apworld Python lit ce JSON
au moment de l'import et construit ses tables d'items et de locations à
partir de lui.

Précisément :

- Chaque table du `Items.py` d'`ALPHAMARIOX` est portée vers des structs
  Go, pas vendue telle quelle en Python. Le bitmask `Group` est porté
  comme un bitmask Go avec les mêmes noms de membres pour que les deux
  restent comparables.
- **Les ids sont assignés en Go et sont append-only.** Chaque entité porte
  un littéral d'id explicite dans le source. Les ids ne sont jamais
  renumérotés, jamais réutilisés après suppression, et une entité retirée
  garde son id réservé avec une tombstone. Un test vérifie qu'aucune
  entité ne partage un id et qu'aucun id présent dans l'export commité n'a
  changé. Renuméroter un id invalide silencieusement chaque seed jamais
  générée, et il n'y a aucun moyen de le détecter en jouant, donc c'est
  gardé au moment du commit à la place.
- Le JSON exporté est **commité**, pas ignoré par git, et l'exporteur
  tourne en CI pour vérifier que la copie commitée correspond au source
  Go. Le commiter veut dire que l'apworld est un artefact autonome :
  quelqu'un peut zipper `apworld/tf2_mvm/` et le donner à un ami sans
  toolchain Go.
- La chaîne de nom du jeu, le décalage d'id de base AP, et la version du
  format des données vivent tous dans `gamedata/` et sont exportés avec
  les tables. L'apworld refuse de charger un fichier de données dont il
  ne connaît pas la version de format.
- Python tient la logique qui ne peut pas être de la donnée : la
  construction du graphe de régions, les règles d'accès, les hooks de
  remplissage, et les classes `Options`. `Options.py` est adapté depuis le
  fork plutôt que généré, parce que les classes d'options sont du code
  avec des docstrings que le site AP affiche, pas une table.

## Conséquences

**Positives**

- Un seul endroit à modifier. Ajouter une mission communautaire est un
  littéral de struct Go et une régénération, pas deux modifications qui
  peuvent diverger.
- Le bridge et le générateur ne peuvent pas être en désaccord sur ce que
  signifie « vague 3 de Mean Machines », parce qu'ils lisent les mêmes
  nombres depuis la même origine.
- La stabilité des ids devient une propriété testable dans un langage
  avec un lanceur de tests que nous faisons déjà tourner en CI, plutôt
  qu'une convention que personne ne vérifie.
- Porter les tables du fork vers Go force la lecture des 556 lignes, ce
  qui est comment l'incohérence de nom de jeu dans le fork a été trouvée
  en premier lieu.

**Négatives**

- Une étape de génération existe, et un export périmé est un vrai mode
  d'échec. La CI qui l'attrape est l'atténuation, pas une correction.
- Les contributeurs de la communauté Archipelago s'attendront à un
  apworld normal et trouveront un blob JSON et un paquet Go. Le dossier
  de l'apworld a besoin d'un README qui le dit dans le premier
  paragraphe.
- Deux langages pour une seule couche conceptuelle. Accepté : l'alternative
  est du Python partout, ce qui perd le bridge, ou du Go partout, ce qui
  est impossible.
- Si ceci est un jour soumis en amont, l'apworld qui lit du JSON pourrait
  ne pas être accepté tel quel. C'est un problème de v2 et `spec.md`
  déclare déjà la soumission en amont hors périmètre pour la v1.

## Alternatives considérées

- **Un apworld Python normal écrit à la main** (finir les fichiers
  d'`ALPHAMARIOX`). Rejeté : le bridge a alors besoin de sa propre copie
  des mêmes tables en Go, et les deux dérivent dès que quelqu'un ajoute
  une mission. C'est aussi l'option qui contredit le plus directement la
  convention « le code sur mesure est en Go ».
- **Un apworld Manual** (JSON plus des hooks, entièrement généré depuis
  Go). Rejeté : Manual ne peut pas exprimer de vraies règles
  d'accessibilité, donc une seed pourrait placer le ticket de la mission
  finale derrière cette même mission. Tout l'intérêt des options Mission
  Order et Goal dans `spec.md` est un graphe logique, et Manual n'a pas de
  graphe.
- **Générer le source Python depuis Go** plutôt que du JSON lu par
  Python. Rejeté : le Python généré est illisible en revue, impossible à
  déboguer avec des breakpoints, et le diff à chaque régénération est du
  bruit. Le JSON est de la donnée et se lit comme de la donnée.
- **Garder les tables en Python et les exposer à Go via un socket.**
  Rejeté d'office : une dépendance réseau entre le générateur et
  l'exécution pour quelque chose qui est une table statique.

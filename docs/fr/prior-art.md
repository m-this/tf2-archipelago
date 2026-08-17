# Travaux antérieurs

Ce qui existe déjà, et ce que nous pouvons en prendre. Vérifié le
2026-08-13.

## ALPHAMARIOX/TF2-MvM-Archipelago

<https://github.com/ArchipelagoMW/Archipelago/compare/main...ALPHAMARIOX:TF2-MvM-Archipelago:main>

Un fork d'Archipelago, 3 commits en avance sur `main` et 143 en retard,
touché pour la dernière fois en mars 2026. Trois fichiers, tous sous
`worlds/tf2/` :

| Fichier | Lignes | État |
| --- | --- | --- |
| `Items.py` | 556 | Tables de données, en grande partie remplies. `mvm_data_to_ap_id()` et `ap_id_to_mvm_data()` sont des stubs vides (`""` comme corps). Aucun assemblage de `item_table`, aucune `ItemClassification` assignée. |
| `Options.py` | 214 | Complet et utilisable. 16 classes d'options plus `TeamFortress2Options(PerGameCommonOptions)` et deux `OptionGroup`. |
| `Locations.py` | 4 | Une classe `TFLocation(Location)` vide. Rien d'autre. |

Il n'y a ni `__init__.py`, ni sous-classe `World`, ni régions, ni règles,
ni client. Ça ne génère pas. À traiter comme un dépôt de données, pas du
code qui fonctionne.

### Ce qui vaut la peine d'être pris

**Les tables dans `Items.py`.** C'est la vraie valeur, et c'est
exactement le genre de donnée qui appartient à `gamedata/` (voir ADR
0001). Quatorze dicts :

`credits_table`, `upgrades_table`, `class_table`, `weapon_table` (210
lignes, le gros morceau), `weapon_slot_table`, `canteens_table`,
`robots_table` (169 lignes), `wave_table`, `mission_table`, `map_table`,
`trap_table`.

L'énum `Group(IntFlag)` dans le même fichier est un schéma de
catégorisation propre (`base`, `credits`, `upgrades`, `tf_class`,
`mvm_class`, `shop`, `mvm_map`, `mission`, `wave`, `weapon`,
`weapon_slot`, `canteen`, `robots`, `traps`). À porter vers un bitmask Go
sur les mêmes noms pour que les deux restent comparables.

Porter les tables vers Go, ne pas vendre le Python tel quel. Une fois
traduit, `gamedata/` les possède et le côté Python lit le JSON exporté.

**Les options.** `Options.py` est proche du plan de Roseburst dans le fil
de discussion et peut être adapté presque tel quel. À noter, le nommage
ne correspond pas au fil : `ShuffleMaps` / `ShuffleMissions` /
`LockClasses` / `LockWeapons` / `LockWeaponSlots` / `ShuffleUpgrades` /
`ShuffleRobots` / `ShuffleCanteens` / `AddTraps` / `AddTrapTypes`, plus
`RandomizeMissionCount` et `LockClassesCounter` comme des `Range`. Les
`Mission Order`, `Goal`, `Tour Size`, `Allied Mercs`, `Merc Loadouts`,
`Giants and Bosses` de Roseburst et tout le bloc `Check Options` n'ont pas
encore d'équivalent là-bas.

### Défauts connus à ne pas hériter

- Le nom du jeu se contredit lui-même : `Items.py` déclare
  `"Team Fortress 2 Mann Vs. Machine"`, `Locations.py` déclare
  `"Team Fortress 2"`. La chaîne de nom du jeu est la clé primaire du
  multiworld pour un slot. Elle doit donc porter une seule valeur, à un
  seul endroit. Nous la choisissons une fois, dans `gamedata/`, et
  l'exportons.
- Rien n'impose la stabilité des ids. Une seed figée interdit tout
  renumérotage des ids Archipelago. Le fork visait à résoudre ce problème
  avec les deux fonctions de conversion en stub. Voir l'ADR 0001 pour la
  façon dont nous le gérons à la place.

## Le manuel MvM de Snolid Ice

Le fil le cite deux fois (« i made a mvm manual », « my mvm manual is
kinda rough »), mais personne n'a posté de lien. Les apworlds Manual
suivent du JSON. Ils ne peuvent pas exprimer de vraies règles
d'accessibilité, donc nous ne suivons pas cette voie. Ça vaut quand même
le coup de le trouver, pour le nommage des items et des locations, si
l'auteur l'a encore.

## Archipelago lui-même

- Protocole : `docs/network protocol.md` dans `ArchipelagoMW/Archipelago`.
  L'ensemble de messages dont nous avons besoin est petit : `Connect`,
  `Connected`, `LocationChecks`, `ReceivedItems`, `StatusUpdate`,
  `Bounced` (pour DeathLink), `Say`.
- Ajouter un jeu : `docs/adding games.md` et `docs/world api.md`.
- Le client doit gérer à la fois `ws://` et `wss://`, et doit se
  reconnecter tout seul. Cette logique de reconnexion est la partie la
  plus pénible d'un client écrit à la main. C'est la raison principale
  pour laquelle le bridge est un processus Go de longue durée, plutôt
  qu'un module boulonné au plugin SourceMod (ADR 0002).

## Travaux antérieurs côté Source

Aucun client Archipelago n'existe pour un jeu du moteur Source. Il n'y a
donc aucun plugin à copier. Les références pertinentes sont l'API
SourceMod elle-même (événements de jeu, `SDKHooks`, natives `TF2_`), et
les noms d'entités et d'événements spécifiques à MvM. Les wikis
communautaires documentent ces noms, pas Valve.

Valve a publié le source du client et du serveur TF2 dans la publication
du SDK de février 2025. Nous ne l'utilisons pas. Une intégration
uniquement côté serveur garde les clients vanilla capables de rejoindre,
et ça compte plus que tout ce que le SDK offre.

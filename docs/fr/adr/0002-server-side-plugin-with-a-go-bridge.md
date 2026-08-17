# ADR 0002 — Plugin SourceMod côté serveur qui parle à un bridge Go, aucun mod client

- **Statut** : Accepté
- **Date** : 2026-08-13
- **Décideurs** : le propriétaire du projet
- **Lié à** : `docs/spec.md`, ADR 0001

## Contexte

Quelque chose doit se tenir entre un serveur TF2 en marche et le
multiworld Archipelago. Cette chose doit détecter les événements en jeu et
les transformer en `LocationChecks`, et doit recevoir des `ReceivedItems`
et les transformer en déblocages en jeu.

Trois contraintes forment la réponse :

1. **Les amis doivent pouvoir rejoindre avec un client TF2 standard.**
   C'est un jeu que des gens jouent ensemble sur un coup de tête. Tout ce
   qui demande à chaque participant d'installer un mod ne se jouera pas.
2. **Le protocole Archipelago est une session websocket avec de vraies
   exigences de liveness.** Le client doit gérer `ws://` et `wss://`,
   doit se reconnecter après une coupure, et ne doit ni perdre ni
   appliquer deux fois des items à travers une reconnexion.
3. **Seul SourceMod peut voir le jeu.** La fin d'une vague, la mort d'un
   tank, l'upgrade station, les restrictions d'arme : tout cela est du
   territoire SourcePawn, à l'intérieur du processus `srcds`.

SourcePawn est un mauvais endroit pour la contrainte 2. L'extension
websocket de SourceMod n'est plus maintenue, `ripext` (REST in Pawn) fait
bien du HTTP et du JSON mais son support websocket est de saveur
Socket.IO plutôt que brut, et SourcePawn n'a aucun vrai modèle de
concurrence, aucun stockage durable au-delà d'un handle SQLite, et un
appel bloquant dans une frame de jeu fige le serveur pour tout le monde
dessus.

## Décision

Le séparer en deux, le long de la ligne où les contraintes changent.

**Le plugin SourceMod voit le jeu et rien d'autre.** Il détecte les
objectifs et applique les déblocages et les pièges. Il parle en
vocabulaire MvM (`wave_cleared`, `mission_cleared`, `tank_destroyed`,
`grant_weapon_slot`, `apply_trap`) et ne connaît rien d'Archipelago : pas
d'id d'item, pas d'id de location, pas de slot, pas de seed. Il atteint le
bridge en HTTP et JSON simples via `ripext`, sans bloquer, sur
`127.0.0.1`.

**Le bridge Go possède la session Archipelago.** Il tient le websocket, se
reconnecte, rejoue, déduplique, et persiste. C'est le seul composant qui
connaît le protocole AP, et via `gamedata/` (ADR 0001) c'est le seul
composant qui connaît la correspondance des ids.

**Aucun mod côté client.** Tout ce que le joueur voit est livré par des
canaux qu'un client vanilla affiche déjà : le chat, le texte HUD, les
annotations, l'interface existante de l'upgrade station.

Décisions de support :

- **Le plugin ne tient jamais d'état faisant autorité.** Après un reload
  de plugin, un changement de carte ou un crash de `srcds`, il demande au
  bridge l'ensemble complet des déblocages actuels et l'applique. Il y a
  exactement une source de vérité pour « qu'est-ce que ce slot a reçu »,
  et c'est l'état sur disque du bridge, pas la mémoire du serveur de jeu.
- **Les checks sont mises en file durablement avant d'être acquittées.**
  Le plugin rapporte une check, le bridge l'écrit sur disque, puis renvoie
  200. C'est seulement ensuite qu'il essaie de l'envoyer en amont. Une
  check qui atteint le bridge n'est jamais perdue, même si le serveur AP
  est en panne depuis une heure. Perdre une fin de vague qui a pris dix
  minutes à gagner n'est pas un mode d'échec acceptable.
- **Les checks sont idempotentes par construction.** Une check est
  identifiée par son id de location, pas par un événement. Rapporter la
  même fin de vague deux fois ne fait rien. Cela compte parce que le
  plugin réessaiera après un timeout et ne peut pas savoir si la première
  tentative est arrivée.
- **Le bridge expose du long-poll, pas du push.** Le plugin demande
  « quoi de neuf depuis la séquence N » et le bridge garde la requête
  ouverte. `srcds` n'est pas un serveur que nous voulons voir écouter des
  connexions entrantes au-delà du port de jeu.
- **Le bridge se lie sur loopback uniquement.** Selon la règle maison sur
  les ports. Le port de jeu est la seule exception dans cette stack, et
  c'est le port du jeu, pas le nôtre.

## Conséquences

**Positives**

- Reconnexion, rejeu, déduplication, backoff et état durable sont écrits
  une fois, en Go, avec des tests et un détecteur de race. Rien de tout
  cela n'est exprimable en SourcePawn à une qualité en laquelle qui que ce
  soit aurait confiance.
- Le plugin reste petit et reste sur le jeu. Il peut être rechargé pendant
  une session sans perdre la progression.
- Les clients vanilla rejoignent. C'est la différence entre un projet
  qu'on joue et un projet qu'on démontre une fois.
- Le bridge est testable sans serveur de jeu : pointez-le vers un serveur
  Archipelago et pilotez son API HTTP avec `curl`.

**Négatives**

- Trois processus à faire tourner au lieu d'un, et un fichier compose qui
  doit les garder dans le bon ordre.
- Un saut HTTP local à chaque événement de jeu. Sans importance au rythme
  d'événements de MvM, qui est une poignée d'événements par vague, mais
  cela veut dire que le plugin doit gérer le bridge en panne, ce qui est
  un chemin de code de plus.
- `ripext` devient une dépendance dure du plugin, donc c'est une chose de
  plus à installer dans le conteneur `srcds` et une chose de plus qui peut
  casser à une mise à jour de SourceMod.
- Tout ce que le joueur doit voir doit passer par le chat, le texte HUD
  ou les annotations. Aucune interface sur mesure, jamais, à moins de
  revenir sur la décision d'aucun mod client.

## Alternatives considérées

- **Websocket directement depuis SourcePawn.** Rejeté : l'extension n'est
  plus maintenue, et même si elle fonctionnait, les exigences de
  reconnexion et de durabilité tombent dans un langage sans bon moyen de
  les satisfaire.
- **Un mod côté client, comme la plupart des apworlds.** Rejeté sur la
  contrainte 1. Cela échoue aussi pour une seconde raison : l'état de MvM
  fait autorité côté serveur, donc un mod client devrait déduire la fin
  d'une vague à partir de ce qu'il peut voir, ce qui est strictement une
  moins bonne information que ce que le serveur a déjà.
- **Aucun plugin du tout, jouer sur les serveurs officiels Boot Camp ou
  Mann Up** (Roseburst a soulevé cela comme possiblement « plus
  pratique »). Rejeté : les serveurs Valve ne font tourner aucun plugin,
  donc il n'y a aucun moyen de verrouiller les armes ou les
  améliorations, et tout le côté item de la conception disparaît. Ce qui
  reste est un traqueur de succès.
- **Mettre la logique du client AP dans une extension SourceMod** (C++,
  chargée dans `srcds`). Rejeté : cela gagne la concurrence mais hérite du
  rayon d'explosion. Un crash dans le client AP emporterait le serveur de
  jeu avec lui, et un segfault en plein milieu d'une vague est un pire
  résultat qu'un websocket coupé.
- **Faire lire au bridge le log console de `srcds`** plutôt que de faire
  tourner un plugin. Rejeté pour le sens des items : lire le log donne des
  événements mais il n'y a aucun moyen d'écrire en retour, donc rien ne
  peut être débloqué.

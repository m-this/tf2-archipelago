# Inviter vos amis

## D'où viennent les joueurs

Un serveur est fait pour être rejoint depuis ailleurs, donc il prend les
connexions sur son port de jeu tant que vous ne dites pas le contraire.
Choisissez parmi trois avec `SRCDS_REACH` dans `.env` :

```sh
SRCDS_REACH=port     # directement sur le port du jeu, redirigé sur votre routeur. Le défaut.
SRCDS_REACH=steam    # par le relais Steam, sans port à ouvrir
SRCDS_REACH=lan      # cette machine et le réseau local
```

`lan` est toute la réponse pour des gens qui jouent dans la même maison, et ne
demande rien d'autre. Les deux autres atteignent Internet, et les deux
demandent un token.

Un serveur sans token reste sur le réseau local quoi que dise le reach, et
l'écrit dans le journal. Il n'a pas le choix : sans token il n'obtient jamais
de session Steam, et un serveur sans session refuse tout joueur qui essaie de
rejoindre, y compris ceux de la même maison.

> **`steam` n'est pas fini.** Aucun essai n'a mené le relais jusqu'à un client
> Team Fortress 2 qui a rejoint. Le launcher le propose quand même, dans
> l'onglet **Networking**. `port` est celui sur lequel les joueurs jouent.

## Le token de connexion

`steam` et `port` connectent tous les deux le serveur à Steam. Sans token il
n'obtient jamais de session Steam, et tout client qui essaie de rejoindre est
refusé sans message utile. Récupérez-en un sur
[steamcommunity.com/dev/managegameservers](https://steamcommunity.com/dev/managegameservers)
pour l'app id 440, une fois, et mettez-le dans les réglages ou dans `.env` :

```sh
SRCDS_TOKEN=VOTRETOKENICI
```

Le token n'est pas un mot de passe que quelqu'un tape. Il identifie le serveur
auprès de Steam. `SRCDS_TOKEN=0` veut dire aucun, ce qui n'est la bonne réponse
que pour `lan`, et c'est ce que le serveur a tant que vous ne lui en donnez pas.

## Par Steam, sans port à ouvrir

Avec `SRCDS_REACH=steam` le serveur demande à Valve une adresse sur le Steam
Datagram Relay et l'affiche dès qu'il en a une :

```
FakeIP allocation succeeded: 169.254.13.42:20232, 20233
```

Le launcher la sort du log et la met dans une case au-dessus du log, avec un
bouton **Copy**. Vos amis tapent la première adresse :

```
connect 169.254.13.42:20232
```

Ça marche de n'importe où. Rien n'est redirigé, rien n'est ouvert dans le
pare-feu, et votre propre adresse n'est jamais montrée aux gens qui
rejoignent : le trafic passe par les relais de Valve.

Deux choses à savoir. L'adresse est nouvelle à chaque démarrage du serveur,
donc envoyez la ligne de cette session-ci plutôt qu'une notée la semaine
dernière. Et personne ne peut mettre le serveur en favori, faute d'adresse
stable.

## Par un port redirigé

Avec `SRCDS_REACH=port` les joueurs se connectent directement à vous :

```sh
SRCDS_PORT=27015
```

Redirigez ce port vers la machine sur votre routeur, en UDP et en TCP, et
ouvrez-le dans le pare-feu de la machine. L'UDP porte le jeu, donc un port UDP
fermé veut dire que personne ne peut rejoindre. Vos amis tapent ensuite votre
adresse publique :

```
connect adresse.publique.de.votre.machine:27015
```

Rien d'autre n'a besoin d'être atteignable. Le serveur randomizer et le bridge
restent sur la boucle locale.

## La console développeur

Elle est désactivée par défaut dans Team Fortress 2. Options, puis Avancé, puis
« Activer la console développeur ». La touche qui l'ouvre est `` ` `` sur un
clavier américain.

## Le mot de passe du serveur

```sh
SRCDS_PW=
```

Une valeur vide laisse quiconque connaît l'adresse rejoindre. Réglez-la pour
garder le serveur aux personnes que vous avez prévenues :

```sh
SRCDS_PW=entre-amis
```

Vos amis tapent alors ceci avant de se connecter :

```
password entre-amis
connect ...
```

`SRCDS_PW` n'est pas `SRCDS_RCONPW`. `SRCDS_PW` laisse un joueur entrer.
`SRCDS_RCONPW` laisse un admin lancer des commandes. Ne donnez jamais le
second.

## Rester hors de la liste publique

Un serveur avec `SRCDS_TOKEN=0` ne se connecte jamais et n'apparaît jamais dans
le navigateur public de serveurs. Avec un vrai token il le peut, ce que vous
voulez éviter sur une partie randomisée en cours : réglez aussi `SRCDS_PW`, et
les inconnus qui trouvent le serveur ne pourront quand même pas entrer.

## Ce qu'il faut leur dire

Envoyez-leur ces trois choses :

1. La ligne de connexion correspondant au mode choisi. Par Steam, celle du log
   de cette session-ci.
2. Le mot de passe du serveur, si vous en avez mis un.
3. [Archipelago pour les joueurs MvM](../archipelago-for-mvm-players.md), ou la
   version courte : les classes et les armes commencent verrouillées, réussir
   des vagues les débloque, tout le monde partage.

Ils n'installent rien. Un client Team Fortress 2 standard est tout ce qu'il
faut.

## Quand un joueur ne peut pas se connecter

Le serveur répond à une requête mais refuse la connexion. C'est presque
toujours l'authentification Steam contre un serveur qui n'a pas de session
Steam : `SRCDS_REACH` vaut `steam` ou `port` et `SRCDS_TOKEN` est resté à `0`.
La console le dit :

```
Could not establish connection to Steam servers.  (Result = 8)
version : ... insecure (secure mode enabled, disconnected from Steam3)
```

Mettez un vrai token, ou revenez à `SRCDS_REACH=lan`, où l'authentification est
sautée entièrement et où aucun token n'est nécessaire.

Vérifiez ensuite le reste dans cet ordre :

1. **En `lan`**, le joueur voit `LAN servers are restricted to local clients
   (class C)`. C'est le serveur qui dit que l'adresse d'où il vient n'est pas
   dans la même classe C, les trois mêmes premiers nombres, que la sienne : un
   réseau invité sur le même routeur en est une autre, un VPN aussi, et le
   réseau d'un conteneur aussi. Donnez l'adresse que `ip -4 addr` montre pour
   le réseau où tout le monde est, ou quittez `lan` : `port` avec un token n'a
   pas cette règle.
2. **En `steam`**, l'adresse est celle de cette session-ci. Elle change à chaque
   démarrage, et une ancienne ne mène nulle part.
3. **En `port`**, le port est redirigé en UDP autant qu'en TCP. Une règle TCP
   seule répond à la requête et jette le jeu.
4. La machine hôte est éveillée. C'est un portable, et il se met en veille.

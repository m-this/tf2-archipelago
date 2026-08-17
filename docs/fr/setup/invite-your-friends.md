# Inviter vos amis

## Ouvrir le port

La stack publie `SRCDS_PORT`, `27015` par défaut, en UDP et en TCP.

```sh
SRCDS_PORT=27015
```

Redirigez ce port vers la machine sur votre routeur. Ouvrez-le dans le
pare-feu de la machine. L'UDP porte le jeu, donc un port UDP fermé veut
dire que personne ne peut rejoindre.

Rien d'autre n'a besoin d'être atteignable. Le serveur randomizer et le
bridge restent à l'intérieur de la stack.

## La commande de connexion

Vos amis ouvrent la console développeur dans Team Fortress 2 et tapent :

```
connect adresse.de.votre.serveur:27015
```

Remplacez l'adresse par l'adresse publique de votre machine. Remplacez le
port si vous avez changé `SRCDS_PORT`.

La console est désactivée par défaut dans Team Fortress 2. Options, puis
Avancé, puis « Activer la console développeur ». La touche qui l'ouvre est
`` ` `` sur un clavier américain.

## Le mot de passe du serveur

```sh
SRCDS_PW=
```

Une valeur vide laisse quiconque connaît l'adresse rejoindre. Réglez-la
pour garder le serveur aux personnes que vous avez prévenues :

```sh
SRCDS_PW=entre-amis
```

Vos amis tapent alors ceci avant de se connecter :

```
password entre-amis
connect adresse.de.votre.serveur:27015
```

`SRCDS_PW` n'est pas `SRCDS_RCONPW`. `SRCDS_PW` laisse un joueur entrer.
`SRCDS_RCONPW` laisse un admin lancer des commandes. Ne donnez jamais le
second.

## Cacher le serveur de la liste publique

```sh
SRCDS_TOKEN=0
```

Un Game Server Login Token est ce qui met un serveur dédié dans le
navigateur public de serveurs. La valeur `0` veut dire que le serveur n'en
a pas. Il tourne, vos amis peuvent s'y connecter avec l'adresse, et il
n'apparaît pas dans la liste publique.

C'est ce que vous voulez pour une soirée entre amis. Laissez `0`.

Pour lister le serveur publiquement, récupérez un token sur
[steamcommunity.com/dev/managegameservers](https://steamcommunity.com/dev/managegameservers)
et mettez-le ici. Réglez aussi `SRCDS_PW`, sinon des inconnus rejoindront
une partie randomisée en cours.

## Ce qu'il faut leur dire

Envoyez-leur ces trois choses :

1. La commande de connexion avec votre adresse.
2. Le mot de passe du serveur, si vous en avez mis un.
3. [Archipelago pour les joueurs MvM](../archipelago-for-mvm-players.md), ou
   la version courte : les classes et les armes commencent verrouillées,
   réussir des vagues les débloque, tout le monde partage.

Ils n'installent rien. Un client Team Fortress 2 standard est tout ce qu'il
faut.

## Quand un joueur ne peut pas se connecter

Le serveur répond à une requête mais refuse la connexion. C'est presque
toujours l'authentification Steam contre un serveur qui n'a pas de session
Steam.

Un serveur sans Game Server Login Token (`SRCDS_TOKEN=0`, le défaut) ne se
connecte jamais à Steam. Sa console le dit :

```
Could not establish connection to Steam servers.  (Result = 8)
version : ... insecure (secure mode enabled, disconnected from Steam3)
```

Avec `SRCDS_LAN=1`, qui est le défaut, cela n'a pas d'importance : le mode
LAN saute complètement l'authentification. Si quelqu'un l'a changé,
remettez-le :

```
rcon sv_lan 1
```

Vérifiez le reste dans cet ordre :

1. Le serveur répond sur l'adresse que vous donnez aux gens. Depuis
   l'hôte : `docker run --rm --network container:tf2-archipelago-srcds-1
   curlimages/curl -s ifconfig.me` n'est pas la réponse que vous voulez.
   Utilisez l'adresse LAN de la machine, celle que `ip -4 addr` montre sur
   le réseau où sont vos amis.
2. Tout le monde est sur le même réseau. Le mode LAN refuse les adresses
   hors des plages privées, et un réseau invité sur le même routeur en est
   une autre.
3. La machine hôte est éveillée. C'est un portable, et il se met en veille.

Passer en ligne plus tard demande un vrai `SRCDS_TOKEN` depuis
[steamcommunity.com/dev/managegameservers](https://steamcommunity.com/dev/managegameservers)
et `SRCDS_LAN=0`. L'un sans l'autre refuse tous les joueurs.

# Installation

Lancez tout depuis la racine du dépôt.

## 1. Écrire le fichier de configuration

```sh
cp deploy/.env.example .env
```

`.env` est le seul fichier que vous modifiez. Git l'ignore.

## 2. Régler le mot de passe de la console

Ouvrez `.env` et réglez `SRCDS_RCONPW` avec un mot de passe de votre choix :

```sh
SRCDS_RCONPW=choisissez-quelque-chose-de-long
```

C'est la seule valeur sans défaut. La stack refuse de démarrer sans elle
et affiche `set SRCDS_RCONPW in .env`.

Ce mot de passe déverrouille la console distante du serveur de jeu. Il vous
le faut pour les commandes admin. Personne d'autre n'en a besoin.

C'est aussi le moment de régler la forme de la partie. La stack génère la
session randomisée au premier démarrage puis la garde, donc un changement
fait plus tard ne fait rien tant que vous ne démarrez pas une nouvelle
partie. Voir [La forme de la partie](shape-of-the-run.md).

## 3. Démarrer la stack

```sh
make up
make logs
```

`make up` construit trois images et démarre trois conteneurs. `make logs`
suit la sortie des trois. Arrêtez de suivre avec Ctrl-C. Cela n'arrête pas
la stack.

## À quoi ressemble le premier démarrage

Le build compile le plugin de jeu et le bridge. Cela prend quelques
minutes.

Puis, dans cet ordre :

1. Le serveur randomizer voit un dossier de sortie vide et génère la
   session. Le log dit `no seed in /ap/output, generating one`, puis
   `hosting /ap/output/AP_<number>.zip on port 38281`. Cela prend moins
   d'une minute.
2. Le serveur de jeu démarre et télécharge environ 14 Go de fichiers de
   jeu. C'est la partie longue. Sa durée dépend de votre connexion.
3. Le plugin est installé dans les fichiers du jeu dès qu'ils arrivent. Le
   log dit `[AP] installed the plugin and ripext into
   /home/steam/tf-dedicated/tf`.
4. Le bridge se connecte au serveur randomizer. Le log du randomizer dit
   `tf2 (Team #1) playing Team Fortress 2 Mann vs Machine has joined`.

Chaque démarrage suivant prend quelques secondes. Rien n'est retéléchargé
et aucune nouvelle session n'est générée.

## Les trois services

| Service | Ce qu'il fait | Ports |
| --- | --- | --- |
| `archipelago` | Génère la session randomisée au premier démarrage, puis l'héberge | aucun, interne à la stack |
| `srcds` | Fait tourner le serveur dédié Team Fortress 2 et le plugin | `27015/udp` et `27015/tcp`, les seuls ports publics |
| `bridge` | Tient la session avec le serveur randomizer et répond au plugin | aucun, loopback à l'intérieur de l'espace réseau du serveur de jeu |

Le bridge partage l'espace réseau du serveur de jeu. C'est ainsi que le
plugin l'atteint sur `127.0.0.1` et que rien à l'extérieur de la machine ne
le peut. Redémarrer le serveur de jeu redémarre le bridge avec lui. Cela
coûte des secondes, pas de progression : le bridge écrit chaque check sur
le disque.

## Les commandes

| Commande | Ce qu'elle fait |
| --- | --- |
| `make up` | Démarrer la stack |
| `make logs` | Suivre la sortie des trois services |
| `make ps` | Lister les conteneurs et leur état |
| `make down` | Arrêter la stack. Garde les fichiers du jeu, la session et la partie. |
| `make restart` | `make down` puis `make up` |
| `make build` | Reconstruire les trois images |
| `make clean` | Arrêter la stack et supprimer chaque volume, y compris les 14 Go de fichiers de jeu |
| `make check` | Lancer tout ce que l'intégration continue lance |
| `make integration` | Démarrer un vrai serveur randomizer et un vrai bridge, et les piloter comme le plugin le fait |

`make clean` supprime les fichiers du jeu. Utilisez `make down` sauf si
c'est vraiment ce que vous voulez.

## Où la stack garde les choses

| Volume | Contient | Le supprimer sert à |
| --- | --- | --- |
| `tf2-archipelago_tf2game` | Les 14 Go de fichiers de jeu, SourceMod et le plugin | Tout retélécharger |
| `tf2-archipelago_apoutput` | La session générée | [Démarrer une nouvelle partie](../operate/start-a-new-run.md) |
| `tf2-archipelago_bridgestate` | Les checks et les déblocages de la partie en cours | Rien d'utile. Le bridge reconstruit les checks à partir du serveur randomizer. |

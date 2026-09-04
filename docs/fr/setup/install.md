# Installation

Lancez tout depuis la racine du dépôt. [Sans le dépôt](#sans-le-dépôt), à la
fin de cette page, fait la même chose avec deux fichiers téléchargés.

## 1. Écrire le fichier de configuration

```sh
cp deploy/.env.example .env
```

`.env` est le seul fichier que vous modifiez. Git l'ignore.

La pile joue une room Archipelago. Il lui faut l'adresse de la room et un slot,
c'est-à-dire `AP_HOST`, `AP_PORT` et `AP_SLOT_NAME` dans `.env`, et rien d'autre
du côté d'Archipelago. Le `make seed` plus bas sert à ceux qui n'ont pas encore
de room ; sautez-le si vous générez déjà vos propres multiworlds.

## 2. Régler le mot de passe de la console

Ouvrez `.env` et réglez `SRCDS_RCONPW` avec un mot de passe de votre choix :

```sh
SRCDS_RCONPW=choisissez-quelque-chose-de-long
```

Ce mot de passe déverrouille la console distante du serveur de jeu. Il vous
le faut pour les commandes admin. Personne d'autre n'en a besoin.

## 3. Donner une session à jouer à la stack

La stack joue une session qui existe déjà. [Créer la
session](create-the-session.md) en fabrique une, l'héberge sur
`archipelago.gg`, et vous donne l'adresse d'une room. Écrivez cette adresse
dans `.env` :

```sh
AP_HOST=archipelago.gg
AP_PORT=12345
AP_TLS=true
```

`SRCDS_RCONPW`, `AP_HOST` et `AP_PORT` sont les trois valeurs sans défaut. La
stack refuse de démarrer s'il en manque une, et elle affiche laquelle.

## Jouer sans Archipelago

`TF2AP_TEST_MODE=1` dans `.env` dispense la stack de room. Le bridge sert un
multiworld d'un seul joueur sur la loopback. Il invente une seed depuis le
nombre de missions et le but que vous avez réglés, puis donne un déblocage à
chaque vague réussie.

Il joue aussi les autres joueurs : ils trouvent des objets, vous en envoient et
meurent. Chaque ligne arrive dans le journal du bridge et dans le chat du jeu.

Le bridge ignore `AP_HOST` et `AP_PORT` tant que c'est actif, et rien ne quitte
la machine. Servez-vous en pour essayer la stack, et pour tester quand quelque
chose cloche.

## 4. Démarrer la stack

```sh
make up
make logs
```

`make up` construit deux images et démarre deux conteneurs. `make logs` suit
la sortie des deux. Arrêtez de suivre avec Ctrl-C. Cela n'arrête pas la stack.

## À quoi ressemble le premier démarrage

Le build compile le plugin de jeu et le bridge. Cela prend quelques
minutes.

Puis, dans cet ordre :

1. Le serveur de jeu démarre et télécharge environ 14 Go de fichiers de
   jeu. C'est la partie longue. Sa durée dépend de votre connexion.
2. La stack installe le plugin dans les fichiers du jeu dès qu'ils arrivent.
   Le log dit `[AP] installed the plugin and ripext into
   /home/steam/tf-dedicated/tf`.
3. Le bridge rejoint la room. Son log dit `connected to archipelago slot=tf2
   missions=8`, et la page de la room dit
   `tf2 (Team #1) playing Team Fortress 2 Mann vs Machine has joined`.

Chaque démarrage suivant prend quelques secondes. La stack ne retélécharge
rien.

## Les services

| Service | Ce qu'il fait | Ports |
| --- | --- | --- |
| `srcds` | Fait tourner le serveur dédié Team Fortress 2 et le plugin | `27015/udp` et `27015/tcp`, les seuls ports publics |
| `bridge` | Tient la session avec la room et répond au plugin | aucun, loopback à l'intérieur de l'espace réseau du serveur de jeu |

Un troisième service, `archipelago`, héberge la session sur cette machine
plutôt que sur `archipelago.gg`. Il ne démarre qu'avec
`COMPOSE_PROFILES=selfhost` dans `.env`. Voir [Créer la
session](create-the-session.md).

Le bridge partage l'espace réseau du serveur de jeu. Le plugin l'atteint donc
sur `127.0.0.1`, et rien à l'extérieur de la machine ne le peut. Redémarrer le serveur de jeu redémarre le bridge avec lui. Cela
coûte des secondes, pas de progression : le bridge écrit chaque check sur
le disque.

## Les commandes

| Commande | Ce qu'elle fait |
| --- | --- |
| `make seed` | Fabriquer une session dans `seed/`, à envoyer sur `archipelago.gg` |
| `make up` | Démarrer la stack |
| `make logs` | Suivre la sortie des services |
| `make ps` | Lister les conteneurs et leur état |
| `make down` | Arrêter la stack. Garde les fichiers du jeu et la partie. |
| `make restart` | `make down` puis `make up` |
| `make build` | Reconstruire les images |
| `make clean` | Arrêter la stack et supprimer chaque volume, y compris les 14 Go de fichiers de jeu |
| `make check` | Lancer tout ce que l'intégration continue lance |
| `make integration` | Démarrer un vrai serveur randomizer et un vrai bridge, et les piloter comme le plugin le fait |
| `make dist` | Construire dans `dist/` tout ce qu'une release attache : le `.apworld`, le plugin, les données exportées, et le fichier compose ci-dessous |

`make clean` supprime les fichiers du jeu. Utilisez `make down` sauf si
c'est vraiment ce que vous voulez.

## Où la stack garde les choses

| Volume | Contient | Le supprimer sert à |
| --- | --- | --- |
| `tf2-archipelago_tf2game` | Les 14 Go de fichiers de jeu, SourceMod et le plugin | Tout retélécharger |
| `tf2-archipelago_bridgestate` | Les checks et les déblocages de la partie en cours | Rien d'utile. Le bridge reconstruit les checks à partir de la room. |
| `tf2-archipelago_apoutput` | La session, avec `COMPOSE_PROFILES=selfhost` seulement | [Démarrer une nouvelle partie](../operate/start-a-new-run.md) |

Les sessions elles-mêmes sont des fichiers dans `seed/`, dans le dépôt. Git
ignore ce dossier, et rien ne le supprime pour vous.

## Sans le dépôt

Chaque release attache un `compose.yaml` qui nomme des images publiées au lieu
de les construire, et le `env.example` qui va avec. Une machine avec Docker n'a
besoin de rien d'autre : ni clone, ni Go, ni compilateur.

```sh
mkdir mann-vs-archipelago && cd mann-vs-archipelago
base=https://github.com/m-this/tf2-archipelago/releases/latest/download
curl -fsSLO "$base/compose.yaml"
curl -fsSL -o .env "$base/env.example"
```

Réglez `SRCDS_RCONPW` dans `.env`, puis fabriquez une session et démarrez :

```sh
docker compose --profile seed run --rm seed   # écrit ./seed
docker compose up -d
docker compose logs -f
```

Les étapes 3 et 4 de [Créer la session](create-the-session.md) s'appliquent
telles quelles. Envoyez le fichier de `seed/`, créez une room, et écrivez le
port de la room dans `AP_PORT`.

Les images viennent de `ghcr.io/m-this/tf2-archipelago`. Le `compose.yaml` que
vous téléchargez les fixe à la release dont il vient, donc la stack garde la
version que vous avez installée. `TF2AP_VERSION` dans `.env` en choisit une
autre :

```sh
TF2AP_VERSION=v1.0.0
```

```sh
docker compose pull && docker compose up -d
```

Les commandes du tableau ci-dessus sont des cibles `make`, et elles ont besoin
du dépôt. `docker compose` fait chacune d'elles seul : `up -d`, `logs -f`, `ps`,
`down`, et `down -v`.

La [page des releases](https://github.com/m-this/tf2-archipelago/releases)
attache aussi `tf2_mvm.apworld` et `tf2_archipelago.smx`, pour une installation
Archipelago ou un serveur de jeu que ce fichier compose ne fait pas tourner.

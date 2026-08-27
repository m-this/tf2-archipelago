# Installer sur Linux

Un seul fichier. Pas de Docker, pas de clone, pas de compilateur. C'est le même
programme que le lanceur Windows. Il dessine la même interface dans le terminal
plutôt que dans une fenêtre. Vous avez le journal, les missions de la partie,
le Bot Switcher, la ligne rcon et les sept mêmes onglets de réglages. Rien à installer pour ça,
et ça marche à travers SSH.

Téléchargez `tf2ap-linux-amd64` depuis la
[dernière version](https://github.com/m-this/tf2-archipelago/releases/latest),
rendez-le exécutable, et lancez-le.

```sh
curl -fsSLO https://github.com/m-this/tf2-archipelago/releases/latest/download/tf2ap-linux-amd64
chmod +x tf2ap-linux-amd64
./tf2ap-linux-amd64
```

## Ce qui se passe

Il demande l'adresse de votre room Archipelago. Puis il installe tout :
SteamCMD, le serveur dédié TF2, Metamod:Source, SourceMod, le plugin, et les
bots qui remplissent votre équipe. Le serveur de jeu pèse environ 14 Go, et le
premier démarrage prend du temps à cause de ça. Chaque démarrage suivant prend
quelques secondes.

Puis l'interface s'ouvre dessus.

| Touche | Ce qu'elle fait |
| --- | --- |
| `s` | Démarre le serveur, ou l'arrête |
| `r` | Le redémarre |
| `j` | Lance Team Fortress 2 et rejoint |
| `c` | Copie la ligne de connexion, à envoyer à un ami |
| `,` | Les réglages, dans les huit onglets de la fenêtre |
| `tab` | Entre la partie, le Bot Switcher et le journal |
| `i` | La ligne rcon. `esc` rend les touches |
| `p` | Sur l'onglet de la partie, charge la mission sous le curseur |
| `a` | Sur le Bot Switcher, donne l'équipe de bots au serveur en cours |
| `q` | Quitte, ce qui arrête le serveur |

`tf2ap-linux-amd64 -console` écrit le journal et rien d'autre, ce que veut un
service ou une session `screen` : une interface qui dessine sur tout l'écran
n'écrit rien d'utile dans un fichier. Celle-là s'arrête avec Ctrl+C.

## Ce qu'il vous faut

| Élément | Ce qu'il faut |
| --- | --- |
| Linux | 64 bits, glibc |
| Disque | Environ 20 Go libres |
| Mémoire | 4 Go pour six joueurs |
| Processeur | Deux cœurs |
| Réseau | Rien, pour des amis sur le même réseau. Un seul port à ouvrir si vous choisissez cette voie. |

Le serveur dédié TF2 est un programme 32 bits. Sur une distribution 64 bits, il
lui faut les bibliothèques C, C++ et curl en 32 bits. Sur Debian et Ubuntu :

```sh
sudo dpkg --add-architecture i386
sudo apt update
sudo apt install lib32gcc-s1 lib32stdc++6 libcurl3t64-gnutls:i386
```

Fedora appelle la bibliothèque C `glibc.i686`, Arch l'appelle `lib32-glibc`.
SteamCMD et le serveur nomment clairement une bibliothèque qu'ils ne trouvent
pas.

Pas de Docker, pas de client Steam, pas de compte Steam pour le serveur
lui-même.

## La session Archipelago

Le lanceur fait tourner le serveur TF2. La session multiworld est séparée. Mann
vs Machine ne fait pas partie des jeux livrés avec Archipelago, donc le
générateur de seed reste dans l'app officielle.

1. Installez l'app officielle
   [Archipelago](https://github.com/ArchipelagoMW/Archipelago/releases). Le
   lanceur la cherche dans `~/Archipelago`, `/opt/Archipelago` et `/ap`.
2. Lancez `./tf2ap-linux-amd64 -yaml tf2.yaml` pour écrire le fichier joueur,
   et déposez-le dans le dossier `Players/` de l'app.
3. Générez là-bas, puis envoyez l'archive sur
   [archipelago.gg/uploads](https://archipelago.gg/uploads) pour ouvrir une
   room.
4. Donnez l'adresse de la room au lanceur.

Voir [Créer la session](create-the-session.md) pour le détail complet.

## Les réglages

`./tf2ap-linux-amd64 -configure` parcourt tous les réglages dans le terminal.
Chacun lit aussi une variable d'environnement, sous le nom qu'emploie
`deploy/.env.example` :

```sh
AP_ROOM=archipelago.gg:12345 SRCDS_BOT_TEAM_SIZE=4 ./tf2ap-linux-amd64
```

`./tf2ap-linux-amd64 -env` affiche chaque nom lu et marque ceux que votre
environnement donne déjà.

## Les commandes

| Commande | Ce qu'elle fait |
| --- | --- |
| `tf2ap-linux-amd64` | Installe ce qu'il faut au serveur, puis le lance |
| `tf2ap-linux-amd64 -room <hôte:port>` | Règle l'adresse, puis lance |
| `tf2ap-linux-amd64 -configure` | Édite tous les réglages, puis quitte |
| `tf2ap-linux-amd64 -install` | Installe ou répare le serveur, puis quitte |
| `tf2ap-linux-amd64 -status` | Affiche les réglages et l'état de l'installation |
| `tf2ap-linux-amd64 -yaml <chemin>` | Écrit le fichier joueur Archipelago, puis quitte |
| `tf2ap-linux-amd64 -env` | Liste les variables d'environnement, puis quitte |
| `tf2ap-linux-amd64 -version` | Affiche la version et les versions des outils |

## Où le lanceur range ses fichiers

| Chemin | Contenu |
| --- | --- |
| `~/tf2-archipelago/` | Les fichiers du jeu, SourceMod et SteamCMD |
| `~/tf2-archipelago/tf2.yaml` | Le fichier joueur |
| `~/tf2-archipelago/bridge-state/` | Les checks et les déblocages |
| `~/.config/tf2ap/config.json` | Vos réglages |

`TF2AP_INSTALL_ROOT` déplace les trois premiers, pour un second disque.

## L'autre façon

[Installer avec Docker](install.md) fait tourner le même logiciel en deux
conteneurs. Prenez celle-là pour garder le serveur hors de votre compte, ou
pour une machine qui a déjà une stack Docker. Cette page est le chemin le plus
court.

Voir [Inviter vos amis](invite-your-friends.md) pour ouvrir le serveur, et
[Dépannage](../operate/troubleshooting.md) quand quelque chose semble aller de
travers.

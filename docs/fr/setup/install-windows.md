# Installer sur Windows

C'est la façon la plus simple de lancer un serveur Mann vs Archipelago. Un seul
fichier, sans Docker, sans clone, sans compilateur.

Téléchargez `tf2ap.exe` depuis la
[dernière version](https://github.com/m-this/tf2-archipelago/releases/latest),
puis lancez-le.

## Ce que tf2ap.exe fait

Double-cliquez dessus : une fenêtre s'ouvre. Elle porte tout ce qu'une soirée
demande.

- **Start**, **Stop** et **Restart** : le serveur monte et descend sans
  terminal.
- Un journal, où le serveur de jeu, le bridge et l'installation écrivent. C'est
  ce que vous lisez quand quelque chose cloche.
- **Settings** : l'adresse de la room, la carte, les bots et la forme de la
  partie.
- Une case **rcon** en bas. Elle envoie une commande au serveur et affiche la
  réponse dans le journal. `sm_ap_status` est celle à connaître.

Le premier Start installe SteamCMD, le serveur dédié TF2, Metamod:Source,
SourceMod, le plugin et les bots défenseurs. Le serveur fait environ 14 Go.
C'est la partie longue, et elle n'arrive qu'une fois. Le journal la suit.

Fermer la fenêtre arrête le serveur. Chaque démarrage suivant prend quelques
secondes, et le lanceur garde vos réponses dans `%APPDATA%\tf2ap\config.json`.

## Ce qu'il vous faut

| Élément | Ce qu'il vous faut |
| --- | --- |
| Windows | 10 ou 11, 64 bits. |
| Disque | Environ 20 Go libres. Le serveur fait 14 Go. |
| Mémoire | 4 Go pour six joueurs. |
| Processeur | Deux cœurs. |
| Réseau | Un port que vos amis atteignent, UDP et TCP. Le défaut est 27015. |

Vous n'avez besoin ni de Docker, ni du client Steam, ni d'un compte Steam pour
le serveur.

## La session Archipelago

Le lanceur gère le serveur TF2. La session multiworld reste séparée. Mann vs
Machine ne fait pas partie des jeux livrés avec Archipelago. Le générateur est
en Python, donc il reste dans l'application officielle.

1. Installez l'application officielle
   [Archipelago](https://github.com/ArchipelagoMW/Archipelago/releases). Le
   lanceur la trouve là où l'installeur la met.
2. Dans le lanceur, ouvrez **Settings**, réglez les options de partie, puis
   appuyez sur **Generate seed**. Le lanceur place le fichier du monde dans
   l'application et écrit le fichier joueur. Il lance ensuite le générateur et
   ouvre le dossier qui contient l'archive.
3. Envoyez cette archive sur
   [archipelago.gg/uploads](https://archipelago.gg/uploads) pour ouvrir une
   room.

La page de la room donne une adresse comme `archipelago.gg:12345`. Collez-la
dans l'onglet **Archipelago room**.

Le fichier joueur est aussi dans `%USERPROFILE%\tf2-archipelago\tf2.yaml`, et
**Open tf2.yaml** l'affiche. C'est pour qui veut générer à la main dans
l'application.

Voir [Créer la session](create-the-session.md) pour le détail de chaque étape.

## Le premier démarrage

Double-cliquez sur `tf2ap.exe`, ou lancez-le depuis un terminal :

```
tf2ap.exe
```

Le premier démarrage se déroule ainsi :

1. La fenêtre Settings s'ouvre d'elle-même : vous n'avez pas encore de room.
   Collez la ligne de votre page de room dans **Room address**, par exemple
   `archipelago.gg:12345`. Le bouton Save reste gris tant que l'adresse ne se
   lit pas comme `hôte:port`. Tout le reste a déjà une valeur qui marche.
2. Appuyez sur **Save**. L'installation démarre toute seule.
3. SteamCMD, puis les 14 Go du serveur de jeu, puis Metamod:Source, SourceMod,
   le plugin et les bots. Le journal montre chaque étape. La longue est le
   serveur de jeu, et la durée dépend de votre connexion.
4. Le serveur démarre. Le journal affiche `connected to archipelago slot=tf2`.

Les démarrages suivants vont directement à cette dernière étape.

Vos amis se connectent depuis la console développeur :

```
connect votre.adresse.serveur:27015
```

Voir [Inviter vos amis](invite-your-friends.md) pour le reste.

## Les bots de votre équipe

Le serveur remplit l'équipe RED avec des bots qui jouent. Team Fortress 2
équilibre chaque vague pour six défenseurs, donc deux joueurs gagnent une
partie. Les bots choisissent leur classe, se battent, achètent leurs
améliorations et se déclarent prêts.

`tf2ap.exe -configure` les coupe, ou remplit RED avec moins de six joueurs pour
une partie plus dure. Voir [Les bots de votre équipe](../play/defender-bots.md).

## Essayer sans Archipelago

La fenêtre Settings porte une case **Test mode**. Cochez-la : le lanceur sert
alors un multiworld d'un seul joueur sur votre machine. Il invente une seed depuis vos
options de partie et donne un déblocage à chaque vague réussie.

Il joue aussi les autres joueurs : ils trouvent des objets, vous en envoient et
meurent. Chaque ligne arrive dans le journal et dans le chat du jeu.

Rien ne quitte la machine, et vous n'avez besoin ni de room ni de seed. Servez-
vous en pour essayer le serveur, et pour tester quand quelque chose cloche.

## Quand vous avez besoin d'aide

La fenêtre Settings porte deux boutons pour ça.

**Debug logs** écrit `debug-logs-<date>.zip` à côté des fichiers du jeu, puis
ouvre le dossier. Il contient le journal du lanceur, les journaux de SourceMod,
la console du serveur, le fichier joueur et vos réglages, sans aucun mot de
passe. Envoyez ce fichier à qui vous aide.

**Repair** jette SteamCMD et les mods, puis les récupère. Il arrête d'abord le
serveur. Il garde les fichiers du jeu et la partie : pas de 14 Go à retélécharger
et les checks restent.

L'onglet **Player options** porte aussi **Open tf2.yaml**. Ce bouton écrit le
fichier joueur depuis les valeurs de la fenêtre, puis l'ouvre.

## Les commandes

La fenêtre couvre une soirée. Ces commandes servent à un script, ou à un
réglage que la fenêtre n'affiche pas. Lancez-les depuis un terminal : l'exe s'y
rattache.

| Commande | Ce qu'elle fait |
| --- | --- |
| `tf2ap.exe` | Ouvre la fenêtre |
| `tf2ap.exe -room <hôte:port>` | Règle l'adresse, puis ouvre la fenêtre |
| `tf2ap.exe -console` | Tourne dans le terminal, sans fenêtre |
| `tf2ap.exe -configure` | Édite tous les réglages dans le terminal, puis quitte |
| `tf2ap.exe -install` | Installe ou répare le serveur, puis quitte |
| `tf2ap.exe -status` | Affiche les réglages et l'état de l'installation |
| `tf2ap.exe -yaml <chemin>` | Écrit le fichier joueur Archipelago, puis quitte |
| `tf2ap.exe -env` | Liste les variables d'environnement, puis quitte |
| `tf2ap.exe -version` | Affiche la version et les versions des outils |

## Les réglages par l'environnement

Chaque réglage lit aussi une variable d'environnement, sous le nom qu'emploie
`deploy/.env.example`. Une variable l'emporte sur le fichier pour cette
exécution, et le lanceur ne la garde jamais. Démarrez un serveur sans aucune
question, depuis un raccourci ou un fichier `.bat` :

```bat
set AP_ROOM=archipelago.gg:12345
set SRCDS_BOT_TEAM_SIZE=4
tf2ap.exe
```

`AP_ROOM` porte l'adresse entière. `AP_HOST`, `AP_PORT` et `AP_TLS` règlent les
trois parties séparément, pour un fichier `.env` partagé avec le stack Docker.

`tf2ap.exe -env` affiche chaque nom lu et marque ceux que votre environnement
donne déjà.

## Où le lanceur range ses fichiers

| Chemin | Contenu |
| --- | --- |
| `%USERPROFILE%\tf2-archipelago\` | Les fichiers du jeu, SourceMod et SteamCMD. |
| `%USERPROFILE%\tf2-archipelago\tf2.yaml` | Le fichier joueur pour Archipelago. |
| `%USERPROFILE%\tf2-archipelago\bridge-state\` | Les checks et les débloquages. |
| `%APPDATA%\tf2ap\config.json` | Vos réglages. |

`TF2AP_INSTALL_ROOT` déplace les trois premiers, pour un second disque.

## En cas de problème

Le lanceur écrit son journal dans le terminal. Remontez pour lire ce que le
serveur de jeu et le bridge ont dit.

Le port est la cause habituelle. Redirigez 27015, ou le port que vous avez
choisi, vers votre machine sur votre box. Ouvrez-le ensuite dans le pare-feu
Windows. UDP porte le jeu, donc un port UDP fermé empêche toute connexion.

Voir [Dépannage](../operate/troubleshooting.md) pour le reste.

## L'autre façon

Si vous avez une machine Linux avec Docker, voir
[Installer avec Docker](install.md). Les deux font tourner le même logiciel. Le
lanceur porte le même bridge et le même plugin, emballés pour Windows sans
Docker.

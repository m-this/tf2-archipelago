# Installer sur Windows

C'est la façon la plus simple de lancer un serveur Mann vs Archipelago. Un seul
fichier, sans Docker, sans clone, sans compilateur.

Téléchargez `tf2ap.exe` depuis la
[dernière version](https://github.com/m-this/tf2-archipelago/releases/latest),
puis lancez-le.

## Ce que tf2ap.exe fait

1. Il installe SteamCMD, le serveur dédié TF2, Metamod:Source, SourceMod, le
   plugin et les bots défenseurs. Le serveur fait environ 14 Go. C'est la
   partie longue, et elle n'arrive qu'une fois.
2. Il demande les réglages d'une soirée. L'adresse de votre room Archipelago,
   un mot de passe RCON, et la forme de la partie.
3. Il démarre le serveur de jeu et le bridge ensemble. Appuyez sur Ctrl-C pour
   arrêter les deux.

Chaque démarrage suivant prend quelques secondes. Les fichiers du jeu restent
sur le disque, et le lanceur garde vos réponses dans
`%APPDATA%\tf2ap\config.json`.

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
Machine ne fait pas partie des jeux livrés avec Archipelago, et le générateur
est en Python.

1. Installez l'application officielle
   [Archipelago](https://github.com/ArchipelagoMW/Archipelago/releases). Elle
   porte son propre Python.
2. Téléchargez `tf2_mvm.apworld` depuis la même version que `tf2ap.exe`.
3. Placez `tf2_mvm.apworld` dans le dossier `custom_worlds/` de cette
   installation.
4. Copiez `%USERPROFILE%\tf2-archipelago\tf2.yaml` dans le dossier `Players/`
   de la même installation. Le lanceur écrit ce fichier au premier démarrage,
   depuis vos réglages. `tf2ap.exe -yaml <chemin>` l'écrit où vous voulez.
5. Ouvrez l'application Archipelago et générez. Envoyez l'archive sur
   [archipelago.gg/uploads](https://archipelago.gg/uploads) pour ouvrir une
   room.

La page de la room donne une adresse comme `archipelago.gg:12345`. Donnez les
deux moitiés au lanceur quand il les demande.

Voir [Créer la session](create-the-session.md) pour le détail de chaque étape.

## Le premier démarrage

Double-cliquez sur `tf2ap.exe`, ou lancez-le depuis un terminal :

```
tf2ap.exe
```

Une fenêtre de terminal s'ouvre. Le premier démarrage se déroule ainsi :

1. Le lanceur installe SteamCMD. Quelques secondes.
2. Le lanceur installe le serveur dédié TF2. C'est le téléchargement de 14 Go.
   La durée dépend de votre connexion.
3. Le lanceur installe Metamod:Source, SourceMod, le plugin et les bots.
4. Le lanceur demande les réglages. Répondez aux questions. Vos réponses
   précédentes sont entre crochets, et Entrée garde la réponse affichée.
5. Le lanceur démarre le serveur. Le journal affiche
   `connected to archipelago slot=tf2`.

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

## Les commandes

| Commande | Ce qu'elle fait |
| --- | --- |
| `tf2ap.exe` | Installe si besoin, demande les valeurs manquantes, démarre |
| `tf2ap.exe -configure` | Édite tous les réglages, puis quitte |
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
set AP_HOST=archipelago.gg
set AP_PORT=12345
set SRCDS_RCONPW=un-mot-de-passe-long
set SRCDS_BOT_TEAM_SIZE=4
tf2ap.exe
```

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

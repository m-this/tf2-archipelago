# Installer sur Windows

C'est la façon la plus simple de lancer un serveur Mann vs Archipelago. Un
seul fichier, sans Docker, sans clone, sans compilateur.

Téléchargez `tf2ap.exe` depuis la
[dernière version](https://github.com/m-this/tf2-archipelago/releases/latest) et
lancez-le. C'est un fichier unique qui fait tout.

## Ce que tf2ap.exe fait

1. Il installe SteamCMD, le serveur dédié TF2, SourceMod et le plugin la
   première fois. Le serveur fait environ 14 Go ; c'est la partie longue, et
   ça n'arrive qu'une fois.
2. Il demande la configuration : l'adresse de votre room Archipelago, un mot de
   passe RCON, et la forme de la partie.
3. Il démarre le serveur de jeu et le bridge dans un seul processus. Appuyez
   sur Ctrl-C pour tout arrêter.

Chaque démarrage suivant prend quelques secondes. Les 14 Go de fichiers restent
sur le disque, et la configuration est sauvegardée dans
`%APPDATA%\tf2ap\config.json`.

## Ce qu'il vous faut

| Élément | Ce qu'il vous faut |
| --- | --- |
| Windows | 10 ou 11, 64 bits. |
| Disque | Environ 20 Go libres. Le serveur fait 14 Go. |
| Mémoire | 4 Go pour six joueurs. |
| Processeur | Deux cœurs. |
| Réseau | Un port que vos amis peuvent atteindre, UDP et TCP. Le défaut est 27015. |

Vous n'avez pas besoin de Docker, ni du client Steam, ni d'un compte Steam pour
le serveur.

## La session Archipelago

Le lanceur gère le serveur TF2. La session multiworld est séparée, parce que
Mann vs Machine n'est pas un des jeux livrés avec Archipelago et le générateur
est en Python.

1. Installez l'application officielle
   [Archipelago](https://github.com/ArchipelagoMW/Archipelago/releases) sur
   Windows. Elle inclut son propre Python.
2. Téléchargez `tf2_mvm.apworld` depuis la même version que `tf2ap.exe`.
3. Placez `tf2_mvm.apworld` dans le dossier `custom_worlds/` de votre install
   Archipelago.
4. Ouvrez l'application Archipelago, générez une seed, et envoyez-la sur
   [archipelago.gg/uploads](https://archipelago.gg/uploads) pour créer une room.

La page de la room donne une adresse comme `archipelago.gg:12345`. Écrivez les
deux moitiés dans le lanceur quand il les demande.

Voir [Créer la session](create-the-session.md) pour le détail de chaque étape.

## Le premier démarrage

Double-cliquez sur `tf2ap.exe`, ou lancez-le depuis un terminal :

```
tf2ap.exe
```

1. Le lanceur installe SteamCMD. Quelques secondes.
2. Le lanceur installe le serveur dédié TF2. C'est le téléchargement de 14 Go.
3. Le lanceur installe SourceMod et le plugin. Quelques secondes.
4. Le lanceur demande la configuration. Répondez aux invites.
5. Le lanceur démarre le serveur.

Vos amis se connectent avec la console développeur :

```
connect votre.adresse.serveur:27015
```

Voir [Inviter vos amis](invite-your-friends.md) pour le reste.

## Les commandes

| Commande | Ce qu'elle fait |
| --- | --- |
| `tf2ap.exe` | Installe si besoin, demande les valeurs manquantes, démarre |
| `tf2ap.exe -configure` | Édite tous les réglages, puis quitte |
| `tf2ap.exe -install` | Installe ou répare le serveur, puis quitte |
| `tf2ap.exe -status` | Affiche la configuration et l'état de l'install |
| `tf2ap.exe -version` | Affiche la version et les versions des outils |

## L'autre façon

Si vous avez déjà une machine Linux avec Docker, ou si vous préférez le stack
conteneur, voir [Installer avec Docker](install.md). Les deux font tourner le
même logiciel : le lanceur est le même bridge et le même plugin, juste emballé
pour Windows sans Docker.

# Mann vs Archipelago

Ce projet transforme un serveur Team Fortress 2 Mann vs Machine en
randomizer. Les classes, les emplacements d'arme et les missions
commencent verrouillés. L'équipe les débloque quand elle réussit des vagues.
Tout le monde sur le serveur partage les mêmes déblocages.

Vous l'hébergez de deux façons. Sur Windows, un seul exe installe et lance
l'ensemble. Partout où Docker tourne, une stack de deux conteneurs
fait la même chose. Un serveur dédié Team Fortress 2, et un petit programme
qui le relie à la session. La session tourne sur `archipelago.gg`, et la stack
l'héberge pour vous si vous préférez. Vos amis rejoignent avec un client
Team Fortress 2 standard et n'installent rien.

Le serveur remplit aussi l'équipe RED avec des bots qui jouent, donc deux
joueurs gagnent une partie que Valve a calibrée pour six.

Ce livre s'adresse à l'hébergeur. Il suppose que vous connaissez Mann vs
Machine et que vous n'avez jamais utilisé un randomizer de ce genre. Il
définit chaque mot avant de l'utiliser.

## À lire dans cet ordre

1. [Archipelago pour les joueurs MvM](archipelago-for-mvm-players.md) vous
   donne le vocabulaire. Lisez-le en premier.
2. [Ce que le randomizer change](what-the-randomizer-changes.md) dit ce qui
   diffère d'un serveur MvM normal.
3. [Prérequis](setup/requirements.md) dit ce qu'il faut à la machine.
4. [La forme de la partie](setup/shape-of-the-run.md) fixe la longueur et
   la difficulté d'une soirée.
5. [Créer la session](setup/create-the-session.md) fabrique la partie et la
   met sur `archipelago.gg`.
6. [Installer sur Windows](setup/install-windows.md) ou
   [Installer avec Docker](setup/install.md) fait tourner le serveur.
7. [Inviter vos amis](setup/invite-your-friends.md) ouvre le serveur.
   [Les bots de votre équipe](play/defender-bots.md) dit qui remplit les
   places vides.

## La version courte

Sur Windows, téléchargez `tf2ap.exe` depuis la
[dernière version](https://github.com/m-this/tf2-archipelago/releases/latest),
puis lancez-le. Il demande l'adresse de la room et installe le reste.

Avec Docker :

```sh
cp deploy/.env.example .env   # puis réglez SRCDS_RCONPW
make seed                     # envoyez le fichier sur archipelago.gg, ouvrez
                              # une room, puis réglez AP_HOST et AP_PORT
make up
make logs
```

Le premier démarrage télécharge environ 14 Go de fichiers de jeu.

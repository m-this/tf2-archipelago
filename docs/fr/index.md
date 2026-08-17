# Mann vs Archipelago

Ce projet transforme un serveur Team Fortress 2 Mann vs Machine en
randomizer. Les classes, les emplacements d'arme et les missions
commencent verrouillés. L'équipe les débloque quand elle réussit des vagues.
Tout le monde sur le serveur partage les mêmes déblocages.

Vous l'hébergez avec Docker. La stack fait tourner trois conteneurs : un
serveur dédié Team Fortress 2, un serveur randomizer, et un petit
programme qui relie les deux. Vos amis rejoignent avec un client Team
Fortress 2 standard et n'installent rien.

Ce livre s'adresse à l'hébergeur. Il suppose que vous connaissez Mann vs
Machine et que vous n'avez jamais utilisé un randomizer de ce genre. Il
définit chaque mot avant de l'utiliser.

## À lire dans cet ordre

1. [Archipelago pour les joueurs MvM](archipelago-for-mvm-players.md) vous
   donne le vocabulaire. Lisez-le en premier.
2. [Ce que le randomizer change](what-the-randomizer-changes.md) dit ce qui
   diffère d'un serveur MvM normal.
3. [Prérequis](setup/requirements.md) et [Installation](setup/install.md)
   font tourner la stack.
4. [La forme de la partie](setup/shape-of-the-run.md) fixe la longueur et
   la difficulté d'une soirée.
5. [Inviter vos amis](setup/invite-your-friends.md) ouvre le serveur.
6. [Tests](operate/testing.md) donne la marche à suivre pour confirmer
   chaque comportement sur un serveur réel. À lire avant la première
   session.

## La version courte

```sh
cp deploy/.env.example .env   # puis réglez SRCDS_RCONPW
make up
make logs
```

Le premier démarrage télécharge environ 14 Go de fichiers de jeu.

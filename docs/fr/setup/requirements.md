# Prérequis

## La machine

| Chose | Ce qu'il vous faut |
| --- | --- |
| Disque | Environ 20 Go libres. Le serveur de jeu télécharge environ 14 Go au premier démarrage. |
| Mémoire | 4 Go pour six joueurs. |
| Processeur | Deux cœurs. |
| Docker | Docker avec le plugin compose. |
| Réseau | Un port que vos amis peuvent atteindre, UDP et TCP. |

Les fichiers du jeu restent dans un volume Docker nommé
`tf2-archipelago_tf2game`. Gardez ce volume. Le supprimer retélécharge
14 Go.

## Le réseau

La stack publie un port, `27015` par défaut, en UDP et en TCP. Réglez-le
avec `SRCDS_PORT` dans `.env`.

- L'UDP porte le jeu. Sans lui, personne ne peut rejoindre.
- Le TCP porte la console distante du serveur de jeu. Il vous le faut pour
  lancer les commandes admin dans [Dépannage](../operate/troubleshooting.md).

Rien d'autre n'est publié. Le serveur randomizer et le bridge entre les
deux restent à l'intérieur de la stack. Voir [Installation](install.md)
pour les trois services.

Si la machine est derrière un routeur, redirigez ce port vers elle. Si la
machine a un pare-feu, ouvrez ce port.

## Ce dont vous n'avez pas besoin

- Aucun compte Steam pour le serveur. `SRCDS_TOKEN=0` fait tourner le
  serveur sans en avoir un et le garde hors de la liste publique des
  serveurs. Voir [Inviter vos amis](invite-your-friends.md).
- Aucune installation de Team Fortress 2 sur l'hôte. Le conteneur télécharge
  la sienne.
- Aucun compte sur un site Archipelago. Le serveur randomizer tourne sur
  votre machine.

## Remarque sur la sécurité

Le serveur de jeu est un gros processus C++ qui lit le trafic réseau de
quiconque connaît l'adresse. Faites-le tourner sur une machine où c'est
acceptable. Si la même machine fait tourner quelque chose qui compte pour
vous, décidez-le exprès plutôt que par défaut.

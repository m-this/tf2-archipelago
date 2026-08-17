# Créer la session

La session randomisée tourne sur `archipelago.gg`. Votre machine fabrique la
session et l'écrit dans un fichier. Le site prend ce fichier, l'héberge, et
vous donne une adresse.

Votre machine la fabrique parce que Mann vs Machine ne fait pas partie des jeux
livrés avec Archipelago. Le site génère ces jeux-là seulement. Il héberge toute
session que vous lui envoyez.

## 1. Régler la forme de la partie

[La forme de la partie](shape-of-the-run.md) tient la longueur, la difficulté
et le but d'une soirée. Réglez ces valeurs dans `.env` maintenant. La session
les garde, et un changement plus tard demande une nouvelle session.

## 2. Fabriquer la session

```sh
make seed
```

Le premier lancement construit l'image du randomizer et prend quelques minutes.
Un lancement suivant prend moins d'une minute.

La commande écrit un fichier et en affiche le nom :

```
generated /ap/output/AP_53174869021847362095.zip
upload it at https://archipelago.gg/uploads, then create a room
```

Sur votre machine, ce fichier est dans `seed/`. Git ignore ce dossier. Gardez
les fichiers qu'il contient. Le même fichier redonne la même session, donc une
room perdue revient à partir de lui.

## 3. Envoyer le fichier à archipelago.gg

1. Ouvrez [archipelago.gg/uploads](https://archipelago.gg/uploads).
2. Envoyez le fichier qui est dans `seed/`.
3. Choisissez **Create New Room**.

Le site ne demande aucun compte.

La page de la room tient l'adresse de la room et un lien vers le suivi de la
partie, que le site appelle « Tracker ». Envoyez cette page à vos joueurs. Ils
suivent la partie depuis elle et n'installent rien.

## 4. Pointer la stack sur la room

La page de la room donne une adresse de la forme `archipelago.gg:12345`.
Écrivez ses deux moitiés dans `.env` :

```sh
AP_HOST=archipelago.gg
AP_PORT=12345
AP_TLS=true
```

Chaque nouvelle room prend un nouveau port. Réglez `AP_PORT` de nouveau après
chaque nouvelle room.

N'importe qui atteint une room dont il connaît l'adresse. Mettez un mot de
passe sur la room, puis le même mot de passe dans `AP_PASSWORD`.

Démarrez ensuite la stack. Voir [Installation](install.md).

## La version d'Archipelago

`deploy/env/versions.env` fixe la version qui fabrique le fichier.
`archipelago.gg` fait tourner sa propre version, et il refuse un fichier que
cette version ne sait pas lire.

Si l'envoi échoue, lisez la version dans le pied de page du site. Réglez
ensuite `ARCHIPELAGO_VERSION` sur cette version, puis relancez `make seed`.

## Héberger la session vous-même

La stack héberge aussi la session elle-même. Cela demande quatre lignes dans
`.env` :

```sh
COMPOSE_PROFILES=selfhost
AP_HOST=archipelago
AP_PORT=38281
AP_TLS=false
```

`make up` démarre alors un troisième conteneur, fabrique la session au premier
démarrage, et l'héberge à côté du serveur de jeu. Vous n'envoyez rien, et les
étapes 2 à 4 ci-dessus ne s'appliquent pas.

Ce que cela coûte :

- Vos joueurs n'ont pas de page de room, donc pas de suivi de la partie.
- Un conteneur de plus tourne sur votre machine.
- Un joueur dans un autre jeu demande un deuxième port public.
  `deploy/compose.yml` dit où.

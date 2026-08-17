# Démarrer une nouvelle partie

Une session ne change jamais une fois qu'elle existe. Une partie différente
demande une nouvelle session, une nouvelle room, et une modification dans
`.env`.

## Les quatre étapes

1. Modifiez `.env` si la forme de la partie change. Voir
   [La forme de la partie](../setup/shape-of-the-run.md).
2. Lancez `make seed`. La commande écrit un fichier de plus dans `seed/`.
3. Envoyez ce fichier et créez une room. Voir
   [Créer la session](../setup/create-the-session.md).
4. Réglez `AP_PORT` sur le port de la nouvelle room, puis lancez
   `make restart`.

Les fichiers du jeu ne sont pas touchés, donc le redémarrage prend quelques
secondes.

Gardez les anciens fichiers dans `seed/`. Chacun est une partie entière, et la
room d'une partie revient à partir de son fichier.

## Ce que le bridge fait de l'ancienne partie

Le bridge remarque que la session n'est pas celle sur laquelle il
travaillait. Il fait alors ceci :

1. Met son fichier d'état de côté sous `bridge.<seed>.json`, dans le même
   dossier.
2. Repart sans checks et sans déblocages.
3. Dit au plugin que la partie a redémarré, pour que le plugin abandonne sa
   propre copie et redemande le nouvel ensemble de déblocages.

L'ancien fichier n'est jamais écrasé. Si vous pointez le bridge par accident
vers la mauvaise room, la partie précédente est toujours sur le disque, dans le
volume `tf2-archipelago_bridgestate`.

Rien d'autre ne fait perdre une partie. Redémarrer un service, redémarrer
la machine et s'arrêter pendant une semaine la gardent tous.

## Ce que vous n'avez pas besoin de supprimer

| Volume | Laissez-le tranquille |
| --- | --- |
| `tf2-archipelago_tf2game` | 14 Go de fichiers de jeu. Le supprimer les retélécharge. |
| `tf2-archipelago_bridgestate` | Le bridge y archive l'ancienne partie tout seul. |

## Si vous hébergez la session vous-même

`COMPOSE_PROFILES=selfhost` met la session dans le volume
`tf2-archipelago_apoutput`, et vous n'avez rien à envoyer. Une nouvelle partie
tient en trois commandes :

```sh
make down
docker volume rm tf2-archipelago_apoutput
make up
```

Modifiez `.env` entre la première et la troisième commande. Le conteneur
`archipelago` trouve un dossier de sortie vide, fabrique une session à partir
du `.env` actuel, et l'héberge. Cela prend moins d'une minute.

## Tout recommencer complètement

```sh
make clean
```

Cela arrête la stack et supprime chaque volume, y compris les fichiers du
jeu. Cela laisse `seed/` tranquille. Utilisez-le quand vous en avez fini avec
le projet, pas entre deux parties.

# Démarrer une nouvelle partie

La stack génère la session randomisée au premier démarrage puis la garde.
Modifier `.env` ensuite ne change rien. Pour jouer une partie différente,
supprimez la session générée et recommencez.

## Les trois commandes

```sh
make down
docker volume rm tf2-archipelago_apoutput
make up
```

Modifiez `.env` entre la première et la troisième commande. Voir
[La forme de la partie](../setup/shape-of-the-run.md).

Le conteneur randomizer trouve un dossier de sortie vide, génère une
nouvelle session à partir du `.env` actuel, et l'héberge. Cela prend moins
d'une minute. Les fichiers du jeu ne sont pas touchés, donc le démarrage
est rapide.

## Ce que le bridge fait de l'ancienne partie

Le bridge remarque que la session n'est pas celle sur laquelle il
travaillait. Il fait alors ceci :

1. Met son fichier d'état de côté sous `bridge.<seed>.json`, dans le même
   dossier.
2. Repart sans checks et sans déblocages.
3. Dit au plugin que la partie a redémarré, pour que le plugin abandonne sa
   propre copie et redemande le nouvel ensemble de déblocages.

L'ancien fichier n'est jamais écrasé. Si vous pointez le bridge par
accident vers le mauvais serveur randomizer, la partie précédente est
toujours sur le disque, dans le volume `tf2-archipelago_bridgestate`.

Rien d'autre ne fait perdre une partie. Redémarrer un service, redémarrer
la machine et s'arrêter pendant une semaine la gardent tous.

## Ce que vous n'avez pas besoin de supprimer

| Volume | Laissez-le tranquille |
| --- | --- |
| `tf2-archipelago_tf2game` | 14 Go de fichiers de jeu. Le supprimer les retélécharge. |
| `tf2-archipelago_bridgestate` | Le bridge y archive l'ancienne partie tout seul. |

Seul `tf2-archipelago_apoutput` tient la session.

## Tout recommencer complètement

```sh
make clean
```

Cela arrête la stack et supprime chaque volume, y compris les fichiers du
jeu. Utilisez-le quand vous en avez fini avec le projet, pas entre deux
parties.

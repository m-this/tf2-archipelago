# Dépannage

Trois choses peuvent clocher : le serveur de jeu ne voit pas la partie, le
plugin ne peut pas atteindre le bridge, ou le bridge ne peut pas atteindre
le serveur randomizer. Cette page trouve laquelle.

## Lire les logs

```sh
make logs
```

Cela suit tous les services que la stack fait tourner. Pour un seul service,
utilisez la commande compose complète. La stack a besoin de deux fichiers d'environnement, donc
la forme courte ne fonctionne pas :

```sh
docker compose --project-directory . \
  --env-file deploy/env/versions.env --env-file .env \
  -f deploy/compose.yml logs -f bridge
```

Remplacez `bridge` par `srcds`, ou par `archipelago` quand la stack héberge
elle-même la session.

```sh
make ps
```

Cela liste les conteneurs. Le bridge rapporte `healthy` quand sa propre
interface répond.

## Interroger le serveur de jeu

```
rcon_password votre-SRCDS_RCONPW
rcon sm_ap_status
```

La réponse tient en cinq lignes :

```
[AP] version 0.1.0, mvm yes, mission mvm_decoy, wave 3 of 8
[AP] events: begin_wave yes, wave_complete yes, mission_complete yes
[AP] unlocks held at sequence 6, 0 objective(s) waiting to be sent
[AP] classes: scout, medic
[AP] slots: primary
[AP] Last bridge error: ...
```

Lisez-les dans cet ordre :

- **`mvm no`** veut dire que le plugin ne pense pas être dans Mann vs
  Machine. Rien n'est rapporté sur une carte qui n'est pas une carte MvM.
- **`events: ... no`** est la ligne importante. Elle nomme lesquels des
  trois événements de jeu Mann vs Machine ce serveur envoie vraiment. Un
  `no` ici veut dire que votre version du jeu n'envoie pas cet événement ;
  signalez-le. `wave_complete no` fait surveiller le compteur de vagues au
  plugin à la place.
- **`unlocks NOT FETCHED`** veut dire que le plugin n'a jamais eu de
  réponse du bridge. Tant que ce n'est pas le cas, il n'impose rien : un
  serveur où personne ne peut tenir une arme est pire qu'une vague jouée
  avec trop d'équipement.
- **`N objective(s) waiting to be sent`** compte les checks que le plugin
  tient. Tout ce qui est au-dessus de zéro veut dire que le bridge ne
  répond pas. Il réessaie toutes les cinq secondes.
- **`Last bridge error`** est la dernière chose qui a mal tourné, dans les
  mots du plugin.

`rcon sm_ap_resync` redemande l'ensemble des déblocages au bridge. C'est
la première chose à essayer quand les déblocages dans le chat semblent
périmés.

## Interroger le bridge

Le bridge sert une page avec tout ce qu'il sait. Elle vit sur l'interface
loopback à l'intérieur de l'espace réseau du serveur de jeu, donc la
requête doit être faite depuis là :

```sh
docker run --rm --network container:tf2-archipelago-srcds-1 \
  curlimages/curl:latest -s 127.0.0.1:24680/healthz
```

Le nom du conteneur vient de `make ps`.

La réponse contient :

| Champ | Ce qu'il vous dit |
| --- | --- |
| `api_version` | La version du protocole. Le plugin le dit dans le chat en cas de désaccord. |
| `connected` | Si la session avec le serveur randomizer est active en ce moment |
| `slot` | Le nom de votre serveur dans la session |
| `missions` | Les missions que la partie a tirées |
| `seed` | L'identité de la session en cours |
| `checks` | Combien de checks la partie tient |
| `items` | Combien d'items la partie a reçus |
| `acked_seq` | Jusqu'où le plugin a confirmé avoir appliqué |
| `goal_sent` | Si la partie a été déclarée terminée |
| `last_check` et `last_check_at` | La dernière check et quand elle est arrivée |
| `wave_drift` | Les missions dont la longueur diffère de ce que dit le jeu |
| `last_error` | Le dernier échec du côté randomizer |

`last_check` répond à « est-ce que cette vague a compté ».

## Surveiller la partie dans le temps

Les mêmes chiffres sont servis comme métriques Prometheus, sur leur
propre port, pour qu'un tableau de bord puisse les tracer plutôt qu'une
personne qui relance la commande ci-dessus sans arrêt. Ce port **est**
publié sur l'hôte — `BRIDGE_METRICS_BIND` décide qui peut l'atteindre,
loopback par défaut :

```sh
curl -s 127.0.0.1:24681/metrics
```

| Métrique | Ce qu'elle vous dit |
| --- | --- |
| `tf2ap_session_connected` | 1 tant que la session avec le serveur randomizer est active |
| `tf2ap_session_missions` | Combien de missions la partie a tirées |
| `tf2ap_run_checks_total` / `tf2ap_run_items_total` | Checks envoyées, items reçus |
| `tf2ap_run_acked_seq` | Jusqu'où le plugin a confirmé avoir appliqué. Bloqué derrière le nombre d'items veut dire que le serveur de jeu n'applique pas les grants |
| `tf2ap_run_goal_sent` | 1 une fois la partie terminée |
| `tf2ap_run_last_check_timestamp_seconds` | Quand la dernière check est arrivée. Absent tant qu'aucune n'est arrivée |
| `tf2ap_mission_wave_drift` | Une série par mission où le jeu et les tables sont en désaccord, valant la différence. Aucune série est le cas sain |
| `tf2ap_run_info` | La seed et le slot auxquels appartiennent les chiffres |
| `tf2ap_game_up` | 1 quand le serveur de jeu a répondu à une requête A2S sur ce scrape |
| `tf2ap_game_players` / `_bots` / `_players_human` | Qui est sur le serveur. MvM compte ses vagues de robots comme des bots, donc les gens qui jouent sont les joueurs moins les bots |
| `tf2ap_game_players_max` | Ce que le serveur annonce — six, les emplacements RED. Pas les 32 avec lesquels il doit démarrer pour héberger MvM du tout |
| `tf2ap_game_map` | La mission en cours, comme étiquette |

Les comptes de joueurs viennent d'une requête A2S que le bridge envoie au
serveur de jeu, la même chose qu'un navigateur de serveurs demande. Un
serveur qui ne répond pas rapporte `tf2ap_game_up 0` et **aucun** compte,
pour qu'un srcds en train de redémarrer se lise comme absent plutôt que
comme un serveur vide.

Deux pièges, tous deux mesurés sur un serveur en marche plutôt que
devinés :

- srcds se lie sur `0.0.0.0:27015` et répond à une requête envoyée à
  n'importe laquelle de ses adresses d'interface, mais **rejette** celle
  envoyée à `127.0.0.1`. Adressez-le par son nom (`srcds:27015`), même
  depuis l'intérieur de son propre espace réseau — c'est le défaut de
  `BRIDGE_GAME_QUERY`.
- srcds arrête de répondre à une source qui l'interroge plus de quelques
  fois par seconde (`sv_max_queries_sec`, sur une fenêtre de 30 secondes),
  et continue de refuser jusqu'à ce que cette fenêtre se vide. La réponse
  est donc mise en cache dix secondes : interrogez `/metrics` en boucle et
  vous obtenez quand même une requête toutes les dix secondes. Sans cela,
  interroger cet endpoint plusieurs fois de suite fait dire au tableau de
  bord que le serveur est en panne.

`tf2ap_mission_wave_drift` est celle qui mérite une alerte : un mauvais
nombre de vagues sur la mission objectif est ce qui rend une seed
imperdable, et cela ne se répare pas en cours de partie.

## Quand le serveur randomizer est en panne

Rien n'est perdu. Le bridge écrit chaque check sur le disque avant de
répondre au serveur de jeu, et l'envoie en amont ensuite. Un serveur
randomizer en panne pendant une heure ne coûte rien : les checks arrivent
à son retour. Le bridge se reconnecte tout seul, en attendant de plus en
plus longtemps entre les tentatives, jusqu'à trente secondes.

Les items reçus arrêtent d'arriver pendant la panne. Les vagues réussies
continuent de compter.

Un bridge qui ne se connecte jamais est un autre problème. Vérifiez `AP_HOST`,
`AP_PORT` et `AP_TLS`. Une room sur `archipelago.gg` répond en `wss://` et
demande `AP_TLS=true` ; une session dans la stack répond en `ws://` et demande
`AP_TLS=false`. La mauvaise valeur fait échouer chaque tentative, et le bridge
écrit l'échec dans son log à chaque fois.

## Quand le bridge est en panne

Le plugin tient ses checks en mémoire et réessaie toutes les cinq
secondes. Le chat dit que le bridge est injoignable, une fois, pour que
personne ne décide que le randomizer est cassé.

Le bridge partage l'espace réseau du serveur de jeu, donc redémarrer le
serveur de jeu redémarre le bridge aussi. Il revient tout seul en
quelques secondes. Les checks sont sur le disque et l'ensemble des
déblocages est reconstruit à partir d'elles.

Si le fichier d'état du bridge est perdu, la partie n'est pas perdue non
plus. Le serveur randomizer tient la même liste de checks et la renvoie à
chaque connexion. Le bridge adopte ce qui lui manque. Perdre le fichier
coûte l'historique des items, pas les checks.

## Récupérer une check à la main

Il y a un seul trou dans tout cela : les secondes entre une vague réussie
et le moment où le bridge prend la check. La file du plugin est en
mémoire et tient au plus 64 objectifs. Si le serveur de jeu plante pendant
que le bridge est injoignable, ce qui est dans cette file est perdu.

Le plugin écrit chaque objectif dans le log SourceMod deux fois : une fois
quand il le met en file, et une fois quand le bridge l'a sur son disque.

```sh
docker compose --project-directory . \
  --env-file deploy/env/versions.env --env-file .env \
  -f deploy/compose.yml exec srcds \
  bash -c 'grep objective /home/steam/tf-dedicated/tf/addons/sourcemod/logs/L*.log'
```

```
objective wave_cleared mvm_decoy wave 3 (mission length 8) queued for the bridge
objective wave_cleared mvm_decoy wave 3 is on the bridge's disk
```

Une ligne `queued` sans ligne `on the bridge's disk` correspondante est
une check qui n'est jamais arrivée. Rejouez-la :

```
rcon sm_ap_report wave_cleared 3
```

Lancez-la sur la carte à laquelle la check appartient. Le plugin envoie
la mission sur laquelle le jeu se trouve, donc rejouer une check de Decoy
pendant que le serveur fait tourner Coal Town enregistre le mauvais
endroit.

## Ne jamais redémarrer le serveur de jeu seul

Le bridge vit à l'intérieur de l'espace réseau du serveur de jeu, ce qui
met son API sur un loopback que rien d'autre ne peut atteindre. Le coût
est que le serveur de jeu possède cet espace réseau.

Donc `docker compose up -d srcds` seul laisse le bridge attaché à un
espace réseau qui n'existe plus. Il continue de tourner, se rapporte
toujours en bonne santé, et ne peut plus rien atteindre : le plugin reçoit
une connexion refusée et le serveur randomizer voit le slot se
déconnecter. `docker restart` ne le répare pas et échoue avec
`joining network namespace: No such container`.

Recréez-le :

```sh
make up            # recrée la stack entière, ce qui est toujours sûr
```

Ou, si seul le bridge en a besoin :

```sh
docker compose --project-directory . \
  --env-file deploy/env/versions.env --env-file .env \
  -f deploy/compose.yml up -d --force-recreate bridge
```

`make up` et le rôle Ansible recréent tous les deux le projet entier,
donc cela n'arrive que si un seul service est redémarré à la main.

## Quand les décomptes de vagues sont faux

Chaque décompte de vagues de ce projet vient du wiki. Personne ne l'a
vérifié contre le jeu. Un mauvais décompte fait qu'une fin de mission se
déclenche une vague trop tôt, ou jamais.

Le plugin envoie la longueur de mission que le jeu rapporte avec chaque
check. Le bridge la compare à sa propre table et sert les désaccords comme
`wave_drift` :

```json
"wave_drift": [
  {"popfile": "mvm_decoy", "tables": 8, "observed": 7}
]
```

Un `wave_drift` vide après une mission entière veut dire que la table est
juste pour cette mission. Une mission qui y apparaît est une ligne à
corriger dans `gamedata/missions.go`. La check compte quand même : la
vague a été réussie dans les deux cas.

## Quand une mission ne fait pas partie de la partie

```
[AP] The run did not unlock mvm_decoy. Its checks still count.
```

Le serveur fait tourner une mission dont la partie n'a pas trouvé le
ticket. C'est un avertissement, pas un refus. La rotation des cartes vous
appartient.

## Quand le plugin et le bridge sont en désaccord

```
[AP] The bridge speaks API version 2 and this plugin speaks 1. Update the one that is behind.
```

Une moitié de la stack a été reconstruite et pas l'autre. Lancez
`make build` puis `make up`.

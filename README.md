# tf2-archipelago

Intégration [Archipelago](https://archipelago.gg) pour un serveur Team Fortress 2
auto-hébergé, en mode Mann vs Machine.

**État : les quatre composants sont écrits ; seul le plugin n'a jamais tourné.**
La génération de seed, le serveur Archipelago et le bridge sont vérifiés bout en
bout à chaque `make integration`. Le plugin SourceMod compile, mais aucun
serveur TF2 n'était disponible pour l'exécuter : les événements de jeu et les
propriétés réseau qu'il lit restent des suppositions, avec des replis et des
messages d'erreur en jeu pour que la première session dise lesquelles sont
bonnes.

## Démarrer

```sh
cp deploy/.env.example .env   # SRCDS_RCONPW n'a pas de défaut
make up
make logs
```

Le premier démarrage télécharge environ 14 Go de fichiers de jeu.
[`docs/`](./docs/) est un livre complet pour l'hébergeur :
[`docs/setup/install.md`](./docs/setup/install.md) donne le détail, et
[`docs/archipelago-for-mvm-players.md`](./docs/archipelago-for-mvm-players.md)
est ce qu'il faut envoyer aux joueurs qui n'ont jamais touché à un multiworld.

```sh
make check        # tout ce que la CI lance
make integration  # Archipelago + bridge en vrai, pilotés comme le plugin le fait
```

## Pourquoi MvM

MvM est le seul mode TF2 qui a déjà une progression : une mission est une suite
ordonnée de vagues, une vague se réussit ou se rate, il y a une boutique
d'améliorations persistantes, et un tour est une suite ordonnée de missions.
Ça se projette directement sur les régions et les locations d'Archipelago. Le
TF2 classique n'a rien de tout ça.

## Comment ça marche

Trois processus, une seule source de vérité.

```
  gamedata/ (Go)  ──génère──>  apworld/tf2_mvm/data/*.json
        │                                 │
        │ compilé dans                    │ lu à la génération
        v                                 v
    bridge (Go)  <──websocket──>  serveur Archipelago (conteneur)
        ^
        │ HTTP + JSON sur 127.0.0.1
        v
  plugin SourceMod  (dans le conteneur srcds)
```

Le plugin SourceMod est la seule chose qui voit le jeu. Le bridge Go est la
seule chose qui parle Archipelago. `gamedata/` en Go est la seule chose qui
sait ce qu'est une mission ou une arme, et il exporte ça en JSON pour l'apworld
Python.

Les joueurs se connectent avec un client TF2 vanilla. Rien à installer.

## Arborescence

| Répertoire | Langage | Rôle |
| --- | --- | --- |
| [`gamedata/`](./gamedata/) | Go | Source de vérité : maps, missions, vagues, armes, upgrades, robots, et les ids. Exporte le JSON. |
| [`bridge/`](./bridge/) | Go | Client Archipelago. Websocket, reconnexion, file durable, API HTTP loopback pour le plugin. |
| [`apworld/`](./apworld/) | Python | apworld mince : lit le JSON exporté, pose les régions, les règles et les options YAML. |
| [`plugin/`](./plugin/) | SourcePawn | Détecte les objectifs, applique les déblocages et les pièges. |
| [`deploy/`](./deploy/) | Compose | Serveur Archipelago + srcds + bridge. |
| [`docs/`](./docs/) | Markdown | Spec, ADR, guide d'hébergement, état de l'art, fil Discord d'origine. |

## Documentation

- [`docs/SUMMARY.md`](./docs/SUMMARY.md) : le sommaire du livre. Héberger,
  configurer la partie, jouer, dépanner.
- [`docs/archipelago-for-mvm-players.md`](./docs/archipelago-for-mvm-players.md) :
  pour un joueur qui n'a jamais fait de multiworld. Le vocabulaire et les
  commandes de chat.
- [`docs/operate/what-nobody-tested.md`](./docs/operate/what-nobody-tested.md) :
  ce qui est vérifié et ce qui ne l'est pas. À lire avant la première session.
- [`docs/spec.md`](./docs/spec.md) : la conception. Périmètre, locations,
  items, objectifs, questions ouvertes.
- [`docs/adr/`](./docs/adr/) : les décisions et pourquoi les alternatives ont
  été écartées.
- [`docs/discord-mvm-thread.md`](./docs/discord-mvm-thread.md) : le fil
  Discord Archipelago à l'origine du projet, recopié verbatim. La conception
  vient de **Damonj17** et **Roseburst**.
- [`docs/prior-art.md`](./docs/prior-art.md) : ce qui existe déjà, notamment le
  fork [ALPHAMARIOX/TF2-MvM-Archipelago](https://github.com/ArchipelagoMW/Archipelago/compare/main...ALPHAMARIOX:TF2-MvM-Archipelago:main).
- [`CONTEXT.md`](./CONTEXT.md) : glossaire. Les vocabulaires Archipelago et MvM
  partagent des mots sans partager les sens, les deux y sont fixés.
- [`TODO.md`](./TODO.md) : ce qui bloque quoi.

## Crédits

La conception vient du fil Discord Archipelago : **Damonj17** a posé la
prémisse et la forme des items et des checks, **Roseburst** a écrit
l'intégralité du plan d'options YAML. Contributions de **adeleine64DS**,
**Amazia**, **Snolid Ice**, **mudkipslike**, **TheBreadstick**, **CrystalClear**
et **Pixel Silzavon**. Les tables de données de départ viennent du fork
d'**ALPHAMARIOX**.

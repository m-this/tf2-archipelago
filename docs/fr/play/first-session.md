# La première session

## Avant que quelqu'un rejoigne

Lisez [Tests](../operate/testing.md) pour savoir ce qui a encore besoin
d'un premier vrai passage. Cela permet de distinguer un bug d'un chemin
jamais testé.

Activez le mode bavard avant la première vague :

```
rcon_password votre-SRCDS_RCONPW
rcon tf2ap_debug 1
```

Tapez cela dans la console Team Fortress 2 après vous être connecté. Une
fois activé, le serveur écrit chaque événement de jeu qu'il voit et chaque
appel qu'il fait dans le chat. C'est bruyant. C'est aussi le seul moyen de
savoir quels événements cette version du jeu envoie vraiment.

Gardez `make logs` ouvert dans un terminal sur l'hôte en même temps.

## Ce qu'un joueur voit

Huit secondes après avoir rejoint, chaque joueur reçoit ceci dans le chat :

```
[AP] This server runs an Archipelago randomizer.
[AP] The run locks the classes and the weapon slots until it finds them. All players share the unlocks.
[AP] Mission: mvm_decoy. Each wave you clear is a check.
[AP] Unlocked classes: scout, medic
[AP] Unlocked slots: primary
[AP] Type !ap to speak to the multiworld. Examples: !ap hint Class: Scout and !ap missing.
```

Les deux lignes de déblocage sont l'état de la partie à ce moment-là. Un
joueur qui rejoint tard voit ce que l'équipe a déjà trouvé.

Le délai garde le message d'accueil hors du chargement de la carte, où il
défilerait sans être lu.

## Ce qui se passe pendant une vague

- Le menu de classe refuse une classe que la partie n'a pas débloquée, avec
  une ligne dans le chat.
- Les emplacements d'arme verrouillés restent vides à chaque spawn, au
  casier de ravitaillement et à l'upgrade station.
- Chaque vague que l'équipe réussit écrit `[AP] Wave 3 cleared.` dans le
  chat.
- Chaque item que la partie reçoit écrit `[AP] Unlocked: Class: Pyro` ou
  `[AP] The run received 200 credits for 4 player(s).`
- Les lignes des autres joueurs de la session randomisée arrivent dans le
  même chat.
- Tout ce qui tourne mal est écrit en rouge, quels que soient les autres
  réglages. Un joueur le verra avant vous.

## Qui fait quoi

**L'hébergeur** se connecte comme tout le monde, et tient en plus le mot
de passe de la console. L'hébergeur lance les commandes admin, choisit la
carte entre les missions, et lit les logs sur la machine.

**Les joueurs** réussissent les vagues. Il n'y a rien à configurer pour
eux. Leurs seules commandes sont `!ap` et `!apchat`, dans
[Commandes de chat](chat-commands.md).

## La première check

Le moment qui prouve toute la chaîne est la première vague réussie.
Surveillez trois choses, dans cet ordre :

1. `[AP] Wave 1 cleared.` dans le chat du jeu. Le plugin a vu la vague.
2. `check recorded` dans le log du bridge, sur l'hôte. La check est sur le
   disque.
3. `tf2 sent <item> to <somebody>` dans le log du randomizer. Le multiworld
   l'a.

Si l'étape 1 ne se produit pas, le plugin n'a pas vu la vague. Lancez
`rcon sm_ap_status` et lisez la ligne `events:`. Voir
[Dépannage](../operate/troubleshooting.md).

Si l'étape 1 se produit et pas l'étape 2, le plugin ne peut pas atteindre
le bridge. Le chat le dit en rouge.

## Terminer la soirée

`make down` arrête la stack et garde tout. Le prochain `make up` continue
la même partie avec la même session, les mêmes checks et les mêmes
déblocages.

Rien n'expire. Une partie peut attendre une semaine.

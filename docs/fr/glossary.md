# tf2-archipelago — Contexte du domaine

Glossaire des termes utilisés dans `gamedata/`, `bridge/`, `apworld/` et
`plugin/`. Deux vocabulaires se rencontrent dans ce projet. Ils partagent
parfois les mêmes mots pour des choses différentes. Ce glossaire fixe les
deux. Les termes restent en anglais. C'est ce que le jeu écrit dans le
chat, ce que le code utilise, et ce que vous verrez dans les logs.

Dernière mise à jour : 2026-08-16. Traduction, pas une deuxième source de
vérité : voir [`CONTEXT.md`](https://github.com/m-this/tf2-archipelago/blob/main/CONTEXT.md)
à la racine du dépôt pour le texte de référence.

## Vocabulaire Archipelago

**Multiworld**
Une session de jeu générée qui réunit plusieurs joueurs et plusieurs jeux.
Les items d'un joueur peuvent se trouver dans le monde d'un autre.

**Slot**
Un participant à un multiworld, identifié par un nom. Un slot joue
exactement un jeu avec un YAML. Dans ce projet, un slot est **le serveur
TF2**, pas un compte Steam. Voir `docs/en/spec.md`, « Slot model ».

**Seed**
Le multiworld généré. Immuable une fois généré. Chaque id référencé par une
seed doit garder le même sens tant que la seed se joue. Cette contrainte
dicte les règles d'id de l'ADR 0001.

**Location** (aussi **check**)
Un endroit d'un monde où un item peut être placé. Faire la check d'une
location signale au serveur qu'un joueur a trouvé ce qui s'y trouvait. Dans
ce projet, une location est un objectif MvM : une vague réussie, une
mission réussie, un tank détruit.

**Item**
Ce que contient une location. Peut appartenir à n'importe quel slot du
multiworld : réussir une vague dans TF2 peut donner une épée à quelqu'un qui
joue à Zelda. Classé `progression`, `useful`, `filler` ou `trap`.

**Progression item**
Un item que la logique du générateur peut exiger pour atteindre une autre
location. Les emplacements d'arme, les tickets de mission et les paquets
d'amélioration sont progression ici. Se tromper de classification produit
des seeds imperdables ou triviales.

**apworld**
Le paquet Python qui apprend au générateur Archipelago un jeu donné : ses
items, ses locations, ses régions, ses règles et ses options YAML. Livré
comme un fichier `.apworld` compressé. Le nôtre est `apworld/tf2_mvm/`.

**Region / access rule**
Le graphe sur lequel le générateur raisonne. Une région contient des
locations ; une access rule est la condition pour en atteindre une.
Exemple : « la vague 4 de la mission M est atteignable une fois que vous
tenez le ticket de M et un emplacement d'arme principale ».

**Sphere 0**
Tout ce qui est atteignable sans aucun item. Si la sphère 0 ne contient
aucune location, la seed est morte-née. Voir l'exigence d'état de départ
dans `docs/en/spec.md`.

**DeathLink**
Une convention optionnelle où une mort dans un monde tue tous les autres
participants DeathLink. Ici une mort est une vague perdue par l'équipe :
une mort individuelle en MvM est routinière, et seul un anéantissement
complet fait échouer la vague. En sortie, une vague perdue est un
`Bounce` ; en entrée, un `Bounced` tue tout RED, bots compris, ce qui fait
échouer la vague. Le plugin ne renvoie pas cette perte-là. Voir
`docs/en/spec.md`, « Traps and DeathLink ».

**Trap**
Un item à effet négatif. Une classification de premier ordre dans
Archipelago, pas un contournement.

## Vocabulaire Mann vs Machine

**Mission**
Une suite ordonnée de vagues sur une carte, définie par un fichier `.pop`.
A un palier de difficulté : Normal, Intermediate, Advanced, Expert ou
Nightmare. « Mean Machines » est une mission.

**Wave**
Un assaut à l'intérieur d'une mission. Réussie ou ratée comme un tout : une
équipe anéantie rejoue la vague, elle ne perd pas la mission. L'unité
atomique de progression, donc le groupe de locations de base.

**Tour**
Un ensemble ordonné de missions joué comme une campagne. Les tours de Valve
(Operation Two Cities et les autres) sont fixes. Le générateur crée les
nôtres quand une partie choisit l'ordre de mission `Campaign`.

**Upgrade station**
La boutique de la mission. Les joueurs dépensent les crédits collectés en
améliorations persistantes par arme, entre les vagues. Les améliorations
tiennent pour toute la mission et se réinitialisent à sa fin.

**Credits / money**
L'argent que les robots détruits laissent tomber. Un joueur le ramasse
quand il passe dessus. L'argent non ramassé disparaît à la fin de la vague.
Ramasser tout l'argent d'une vague donne une **A+ rating**, qui déclenche
le groupe de locations du bonus d'argent.

**Canteen**
La Power Up Canteen, une consommable rechargée à l'upgrade station :
Übercharge, Critical Hits, Ammo Refill, Building Upgrade, Recall (Return to
Spawn). Recall et ses semblables rendent possible un piège « forced bad
canteen ».

**Giant / boss / tank**
Des variantes de robots surdimensionnées, des robots-boss nommés, et le tank
qui traîne lentement une bombe vers la trappe. Les trois sont des événements
de mise à mort distincts et observables, ce qui en fait de bonnes locations.

**Sentry Buster**
Un robot suicide qui apparaît pour détruire la sentry d'un Engineer. Listé
comme un piège dans le fil de discussion d'origine.

**Robot template**
Une définition d'équipement pour un robot : classe, arme, attributs.
Utilisée par le jeu pour les robots ennemis, et par nous pour les **allied
bots**, où un template devient un item à débloquer.

**Allied merc / RED bot**
Un `tf_bot` dans l'équipe du joueur, généré par le plugin pour compléter
une équipe plus petite que six. Son équipement vient d'un robot template
débloqué. Il partage directement les améliorations débloquées du joueur,
puisque les bots RED n'en achètent pas par eux-mêmes.

**Fichier `.pop`**
Le fichier texte de population qui définit les vagues et les robots d'une
mission. La seule autorité sur le nombre de vagues d'une mission.
`gamedata/` code en dur les nombres de vagues des missions Valve, tirés du
wiki, plutôt que de les analyser. La v1 ne prend pas en charge les missions
communautaires.

## Vocabulaire du projet

**gamedata**
Le paquet Go qui est la seule source de vérité pour chaque fait MvM et
chaque id. Compilé dans le bridge, exporte du JSON pour l'apworld. ADR 0001.

**Export**
Le JSON commité sous `apworld/tf2_mvm/data/`, produit par `gamedata/` et
vérifié en CI. Commité plutôt que généré au build pour que l'apworld reste
un artefact autonome.

**Bridge**
Le processus Go de longue durée qui tient la session websocket Archipelago
et expose une API HTTP en loopback pour le plugin. Le seul composant qui
connaît à la fois le protocole AP et la correspondance des ids. ADR 0002.

**Plugin**
Le plugin SourcePawn à l'intérieur du conteneur `srcds`. Voit le jeu, ne
connaît rien d'Archipelago, ne tient aucun état faisant autorité.

**Objective**
Le vocabulaire du plugin pour ce que le bridge appelle une location. Le
plugin rapporte `wave_cleared{mission, wave}` ; le bridge en fait un id de
location. Séparer les deux vocabulaires est ce qui permet au plugin de
rester agnostique d'Archipelago.

**Grant**
Le vocabulaire du plugin pour ce que le bridge appelle un item reçu. Le
bridge envoie `grant_weapon_slot{slot}` ; le plugin ignore quel id d'item
se trouve derrière.

**State grant / effect grant**
Les deux types de grant. `ItemKind.OneShot` dans `gamedata` distingue l'un
de l'autre. Un state grant est un fait qui reste vrai : une classe est
jouable, un emplacement d'équipement est ouvert, une mission est
disponible. Appliquer un state grant deux fois revient à l'appliquer une
fois. Le bridge peut donc le renvoyer chaque fois que le plugin le demande.

Un effect arrive et se termine : le bridge paie des crédits, ou déclenche
un piège. Appliquer un effect deux fois paie de l'argent que personne n'a
gagné, ou déclenche un piège que personne n'a mérité. Le bridge l'envoie
donc une fois, puis arrête de l'envoyer dès que le plugin l'acquitte. Seuls
les state grants apparaissent dans l'unlock set.

**Acknowledgement**
Le numéro de séquence que le plugin renvoie au bridge pour dire ce qu'il a
appliqué. Le bridge ne renvoie jamais les effects à ce numéro ou en
dessous. L'acknowledgement vit sur le bridge, sur disque. Le plugin ne se
souvient de rien après un reload, et un reload est justement le moment où
un effect se répète sans lui.

C'est aussi le curseur d'où le plugin reprend. Voilà pourquoi l'unlock set
rapporte cet acknowledgement plutôt que la longueur de la liste d'items. Un
curseur placé au-dessus d'un effect non appliqué perd cet effect pour de
bon.

**Wave drift**
Le jeu qui rapporte une longueur de mission en désaccord avec `gamedata`.
Chaque nombre de vagues dans les tables vient du wiki, et un nombre faux
fait qu'une mission se termine tôt ou jamais. Le plugin envoie ce que dit
le jeu à chaque check et le bridge sert les désaccords sur `/healthz`. Il ne
refuse jamais la check pour autant.

**Sequence**
Le curseur du plugin sur ce que le bridge a accordé. Il compte les items
reçus plutôt que les grants. Un item que le bridge ne sait pas lire ne
compte pas dans la séquence : il laisse un trou. Une sequence garde ainsi
le même sens après une mise à jour du bridge. `GET /grants?since=N` et le
`seq` d'un unlock set sont ce même nombre.

**Unlock set**
L'ensemble complet des grants actuellement en vigueur pour le slot. Fait
autorité sur le bridge, sur disque. Le plugin le redemande après chaque
reload ou changement de carte plutôt que d'essayer de s'en souvenir.

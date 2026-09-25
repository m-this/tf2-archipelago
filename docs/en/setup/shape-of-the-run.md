# Run options

The run options are the Archipelago options of your seed. They decide how
long the run is, how hard it is, what ends it, and what the checks pay.

Each option has three names, for the three places you can set it:

- the label in the launcher's **Settings**,
- the key in the YAML player file, for the Archipelago app,
- the variable in `.env`, for Docker.

**Set them before you generate the seed.** The seed keeps them. A change made
later does nothing until you [start a new run](../operate/start-a-new-run.md).

The launcher's **Check Run Selection** button is on the **Missions** page.
It tells you whether the missions you picked hold enough checks for the items
of the run. The Archipelago generator repeats that check and has the final say.

## The run

Launcher page: **Player options**.

| Launcher | YAML | `.env` | Default | What it decides |
| --- | --- | --- | --- | --- |
| Easiest tier | `difficulty_pool` | `MVM_DIFFICULTY` | `intermediate` | The easiest tier the run draws from. Harder tiers are always in. |
| Missions used | `mission_count` | `MVM_MISSION_COUNT` | `8` | How many missions the run draws, up to the size of the pool. |
| Goal | `goal` | `MVM_GOAL` | `final_boss` | What ends the run. |
| Missionsanity share | `missionsanity_percentage` | `MVM_MISSIONSANITY_PERCENTAGE` | `80` | The share of missions the Missionsanity goal asks for. |
| Death Link | `death_link` | `MVM_DEATH_LINK` | off | Share deaths with the multiworld. |
| Weapon slots per class | `class_weapon_slots` | `MVM_CLASS_WEAPON_SLOTS` | `off` | Each class earns its own slots. |
| Mission modifiers | `mission_modifiers` | `MVM_MISSION_MODIFIERS` | off | Give each mission a fixed set of modifiers. |
| Minimum modifiers | `minimum_mission_modifiers` | `MVM_MINIMUM_MISSION_MODIFIERS` | `1` | Fewest modifiers per mission. |
| Maximum modifiers | `maximum_mission_modifiers` | `MVM_MAXIMUM_MISSION_MODIFIERS` | `2` | Most modifiers per mission, up to 3. |

### Easiest tier

The run draws from this tier and every tier above it.

| Value | Missions in the pool | The team starts with |
| --- | --- | --- |
| `normal` | all 29 Valve missions | 1 class, 1 weapon slot |
| `intermediate` | 25 | 2 classes, 1 weapon slot |
| `advanced` | 17 | 3 classes, 2 weapon slots |
| `expert` | 4 | 4 classes, 3 weapon slots |
| `haunted` | 1, Caliginous Caper | 5 classes, 3 weapon slots |

The starting kit follows the tier of the easiest mission the run drew.

Do not pick `haunted` for a real run. It is one mission of 666 robots, and it
holds too few checks for the items of a run. The launcher does not offer it.

### Missions used

Most Valve missions have six or seven waves. Eight missions is about 50 waves,
which is one evening for a team that knows the mode.

If you ask for more missions than the pool holds, you get the whole pool. The
generator draws more missions than you asked for when the run needs more
checks, and it says so in its log.

### Goal

- `final_boss` marks the hardest mission the run drew. Clear it to win.
- `missionsanity` asks for a share of the missions, in any order.
  **Missionsanity share** is that share, rounded up. The default of 80 on
  eight missions means seven missions.

The Final Boss goal ignores the Missionsanity share.

### Death Link

Off by default. With it on:

- a lost wave kills every other DeathLink player in the multiworld;
- a death from another DeathLink player kills everybody on RED, bots included.

Expect a harder run. A wave can be lost to somebody else's mistake.

### Weapon slots per class

- `off`: one `Progressive Weapon Slot` item for everybody, three copies. The
  default.
- `progressive`: each class has its own item, two copies. Its first slot comes
  free with the class. Each class opens its slots in its own order.
- `any_order`: each slot of each class is a named item, found in any order.

Both other values put eighteen items in the pool where there were three. A
short run does not always have the checks to hold them. Generation says so.

### Mission modifiers

With this on, the seed gives every mission a fixed combination of modifiers:
changes to the map, the robots, the players or the waves. A mission keeps the
same combination when you replay it. **Minimum** and **Maximum** bound how
many modifiers a mission gets.

## More checks

Launcher page: **Player options**. All off by default.

| Launcher | YAML | `.env` | What it adds |
| --- | --- | --- | --- |
| Australium Medal on clear | `medal_on_clear` | `MVM_MEDAL_ON_CLEAR` | A medal item of your own on every mission clear. The goal then counts medals. Costs the multiworld one check per mission. |
| Victory caches | `victory_caches` | `MVM_VICTORY_CACHES` | A mission clear pays checks by tier: 1 at normal, 2 at intermediate, 3 at advanced, 5 at expert and haunted. |
| Milestone checks | `milestone_checks` | `MVM_MILESTONE_CHECKS` | Fifteen checks for running totals over the run: robots, giants and tanks destroyed, in any mission. |
| Giantsanity | `giantsanity` | `MVM_GIANTSANITY` | A check for every giant of every wave. Empire Escalation alone holds 82. |
| Tanksanity | `tanksanity` | `MVM_TANKSANITY` | A check for every tank of every wave. |

Use these when the run is short on checks, or when your team wants more to
find. Victory caches add checks without adding missions.

Australium Medal on clear fixes one problem. When another player finishes and
releases their items, you can receive a mission clear before you played the
mission. With a medal on each clear, the goal counts the medals you
hold, and nobody can hand you one.

## The missions

Launcher page: **Missions**.

| Launcher | YAML | `.env` | Default | What it decides |
| --- | --- | --- | --- | --- |
| The mission list, with a box per mission | `excluded_missions` | `MVM_EXCLUDED_MISSIONS` | none excluded | Missions the run never draws. |
| Community missions | `community_missions` | `MVM_COMMUNITY_MISSIONS` | on | Whether the run draws community missions at all. |
| Start mission | `start_mission` | `MVM_START_MISSION` | `random` | The mission the run starts on. |
| Start class | `start_class` | `MVM_START_CLASS` | `random` | One of the classes the run starts with. |
| The server mods list | `server_mods` | `SRCDS_MODS` | none | Mods the server loads. The run draws a mission that needs a mod only when the mod is on. |

### Excluded missions

Untick a mission in the launcher, or name it in the YAML or `.env` by its
file name. `mvm_ghost_town_666` keeps out Caliginous Caper, one wave of 666
robots that takes an hour on its own.

### Start mission and start class

`random` leaves both to the seed. The run then starts on the easiest mission
it drew, with random classes.

Name a mission, and the run always draws it and starts there. Name a class,
and the run always starts with it. The tier of the start mission still decides
how many classes the run starts with.

Do not name the hardest mission of the run as the start with the Final Boss
goal. Clearing it wins on the spot, so generation stops.

### Community missions

Community missions come from asset packs that you download in the launcher,
on the **Missions** page. Once a pack is on disk, its missions appear in the
list. See the
[community content guide](https://github.com/m-this/tf2-archipelago/blob/main/community-content/README.md).

Some community missions need a server mod. The one mod so far is SigMod
(`sigsegv-mvm`). On the **Missions** page each mod has three answers:

| Answer | What the server does |
| --- | --- |
| **off** | Never loads the mod. Its missions leave the pool. |
| **only when a mission needs it** | Loads it while the pool holds a mission that names it. The default. |
| **always, on every mission** | Loads it on every map. |

The native launcher installs the mod when it is needed. Docker's server image
already includes it; save the selection and recreate the containers to apply
it. Turning it off leaves the files in place, so turning it back on costs no
download.

On Windows the row reads **SigMod (beta)**. Linux and Docker use the
mod's own release, which players run every day. Windows has no such release,
so it downloads this project's port instead. That port crashed one player's
server. Leave it on **only when a mission needs it** unless you test it. To
play the missions that need it today, run the server on Linux, in Docker, or
under WSL.

In `.env` the answer is `SRCDS_MOD_LOADING`, and Docker reads it as on or off:
the image cannot tell which missions the pool holds.

## Rewards

Launcher page: **Rewards**. These options decide what fills the checks left
after the classes, the slots and the tickets.

| Launcher | YAML | `.env` | Default | What it decides |
| --- | --- | --- | --- | --- |
| Mission tickets | `mission_ticket_importance` | `MVM_MISSION_TICKET_IMPORTANCE` | `progression` | Whether tickets gate missions. |
| Class unlocks | `class_unlock_importance` | `MVM_CLASS_UNLOCK_IMPORTANCE` | `progression` | Whether classes count for the tier requirements. |
| Weapon slots | `weapon_slot_importance` | `MVM_WEAPON_SLOT_IMPORTANCE` | `progression` | Whether slots count for the tier requirements. |
| Weapon buffs | `weapon_buff_importance` | `MVM_WEAPON_BUFF_IMPORTANCE` | `useful` | Whether harder tiers need some buffs. |
| Cash rewards | `cash_rewards` | `MVM_CASH_REWARDS` | off | Let spare checks pay cash. |
| Unlockable bot cards | `bot_cards` | `MVM_BOT_CARDS` | off | Put every distinct non-starting card on a spare check when the seed has room; cards never gate progress. |
| Starting card mode | `starting_bot_card_mode` | `MVM_STARTING_BOT_CARD_MODE` | `draw_random` | Random cards or one stock-loadout card for each class. |
| Random starting cards | `starting_bot_cards` | `MVM_STARTING_BOT_CARDS` | `0` | Distinct cards received at the start; ignored in stock-class mode. |
| Buff share | `weapon_buff_percentage` | `MVM_WEAPON_BUFF_PERCENTAGE` | `75` | With cash on, the share of spare checks that pay a buff. |
| Buff stack chance | `weapon_buff_stack_chance` | `MVM_WEAPON_BUFF_STACK_CHANCE` | `25` | Chance that a buff adds a level to one already in the seed. |
| Traps (%) | `trap_percentage` | `MVM_TRAP_PERCENTAGE` | `1` | The share of spare checks that hold a trap. |
| Grappling Hook | `server_settings` | `MVM_SERVER_SETTINGS` | off | Put the Grappling Hook in the pool. |

Each card's rarity and form are separate seed rolls: Common / Elite / Legendary
are 50% / 40% / 10%, and Human / RED robot / Giant are 50% / 40% / 10%.
Only cards received by the room can be recruited in the admin UI. Test mode
continues to offer the full collection.

### Importance

`progression` means the generator can put the item behind a check you need
it for. `useful` means the item never gates anything.

- With tickets on `useful`, every mission of the run is open from the start.
- With classes or slots on `useful`, the tiers ask for nothing.
- With buffs on `progression`, harder tiers ask for some buffs.

### Cash and buffs

With **Cash rewards** off, every spare check pays a weapon buff. With it on,
**Buff share** decides the split, and the rest pays cash.

A `Cash Bundle` pays 200 credits to each player on RED. It is paid at the
upgrade station, between waves, so a lost wave cannot take it back.

A weapon buff is a permanent bonus on one weapon family. Numeric buffs stack a
level each time. On/off buffs never repeat.

### Traps

A trap is an item with a bad effect. Another player finds it, and your team
pays for it. The one trap so far is `Trap: Team Jarate`: ten seconds of Jarate
for everybody on RED, during the next wave. A trap never takes back an unlock.

## Balancing

Launcher page: **Balancing**. This is a server setting, not a seed option.
It applies at the next map load.

| Launcher | `.env` | Default | What it decides |
| --- | --- | --- | --- |
| Robot health (%) | `SRCDS_BLU_HEALTH_PCT` | `100` | A multiplier on the health of every robot, from 10 to 1000. |

Lower it for a short team. See [The bots on your team](../play/defender-bots.md#a-short-team).

## The room

Launcher page: **Archipelago room**.

| Launcher | `.env` | Default | What it decides |
| --- | --- | --- | --- |
| Room address | `AP_ROOM`, or `AP_HOST` and `AP_PORT` | none | The line from the room page, like `archipelago.gg:12345`. |
| Room password | `AP_PASSWORD` | empty | The password of the room, if you set one. |
| Slot name | `AP_SLOT_NAME` | `tf2` | The name of your server in the session. It has to match the `name` in the player file. |
| Test mode | `TF2AP_TEST_MODE` | off | Play without a room. The launcher fakes a multiworld of one. |
| | `AP_TLS` | `true` | Docker only. `true` for a room on `archipelago.gg`, `false` for a room hosted in the stack. |

## The game server

Launcher page: **Game server**.

| Launcher | `.env` | Default | What it decides |
| --- | --- | --- | --- |
| Server name | `SRCDS_HOSTNAME` | `Mann vs Archipelago` | The name in the server browser. |
| Server password | `SRCDS_PW` | empty | The password players type to join. Empty lets anybody with the address in. |
| Game port | `SRCDS_PORT` | `27015` | The port players connect to, UDP and TCP. |
| Join address | `TF2AP_JOIN_HOST` | this machine | The address shown on the Join line. |
| Admins by Steam id | `SRCDS_ADMIN_STEAMIDS` | empty | Who can switch missions and bots from the chat. See [Chat commands](../play/chat-commands.md#for-the-admin). |
| | `SRCDS_RCONPW` | none | Docker only. The remote console password. The launcher picks one for you. |
| | `SRCDS_START_MISSION` | empty | The mission the server loads at start. The launcher sets it from **Start mission**. |
| | `TF2AP_NEXT_MISSION_DELAY` | `30` | Seconds between a mission clear and the next mission. |
| | `SRCDS_MAXPLAYERS` | `32` | Do not lower it. TF2 refuses to host MvM with fewer slots. |

The page **Networking** decides who can reach the server. See
[Invite your friends](invite-your-friends.md). [The bots on your team](../play/defender-bots.md).

Next: [Create the session](create-the-session.md).

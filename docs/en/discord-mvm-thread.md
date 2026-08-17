# Discord thread: TF2 Mann vs Machine apworld

Verbatim copy of the Archipelago Discord thread that started this project.
Kept unedited on purpose: `spec.md` is the interpretation, this file is the
source. Do not rewrite it, only append if the thread grows.

Two people carry the design here. **Damonj17** opened the thread with the
premise and the item/check shape. **Roseburst** posted the full YAML option
outline that `spec.md` is built on.

---

**Damonj17 (She/Her):**

> DISCLAIMER: I do not have the ability to create this implementation. I am posting this mostly in the hopes of creating discussion/catching the eyes of an actual dev.
> I apologize if this game belongs in the AD server instead of here, but it Should be fine? Anyhoot, this would be an apworld for the Gamemode Mann vs Machine.
>
> How would it work?
>
> Sourcemod server plugin, connect locally/portforward similar to Minecraft
> Damage multiplier/Some Mechanic to compensate for not having a full team of 6?
> Goal could be number of waves completed, missions completed, etc.
>
> Send out items for the following:
>
> Wave Completion (Additional items depending on credit score?)
> MvM achievements
> Bosses/Giants/Tanks defeated
>
> Receive the following as items:
>
> Weapon Slots (Primary, Secondary, Melee)
> Access to Canteens/Upgrades
>
> Bad Ideas That Someone Would Probably Want:
>
> Randomize what Classes are available?
> Cashsanity (Every 100 dollars is a check, and you need your fellow players in other worlds to help you Get That Money !)
>
> Traps:
>
> Deathlink/Traps on map events (Rottenburg Barrier, Mannhattan captures)
> Forced Bad Canteens/Upgrades, like Return to Spawn/Heavy Rage
> Sentry Buster/Engineer/Sniper/Spy Traps

**[10:49 AM] adeleine64DS 🐵 [DK64]:** looks to be fine for here *(image)*

**[11:08 AM] mudkipslike (elebits360) [₠ʟɪᴀ]:** Heavy Rage is useful for giant robots

**[12:45 PM] adeleine64DS 🐵 [DK64]:** Im pretty sure a manual has been made too

**[12:56 PM] adeleine64DS 🐵 [DK64]:** @Snolid Ice I know you want this

**[12:57 PM] adeleine64DS 🐵 [DK64]:** Do you think it would be viable to have weapons in the item pool? I know most servers can let you use whatever weapon

**[1:07 PM] Amazia [AP]:** Does MvM support bot teammates and would they even be worth having as items? (Yes the irony of bot teammates in the "vs Machines" mode is not lost on me)

**[1:14 PM] adeleine64DS 🐵 [DK64]:** All depends on any damage multiplier being done

**[1:40 PM] Damonj17 (OP):** They definitely used to work in MvM, although I doubt that they would really work for classes that do more than just damage (Engineer/Medic/Scout) if they still worked

**[1:40 PM] Damonj17 (OP):** The damage multiplier was more proposed as "okay, since you're missing a damage class you do more damage to compensate" since the waves would be balanced around 6 players

**[1:44 PM] Damonj17 (OP):** There is also a difficulty setting for tf_bots iirc, which might be useful, although there's also the question of Getting Red Bots to Buy Upgrades

**[1:45 PM] adeleine64DS 🐵 [DK64]:** They could share your upgrades maybe

**[1:45 PM] adeleine64DS 🐵 [DK64]:** Like you just have an all class shop and each upgrade only has to be gotten once for everyone

**[1:46 PM] Damonj17 (OP):** Oh god, i just had the thought of upgrades counting as checks like shops in other games

**[1:46 PM] Damonj17 (OP):** You open the upgrade station and automatically get a bajillion hints

**[1:47 PM] Damonj17 (OP):** Lingo colors are on demoknight shield upgrades...

**[1:47 PM] adeleine64DS 🐵 [DK64]:** You could also require that you have that class/weapon before you can buy the upgrade

**[1:47 PM] adeleine64DS 🐵 [DK64]:** Its all coming together

**[5:19 PM] Snolid Ice:** uh kinda

**[5:40 PM] TheBreadstick (Ping Scooty):** I would be so down for this

**[5:40 PM] TheBreadstick (Ping Scooty):** More reasons to play MvM is always a good thing

**[12:13 AM] Snolid Ice:** oh yeah if u do guys do base it off of my mvm manual my mvm manual is kinda rough

**[7:34 AM] Pixel Silzavon:** I'm curious about this. o:

**[7:34 AM] Snolid Ice:** i made a mvm manual

**[12:23 AM] CrystalClear:** This would be super cool if it could be implemented

---

## Roseburst's design outline

**[8:40 AM] Roseburst:**

> Had a personal idea for how this could work. My thoughts were to develop a Sourcemod plugin that could be added onto either a local or personal server, allowing us to have much more freedom with what the plugin can do. However, it could be argued that a plugin-less approach using either official boot camp or community servers would be more practical.
>
> Here is a a possible design outline of how this experience would work:
>
> **YAML Parameters**
>
> **Maps**
> Defines what maps are included in the pool. Mainly affects variety and not run length. Selection of the following:
> - **Base Maps** - Includes the maps Decoy, Coaltown, and Mannworks, the three original maps for the mode.
> - **Base +** - Adds the additional maps Bigrock, Mannhattan, and Rottenburg to the pool.
> - **Base + Event** - Adds Wave 666 to the mission pool.
> - **Community** - Includes community maps that have been officially promoted in Potato.tf/Moonlight.tf's tours. Does not include Halloween/April Fools event maps, such as Swirl Event.
> - **Community Event** - Adds on additional maps featured in the Halloween/April Fools tours.
> - **All Basic** - A combination of Base + and Community
> - **All** - A combination of Base + Event and Community Event.

**[8:40 AM] Roseburst:**

> **Missions**
> Defines what missions are includes in the pool. Essentially defines the difficulty of the run. Selection of the following:
> - **Boot Camp** - Only includes Normal and Intermediate missions.
> - **Mann Up** - Includes Intermediate, Advanced, and Expert missions.
> - **Nightmare** - Includes Intermediate, Advanced, Expert, and Nightmare missions.
>
> **Mission Number**
> Defines how many missions are included in the run. A slider that can select anywhere from 1 to 50 missions. If the map/difficulty selection can't find enough maps, it will lower the size to include the maps it can.
> (For instance, if the player selects Base Maps + and Boot Camp but chooses 20 missions, it would get downsized to 13, as there are only a combined 13 Normal/Intermediate maps in that selection.)
>
> **Mission Order**
> In a similar system to Starcraft 2's Archipelago implementation, this would define how the run is played. Selection of the following:
> - **Tour of Duty** - Forms a Tour of the selected missions. All are available to start.
> - **Campaign** - Organizes the selected missions into several tours. Each tour is given a randomly generated name, with one being given to the player and the others being locked behind a Ticket to that respective tour, which is given its own check.
> - **Gauntlet** - The player is given access to a certain number of starting mission(s). Other missions are unlocked through Ticket checks respective to the mission.
>
> **Goal**
> Defines how a run is won. Selection of the following:
> - **Final Boss** - Selects a mission of the hardest available difficulties to be the 'Final Boss'. Completing that mission wins the run.
> - **Missionsanity** - Completing a certain percentage of missions results in a win.
> - **Australium Hunt** - Creates 'Junk' Australium Weapon checks, a percentage of which are required to win the run.
>
> **Tour Size**
> Defines how large a tour is. Used for the Campaign setting. Slider from 1-10.

**[8:40 AM] Roseburst:**

> **Allied Mercs**
> If the plugin sees that there are less than 6 players on Red Team, it spawns in bots to assist you. These default to Bat Scouts, though the player is able to adjust each bot through a menu. Enemies killed by bots drop Red Money which is automatically collected. Options are the following:
> - **Off** - No allied bots will spawn. Not recommended outside of having a full team of 6 players in the server.
> - **Fill 6** - Allied bots will spawn until there are 6 total players on Red Team.
> - **Fill 10** - Allied bots will spawn until there are 10 total players on Red Team.
> - **Scavenge** - Start with a number of bot teammates to start, with checks that give you access to additional teammates.
>
> **Merc Loadouts**
> Defines how the player gets access to friendly equipment. Includes the following:
> - **Robot Sync** - Enables bots that use weapons and/or upgrades that the player has access to. Ex. If the player has the Fists of Steel check, they may change their bots to Fists of Steel Heavies.
> - **Robot Templates** - Assigns individual bot loadouts to checks. Once unlocked, any number of bots may use that loadout. Ex. By completing the first wave of Empire Escalation, the player gains access to Uber Medics.
> - **Single Templates** - Same as above, but each template may only be used by a single robot at a time.
>
> **Giants and Bosses**
> Defines wether or not friendly Giant Robots and/or Boss Robots can be used. Selection of one of the following:
> - **Off** - Giants and Bosses are not included.
> - **Limit 1 Boss** - The player may have one Giant Robot of their choosing on their team.
> - **Limit 1 Any** - The player may have one Giant Robot OR Boss of their choosing on their team.
> - **Limit 1 Each** - The player may have one Giant Robot AND one Boss of their choosing on their team.
> - **Unlimited Giants** - The player may have any number of Giants on their team.
> - **Any Giant, Limit Boss** - The player may have any number of Giant Robots on their team, but only 1 Boss.
> - **Unlimited Any** - No limits for Giants and Bosses.

**[8:40 AM] Roseburst:**

> **Player Upgrades**
> Defines how player upgrades are handled. Selection of one of the following:
> - **Off** - Players have access to all the upgrades they would usually have without any restrictions.
> - **Weapon Unlocks** - Each weapon is given its own check, which when unlocked gives full access to that weapon and all of its upgrades.
> - **Upgrade Packages** - Each upgrade is given its own check, but these are consistent on all weapons with that upgrade. Ex. If a player gets Primary Damage, they can get damage for the Minigun, Sydney Sleeper, and Flamethrower.
>
> **Initial Upgrades**
> Defines how many upgrades the player is given at the start of the run. By default, gives 1 random weapon/weapon package.
>
> **Initial Robots**
> Defines how many Robot templates are given at the start of the run, if that setting is enabled. Only gives smallbots.

**[8:40 AM] Roseburst:**

> **Trap Checks**
> Enables recieved checks that can include negative effects, such as applying Jarate to you and your team, stunning your robot allies, spawning extra (Boss) robots, and more.
>
> **Check Options**
>
> **Wave Checks**
> Choose if completing a mission wave gives a check.
>
> **Mission checks**
> Choose if completing a mission gives a check.
>
> **Tour checks**
> Choose if completing a tour gives a check. Only available for the Campaign setting.
>
> **Money Checks**
> Choose if getting a perfect rating of A+ by collecting all money in a wave gives a check.
>
> **Shop Checks**
> Choose if each mission can have purchasable checks in the shop for $100-$400 of dropped cash. Options are:
> - **Off** - No checks are sold.
> - **Wave Turn-in** - Checks are sold in the shop, and completing a wave with one active rewards the check.
> - **Mission Turn-in** - Checks are sold in the shop, but you must complete the mission with them having been purchased.
>
> **Example Run**
>
> Base Maps +
> Mann Up
> 8 Missions
> Campaign
> Final Boss
> 3 Mission Tour
> Fill 10
> Robot Templates
> Limit 1 Each
> Weapon Unlocks
> 1 Initial Upgrade
> 5 Random Robots
> Wave Checks, Mission Checks, Tour Checks, No Shop Checks.
>
> When loading in, the player finds their tours are the following:
>
> (UNLOCKED) **Diamond Disintegration** - Mean Machines, Big Apple Barricade, Halmet Hostility
> **Ruby Ruin** - Broken Parts, CPU Slaughter, Cave-In
> **Thermite Thurable** - Bavarian Botbash, (FINAL BOSS) Mannslaughter
>
> Their initial weapon is the Cow Mangler.
>
> Their 5 random robots are Blast Soldiers, Samurai Demomen, Rapid Fire Bowmen, Shortstop Scouts and Charged Giant Flare Pyros.
>
> To unlock the Thermite Thurable ticket, they would have to complete Hamlet Hostility.

**[5:33 PM] Snolid Ice:** I agree with all of this
>
> Also making it a source plugin is smart I didn't think of that

**[4:07 AM] Roseburst:** It would enable you to play with friends, or on your own on a private server

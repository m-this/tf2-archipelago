# Changes

What each release changes, for somebody who plays the game. The workflow in
`.github/workflows/release.yml` reads the section matching the tag and puts it
in the release notes, so this file is the only place to write it.

## Unreleased

### Fixed

- **Make it Count works on a Windows server.** It switched itself off there,
  because the plugin could not find the dispenser's ammo function on Windows.
- **A server whose bridge stops answering no longer runs out of memory.** The
  plugin sent every check it was still holding again every five seconds, so
  each giant and tank killed while the bridge was down made the traffic grow,
  until the server stopped with "Out of memory" a few missions later. It now
  retries the oldest check alone until the bridge answers, then sends the rest.

## v1.17.3

### Added

- **Server mods have three answers instead of a tick: off, only when a mission
  needs it, or always.** Installing a mod used to be the same thing as loading
  it, because the game server loads an extension whenever its file is on the
  disk. They are separate now. The default loads a mod only while your mission
  pool holds a mission that names it, so a run that never asks for SigMod
  never has it in the server. Turning a mod off leaves the download in place,
  so turning it back on costs nothing.
- **SigMod is offered on Windows again**, through this project's own port,
  since upstream publishes no Windows build. It is off by default and the row
  reads **SigMod (beta)**: the port crashed one player's server, and the new
  loading setting is what makes it safe to offer. Leave it on **only when a
  mission needs it** unless you are testing it. Linux, Docker and WSL download
  upstream's build and are the safe way to play those missions today.
- **Each reward family can be turned off.** Mission Ticket, Class Unlock and
  Weapon Slot Importance take **Disabled - All Unlocked**: everything starts
  unlocked and leaves the item pool, per-class slots included. Weapon Buff
  Importance takes **Disabled - No Buffs**, and the spare checks become cash.
  The defaults are unchanged. By kelly-cs.
- **Remedic is labelled Medic only** in the mission table, the mission pool and
  the start-mission choices, since its population file bans every other class.
  By kelly-cs.
- **Community archives are checked against a pinned hash.** The Potato and
  Moonlight snapshots now download from this project's GitHub release first,
  with the Potato servers as fallback, and a file whose hash differs is held
  back until you press **Ignore hash mismatch** on the Missions page. By
  kelly-cs.

### Fixed

- **A server mod can be turned off.** Whether you run the launcher or the
  Docker stack, taking a mod out of the settings left it loading from the game
  files it had already been installed into, with nothing on screen admitting
  it. Both now take the marker away.
- **Modifiers work on a Windows server.** Thirteen of the plugin's game hooks
  had no Windows signature, so a modifier could be drawn and announced while
  half of it did nothing. Faulty Calibration and Mental's cloak now work there,
  and a refused mission file is reported. Bot Surge's support fillers and Make
  it Count's dispenser limit still do nothing on Windows.
- **A robot stuck protected in its spawn no longer holds the wave open.** After
  two minutes the plugin removes it. Condemned Trespasser wave 2 could not end
  without this. By kelly-cs.
- **Bot Surge no longer loses scripted waves at once.** It shortened every
  spawn delay, including the timers some missions use to open the hatch, so
  Homestead Happenings wave 5 was lost after seven seconds. Only robot spawns
  are sped up now. By kelly-cs.
- **Caliginous Caper wave 666 has its tanks and its Halloween look.** The stock
  file sends its tanks down a path the map does not have. The server now
  repairs it from your game files at start. By kelly-cs.
- **SigMod no longer kicks RED defender bots.** Its robot limit counted them,
  so a crowded wave kicked one and the bots added it back, over and over. By
  kelly-cs.
- **Ten community missions are marked as needing SigMod**, among them Bionic
  Breach, Nuclear Threat and Twilight Of Terror. They were offered without it
  and could not run. By kelly-cs.
- **SigMod's extra loadout menu shows its item names and title** instead of
  `%t: %s %s %t`. On the Windows and Linux launchers, SigMod shows as not ready
  once after this update: press **Download / set up selected server mods** to
  apply the fix. By kelly-cs.
- **Docker: editing SigMod's config no longer hides its missions**, and the
  settings page no longer points at a setup button that does nothing in Docker.
  By kelly-cs.
- **Docker: Test mode uses the whole mission pool you asked for**, not eight,
  and the server picks up a new test run without a restart. By kelly-cs.
- **The item server line no longer shows terminal color codes.** By kelly-cs.

## v1.17.2

### Added

- **A Documentation link in the launcher's footer.** It opens the book, which
  now walks every tab and every settings page with a picture of each, and has a
  page on the campaign tracker.

### Fixed

- **SigMod is off on Windows again, and a Windows server that has it installed
  starts.** v1.17.0 offered SigMod on the Missions page on Windows. Ticking it
  installed a build whose game server died with a corrupted heap before it
  finished loading, on every start, and the launcher only said the server had
  crashed. Unticking the mod could not undo it: the extension installs a file
  that makes the server load it whatever the launcher's settings say, so the
  crash outlived the setting. Start now takes that installation off the disk
  and runs without it, the row on the Missions page says there is no Windows
  build, and a run whose missions need SigMod no longer refuses to start on a
  machine that cannot have it. Linux is unchanged. Reported by [-SAM-].

## v1.17.1

### Fixed

- **A mission file the server cannot read now says so, and the run moves on.**
  A community mission whose file uses syntax stock TF2 rejects left the map up
  with no waves in it and the team on wave 0, with nothing saying why. The
  server now names the mission, points at the console line that names what it
  refused, says when SigMod is the missing piece, and moves to the next mission
  in the run. Reported by Doctor Pyro on Breaking Walls.
- **A long kill tally can no longer hold up a multiworld.** Every tally was
  reachable from the first mission, so an item somebody else needed could be
  placed behind "40 Tanks Destroyed" and the only way to it was replaying one
  mission ten times over. The first tally of each kind still counts from the
  start; the longer ones now wait for their share of the run. Reported by
  Roseburst.

## v1.17.0

### Two new mission modifiers

- **Mental.** The robots, the tanks and their buildings stay cloaked for the
  whole wave. A normal hit makes one shimmer for a moment; a reveal-on-hit
  weapon buff decloaks it properly for five seconds before it fades back.
- **Make it Count.** Your non-melee weapons and your sentries hit for two and
  a half times as much, and in exchange RED carries a quarter of the reserve
  ammo and the Engineer a quarter of the metal. Dispensers top you up every
  four seconds rather than continuously.
- Both by kelly-cs, and both are drawn per mission like the rest: a mission
  keeps the modifiers it was given for the whole run.

### SigMod missions on a Windows server

- **The 99 community missions that need SigMod now run on Windows.** They were
  listed and locked there because rafradek publishes a Linux server build only.
  The extension is now cross-compiled for Windows in `m-this/sigsegv-mvm-win`,
  and the launcher downloads and verifies it the same way it does the Linux one.
- Turn on **SigMod** on the Missions page and press **Download / set up
  selected server mods**, exactly as on Linux. Nothing installs it otherwise:
  it is only needed by missions that ask for it, and a run without one never
  downloads it.

### Fixed

- **Defender bots ready up again after a map change.** A per-client
  ready-command deadline kept the previous map's game clock, so the bots
  stopped pressing F4 until the new map's clock had caught up with it. On a
  mission switch that could leave the team standing in the spawn room.
- **A bot is no longer called "// Managed by tf2ap. Edits here".** The names
  file the launcher writes starts with a managed-file comment, and the mod read
  that line as a name and let the game cut it at 31 characters. Reported with a
  screenshot by Cowser the Khelinace.
- **Adding a bot name works again.** Typing a name on the Bots page and pressing
  Add this name answered `settings action "bots.name_add" is not wired` and
  added nothing. Reported by Cowser the Khelinace.
- **A Windows server no longer hangs when the mission changes.** The bots walked
  navigation areas left over from the previous map and the server never came
  back, with nothing in the log after the last bot finished shopping. Since a
  run changes mission by changing map, every switch could meet it.
- **Bot Surge waves that stalled now recover.** On some wave patterns the
  endless support robots respawned fast enough to hold every enemy slot, so the
  wave itself stopped progressing, Cybernetic Carnage wave 1 among them. After
  twenty seconds with no progress and no tank alive, the support is cleared for
  four seconds to let the wave through. By kelly-cs.
- **Docker: the defender bots load again.** The v1.16.1 server image built its
  two native extensions against a newer C library than the image runs, so
  SourceMod could load neither and no bots joined. By kelly-cs.

## v1.16.2

### Fixed

- Turning off SigMod or Community missions on the Missions page now takes their
  missions out of the pool. Before, they stayed ticked: the pool looked full,
  and Check Run Selection still asked for SigMod after you had turned it off.
  If the start mission was one of them, the start goes back to Any and the
  server boots on its default map again. By kelly-cs.
- A mission that stayed ticked after its requirement went away can be unticked.
  Its row used to be locked, so it could not be taken out of the pool.
- All in the pool no longer puts community missions back while Community
  missions is off.

## v1.16.1

### Weapon slots, found in any order

- **Weapon slots per class is now three ways, not two.** Off is one progressive
  item for everybody. Progressive is what v1.16.0 shipped: an item per class,
  opening that class's slots in its own order. **Any order** is the same
  eighteen slots as items that name them, found in whatever order the
  multiworld puts them in, so a Scout can find his melee before his secondary.
- Safe because the slot a class is played with still comes free with the class:
  the Medigun for a Medic, the Knife for a Spy. Nobody is left holding only
  their worst weapon.
- The items read as what they are: "Spy Revolver Slot" in somebody else's
  spoiler log, rather than a progressive item nobody outside the run can place.
- Same eighteen items either way, so a run needs no more room for them than it
  did. A yaml saying `true` still means progressive.

### Fixed

- A mission whose file this server does not have is said so and skipped, rather
  than loading its map and sitting on wave 0 for ever. TF2 ignores a mission
  file it cannot find and keeps the map's own, so a run that drew a community
  mission without the pack that carries it stopped dead with nothing saying
  why. `!missions` marks those `[not installed]`.

### The bots have names you choose

- **A Names section on the Bots page.** The bots draw from a pool of names as
  they take their seats, and the pool is yours: add your own, and leave out any
  of the ones this launcher ships. The list is kept as what you changed rather
  than as a copy, so names added in a later release still reach you.
- It works the same in the Docker stack, through `SRCDS_BOT_NAMES_EXCLUDED` and
  `SRCDS_BOT_NAMES_ADDED` in `.env`.
- **A name for a seat.** Seat 3 can be called Gravel Pit Gary every run: the bot
  that sits there is given the name instead of drawing one. The rest of the
  seats keep drawing.
- Renaming a seat costs the team nothing. The bots are renamed where they stand,
  rather than rebuilt: a new team would lose the upgrades it had bought, which
  is a steep price for a change of name. Bots at 0.16.0.
- The pool itself is read when a map starts, so adding or removing a name
  reaches the bots on the next mission.

### Fixed

- In the Docker stack the bots played stock weapons whatever the admin page
  said. The loadouts reached `.env` and stopped there: nothing wrote the file
  the mod reads, and nothing turned on the setting that makes it read one. A
  loadout built on the Loadouts page was also forgotten at the next restart,
  because `.env` is the whole of what the page remembers and the built ones
  were never written to it.
- A team saved on the admin page was forgotten the same way, for the same
  reason.

## v1.16.0

The Docker stack gets the launcher's browser interface, five new ways to pay
checks, and the bugs that came with running the stack on a plain Docker Engine.
Nothing here changes a run in progress: a seed keeps the options it was made
with. Bots at 0.15.2.

### The admin page

- **A browser interface for Compose.** `docker compose up -d` now serves the
  same page the native launcher opens, on **http://127.0.0.1:8477**: the run
  and its missions, what the multiworld has unlocked, your bot team, and the
  server and bridge logs with the search over them. `docker compose logs admin`
  prints the address, and `TF2AP_ADMIN_PORT` in `.env` moves it.
- It stays on loopback, because the page can send server commands. For a
  server on another machine, forward it: `ssh -L 8477:127.0.0.1:8477 user@host`.
- Docker keeps the server's lifecycle. Start, Stop and Restart print the
  Compose command to run rather than pretending to do it themselves: the page
  has no Docker socket, and giving it one would give it the whole host.
- The Settings tab writes `.env`, the file Compose already reads, keeping its
  comments and everything in it the launcher does not own. Settings the
  containers read apply on `docker compose up -d --force-recreate`; seed
  options apply the next time the seed runs.
- **Join address.** A setting for the address the Join button hands to players.
  Blank discovers a local one. Name your public IP or DNS for a forwarded
  server, or the WSL address when Docker runs in WSL and TF2 runs on Windows.
- Community packs download from the page, with the progress on the page rather
  than buried in the log, and the running server picks the extracted files up
  within about thirty seconds. Nothing to rebuild.
- SigMod ships in the image, and the page now recognises it as installed
  instead of offering to install what is already there.
- **Tailscale needs no auth key.** Set up / check Funnel signs the bundled
  service in through a link in the browser, and the login survives restarts in
  a volume of its own. Its Compose profile and `TAILSCALE_AUTHKEY` are both
  gone; the switch on its own is enough.

### More checks, without more missions

Five options, all off by default. Each one adds checks to a seed, so a short
run has something to fill it and a long one has more to find.

- **Victory caches.** A mission clear is worth checks scaled by its tier: one
  at normal, two at intermediate, three at advanced, five at expert and
  haunted.
- **Milestone checks.** Running totals over the whole run, whatever mission
  they happened in: robots, giants and tanks destroyed. Something to grind when
  a wave will not fall.
- **Giantsanity** and **Tanksanity.** A check for every giant, and for every
  tank, of every wave, beside the mission's own first of each. The counts come
  out of Valve's own mission files, so community missions pay only their first.
  Empire Escalation alone holds eighty-two giants, Cataclysm eleven tanks.
- **Weapon slots per class.** Each class earns its own two slots instead of one
  item opening a slot for everybody, and its first slot comes free with the
  class: the Medigun for a Medic, the Knife for a Spy. Eighteen items in the
  pool where there were three, so a run with few missions may not fit them.

### The tracker has an address

- The visual campaign tracker is published at
  <https://m-this.github.io/tf2-archipelago/tracker/>. Paste a room or tracker
  link and share the page; nobody has to host it.

### The bots

The mod moves to 0.15.2.

- The mercenaries stopped shouting over each other in the break. A bot pressed
  ready on every frame the game refused one, which is every frame for the first
  seconds after a wave ends, and every press spoke its Ready line. It presses
  once a second now.
- In the Docker stack they stopped moving after a map change. The navigation
  they read was emptied by the changelevel and nothing filled it again, so only
  a restart brought them back. By kelly-cs.

### Fixed

- The Docker stack finds the Steam relay address, the way the launcher started
  doing in v1.14.0: from the server's own console file, the only place it
  lands. Join could say "waiting" for a whole session on a relay server that
  was working perfectly.
- On Docker Engine the page could not save anything at all: every Save and
  every pack download failed with a permission error. Docker Desktop hides the
  file ownership behind it, so testing there never hit it.
- Saving any setting, against a multiworld hosted inside the stack, turned TLS
  on for it. The bridge then never reconnected.
- A stack that kept the example's `MVM_START_MISSION=random` could not generate
  a seed. It had been refusing since v1.11.0.
- A crash in the container leaves a `debug.log` with a stack in it. The image
  asked the server for one and had no debugger to produce it.
- Admins listed in `.env` got no permissions until somebody reloaded the admin
  cache by hand. The server is told about the list as soon as it is written.
- Test mode left missions locked until the starting inventory arrived, so the
  first map change could be refused.
- A mission drawn with zero modifiers still drew one.
- Weapon buffs offered flags the weapon already carries, and five weapons were
  offered buffs that do nothing on them. A buff in the pool now changes
  something on the weapon it names.
- Max health buffs followed the class rather than the player holding them.

## v1.15.0

- The mission list could vanish mid-run. Any mission nobody had cleared a
  wave in yet, which is most of them on a normal run, threw an error the
  plugin never recovered from, leaving "the bridge has not answered" for the
  rest of the session.
- Rootless Podman, the default on Bazzite, could fail to write the server's
  admin list on start and stop picking up plugin updates until the container
  was restarted by hand.

## v1.14.0

Missions that change their own rules, a tracker your spectators can open, SigMod
missions the launcher installs itself, and the Grappling Hook fix the last
release left on nightly. Bots at 0.15.1.

### The run

- **Mission modifiers.** A `mission_modifiers` option, off by default, gives
  every drawn mission its own rules: low or high gravity, robots plated against
  blast, fire or bullets, robots 15% faster or half size, robots that can crit,
  tanks carrying an invulnerable sentry, waves that refill the moment a robot
  dies, aim that wanders off the crosshair, ground with no grip, and
  mercenaries who take a quarter more damage. The seed chooses once per mission
  and keeps the combination, so a mission you return to plays the same twists
  again. One or two a mission by default, and `minimum_mission_modifiers` and
  `maximum_mission_modifiers` set the range.
- **A tracker for spectators.** A web page anybody with the link can open:
  which missions are cleared, what has been checked, what the multiworld has
  unlocked, and the weapon buffs each class carries and at what level. Viewers
  need no room password, the page needs no server of its own, and a live
  tracker refreshes once a minute. The room has to have tracking enabled, and
  a seed made with an older apworld cannot show its starting inventory.
- Test mode hands you the missions you picked. The missions you ticked into
  the pool are unlocked from the start, so `!mission` can switch to any of
  them straight away instead of only the one you start on.
- A wave picker beside Play on the mission list. Pick a wave and the mission
  starts there, with the money the game gives for having won every wave
  before it. It opens on the wave your team got to, so the mission you left
  part way through is one click from where you left it.
- The run remembers the highest wave you reached in every mission, not only
  the one you were on. Going to look at another mission no longer throws the
  first one's progress away.
- The wave restore now waits for the game to agree before it says it worked.
  It used to announce the wave it had asked for, which is why a player could
  read "Restored: Quarry at wave 3 of 6" and be standing in wave one. If the
  jump never takes, the log says so and names the wave the game is actually
  on.

### Weapon buffs

- Skins and decorated weapons draw from the gun they are a copy of. A buff
  rolled for the Holy Mackerel used to belong to the Holy Mackerel alone; it
  is a Bat buff now, and the same goes for the decorated versions of weapons
  you already own. The Self-Aware Beauty Mark counts as Jarate.
- Weapons that are the same gun under different names share one pool. The
  Saxxy, the Conscientious Objector and the other all-class melees draw one
  "All-Class Melee" buff between them instead of one each, and the Bootlegger
  draws what Ali Baba's Wee Booties draws. Fewer duplicate rolls, and every
  roll lands on a weapon that can use it.
- Your own explosions no longer bleed, ignite, milk or Jarate you. The hit
  effects were written where the game could read them on self-inflicted blast
  damage, so rocket jumping with a bleed rocket launcher bled its owner.

### The launcher

- SigMod community missions install themselves. On Linux the launcher
  downloads SigMod, loads it, and the missions that need it come off their
  lock. The Docker image already carried it. On Windows they stay locked:
  SigMod has no Windows build, and porting it is its own project.
- A seed made by a different apworld version says what to do about it. The
  refusal named two numbers and no action, and two people a week apart asked
  what they meant. It names the two ways out now: regenerate the seed with the
  apworld that shipped with this launcher, or run the launcher the seed was
  made for.
- The Rewards toggles say what they switch. "Rewards" is "Grappling Hook", and
  the labels read "enabled" and "disabled" instead of "in the pool" and
  "include cash filler".
- Joining from the launcher works when Steam's relay is in use. The relay
  address sometimes reached the launcher and sometimes only the game's own
  console log, which the launcher reads now.
- Community missions hand their wave icons to joining players over the HTTP
  download. They showed on the machine running the server and nowhere else.

### Fixed

- The Grappling Hook can be turned on. The launcher wrote every option the
  apworld takes except that one, so a run generated from its tf2.yaml never
  had the hook in it and hand-editing the file was the only way in. It is a
  row on the Rewards page now, under Server levers.
- A held Grappling Hook shows on the Unlocks screen. The screen listed four
  kinds of unlock and the hook is a fifth, so it arrived, the plugin turned
  it on, and nothing said so.

### The bots

The mod moves to 0.15.1.

- A robot sent where the navigation mesh has no ground no longer takes the
  wave down with it. The list of walkable tiles around a point came back
  empty and was read as if it had entries. By kelly-cs.
- The medic spends the break healing, so his über is full when the wave
  starts. The charge never built: every wave opened with it empty, and a
  mission got one über where it now gets seven. Measured on Decoy.
- A released über keeps the beam on the player it started on, instead of
  drifting to whoever was nearest when the button went down.
- The medic heals the biggest body, and only a player who calls for him jumps
  the queue. A quiet player standing in spawn used to hold the top of the
  list for a whole wave while the beam landed on whoever was nearby.

## v1.13.0

The launcher has one screen now, and it opens in your browser.

### The new interface

- Double-click the launcher and a browser opens on it. Same screen on Windows
  and on Linux, so a screenshot in Discord is a screenshot of what you have.
- Everything is on it: the connect line, the run and its missions, what the
  multiworld has unlocked, your bot team, the log with a search over it, and
  every setting.
- The log holds twenty thousand lines and stays quick to scroll, and the search
  narrows it as you type. The server command box is under it.
- The mission table sorts on any column, searches, and says why a mission is not
  ready rather than leaving it out.
- Leaving the tab open behind the game costs nothing: it stops listening while
  it is hidden and picks the state back up when you come back to it.
- Closing the tab leaves the server running. On Windows the launcher sits in
  the notification area while it does: click the icon to get the page back,
  right-click it for Quit. Quit on the screen stops it too.
- The settings open when you go to them. There is no button to press first.
- A settings page with sections shows them as tabs across the top, one at a
  time. Bots is Team, Classes, Looks and Loadouts, each a screen rather than
  a scroll; a search still finds a setting whatever tab it is on.
- The Team tab is a grid of six seats with the fill controls above it; the
  Classes tab is one line per class, allowed or not and what it carries.
- Loadouts is a builder: pick the class, a weapon per slot, name it. The
  saved ones are a list with Load and Remove on each line.
- The mission pool is one table, the same one the Play screen has, with a
  tick per mission and the tier beside it. It was drawn twice.
- The console is under the missions on the Play screen, the same width, with
  the search, the command box, Save debug logs and Clear this view. There is
  no Console tab.
- Join is in the header, left of Start, and the Bot Switcher has Apply:
  the seats are a draft, and a lineup built there was gone on the next
  visit because nothing wrote it.
- Star on GitHub, in a footer that stays put.

### Gone

- The old settings window and the terminal interface are both gone. The
  browser is what you get.
- `-tui` and `-web` are gone with them. `-console` still prints the log and
  nothing over it, for a machine with no desktop; `-configure` still asks the
  questions in a terminal.
- New: `-no-browser` prints the address instead of opening one, and `-addr`
  says where to serve.

### If the browser does not open

The address is printed either way. Open it yourself. On WSL the launcher asks
the Windows browser rather than one inside the distribution, where you could
not see it.

### Fixed

- A saved loadout handed to a seat or a class plays that loadout. It played
  stock: the menu wrote the loadout's name where its key belongs. A settings
  file written by 1.12 still works, the name is understood too.

## v1.12.0

Traps, a medal on every mission clear, a Grappling Hook the multiworld can hand
you, maps that download over HTTP, thirty more community missions, and a long
pass over the weapon buffs from Cowser's sheet and kelly-cs's issues. Bots at
2.52.3. The settings screen says what it did with a Save and where it put the
file, and a save stops ending the mission you are four waves into.

### The run

- **Traps.** A `trap_percentage` option puts traps in the pool, one percent by
  default and zero for none. The one that exists soaks the whole team in
  Jarate: ten seconds of 35% more damage taken and no crits, bots included. It
  belongs to the multiworld like any other item, so it is another player
  opening a chest that does it to you. One that arrives between waves waits
  for the next one: Jarate on a team standing at the upgrade station is no
  trap at all.
- **Australium Medals.** `medal_on_clear`, off by default, locks a medal onto
  every mission clear and takes those checks out of the pool. Both goals read
  them: Final Boss asks for the goal mission's own, missionsanity counts them.
- **The Grappling Hook.** Turn on `server_settings` in your YAML and the
  multiworld can hand your team Mannpower's hook for the rest of the run. It is
  useful, never progression: no wave needs it.
- A `community_missions` option keeps every community mission out of a run in
  one line, instead of naming each in `excluded_missions`.
- Thirty community missions that play on a Valve map are on the lists. Every
  mission in the catalogue named one of its own community maps, so Moonlight's
  Trouble in Mann Town on Coaltown and Manntenance on Mannworks were in the
  downloaded assets and on no list. They need their pack for the population
  file and nothing else. A run that keeps every mission draws from 112 at the
  normal floor instead of 82, and the defaults do not move: community missions
  still start excluded. Reported by Glitch.
- Community missions can name a server mod they need. The Docker image carries
  SigMod and loads it with `SRCDS_MODS=sigsegv-mvm`; the seed's `server_mods`
  option draws those missions only for a server that has it. The Windows
  launcher shows them locked: SigMod has no Windows build.
- Wicked Wizardry and Fiefdom Fiasco on Frostwynd are Medieval missions: the
  map keeps only melee and medieval-era weapons, so most weapon slot unlocks do
  nothing there. The launcher says so where you pick missions for the pool,
  where you choose the starting mission, and on the session list, so a run
  does not walk into one by surprise. By kelly-cs.
- A server that restarts goes back to the wave the team had reached. The bridge
  writes the wave down as checks come in, and the plugin reloads the mission and
  jumps forward, never back, when the server comes up again. A crash cost the
  team every wave it had cleared.
- A run that crossed the 1.10 upgrade had every mission it had cleared read as
  "collected", and a missionsanity goal counted none of them. The old state file
  could not say which checks were yours, and they all were. kelly-cs's run.
- The bridge no longer drops the room every second in a multiworld with many
  games. It asked for every game's item names at once and the reply was too big
  to read, so nothing you checked ever reached the room. EMann's bundle showed
  it fifty-nine times.
- Test mode draws its missions at random from the pool, the way a real seed
  does. It took the first ones in the settings list, which looked like a
  randomizer that does not randomize.

### Weapon buffs

- The run's buffs are yours and not the station's. Refunding an upgrade used to
  take them, and they came back as purchased, infinitely refundable upgrade
  levels that stacked again on each refund. They are held apart from MvM's
  shopping now and survive refunds, wave transitions and loadout changes.
- The upgrade station no longer opens the buff window by itself. That window
  took the number keys, which are your weapon slots, for twenty seconds. The
  station lists your loadout's buffs in chat, and `!ap buffs` opens the window
  when you want it.
- Explode on ignite is gone from the pool: on a minigun it ended a wave on its
  own.
- Über on hit is a Medic buff. It used to be offered on mediguns only, which
  never land a hit, so it did nothing wherever it fell. It is on his syringe
  guns, his saws and the Crusader's Crossbow now: 1% a syringe, because they
  land far faster, and 5% everywhere else.
- Ninety-six buff and weapon pairs that did nothing are out of the pool, from
  Cowser's sheet: accuracy on weapons with no spread, reload and clip buffs on
  weapons with no clip, kill buffs on the two jumpers, fire rate on sniper
  rifles, and the like. Existing seeds keep their ids.
- Eight pairs came in from the same sheet. A thrown jar counts as a hit, so
  Jarate and Mad Milk draw ignite and heal on hit. The Gas Passer never fires
  and still gets kills through its afterburn, so it draws the on-kill buffs.
- Projectile penetration is no longer offered on explosives: a rocket that
  penetrates does not explode where it was aimed. Armor piercing is a knife buff
  only, since the game reads it on backstabs and nowhere else.
- Bleed lands one stack per enemy instead of one per hit, which made a syringe
  gun a wave-ender. Melee fire rate stops at +50%: past it the swing outran its
  own animation and hit nothing.
- "No self-inflicted blast damage" and rocket-jump protection work on rocket,
  grenade and stickybomb launchers, and zero self-damage keeps the explosion and
  the blast-jump push. The game reads both on the player, not the weapon, so the
  buff on the weapon never reached them.
- Bleed, milk, Jarate and the marks land on a direct hit with the jar itself.
- A locked weapon slot takes the wearable in it too. A Demoman with a locked
  secondary charged with a shield, and one with a locked primary kept the boots
  and held nothing. Gunboats, Mantreads, the Razorback and the other
  slot-filling wearables follow the same rule.
- Linux servers get the extra arrows and flares. The projectile-count fanout had
  Windows gamedata only, so a Docker server logged both fanouts disabled and
  every extra bolt was a plain clone.
- Extra Crusader's Crossbow bolts heal. The fanout built them as plain arrows,
  which hurt a robot and do nothing for a teammate. They now come off the game's
  own arrow factory with their launcher set, so a bolt that reaches a teammate
  heals him, penetration carries over, and the weapon still fires one sound
  however many bolts leave it. By kelly-cs.
- Heal on hit and speed boost on hit work on Jarate and Mad Milk. The game
  reads both off a weapon attack, and a jar splash is not one, so a soaked
  robot gave the thrower nothing. The plugin pays them now, once per soaked
  enemy.
- The on-kill buffs credit the weapon that killed, not the one in hand. A robot
  that burned down after the Pyro put the Gas Passer away paid the flamethrower's
  buffs, or nothing.
- `sm_ap_buff_slot` puts a test buff on a holstered weapon, a shield or a
  watch, and `tf2ap_buffs_for_defender_bots` lets the bots carry the run's
  buffs so a buff can be measured on a server with no player.
- Movement speed, jump height and health regeneration are passive. They used to
  work only while the weapon that carried them was out, which is not how MvM's
  own class upgrades read; a Scout who rolls movement speed on his bat is faster
  with the scattergun in his hands too, and two weapons carrying the same buff
  add up.
- Recharge speed is off the Gas Passer. It fills the item meter and the buff
  scales the effect bar, which is a different meter, so it never reached it.
- The Flying Guillotine no longer draws clip size, reserve ammo or reload speed.
  It recharges, so it holds no rounds for any of them to act on.
- A server where no buff applies at all says why in its log. The plugin hangs
  the buffs off an attribute provider it attaches through VScript, because
  TF2Utils' equip offset is stale on current Windows TF2, and a server with no
  VScript VM refuses that input. The refusal was dropped and came back a step
  later as the provider not providing attributes, which is a symptom no debug
  bundle could explain. Nine of those lines in four minutes sat behind Likai's
  screenshot.

### Balancing

- Robot health scaling works, and holds. Setting it above 100% left the robots
  at the mission's health and setting it below moved nothing: the number was
  written somewhere the game recomputes a moment later. It is a straight
  multiplier now, 10% to 1000%, applied after the game finishes building the
  robot, read back, and the server log says what each robot ended up worth.

### Getting a server up

- The server hands out maps over HTTP from the machine it runs on, so a friend
  joining without a community map downloads it instead of watching Transmission
  reach the end and start over. On the local network there is nothing to set up.
  A friend joining from outside needs the download port forwarded and your
  public address in `SRCDS_DOWNLOADURL`, or a Tailscale Funnel: the launcher
  publishes the download through it when you have one, on Windows and Linux and
  with the Compose stack. Without either they fall back to the old transfer.
- The launcher says in its log when a newer release is out, with the link. It
  still does not replace itself: download the new exe and run it.
- A server whose Metamod or SourceMod went missing is repaired on the next start
  instead of playing stock Mann vs Machine with every setting ignored, and a
  debug bundle says so when it happened.
- The Docker server plays the randomiser again. The image shipped without the
  plugin's gamedata, so the plugin refused to load and the server came up as
  plain Mann vs Machine with nothing saying so.
- The Docker server writes its player file with the same code as the launcher,
  so both carry the reward importance options and the same defaults.
  `MVM_COMMUNITY_MISSIONS` reaches the launcher's settings too.
- The game server starts with `-debug`, so a crash leaves a stack in
  `debug.log` even when Breakpad never started, and the bundle carries it. Two
  bundles arrived with an access violation and nothing naming the function.
- A debug bundle only carries the game server's own crash dumps, not every
  program's on the machine. Two bundles carried GameBar and a Tony Hawk game.
- The community mission installer skips the missions a pack cannot support.

### The launcher

- An **Unlocks** tab, window and terminal, lists what the multiworld has handed
  the run: classes, weapon slots, missions and weapon buffs, with the level a
  repeated buff reached.
- The install folder is a setting, with Browse beside it, instead of a choice
  the installer screen made once and never offered again. Changing it does not
  move what is already downloaded, and the help says so.
- **Show the settings file** opens the folder holding `config.json`, which is
  under the OS config directory and not in the install folder. Two players went
  looking for it in the install folder, found nothing, and concluded that
  nothing had saved.
- A Save that is refused says so where you are looking: a message box, and the
  page holding the reason. A refusal used to be a small label beside a box on
  another tab, so a player could set a login token three times over while the
  room address two pages away was what stopped every one of them. Every refusal
  is in the log now as well, so a debug bundle carries it.
- A failed write keeps the screen up with your answers on it and names the path
  it could not write.
- Save asks the room whether it is listening, and says which of "live", "not
  there" or "did not answer" it got. None of the three refuses the save: a room
  you have not made yet is not a mistake. Start used to be where you found out,
  with "AP_PORT is not set", and one player spent seventeen minutes on that.
- The mission count you may ask for is the number the seed can actually draw.
  The ceiling counted every mission in the catalogue, including the community
  ones a fresh settings file leaves out and any needing a server mod you do not
  load, so the tier rows offered 82 missions at the normal floor where the run
  held 29. Both numbers now count what the generator counts, and the generator
  says in its log when the pool is smaller than the ask. Reported by [-SAM-].
- The settings window is laid out like a settings window again. Moving the rows
  into one declaration flattened it: the Bots page became one list of
  thirty-nine rows where it had been four sub-tabs, every button took a line
  with its own name in the label column beside it, Debug logs, Repair and Reset
  settings left the bottom bar for a page, and a page of three settings spread
  its rows down the window instead of leaving the space at the bottom. The
  window nests, groups and lays out again: Bots holds Team, Classes, Looks and
  Loadouts; the seats and the classes sit two to a line; buttons share a line;
  numbers are the width of a number rather than of the window; and the three
  buttons that are about the launcher are along the bottom where they were.
- SourceMod's one automatic restart waits for the settings window to close. The
  updater fetches gamedata and asks for a restart whenever it likes, so the
  server came round in the middle of an edit and looked like the edit had done
  it: Sirknobbles reported changing the bot team as restarting the server, but
  only when the window had been open long enough. A save that restarts anyway
  loads the new gamedata itself.
- A save restarts the server only for a setting the server reads at startup.
  Saving a mission count ended the mission the team was four waves into and
  brought the server back exactly as it was. A **Restart on save** tick sits
  beside Save and shows only when what is on screen would need one, never with
  the server down, so a lineup changed between waves costs the run nothing.
  Reported by Likai.
- A run of nothing but community missions has missions again. The community
  switch was absent from every settings file written before the option existed,
  so it loaded as off, and no screen had a row to turn it back on once it was
  the field that empties the pool. It has one on the Missions page, and a
  community mission's Compatibility column reads "community missions are off"
  while the switch is, so the table stops looking full over an empty pool.
  Reported by Peppy.
- A run selection the launcher refuses is reported as refused. The button
  dropped the error naming the empty pool and printed "Selection is valid: 0
  eligible mission(s) provide 0 checks for 0 unlocks" over it. From Peppy's
  screenshot.
- The mission table sorts on its headers, and a row it was told about redraws.
  The "In the pool" column kept its old answer until the mouse passed over the
  line, and a tick the launcher had refused was drawn as taken. The headers
  looked pressable and did nothing, which with eighty-five missions makes
  finding one a nightmare. The table opens sorted by mission name. Reported by
  EZKSupernova.
- The settings window opens on a number saved outside the bounds it has now.
  The mission pool can shrink under a count already saved, and the window
  refuses to build at all for one number box starting outside its range, so
  nothing opened. The stale value sits on the nearest bound, where you can see
  it and save it. By kelly-cs.
- Terminal tabs that list more than fits scroll, and the last line counts what
  is off the screen. The Unlocks and Bot Switcher lists were cut at the body
  height and said nothing about the rest.
- `-web` opts into an experimental browser interface, the same screens on
  Windows and Linux, served on loopback by the launcher itself. The window and
  the terminal interface stay the defaults until a later cutover. By kelly-cs.

### The bots

The mod moves from 2.39.0 to 2.52.3.

- The medic answers a player who calls for him. A human pressed the call and
  the bot medic carried on healing whichever bot it had picked; reported by
  Cowser and by Peppy. Measured with a stand-in player calling on Decoy: the
  caller held the beam up to a quarter of the time against 2% before, every
  wave cleared either way.
- The engineer's break is a plan, not a walk. Every engineer claims his four
  spots before anybody moves, jumps to each rather than walking, and never
  builds on another engineer's ground. On Decoy with two engineers the
  teleporter entrance stands at 21 to 30 seconds of the break against 37 to 45,
  and the exit at 26 to 38 against 50 to 58. The cost is on the record: the
  wrench is what applies a level, and an engineer who jumps swings it less, so
  the sentry sits at level three in 76% of samples rather than 85%.
- Two engineers who pick the same nest no longer land inside each other. The
  sentry was the one building whose jump onto its spot skipped the check for
  room to stand, so a pair holding one nest were put on the same coordinate and
  neither could move until the stuck recovery threw one of them out. Reported by
  Cowser on Mannworks, where five engineers share four nest spots and the pair
  spent the whole break wedged with no sentry between them.
- A teleporter exit puts you down on a side with ground beside it. The exit
  ring never looked down, so a player taking the teleporter could land where the
  engineer never stood.
- A bot's loadout slot is emptied only for an item the game can give. A Heavy
  joined RED on Rottenburg with no weapon at all.
- Halloween cosmetics are out of the bots' hat pool. The game refuses to attach
  a holiday item out of season, and the refusal threw an error that cost the
  bot the rest of its cosmetics.
- A bot moved out of spawn is put where a standing player fits. The recovery
  teleported onto anything and asked for a route in the same tick.
- Five map spots stand on the ground. Four nest and dispenser spots on
  Mannhattan and Mannworks sat in holes beside the mesh, and Rottenburg's fourth
  sniper spot was written 213 units above it, so every side was refused.
- A bot taking a seat no longer inherits the last bot's answers: the health,
  ammo and revive searches around where the previous bot stood cleared with the
  seat, so a new bot stops walking to a pack it cannot reach.
- The server no longer crashes on a map change after a custom loadout file
  was loaded. Two handles the mod had closed were read back after the change.
- The final wave's send-off is a taunt. The last wave neither danced nor
  kicked.

Bigrock's rock-top nest and teleporter spots are still built at the foot of the
rock: the engineer cannot climb, and the building goes where he stands. Open.

## v1.11.0

Change the bot team mid-run, build your own loadouts, scale robot health, and
weapon buffs in place of spare cash. Two of them came from kelly-cs.

### Change the team without ending the run

- A **Bot Switcher** tab, between the mission list and the log, in the window
  and in the terminal. It shows what each seat plays and carries, and one button
  hands a new team to the server.
- Saving a bot setting no longer restarts the server. Every other setting still
  does, because the game reads those once at startup.
- `!ap bots` opens the same team in the game, seat then class.
- The bots keep the money they earned across a switch. Measured: every one came
  back holding its full 400 credits and spent it within seconds.
- A change made during a wave waits for the break. A bot pulled out mid-wave
  drops its buildings, and its replacement starts again from spawn.

### Build your own loadouts

- A **Loadouts** page in the settings. Pick a class, pick a weapon per slot,
  name it, save it. The saved ones join every weapon menu for that class.
- 251 weapons, taken from what the bot mod can hand out and named from the
  game's own files.
- Remove a loadout and any seat still naming it plays stock. The team survives.

### Balancing

- A **Balancing** page with a robot health scale, and the old Balancing page is
  **Rewards**. What the robots are worth and what the run hands out are
  different questions.
- Robot health is what the robots keep with one player on RED, rising back to
  100 at six. A full team plays the mission as Valve wrote it. It starts at 100,
  which changes nothing.
- At 50 the bots killed 108 robots a wave, against 52 to 64 unscaled, and
  cleared three waves in eight where twenty four attempts cleared none.

### Weapon buffs, by kelly-cs

- Spare checks award stackable weapon buffs instead of cash. Cash is spent
  inside the wave it arrives in; a buff lasts the run.
- Extra rockets, grenades, stickies, arrows, bolts, flares, syringes, energy
  projectiles and thrown jars.
- On by default, at three quarters of the spare checks.
- Test mode hands out weapon buffs early enough to try them. They were last in a
  list of seventeen thousand, so the first one arrived about sixty waves in.

### Community maps, by kelly-cs

- Optional community mission support. Nothing bundles third-party assets: the
  launcher fetches what you choose, or imports a ZIP you already have.
- The list keeps Valve and community missions apart.
- A map with no bot navigation file shows red, and the list refuses it. The
  bots cannot walk it.
- The launcher checks the missions you picked can satisfy the run before it
  generates anything.
- Offer every stock-compatible mission found for the 19 supported community
  maps. Mission source checks now also verify wave, tank, and giant metadata so
  Archipelago cannot generate an objective the selected mission lacks.

### The bots

- Snipers with the stock rifle no longer stand where they shopped for the whole
  mission. They were never given the sniper role, so the game never sent them to
  a perch: a sniper carrying any other rifle worked, which is what made it look
  like a weapon problem. Only servers with custom loadouts on were affected.
- A bot hat the class cannot wear is turned down instead of throwing, which used
  to cost that bot the rest of its cosmetics.
- Engineers no longer freeze in one spot and build nothing. A bot stuck three
  times in the same place is moved onto ground around him he can walk on. On
  Mannworks that took a run from crashing before the first wave to four clean
  waves, and a team that cleared none of six waves cleared six of nine.
- The wave-start stutter on Mannhattan is gone with it. The worst frame in a
  wave fell from 1833 ms to 141 ms, because a bot that cannot move asks the game
  for a route every frame.
- The mod moves a defender that cannot leave spawn to the objective. Some
  community maps have navigation the bots will not use, and it happens on Valve
  maps too. Also by kelly-cs.
- One press of the build button makes one building. Engineers built a second
  dispenser and a second sentry.
- A sentry buster blowing itself up is no longer counted as a giant.
- `sm_redbots_feature_watch_idle_bots` sets the switch it names. It set a
  different one before.

Stock snipers standing at the upgrade bench is still open. Nothing in this
release changes it.

### Under it

- The settings window opens on a settings file written before the Balancing
  page existed. It refused to open at all, because the robot health scale read
  back as zero and the window will not take a number below ten.
- The minus button at the upgrade station gives back credits that came from a
  cash bundle. Taking one upgrade back used to leave you with less than you
  started with.
- A crash in the launcher's background work no longer closes the window and
  stops the server you are playing on. It is caught and written to the log a
  debug bundle carries.
- The window and the terminal offer the same tabs and write the same settings.

## v1.10.0

Most of this release is what the 1.9.0 play-tests found.

### The run

- Items and locations print by name in chat. They arrived as bare numbers: the
  bridge never asked the multiworld for the names.
- The mission list says what you cleared. Another world running `!collect` on
  its goal marks every check it still holds. Those missions then read as cleared
  here, though nobody on this server played them. Yours now read "cleared", the
  room's read "collected".
- A sentry buster blowing itself up is no longer the giant kill. It carries the
  giant flag, so it took the mission's giant check with it and the real giant
  that died later reported nothing.
- Cash Bundles pay the bots as well as the players, and a bot that rejoins keeps
  what its bundles paid for.

### Getting a server up

- The connect address is on the Session tab of the terminal interface, as the
  full `connect` command rather than a bare address. It was only reachable by
  pressing C on the log tab.
- Joining from the launcher aims at the address your machine actually answers
  on. Docker, WSL and virtual machines each leave one behind, and the link took
  whichever came first. That gives "connection failed after 4 retries" and a
  stall at two bars. The LAN tab of the server browser finds the same server
  first try.
- Settings changed in the terminal interface reach the server. `server.cfg` was
  written once when the launcher started and never again. The launcher saved a
  class you unticked mid-session, showed it unticked, and played it anyway.
- The launcher says whether Steam's item server answered. Without it everybody
  plays full stock and nothing said why, which is indistinguishable from a setup
  step you missed.
- The Docker quick start downloads a file that exists. The documented URL
  returned 404 on every release, and `curl` fails silently, so you got no `.env`
  and hit a missing password several steps later.
- A crash reads as a crash. The launcher reported one as "bridge stopping",
  which is its own shutdown message, so the component that had not failed was
  the one people reported.

### The bots

- RED holds its size. One request for six bots added nine at mission load, and
  they stayed until somebody restarted a wave. A surplus now goes within three
  seconds. A player who reconnects mid-mission gets their seat back instead of
  spectator.
- A class you untick is never drawn. A lineup that named some seats left the
  rest to the mod, blacklist and all, and an unticked Spy walked onto RED.
- One build press makes one building. The engineer asked whether he had
  succeeded in the same frame as he pressed, and the game answers on the next
  one. So he pressed again, and two dispensers stood under one engineer. Four
  waves of the test bed counted eighteen of those before, and none after.
- Engineers stop buying the disposable mini sentry. Measured over six waves of
  Decoy: defender deaths per wave ran 0 to 10 without it and 11 to 17 with it,
  and sentries lost doubled. It is a switch, `sm_redbots_feature_engineer_disposable`,
  if you want it back.
- Teleporter exits go outside the blast that kills the sentry. The engineer put
  one 150 units from the nest, and a sentry buster reaches 400. One buster took
  the sentry and the team's forward spawn together.
- Decoy has its sniper spots back. A test removed them and every build since
  carried the gap.
- A bot at the upgrade station follows your ready. It had no readiness of its
  own while shopping, so it never pressed F4 and the wave waited on it. A bot
  mid-taunt readies too.
- Raising the RED team size says what the game allows. Asking for twelve
  silently produced no bots at all, which left fewer than asking for six.

### Under it

- Native Linux servers keep the engine watchdog off. It kills the server when a
  frame takes too long, which fires under load a machine survives otherwise.
- Debug bundles open with what looks wrong: crashes, plugin exceptions, stuck
  bots, and the lines a run repeated most. They also say whether `server.cfg`
  still matches the settings beside it, and where to look when a crash left no
  dump.
- The bots mod is on v2.17.1.

## v1.9.0

Most of this release is the bots.

### The run

- Mission switch works from the Session tab and from `!mission`.
- A refund returns your cash bundles. The plugin counts every credit a bundle
  adds, and puts that count back on top of the refund.
- A lost wave returns the bundles you spent. A balance below zero goes to zero.
- The win comes off the locations this server played. Another player's
  `!collect` checks a mission clear without ending your run.
- Bots take their seats as soon as somebody joins the server. They shop at the
  upgrade station and build before the wave starts.
- Bots ready up only after every player on RED is ready. A server with nobody
  on RED never readies at all.
- Bots fill the seat of a player who leaves between waves, and free a seat for
  a player who arrives.
- Test mode answers as a real room does. `!ap unlock mission` hands over the
  next ticket, `!ap missing` and `!ap checked` reply, and you start on the class
  you asked for.

### The bots

- Bots shoot the Medic first, then the Sniper and the Engineer, then giants.
- Ten runs on Decoy measure 25 defender deaths, against 56 for the same waves
  without the target order.
- A bot with nothing to fight holds the hatch. A bot the nav mesh refuses a path
  steps toward its target.
- Every class has upgrade paths, and a bot refuses an upgrade it cannot use. A
  Phlogistinator Pyro has no use for airblast pushback.
- Bots buy blast, bullet and fire resistance against the wave to come.
  Explosions cause 45 to 60 percent of defender deaths on every map measured.
- Leftover credits stay in the wallet.
- Engineers build on the spot they picked and rebuild it there, one nest inside
  and one out, with the teleporter exits apart.
- Engineers keep the sentry on its spot through the wave, finish the nest they
  start, and give up on a spot they cannot reach.
- A sentry that cannot reach the nest goes down beside the engineer. The
  disposable sentry goes beside the real one on purpose.
- An engineer with no sentry left rides his own teleporter home.
- A map can name which dispenser spot belongs to which nest.
- The game walks the Medic, and the mod points the game's own heal action at the
  biggest body nearby.
- On Coal Town the beam connects in 75 percent of samples, and 72 percent of
  those samples show a Heavy on the end of it.
- A Medic deploys six ubers in a mission, and moves 337 units between samples.
- Medics shop before they follow anybody, hold the wave until the charge is
  full, and spread a wave's credits over more than one upgrade.
- Soldiers and Demomen keep their distance from a tank hull.
- The Demoman rates the stickybomb launcher as no tank weapon, and holds the
  detonator while his own pipes sit on a hull.
- Neither aims at the feet of a robot that stands on them, and the Demoman
  throws a pipe as far as it flies.
- A pipe can leave while the aim still moves, a rocket cannot, which is the
  difference between the two arcs.
- Demomen hold the stickybomb launcher, close to the range their pipes arrive
  at, and put an empty launcher down in a fight.
- The Soldier carries the stock rocket launcher.
- The Pyro walks in instead of parking at shotgun range, and bots pick money off
  the ground.
- Each class carries a named loadout and a seat can name its own, so two
  engineers can hold different weapons.
- A bot draws a random cosmetic, and an unusual effect on it, in two ticks, and
  keeps it for the whole mission.
- Bots wear no war paint. It painted the weapons the upgrade station replaces,
  and killed the server when two engineers finished shopping.
- Buildings go on ground that exists, and the wearable sweep ends.
- Nav mesh searches for health, ammo and revive markers run less than once a
  frame.
- The plugin builds the shopping list once a session, and hands out hats one bot
  at a time.
- `sm_redbots_feature_<name>` switches any of this off one feature at a time,
  which is how two ways of playing get compared.

### The window

- The run is the first tab and the log is the second.
- A mission button names the mission rather than its pop file, and a locked
  mission says so on the button.
- A press says what loads until the server confirms it.
- The Bots tab is three pages: Team, Classes and Looks.
- Each seat is one line of what it plays and what it holds, and teams save by
  name. The class pool is two ticks to a line.
- The title bar says which build it is, which matters when several carry the
  same version.
- Save leaves a stopped server stopped. Start is the button that starts the
  server.
- Numbers read from the left, columns line up, and no row runs under the
  scrollbar.
- Over Steam the join line reads `Steam public IP:`, and the Join button goes to
  that address.
- The page that issues a login token is a link on the tab that needs one, with
  app id 440 and a memo.
- The terminal launcher follows the window: the run first, the same seats and
  saved teams, the same Steam address.

### When something goes wrong

- Debug logs hold the game server's own console log, the last launcher log and
  the crash dumps.
- They also hold what the bridge says about the run, and which defender bots
  played.
- The plugin writes what it does to the console and the SourceMod log by
  default.
- The plugin writes down every purchase and sale at the upgrade station, players
  and bots, with the credits held after.
- The bots name the upgrade they bought. `mvm_upgrades.txt` has 64 entries and
  the game loads 63, because one entry carries a comment marker.
- A defender bot version bump builds that version, on any machine and in CI.

## v1.8.2

- `tf2ap.exe` carries the header checksum Windows expects. The Go linker leaves
  it at zero, and zero is one of the things a scanner counts against a file.
- The exe says out loud that it never wants the administrator prompt, instead of
  leaving it to the default.
- Every release now attaches a signed record of which commit and which workflow
  built each binary. `gh attestation verify tf2ap.exe --repo m-this/tf2-archipelago`
  checks a file you already downloaded.

## v1.8.1

- These notes now link the VirusTotal report for `tf2ap.exe` and for the Linux
  binary. The scan already ran on the last two releases, but the links never
  reached the notes.
- The warning about `tf2ap.exe` on the front page is two sentences instead of
  nine, and sits under the download buttons where you meet it.

## v1.8.0

- Medics keep the medigun out. Every robot they could see used to pull them
  onto the syringe gun, which drops the heal and stops the charge building.
- Scouts double jump most of the time, and the second jump goes the other way,
  which is harder to shoot than one long arc.
- Bots that are hurt or low on ammo hold the bomb from a friendly dispenser
  instead of walking off to find a health pack.
- Engineers rate dispenser range far higher and buy it early, now that the
  whole team stands in it.

## v1.7.0

- The server no longer freezes at the end of a wave. Every engineer worked out
  where to move its nest in the same frame; they take turns now.
- Engineers stop holding wave one's nest for the whole mission, and upgrade it
  instead.
- Engineers no longer move their nest between waves. It crashed the server at
  every wave transition, so it is off until it works.
  `sm_redbots_manager_engineer_nest_relocate 1` turns it back on.
- A bot that is hurt or low on ammo guards the bomb from a dispenser beside it,
  instead of walking off to find a health pack.
- `tf2ap.exe` carries an icon and says what it is in its file properties.
- Windows still warns about `tf2ap.exe`. The install guide says why, and every
  release now publishes `SHA256SUMS` and links a VirusTotal report so you can
  check what you downloaded.

## v1.6.0

- The defender bots know six maps by hand: Bigrock, Coal Town, Decoy,
  Mannworks, Mannhattan and Rottenburg. Somebody flew each one and stood on
  every spot, so engineers build where a player would build.
- Engineers move their nest between waves when a better spot opens up, instead
  of holding one place all mission.
- Engineers put the dispenser where it was placed by hand, and keep out of each
  other's way.
- On Rottenburg engineers stay off the tank's path on a tank wave, and use the
  platform spot that only works when a tank is rolling.
- Engineers no longer rebuild a level 3 from nothing every wave when they were
  not going to move anyway.

## v1.5.0

- New defender bots. They evade sentry busters, spy check, deploy the medigun
  by what it is instead of by panic, lay and detonate sticky traps, jump as
  Scout, and aim rockets at the ground when the splash pays.
- Engineers buy for their primary and secondary, and pull the wrangler for the
  shield rather than only for the reach.
- Bots stop walking backwards to ride a teleporter that was not worth it.
- A server with no login token now runs on LAN, which is what it is.
- The launcher says where the server is when Join does nothing.
- The Bots tab scrolls on a short window.

## v1.4.0

- The launcher runs in a terminal with the same tabs as the window, for a
  machine with no desktop.

## v1.3.6

- Stop now waits for the half of the server that holds the ports, so starting
  again straight after works.

## v1.3.5

- Screenshots in the setup guides. Nothing in the game changed.

## v1.3.4

- New defender bots: engineers pick a nest near the hatch.
- Screenshots of the launcher in the guides.

## v1.3.3

- A server set to LAN refused the whole network, including the machines meant
  to reach it. Fixed.
- Giving a login token now means the server is meant to be joined from
  somewhere else, and it is set up that way.

## v1.3.2

- New defender bots: the team you name is the team you get, no phantom
  canteens, no afterburn.
- The plugin says which slot a bot bought an upgrade in.

## v1.3.1

- A native Linux launcher.
- The server runs with a console, binds every interface, and takes its children
  down with it.

## v1.3.0

- The first giant and the first tank of a mission are checks.
- Each class's own first weapon slot opens.
- Choose the mission and the class you start on, and exclude missions you do
  not want in the run.
- Reach the internet over Steam's relay, with no port to open.
- A Join button that puts you on the right server.
- Bots play the loadout they were given, buy damage instead of buying at
  random, and keep your seat on RED.
- Name the classes the bots fill RED with.
- A Cash Bundle now pays where the money survives.
- The run counts the waves the team lost.

## v1.2.0

- Death Link. Dying takes the rest of the multiworld with you, and theirs takes
  you.

## v1.1.0

- A Windows launcher: one file, no Docker, it installs the rest.
- Generate the seed from the launcher.
- Pick the map and the shape of the run from lists.
- Defender bots ship with the server and stay between waves.
- Play the whole stack without an Archipelago room, to try it.

## v1.0.0

First release.

- Mann vs Machine as an Archipelago randomiser: missions and waves are checks,
  and your weapons start locked and arrive as items.
- Chat in game talks to the multiworld.
- A Docker stack that runs the server, the bridge and the game.

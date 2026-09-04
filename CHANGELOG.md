# Changes

What each release changes, for somebody who plays the game. The workflow in
`.github/workflows/release.yml` reads the section matching the tag and puts it
in the release notes, so this file is the only place to write it.

## Unreleased

- Robot health scaling works. Setting it above 100% left the robots at the
  health the mission gives them, and setting it below moved nothing either:
  the server wrote the number somewhere the game recomputes a moment later.
  The server log now says what each scaled robot ended up worth.
- Archipelago weapon buffs now stay active when players refund upgrades at an
  MvM station. They no longer appear as purchased, infinitely refundable MvM
  upgrade levels or stack again after repeated refunds.

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

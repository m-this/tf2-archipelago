package form

import (
	"fmt"
	"slices"
	"strings"

	"github.com/m-this/tf2-archipelago/gamedata"
	"github.com/m-this/tf2-archipelago/launcher/internal/runshape"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

/*
	The pages that describe the run: what it draws from and how it is reached

The IDs are stable. The browser sends them over HTTP and a Change names its
row, so renaming one is a
protocol change rather than a rename.
*/
func runSpecs(s State, env Env) []Spec {
	specs := playerSpecs(s, env)
	specs = append(specs, rewardSpecs()...)
	specs = append(specs, balancingSpecs()...)
	specs = append(specs, missionSpecs(s, env)...)
	specs = append(specs, roomSpecs()...)
	specs = append(specs, serverSpecs()...)
	specs = append(specs, networkingSpecs()...)
	return specs
}

func playerSpecs(s State, env Env) []Spec {
	/* The tiers describe the pool as the settings stand when the page is
	   built, which is the same pool the ceiling below caps against. Turning a
	   mission on is a Missions-page change and the labels catch up when the
	   page is built again. */
	poolSettings := s.Settings
	poolSettings.SrcdsMods = activeServerMods(s, env)
	tiers := runshape.Tiers(settings.MissionPool(poolSettings))
	tierValues := make([]string, 0, len(tiers))
	tierLabels := make([]string, 0, len(tiers))
	for _, tier := range tiers {
		tierValues = append(tierValues, tier.Key)
		tierLabels = append(tierLabels, tier.Label())
	}
	goals := runshape.Goals()
	goalValues := make([]string, 0, len(goals))
	goalLabels := make([]string, 0, len(goals))
	for _, goal := range goals {
		goalValues = append(goalValues, goal.Key)
		goalLabels = append(goalLabels, goal.Label())
	}

	const tab = "Player options"
	rows := []Spec{
		choice("run.tier", tab, "Easiest tier",
			"The easiest tier a mission may come from. Harder tiers are always in as well, so the pool shrinks as this rises. Expert leaves four, because Valve made only three expert missions and one haunted one.",
			options(tierValues, tierLabels),
			func(s State) string { return s.Settings.MvmDifficulty },
			func(s State, v string) State {
				s.Settings.MvmDifficulty = v
				return clearIneligibleStart(s)
			}),

		/* The ceiling is the pool the tier leaves, so raising the tier lowers
		   it. Asking for more than the pool holds gives the whole pool anyway,
		   which is why the window capped it here rather than explaining it. */
		boundedNumber("run.mission_count", tab, "Missions used",
			"How many missions this run uses, out of the pool above. Eight is about fifty waves, which is one evening for a team that knows the mode.",
			func(s State, _ Env) (int, int) {
				return 1, max(runshape.MissionsInPool(settings.MissionPool(s.Settings), s.Settings.MvmDifficulty), 1)
			},
			func(s State) int { return s.Settings.MvmMissionCount },
			func(s State, v int) State { s.Settings.MvmMissionCount = v; return s }),

		choice("run.goal", tab, "Goal", "What ends the run.",
			options(goalValues, goalLabels),
			func(s State) string { return s.Settings.MvmGoal },
			func(s State, v string) State { s.Settings.MvmGoal = v; return s }),

		number("run.missionsanity", tab, "Missionsanity share",
			"How much of the run's checks come from waves rather than whole missions, as a percentage. It rounds up, and the Final Boss goal ignores it.",
			10, 100,
			func(s State) int { return s.Settings.MvmMissionsanityPct },
			func(s State, v int) State { s.Settings.MvmMissionsanityPct = v; return s }),
	}
	rows = append(rows, missionModifierSpecs(tab)...)
	rows = append(rows, checkSpecs(tab)...)
	rows = append(rows, toggle("run.deathlink", tab, "Death Link",
		"A lost wave kills every other player in the multiworld who has Death Link on, and their deaths wipe your team.",
		"share deaths",
		func(s State) bool { return s.Settings.MvmDeathLink },
		func(s State, v bool) State { s.Settings.MvmDeathLink = v; return s }))
	return append(rows, runFolderSpecs(tab)...)
}

// checkSpecs are the rows that decide what pays a check besides the waves: the
// clears, the running totals, and the giants and tanks of every wave.
func checkSpecs(tab string) []Spec {
	return []Spec{
		toggle("run.medal", tab, "Australium Medal on clear",
			"Lock a medal of your own onto every mission clear, and read the goal off the medals you hold. It costs the multiworld one check a mission.",
			"lock the clears",
			func(s State) bool { return s.Settings.MvmMedalOnClear },
			func(s State, v bool) State { s.Settings.MvmMedalOnClear = v; return s }),

		toggle("run.caches", tab, "Victory caches",
			"A mission clear is worth checks scaled by its tier: one at normal, two at intermediate, three at advanced, five at expert and haunted. More checks for the multiworld without more missions.",
			"scale the clears",
			func(s State) bool { return s.Settings.MvmVictoryCaches },
			func(s State, v bool) State { s.Settings.MvmVictoryCaches = v; return s }),
		toggle("run.milestones", tab, "Milestone checks",
			"Checks for running totals over the whole run: robots, giants and tanks destroyed, whatever mission they fell in. Something to grind when a wave will not fall.",
			"count the kills",
			func(s State) bool { return s.Settings.MvmMilestoneChecks },
			func(s State, v bool) State { s.Settings.MvmMilestoneChecks = v; return s }),
		choice("run.class_slots", tab, "Weapon slots per class",
			"Each class earns its own two slots instead of one item opening a slot for everybody. Its first slot comes free with the class: the Medigun for a Medic, the Knife for a Spy. Eighteen items in the pool where there were three, so a run with few missions may not fit them.",
			options(
				[]string{
					string(settings.ClassSlotsOff),
					string(settings.ClassSlotsProgressive),
					string(settings.ClassSlotsAnyOrder),
				},
				[]string{
					"off: one slot for everybody",
					"per class, in the class's own order",
					"per class, found in any order",
				}),
			func(s State) string { return string(s.Settings.MvmClassWeaponSlots.OrOff()) },
			func(s State, v string) State {
				s.Settings.MvmClassWeaponSlots = settings.ClassSlots(v).OrOff()
				return s
			}),
		toggle("run.giantsanity", tab, "Giantsanity",
			"A check for every giant of every wave, beside the mission's first. The counts come out of Valve's own mission files; community missions pay only their first. Empire Escalation alone holds eighty-two.",
			"every giant",
			func(s State) bool { return s.Settings.MvmGiantsanity },
			func(s State, v bool) State { s.Settings.MvmGiantsanity = v; return s }),

		toggle("run.tanksanity", tab, "Tanksanity",
			"A check for every tank of every wave, beside the mission's first. Cataclysm holds eleven.",
			"every tank",
			func(s State) bool { return s.Settings.MvmTanksanity },
			func(s State, v bool) State { s.Settings.MvmTanksanity = v; return s }),
	}
}

func missionModifierSpecs(tab string) []Spec {
	return []Spec{
		toggle("run.mission_modifiers", tab, "Mission modifiers",
			"Give every mission a seed-generated combination of environmental, robot, player, and wave modifiers. A mission keeps the same combination when replayed.",
			"modify missions",
			func(s State) bool { return s.Settings.MvmMissionModifiers },
			func(s State, v bool) State { s.Settings.MvmMissionModifiers = v; return s }),
		boundedNumber("run.modifier_min", tab, "Minimum modifiers",
			"Fewest modifiers assigned to each mission when mission modifiers are enabled.",
			func(s State, _ Env) (int, int) { return 0, max(s.Settings.MvmModifierMax, 0) },
			func(s State) int { return s.Settings.MvmModifierMin },
			func(s State, v int) State { s.Settings.MvmModifierMin = v; return s }),
		boundedNumber("run.modifier_max", tab, "Maximum modifiers",
			"Most compatible modifiers that can stack on one mission.",
			func(s State, _ Env) (int, int) { return max(s.Settings.MvmModifierMin, 0), 3 },
			func(s State) int { return s.Settings.MvmModifierMax },
			func(s State, v int) State { s.Settings.MvmModifierMax = v; return s }),
	}
}

/*
	runFolderSpecs are where things are, and the four buttons for them.

Separate from the run's own options above because they answer a different
question. Those say what the seed holds; these say where it lands and what to
press once it does.
*/
func runFolderSpecs(tab string) []Spec {
	return []Spec{
		/* The 14 GB lives here, and until now nothing on any page said where
		   that was or let anybody move it. A player on Discord went looking
		   for the launcher's config in this folder, could not find it, and
		   concluded nothing saved at all: the config is under the OS's own
		   config directory and this is somewhere else entirely. Showing the
		   folder is half of telling them that. */
		folder("run.install_root", tab, "Install folder",
			"Where the game files, SourceMod, steamcmd, the player file and the run's state live. "+
				"Changing it does not move what is already there: the next Start installs into the new folder from scratch, "+
				"and the old one stays where it is until you delete it. The launcher's own settings are not in here.",
			"",
			func(s State) string { return s.Settings.InstallRoot },
			func(s State, v string) State { s.Settings.InstallRoot = trim(v); return s }),

		folder("run.app_dir", tab, "Archipelago app",
			"Where the Archipelago app is installed. On Linux this may be the AppImage itself. Blank searches PATH and common install, application and download locations.",
			"",
			func(s State) string { return s.Settings.ArchipelagoDir },
			func(s State, v string) State { s.Settings.ArchipelagoDir = trim(v); return s }),

		press("run.generate", tab, "Generate seed",
			"Make and download the seed archive through the browser. Upload that archive at archipelago.gg/uploads to open a room."),
		press("run.open_player_file", tab, "Open tf2.yaml",
			"Write the player file from what is on screen, then show it in the browser."),
		press("run.open_folder", tab, "Browse install files",
			"Browse the folder above through this private launcher page: the game files, player file, log and run state."),

		/* Asked in as many words on Discord, by somebody who had gone looking
		   in the install folder and found nothing. The settings are not there:
		   they are one file under the OS's own config directory, and until now
		   nothing in the launcher said so or would show it. */
		press("run.open_settings_file", tab, "Show the settings file",
			"Show config.json in the browser. It holds everything on these pages and is not in the install folder above."),
	}
}

func rewardSpecs() []Spec {
	return []Spec{
		importance("rewards.mission_tickets", "Mission tickets",
			"Required tickets gate deployment to each mission. Useful tickets leave all drawn missions available but still appear as rewards. Disabled unlocks all drawn missions from the start and removes their tickets from rewards.",
			"Disabled - All Unlocked",
			func(s State) string { return s.Settings.MvmMissionTicketImportance },
			func(s State, v string) State { s.Settings.MvmMissionTicketImportance = v; return s }),

		importance("rewards.class_unlocks", "Class unlocks",
			"Required classes satisfy mission-tier roster checks. Useful classes only expand the playable roster. Disabled makes every class playable from the start and removes class unlock rewards.",
			"Disabled - All Unlocked",
			func(s State) string { return s.Settings.MvmClassUnlockImportance },
			func(s State, v string) State { s.Settings.MvmClassUnlockImportance = v; return s }),

		importance("rewards.weapon_slots", "Weapon slots",
			"Required slots satisfy mission-tier loadout checks. Useful slots only expand the available loadouts. Disabled opens every slot from the start and removes slot unlock rewards, including per-class slots.",
			"Disabled - All Unlocked",
			func(s State) string { return s.Settings.MvmWeaponSlotImportance },
			func(s State, v string) State { s.Settings.MvmWeaponSlotImportance = v; return s }),

		importance("rewards.weapon_buffs", "Weapon buffs",
			"Required buffs gate increasingly difficult tiers by total buff count. Useful buffs remain optional power-ups. Disabled awards no buffs and fills spare checks with cash.",
			"Disabled - No Buffs",
			func(s State) string { return s.Settings.MvmWeaponBuffImportance },
			func(s State, v string) State { s.Settings.MvmWeaponBuffImportance = v; return s }),

		twoWayToggle("rewards.cash", "Rewards", "Cash rewards",
			"When enabled, spare checks can award temporary MvM credits. When disabled, every spare check awards a persistent weapon buff unless weapon buffs are disabled, in which case spare checks award cash.",
			"enabled", "disabled",
			func(s State) bool { return s.Settings.MvmCashRewards },
			func(s State, v bool) State { s.Settings.MvmCashRewards = v; return s }),

		number("rewards.buff_share", "Rewards", "Buff share",
			"Percent of spare checks that award buffs when cash rewards are enabled. The remainder award cash. Ignored when weapon buffs are disabled.",
			0, 100,
			func(s State) int { return s.Settings.MvmWeaponBuffPct },
			func(s State, v int) State { s.Settings.MvmWeaponBuffPct = v; return s }),

		number("rewards.buff_stack_chance", "Rewards", "Buff stack chance",
			"Chance that another buff reward adds a level to a numeric buff already in the seed. Toggle effects never repeat. Ignored when weapon buffs are disabled.",
			0, 100,
			func(s State) int { return s.Settings.MvmWeaponBuffStackChance },
			func(s State, v int) State { s.Settings.MvmWeaponBuffStackChance = v; return s }),

		number("rewards.traps", "Rewards", "Traps (%)",
			"Percent of spare checks that hold a trap instead of a reward. A trap is an item another player finds and your team pays for, such as Jarate on everyone. Zero leaves them out of the seed.",
			0, 100,
			func(s State) int { return s.Settings.MvmTrapPct },
			func(s State, v int) State { s.Settings.MvmTrapPct = v; return s }),

		twoWayToggle("rewards.server_settings", "Rewards", "Grappling Hook",
			"When enabled, put the Grappling Hook in the item pool. Whoever finds it turns Mannpower's hook on for everybody for the rest of the run. It costs one check, like a trap.",
			"enabled", "disabled",
			func(s State) bool { return s.Settings.MvmServerSettings },
			func(s State, v bool) State { s.Settings.MvmServerSettings = v; return s }),
	}
}

func balancingSpecs() []Spec {
	return []Spec{
		number("balancing.robot_health", "Balancing", "Robot health (%)",
			"Direct health multiplier for every robot, from 10% to 1000%.",
			settings.RobotHealthPercentMin, settings.RobotHealthPercentMax,
			func(s State) int { return s.Settings.SrcdsBluHealthPct },
			func(s State, v int) State { s.Settings.SrcdsBluHealthPct = v; return s }),
	}
}

/*
	missionSpecs is the pool, and it is a different shape from the other pages

Every other page has a fixed list of rows. This one has a row per mission, and
which missions exist depends on which community asset packs have a valid ZIP on
disk. That is why Specs is a function of the state rather than a package
variable: ticking a pack adds twenty rows below.
*/
func missionSpecs(s State, env Env) []Spec {
	const tab = "Missions"

	starts := runshape.StartMissionChoicesForPacksAndMods(env.CommunityAvailable, activeServerMods(s, env))
	startValues := make([]string, 0, len(starts))
	startLabels := make([]string, 0, len(starts))
	for _, start := range starts {
		startValues = append(startValues, start.PopFile)
		startLabels = append(startLabels, start.Label)
	}

	/* The menu shows the draw first and then the nine mercenaries, and the
	   draw is the empty name: it is what the apworld's option takes for "the
	   run picks one". So the labels are what StartClassChoices returns and the
	   values are the same with the draw's label replaced by nothing. */
	classLabels := runshape.StartClassChoices()
	classValues := append([]string{""}, classLabels[1:]...)

	specs := []Spec{
		packSpec("missions.potato", "Potato Archive", settings.CommunityPackPotato,
			"Select archive-assets.zip for the explicit download action and for installation when the local ZIP is valid."),
		packSpec("missions.moonlight", "Moonlight Archive", settings.CommunityPackMoonlight,
			"Select mlarchive-assets.zip for the explicit download action and for installation when the local ZIP is valid."),

		/* The generator's own switch for community missions, which is a
		   different question from which packs to install: the packs put the
		   files on disk, this decides whether a seed may draw one at all. */
		toggle("missions.community", tab, "Community missions",
			"Whether a seed may draw a community mission at all. Turning this off removes every community mission from the pool.",
			"let the seed draw them",
			func(s State) bool { return s.Settings.MvmCommunityMissions },
			func(s State, v bool) State {
				s.Settings.MvmCommunityMissions = v
				if !v {
					s = excludePoolMissions(s, func(m gamedata.Mission) bool { return gamedata.IsCommunityMission(m.ID) })
				}
				return clearIneligibleStart(s)
			}),

		serverModSpec("sigsegv-mvm", "SigMod", env),
	}
	specs = append(specs, missionSetupActions(env)...)
	specs = append(specs,
		/* One control for both the seed and the server: the run begins on this
		   mission and srcds boots on it, which is the only way the two cannot
		   disagree. Picking one also puts it back in the pool and turns on the
		   pack it belongs to, because starting on a mission the seed may not
		   draw is not a state anybody asked for. */
		openChoice("missions.start", tab, "Start mission",
			"Where the run begins. The seed starts there and the server boots there.",
			func(State, Env) []Option { return options(startValues, startLabels) },
			func(s State) string { return s.Settings.MvmStartMission },
			startMission),

		choice("missions.start_class", tab, "Start class",
			"The mercenary the run starts with. The tier of the start mission decides how many it starts with.",
			options(classValues, classLabels),
			func(s State) string { return s.Settings.MvmStartClass },
			func(s State, v string) State { s.Settings.MvmStartClass = v; return s }),

		// The window has these two under its table. Without them the only way
		// to a pool of three missions is 26 keystrokes down the list.
		press("missions.pool_all", tab, "All in the pool", "Put every mission back in the pool."),
		press("missions.pool_none", tab, "None in the pool", "Leave every mission out, to tick back the few this run is for."),
	)

	for _, mission := range runshape.VisibleMissions(env.CommunityAvailable) {
		specs = append(specs, poolSpec(mission))
	}
	return specs
}

func communityHashMismatchSpec(env Env) Spec {
	spec := confirm("missions.ignore_hash_mismatch", "Missions", "Ignore hash mismatch",
		"Approve only the downloaded ZIPs currently held for a hash mismatch. Approval applies to these exact bytes; a later change requires another approval.",
		"This archive differs from the version used to build the mission catalog. It may cause missing missions, wave 0, or server instability. Approve this exact downloaded file anyway?")
	if len(env.CommunityHashMismatches) == 0 {
		spec.Unavailable = func(State, Env) string { return "download an archive with a hash mismatch before approving it" }
	} else {
		spec.Help += " Awaiting approval: " + strings.Join(env.CommunityHashMismatches, ", ") + "."
	}
	return spec
}

func missionSetupActions(env Env) []Spec {
	const tab = "Missions"
	var actions []Spec
	if !env.ManagedExternally {
		actions = append(actions, serverModInstallSpec())
	}
	return append(actions,
		press("missions.download_packs", tab, "Download Selected Community Assets",
			"Download only the checked full-with-maps community packs. Live progress remains visible on this page. Start never downloads community content."),
		communityHashMismatchSpec(env),
		press("missions.import_assets", tab, "Import local assets",
			"Choose archive-assets.zip and/or mlarchive-assets.zip from this computer. Valid packs are selected and their missions appear below immediately."),
		press("missions.check_selection", tab, "Check Run Selection",
			"Confirm that the eligible mission pool has enough checks to hold every mission, class, and weapon-slot unlock."),
	)
}

func activeServerMods(s State, env Env) []string {
	var active []string
	for _, key := range settings.ServerModKeys(s.Settings) {
		if slices.Contains(env.ServerModsReady, key) {
			active = append(active, key)
		}
	}
	return active
}

/*
serverModSpec is one mod: off, on for the missions that need it, or on always.

Three answers rather than a tick, because installing a mod and loading it are
different questions and this row used to answer both at once. SourceMod loads
an extension because a file sits beside it, so a mod that crashed the server
could not be turned off here at all: the launcher writes that marker from this
answer now, on every start.
*/
func serverModSpec(key, label string, env Env) Spec {
	const off = "off"
	mod, _ := gamedata.ServerModByKey(key)

	help := "Required by missions that use SigMod population extensions, and used by nothing else. Off removes those missions from the pool. Start downloads, verifies and installs the pinned release when it is needed."
	if env.ManagedExternally {
		help = "The Docker image already includes SigMod. Choose off, only when a mission needs it, or always. Save and recreate the containers to apply the choice."
	}
	if !mod.BuildsOn(env.Platform) {
		help = label + " has no " + env.Platform + " server build, so the missions that need it stay out of the pool."
	}

	values := []string{off}
	labels := []string{"off"}
	for _, loading := range settings.ModLoadings() {
		values = append(values, string(loading))
		labels = append(labels, loading.Label())
	}

	/*
		Beta is the Windows row only. Linux and Docker run upstream's own
		release, which has been played for years; Windows runs this project's
		port, which has crashed a real server. The word belongs where the
		difference is, and a Linux host reading "beta" would be misled about
		the build they actually have.
	*/
	shown := label
	if env.Platform == "windows" && mod.BuildsOn(env.Platform) {
		shown = label + " (beta)"
	}

	spec := choice("missions.mod."+key, "Missions", shown, help,
		options(values, labels),
		func(s State) string {
			if !slices.Contains(s.Settings.SrcdsMods, key) {
				return off
			}
			return string(s.Settings.SrcdsModLoading.OrDefault())
		},
		func(s State, v string) State {
			s.Settings.SrcdsMods = slices.DeleteFunc(slices.Clone(s.Settings.SrcdsMods), func(one string) bool { return one == key })
			if v == off {
				s = excludePoolMissions(s, func(m gamedata.Mission) bool { return gamedata.MissionServerMod(m.ID) == key })
				return clearIneligibleStart(s)
			}
			s.Settings.SrcdsMods = append(s.Settings.SrcdsMods, key)
			if loading := settings.ModLoading(v); loading.Valid() {
				s.Settings.SrcdsModLoading = loading
			}
			return clearIneligibleStart(s)
		})

	/*
		The Windows build is this project's port rather than upstream's, and it
		has crashed a real Windows server. A player is owed that before they
		pick it, and "only when a mission needs it" is why the row is offered
		at all rather than hidden.
	*/
	if env.Platform == "windows" && mod.BuildsOn(env.Platform) {
		spec.Warning = "The Windows build is this project's port, and it has crashed a server on a host who ran it. Leave this on \"only when a mission needs it\" unless you are testing it."
	}
	if !mod.BuildsOn(env.Platform) {
		spec.Unavailable = func(State, Env) string {
			return label + " has no " + env.Platform + " server build"
		}
	}
	return spec
}

func excludePoolMissions(s State, match func(gamedata.Mission) bool) State {
	excluded := slices.Clone(s.Settings.MvmExcludedMissions)
	for _, mission := range gamedata.Missions {
		if match(mission) && !slices.Contains(excluded, mission.PopFile) {
			excluded = append(excluded, mission.PopFile)
		}
	}
	s.Settings.MvmExcludedMissions = excluded
	if start, ok := gamedata.MissionByPopFile(s.Settings.SrcdsStartMission); ok && match(start) {
		s.Settings.SrcdsStartMission = settings.Defaults().SrcdsStartMission
	}
	return s
}

func serverModInstallSpec() Spec {
	return press("missions.install_mods", "Missions", "Download / set up selected server mods",
		"Download the verified pinned release and install its SourceMod extension. On a new setup this also installs TF2 and SourceMod; Start performs the same setup automatically.")
}

// packSpec is one community asset pack, on or off. Off is not "absent": the
// pack is still offered so the player can see what a download would add.
func packSpec(id, label, pack, help string) Spec {
	return toggle(id, "Missions", label, help, "selected",
		func(s State) bool { return slices.Contains(s.Settings.CommunityPacks, pack) },
		func(s State, v bool) State {
			packs := slices.DeleteFunc(slices.Clone(s.Settings.CommunityPacks), func(p string) bool { return p == pack })
			if v {
				packs = append(packs, pack)
			}
			s.Settings.CommunityPacks = packs
			return s
		})
}

/*
	poolSpec is one mission's tick, and the pool is stored as the exclusions

MvmExcludedMissions holds what is out rather than what is in, so a mission added
to the game in a later release is in the pool by default rather than silently
missing from every settings file written before it existed.
*/
func poolSpec(mission gamedata.Mission) Spec {
	played, _ := gamedata.MapByID(mission.Map)
	source := "Valve"
	switch gamedata.MissionPack(mission.ID) {
	case settings.CommunityPackPotato:
		source = "Potato Archive"
	case settings.CommunityPackMoonlight:
		source = "Moonlight Archive"
	}
	tag := ""
	if label := runshape.MissionLoadoutLabel(mission); label != "" {
		tag = " [" + label + "]"
	}
	help := fmt.Sprintf("%s, %d waves. Off means the seed never draws it.", mission.Difficulty.String(), mission.Waves)
	if note := runshape.LoadoutNote(gamedata.MissionLoadout(mission.ID)); note != "" {
		help = note + " " + help
	}

	spec := twoWayToggle("missions.pool."+mission.PopFile, "Missions",
		fmt.Sprintf("[%s] %s (%s)%s", source, mission.Name, played.Name, tag),
		help, "in the pool", "left out",
		func(s State) bool { return !slices.Contains(s.Settings.MvmExcludedMissions, mission.PopFile) },
		func(s State, in bool) State {
			excluded := slices.DeleteFunc(slices.Clone(s.Settings.MvmExcludedMissions),
				func(p string) bool { return p == mission.PopFile })
			if !in {
				excluded = append(excluded, mission.PopFile)
			}
			s.Settings.MvmExcludedMissions = excluded
			return clearIneligibleStart(s)
		})

	/* A mission the game cannot play is shown and refused rather than hidden,
	   so a player looking for one the wiki names finds out why it is not here.
	   The row says what is missing rather than "unavailable": the two reasons
	   are a missing navigation mesh and a server mod whose verified setup is
	   not ready, and the fix is different for each. */
	if gamedata.MissionRequirement(mission.ID) != "" {
		return missionRequirementSpec(spec, mission, help)
	}
	/* A ticked community mission the switch above will not let the seed draw.
	   The tick is not wrong and neither is the pack, so the row says which
	   switch is in the way rather than being hidden or quietly refused. */
	if gamedata.IsCommunityMission(mission.ID) {
		spec.Unavailable = func(s State, _ Env) string {
			if s.Settings.MvmCommunityMissions || !slices.Contains(s.Settings.MvmExcludedMissions, mission.PopFile) {
				return ""
			}
			return "community missions are off"
		}
	}
	return spec
}

func missionRequirementSpec(spec Spec, mission gamedata.Mission, help string) Spec {
	if gamedata.MissionRequirement(mission.ID) == "no_nav" {
		nav := gamedata.MissingNavigationMesh(mission.ID)
		why := "The asset pack has this map's BSP but is missing " + nav + ". Bots cannot navigate the map, so this mission cannot be enabled in a seed."
		short := "missing " + nav
		spec.Help = why
		spec.Get = func(State) string { return "false" }
		spec.Unavailable = func(State, Env) string { return short }
		return spec
	}
	if key := gamedata.MissionServerMod(mission.ID); key != "" {
		spec.Help = gamedata.RequirementLabel(key) + ". " + help
		spec.Unavailable = func(s State, env Env) string {
			// An already selected mission must remain editable so it can be
			// removed even when the mod or community switch is unavailable.
			if !slices.Contains(s.Settings.MvmExcludedMissions, mission.PopFile) {
				return ""
			}
			mod, _ := gamedata.ServerModByKey(key)
			if (env.Platform == "windows" && !mod.Windows) || (env.Platform == "linux" && !mod.Linux) {
				return mod.Name + " has no " + env.Platform + " server build, so this mission cannot run here"
			}
			if !slices.Contains(settings.ServerModKeys(s.Settings), key) {
				if env.ManagedExternally {
					return "turn on " + mod.Name + " above, then save and recreate the containers"
				}
				return "turn on " + mod.Name + " above, then press Download / set up selected server mods"
			}
			if !slices.Contains(env.ServerModsReady, key) {
				return MissingServerModReason(mod.Name, env.ManagedExternally)
			}
			if !s.Settings.MvmCommunityMissions {
				return "community missions are off"
			}
			return ""
		}
		return spec
	}
	return spec
}

// startMission puts the chosen mission back in the pool and turns on the pack
// it came from. The draw leaves the server's own default alone, since srcds
// must boot on something and the plugin moves to the run's mission on its own.
func startMission(s State, popFile string) State {
	s.Settings.MvmStartMission = popFile
	if popFile == "" {
		return s
	}
	s.Settings.SrcdsStartMission = popFile
	s.Settings.MvmExcludedMissions = slices.DeleteFunc(
		slices.Clone(s.Settings.MvmExcludedMissions),
		func(p string) bool { return p == popFile })
	mission, ok := gamedata.MissionByPopFile(popFile)
	if !ok {
		return s
	}
	pack := gamedata.MissionPack(mission.ID)
	if pack != "" && !slices.Contains(s.Settings.CommunityPacks, pack) {
		s.Settings.CommunityPacks = append(slices.Clone(s.Settings.CommunityPacks), pack)
	}
	return s
}

// clearIneligibleStart returns the start choice to "Any" when a higher
// difficulty floor stops that named mission from being drawable. Keeping the
// stale choice makes every pool edit look broken when preflight reports
// the old start mission instead of the rows the player just selected.
func clearIneligibleStart(s State) State {
	if s.Settings.MvmStartMission == "" {
		return s
	}
	floor, valid := gamedata.DifficultyByKey(s.Settings.MvmDifficulty)
	for _, mission := range settings.MissionPool(s.Settings).Missions() {
		if mission.PopFile == s.Settings.MvmStartMission && (!valid || mission.Difficulty >= floor) {
			return s
		}
	}
	s.Settings.MvmStartMission = ""
	return s
}

func roomSpecs() []Spec {
	const tab = "Archipelago room"
	return []Spec{
		toggle("room.test_mode", tab, "Test mode",
			"Play without Archipelago at all: the launcher serves a multiworld of one and simulates the other players.",
			"no room needed",
			func(s State) bool { return s.Settings.TestMode },
			func(s State, v bool) State { s.Settings.TestMode = v; return s }),

		/* Deferred: the address is parsed, and reaching archipelago.gg:12345
		   means passing through "a", which is not one. What is typed is kept
		   and the parse is reported; Save is where a room that never became an
		   address is refused, and test mode never needs one. */
		roomAddress(),
		text("room.tracker_link", tab, "Room page link",
			"Paste the room page URL from archipelago.gg to open this room directly in the campaign tracker. The game connection address above does not contain the room ID.",
			"https://archipelago.gg/room/...",
			func(s State) string { return s.Settings.APRoomURL },
			func(s State, v string) State { s.Settings.APRoomURL = trim(v); return s }),

		secret("room.password", tab, "Room password", "Only if the room asks for one.", "none",
			func(s State) string { return s.Settings.APPassword },
			func(s State, v string) State { s.Settings.APPassword = v; return s }),

		text("room.slot", tab, "Slot name",
			"The name this server plays under in the multiworld. It has to match the name in tf2.yaml.",
			"tf2",
			func(s State) string { return s.Settings.APSlotName },
			func(s State, v string) State { s.Settings.APSlotName = trim(v); return s }),
	}
}

func roomAddress() Spec {
	spec := text("room.address", "Archipelago room", "Room address",
		"The line from your room page on archipelago.gg: host and port.",
		"archipelago.gg:12345",
		func(s State) string { return s.Draft.Room },
		func(s State, v string) State {
			s.Draft.Room = v
			// Best effort: an address that parses is stored, one that does not
			// leaves the last good host and port alone until Save asks again.
			if room, err := settings.ParseRoom(v); err == nil {
				s.Settings.APHost, s.Settings.APPort, s.Settings.APTls = room.Host, room.Port, room.TLS
			}
			return s
		})
	spec.Deferred = true
	return spec
}

func serverSpecs() []Spec {
	const tab = "Game server"
	return []Spec{
		text("server.hostname", tab, "Server name", "What the server calls itself in the player list.",
			"Mann vs Archipelago",
			func(s State) string { return s.Settings.SrcdsHostname },
			func(s State, v string) State { s.Settings.SrcdsHostname = v; return s }),

		secret("server.password", tab, "Server password",
			"What your friends type before connect. Blank means anybody with the address can join.", "none",
			func(s State) string { return s.Settings.SrcdsPw },
			func(s State, v string) State { s.Settings.SrcdsPw = v; return s }),

		number("server.port", tab, "Game port",
			"UDP and TCP, 27015 by default. Who can reach it is on the last page.",
			1024, 65535,
			func(s State) int { return s.Settings.SrcdsPort },
			func(s State, v int) State { s.Settings.SrcdsPort = v; return s }),

		text("server.join_host", tab, "Join address",
			"What the Join button gives players. Blank discovers this machine's local address. Use your public IP or DNS name for a port-forwarded server, or the WSL address when Docker runs in WSL and TF2 runs on Windows.",
			"discover it",
			func(s State) string { return s.Settings.SrcdsJoinHost },
			func(s State, v string) State { s.Settings.SrcdsJoinHost = trim(v); return s }),

		text("server.admins", tab, "Admins by Steam id",
			"Who may run the admin commands, separated by commas. The 17 digit id or SourceMod's STEAM_0:1:26975537.",
			"none",
			func(s State) string { return s.Settings.SrcdsAdminSteamIDs },
			func(s State, v string) State { s.Settings.SrcdsAdminSteamIDs = trim(v); return s }),

		/* The three that are about the launcher rather than about this page.
		   The window puts them along the bottom, where they were before the
		   rows moved into form and where a player looking for the debug bundle
		   still looks. */
		onBar(press("server.debug_bundle", tab, "Debug logs",
			"Download the logs, settings without passwords and player file as one zip for whoever is helping you.")),

		onBar(confirm("server.repair", tab, "Repair",
			"Reset SteamCMD and the mods; also redownload selected community packs whose hashes do not match. Keeps the game files and the run.",
			"this stops the server, removes SteamCMD and the mods, and redownloads any selected community ZIP with a hash mismatch. A large pack may use several GB while downloading. No 14 GB game download and no lost checks.")),

		onBar(confirm("server.reset", tab, "Reset settings",
			"Put every setting back to what a fresh install has. Keeps the game files and where they are.",
			"this puts the room, the passwords, the missions, the bots and who can join back to their defaults.")),
	}
}

func networkingSpecs() []Spec {
	const tab = "Networking"
	reaches := settings.Reaches()
	values := make([]string, 0, len(reaches))
	labels := make([]string, 0, len(reaches))
	for _, reach := range reaches {
		values = append(values, string(reach))
		labels = append(labels, reach.Label())
	}

	return []Spec{
		choice("net.reach", tab, "Who can reach it",
			"Where the server takes connections from. Without a login token it stays on the local network whatever this says.",
			options(values, labels),
			func(s State) string { return string(s.Settings.SrcdsReach) },
			func(s State, v string) State { s.Settings.SrcdsReach = settings.Reach(v); return s }),

		text("net.token", tab, "Login token",
			"A Game Server Login Token for app id 440, from steamcommunity.com/dev/managegameservers.",
			"0",
			func(s State) string { return s.Settings.SrcdsToken },
			func(s State, v string) State { s.Settings.SrcdsToken = trim(v); return s }),

		number("net.fastdl_port", tab, "FastDL port",
			"Where the launcher serves maps and other content to joining clients, on this machine. Zero turns the server off.",
			0, 65535,
			func(s State) int { return s.Settings.FastDLPort },
			func(s State, v int) State { s.Settings.FastDLPort = v; return s }),

		text("net.download_url", tab, "Download URL",
			"The whole sv_downloadurl value, for an operator who knows an address this machine cannot work out. Blank lets the launcher decide.",
			"work it out",
			func(s State) string { return s.Settings.SrcdsDownloadURL },
			func(s State, v string) State { s.Settings.SrcdsDownloadURL = trim(v); return s }),

		toggle("net.tailscale", tab, "Tailscale FastDL",
			"Publish maps through Tailscale Funnel. Only the server needs Tailscale; players use its public HTTPS URL. This does not change the game address. In Docker, the bundled service signs in through Set up / check Funnel. The native launcher uses Tailscale installed on this machine.",
			"use Funnel",
			func(s State) bool { return s.Settings.TailscaleFastDL },
			func(s State, v bool) State { s.Settings.TailscaleFastDL = v; return s }),

		press("net.check_funnel", tab, "Set up / check Funnel",
			"Checks Funnel and provides the approval page when needed. Docker includes its own Tailscale service and remembers the login. With the native launcher, install Tailscale first; headless Linux can run this launcher with -setup-funnel."),
	}
}

// MissingServerModReason is shared by the setting and mission table so both
// explain the same unavailable state in the same words.
func MissingServerModReason(name string, managedExternally bool) string {
	if managedExternally {
		return name + " files are missing or incomplete in the server volume; recreate the container"
	}
	return name + " is selected but its installation is missing or incomplete; press Download / set up selected server mods"
}

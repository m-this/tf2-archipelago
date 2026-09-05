/*
The settings, in the eight tabs the window uses: players, rewards, missions,
the room, the game server, bots, bot loadouts, and who can join.

The window puts these in a modal dialog with a control per answer. Here they
are a list per tab, one row each, and the keys do what the mouse does there.
Nothing is saved until Save: cancelling leaves the file alone, the way closing
a dialog does.
*/
package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/m-this/tf2-archipelago/gamedata"
	"github.com/m-this/tf2-archipelago/launcher/internal/assets"
	"github.com/m-this/tf2-archipelago/launcher/internal/botloadout"
	"github.com/m-this/tf2-archipelago/launcher/internal/debugbundle"
	"github.com/m-this/tf2-archipelago/launcher/internal/generate"
	"github.com/m-this/tf2-archipelago/launcher/internal/installer"
	"github.com/m-this/tf2-archipelago/launcher/internal/runshape"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
	"github.com/m-this/tf2-archipelago/launcher/internal/tailscalefastdl"
	"github.com/m-this/tf2-archipelago/launcher/internal/winproc"
)

// botSeats is how many seats the team composition names, which is the largest
// team the mod will field.
const botSeats = 6

// settingsForm is the settings screen: the tabs, their rows, and the copy of
// the settings the rows write into.
type settingsForm struct {
	edited settings.Settings
	// communityAvailable is derived from valid local ZIPs, never merely from
	// checkbox state. Community mission rows stay absent until their assets
	// can actually be used.
	communityAvailable []string

	tabs    []settingsTab
	tab     int
	focused int
	offset  int

	room     string       // the room address as typed, parsed on save
	teamName string       // what the next Save keeps the bot team under
	draft    loadoutDraft // the loadout the Loadouts page is building
	warn     string
	saved    func(settings.Settings) tea.Cmd
	repair   func() ([]string, error)
	reset    func() (settings.Settings, error)
	closed   bool
}

type settingsTab struct {
	title  string
	fields []field
}

type settingsDeps struct {
	saved  func(settings.Settings) tea.Cmd
	repair func() ([]string, error)
	reset  func() (settings.Settings, error)
}

func newSettingsForm(s settings.Settings, deps settingsDeps) *settingsForm {
	form := &settingsForm{edited: s, saved: deps.saved, repair: deps.repair, reset: deps.reset}
	form.room = settings.Room{Host: s.APHost, Port: s.APPort, TLS: s.APTls}.String()
	form.communityAvailable = availableCommunityPackNames(s.CommunityContentDir)
	form.build()
	return form
}

// build lays the tabs out from what is edited now. Reset settings and the
// pool's All and None all change rows the fields captured when they were made,
// so they call this rather than trying to patch what is on screen.
func (f *settingsForm) build() {
	f.tabs = []settingsTab{
		{title: "Player options", fields: f.playerFields()},
		{title: "Rewards", fields: f.rewardFields()},
		{title: "Balancing", fields: f.balanceFields()},
		{title: "Missions", fields: f.missionFields()},
		{title: "Archipelago room", fields: f.roomFields()},
		{title: "Game server", fields: f.serverFields()},
		{title: "Bots", fields: append(f.botFields(), f.loadoutFields()...)},
		{title: "Networking", fields: f.reachFields()},
	}
}

func (f *settingsForm) rewardFields() []field {
	importance := func(label, help string, value *string) field {
		return &choiceField{
			label: label, help: help,
			options: []string{"Useful", "Required for progression"},
			index:   map[bool]int{true: 1}[*value == "progression"],
			apply: func(i int) {
				if i == 1 {
					*value = "progression"
				} else {
					*value = "useful"
				}
			},
		}
	}
	return []field{
		importance("Mission tickets", "Required tickets gate mission deployment. Useful tickets leave every drawn mission available.", &f.edited.MvmMissionTicketImportance),
		importance("Class unlocks", "Required classes satisfy mission-tier roster checks. Useful classes only expand the roster.", &f.edited.MvmClassUnlockImportance),
		importance("Weapon slots", "Required slots satisfy mission-tier loadout checks. Useful slots only expand loadouts.", &f.edited.MvmWeaponSlotImportance),
		importance("Weapon buffs", "Required buffs gate harder tiers by buff count. Useful buffs are optional power-ups.", &f.edited.MvmWeaponBuffImportance),
		&toggleField{label: "Cash rewards", help: "Allow temporary MvM credits in spare checks. Off makes every spare reward a persistent weapon buff.", value: &f.edited.MvmCashRewards, on: "include cash", off: "include cash"},
		&numberField{label: "Buff share", help: "Percent of spare checks that are buffs when cash rewards are enabled.", value: &f.edited.MvmWeaponBuffPct, low: 0, high: 100},
		&numberField{label: "Buff stack chance", help: "Chance for another level of an already drawn numeric buff. Toggle buffs never repeat.", value: &f.edited.MvmWeaponBuffStackChance, low: 0, high: 100},
		&numberField{label: "Traps", help: "Percent of spare checks that hold a trap rather than a reward. A trap is an item another player finds and this team pays for. Zero leaves them out.", value: &f.edited.MvmTrapPct, low: 0, high: 100},
	}
}

/*
	balanceFields is what the robots are worth, which is a different question

from what the run hands out.

The two were one page and it had to be read twice to find either.
*/
func (f *settingsForm) balanceFields() []field {
	return []field{
		&numberField{label: "Robot health (%)", help: "Direct health multiplier for every robot, from 10% to 1000%.", value: &f.edited.SrcdsBluHealthPct, low: settings.RobotHealthPercentMin, high: settings.RobotHealthPercentMax},
	}
}

func (f *settingsForm) playerFields() []field {
	tiers := runshape.Tiers()
	tierLabels := make([]string, 0, len(tiers))
	for _, tier := range tiers {
		tierLabels = append(tierLabels, tier.Label())
	}
	goals := runshape.Goals()
	goalLabels := make([]string, 0, len(goals))
	for _, goal := range goals {
		goalLabels = append(goalLabels, goal.Label())
	}

	return []field{
		&choiceField{
			label:   "Easiest tier",
			help:    "The easiest tier a mission may come from. Harder tiers are always in as well, so the pool shrinks as this rises.",
			options: tierLabels,
			index:   max(slices.IndexFunc(tiers, func(t runshape.Tier) bool { return t.Key == f.edited.MvmDifficulty }), 0),
			apply:   func(i int) { f.edited.MvmDifficulty = tiers[i].Key },
		},
		&numberField{
			label: "Missions used",
			help:  "How many missions this run uses, out of the pool above. Eight is about fifty waves.",
			// The pool the tier leaves, the way the window caps it. Asking for
			// more than it holds gives the whole pool anyway.
			value: &f.edited.MvmMissionCount, low: 1, high: runshape.MissionsInPool(f.edited.MvmDifficulty),
		},
		&choiceField{
			label:   "Goal",
			help:    "What ends the run.",
			options: goalLabels,
			index:   max(slices.IndexFunc(goals, func(g runshape.Goal) bool { return g.Key == f.edited.MvmGoal }), 0),
			apply:   func(i int) { f.edited.MvmGoal = goals[i].Key },
		},
		&numberField{
			label: "Missionsanity share",
			help:  "How much of the run's checks come from waves rather than whole missions, as a percentage. It rounds up, and the Final Boss goal ignores it.",
			value: &f.edited.MvmMissionsanityPct, low: 10, high: 100,
		},
		&toggleField{
			label: "Death Link",
			help:  "A lost wave kills every other player in the multiworld who has Death Link on, and their deaths wipe your team.",
			value: &f.edited.MvmDeathLink, on: "share deaths", off: "share deaths",
		},
		&textField{
			label:       "Archipelago app",
			help:        "Where the Archipelago app is installed. Blank means the launcher looks where the installer puts it.",
			value:       &f.edited.ArchipelagoDir,
			placeholder: defaultAppDir(),
		},
		&actionField{
			label: "Generate seed",
			help:  "Make the seed with the Archipelago app on this machine, then upload the archive at archipelago.gg/uploads to open a room.",
			hint:  "enter",
			run:   f.generateSeed,
		},
		&actionField{
			label: "Open tf2.yaml",
			help:  "Write the player file and show it. It is what the seed is generated from.",
			hint:  "enter",
			run:   f.openPlayerFile,
		},
		&actionField{
			label: "Open the folder",
			help:  "The install root: the game files, the player file, the log and the run's state.",
			hint:  "enter",
			run:   f.openInstallRoot,
		},
	}
}

func (f *settingsForm) communityMissionFields() []field {
	return []field{
		&textField{
			label:       "Asset pack folder",
			help:        "Folder containing archive-assets.zip and/or mlarchive-assets.zip. Start never downloads community files.",
			value:       &f.edited.CommunityContentDir,
			placeholder: filepath.Join("C:", "Users", "Admin", "tf2"),
		},
		&choiceField{
			label:   "Potato Archive",
			help:    "Select archive-assets.zip for the explicit download action and for installation when the local ZIP is valid.",
			options: []string{"disabled", "selected"},
			index:   boolIndex(slices.Contains(f.edited.CommunityPacks, settings.CommunityPackPotato)),
			apply:   func(i int) { f.setCommunityPack(settings.CommunityPackPotato, i == 1) },
		},
		&choiceField{
			label:   "Moonlight Archive",
			help:    "Select mlarchive-assets.zip for the explicit download action and for installation when the local ZIP is valid.",
			options: []string{"disabled", "selected"},
			index:   boolIndex(slices.Contains(f.edited.CommunityPacks, settings.CommunityPackMoonlight)),
			apply:   func(i int) { f.setCommunityPack(settings.CommunityPackMoonlight, i == 1) },
		},
		&actionField{
			label: "Download Selected Community Assets",
			help:  "Fetch only the checked full-with-maps community packs. This is the launcher's only community download action.",
			hint:  "enter",
			run:   f.downloadSelectedCommunityAssets,
		},
		&actionField{
			label: "Use Local Community Assets",
			help:  "Validate archive-assets.zip and mlarchive-assets.zip already present in the asset pack folder, then show their missions.",
			hint:  "enter",
			run:   f.useLocalCommunityAssets,
		},
		&actionField{
			label: "Check Run Selection",
			help:  "Confirm that the eligible mission pool has enough checks to hold every mission, class, and weapon-slot unlock.",
			hint:  "enter",
			run:   f.checkRunSelection,
		},
	}
}

func (f *settingsForm) missionFields() []field {
	choices := runshape.StartMissionChoicesForPacks(f.communityAvailable)
	choiceLabels := make([]string, 0, len(choices))
	for _, choice := range choices {
		choiceLabels = append(choiceLabels, choice.Label)
	}
	classes := runshape.StartClassChoices()

	fields := append(f.communityMissionFields(),
		&choiceField{
			label:   "Start mission",
			help:    "Where the run begins. The seed starts there and the server boots there.",
			options: choiceLabels,
			index:   max(slices.IndexFunc(choices, func(c runshape.MissionChoice) bool { return c.PopFile == f.edited.MvmStartMission }), 0),
			apply: func(i int) {
				f.edited.MvmStartMission = choices[i].PopFile
				if choices[i].PopFile != "" {
					f.edited.SrcdsStartMission = choices[i].PopFile
					f.edited.MvmExcludedMissions = slices.DeleteFunc(f.edited.MvmExcludedMissions, func(popFile string) bool {
						return popFile == choices[i].PopFile
					})
					if mission, ok := gamedata.MissionByPopFile(choices[i].PopFile); ok {
						f.enableCommunityPack(gamedata.MissionPack(mission.ID))
					}
				}
			},
		},
		&choiceField{
			label:   "Start class",
			help:    "The mercenary the run starts with. The tier of the start mission decides how many it starts with.",
			options: classes,
			index:   max(slices.Index(classes, f.edited.MvmStartClass), 0),
			apply:   func(i int) { f.edited.MvmStartClass = startClass(classes, i) },
		},
		// The window has the two buttons under its table. Without them the only
		// way to a pool of three missions is 26 keystrokes down the list.
		&actionField{
			label: "All in the pool",
			help:  "Put every mission back in the pool.",
			hint:  "enter",
			run:   func() tea.Cmd { return f.setPool(true) },
		},
		&actionField{
			label: "None in the pool",
			help:  "Leave every mission out, to tick back the few this run is for.",
			hint:  "enter",
			run:   func() tea.Cmd { return f.setPool(false) },
		},
	)

	// One row per mission, because the pool is what the seed draws from and
	// the window gives it a table with a tick in every row.
	for _, mission := range runshape.VisibleMissions(f.communityAvailable) {
		if gamedata.IsPlayableMission(mission.ID) {
			fields = append(fields, f.poolField(mission))
		} else {
			fields = append(fields, unavailableMissionField(mission))
		}
	}
	return fields
}

func (f *settingsForm) checkRunSelection() tea.Cmd {
	return func() tea.Msg {
		result, err := settings.CheckRunSelection(f.edited)
		if err != nil {
			return noticeMsg(err.Error())
		}
		return noticeMsg(result.Summary())
	}
}

func (f *settingsForm) downloadSelectedCommunityAssets() tea.Cmd {
	folder := strings.TrimSpace(f.edited.CommunityContentDir)
	selected := f.edited
	selected.CommunityContentDir = folder
	archives := settings.CommunityArchives(selected)
	return func() tea.Msg {
		if folder == "" {
			return noticeMsg("choose an asset pack folder first")
		}
		if len(archives) == 0 {
			return noticeMsg("select at least one community pack first")
		}
		if err := installer.DownloadCommunityArchives(context.Background(), archives, func(string, ...any) {}); err != nil {
			return noticeMsg("community assets: " + err.Error())
		}
		return communityAssetsMsg{
			notice:    "Selected community packs are ready in " + folder,
			available: availableCommunityPackNames(folder),
		}
	}
}

func (f *settingsForm) useLocalCommunityAssets() tea.Cmd {
	folder := strings.TrimSpace(f.edited.CommunityContentDir)
	return func() tea.Msg {
		if folder == "" {
			return noticeMsg("choose an asset pack folder first")
		}
		available := availableCommunityPackNames(folder)
		if len(available) == 0 {
			return noticeMsg("no valid archive-assets.zip or mlarchive-assets.zip was found in " + folder)
		}
		return communityAssetsMsg{
			notice:      "Using local community packs from " + folder,
			available:   available,
			selectPacks: true,
		}
	}
}

func availableCommunityPackNames(folder string) []string {
	paths := installer.AvailableCommunityArchives(settings.KnownCommunityArchives(strings.TrimSpace(folder)))
	packs := make([]string, 0, len(paths))
	for _, path := range paths {
		packs = append(packs, filepath.Base(path))
	}
	return packs
}

type communityAssetsMsg struct {
	notice      string
	available   []string
	selectPacks bool
}

func (f *settingsForm) applyCommunityAssets(msg communityAssetsMsg) {
	f.communityAvailable = slices.Clone(msg.available)
	if msg.selectPacks {
		for _, pack := range []string{settings.CommunityPackPotato, settings.CommunityPackMoonlight} {
			f.edited.CommunityPacks = slices.DeleteFunc(f.edited.CommunityPacks, func(name string) bool { return name == pack })
			selected := slices.Contains(msg.available, pack)
			if selected {
				f.edited.CommunityPacks = append(f.edited.CommunityPacks, pack)
			}
			for _, mission := range gamedata.PlayableMissions() {
				if gamedata.MissionPack(mission.ID) != pack {
					continue
				}
				f.edited.MvmExcludedMissions = slices.DeleteFunc(f.edited.MvmExcludedMissions, func(popFile string) bool {
					return popFile == mission.PopFile
				})
				if !selected {
					f.edited.MvmExcludedMissions = append(f.edited.MvmExcludedMissions, mission.PopFile)
				}
			}
		}
	}
	f.build()
}

func boolIndex(value bool) int {
	if value {
		return 1
	}
	return 0
}

func (f *settingsForm) enableCommunityPack(pack string) {
	if pack != "" && !slices.Contains(f.edited.CommunityPacks, pack) {
		f.edited.CommunityPacks = append(f.edited.CommunityPacks, pack)
	}
}

func (f *settingsForm) setCommunityPack(pack string, enabled bool) {
	f.edited.CommunityPacks = slices.DeleteFunc(f.edited.CommunityPacks, func(name string) bool { return name == pack })
	if enabled {
		f.edited.CommunityPacks = append(f.edited.CommunityPacks, pack)
	}
	for _, mission := range gamedata.PlayableMissions() {
		if gamedata.MissionPack(mission.ID) != pack {
			continue
		}
		f.edited.MvmExcludedMissions = slices.DeleteFunc(f.edited.MvmExcludedMissions, func(popFile string) bool {
			return popFile == mission.PopFile
		})
		if !enabled {
			f.edited.MvmExcludedMissions = append(f.edited.MvmExcludedMissions, mission.PopFile)
		}
	}
	// The mission rows capture their ticks, so rebuild them after changing a
	// whole pack rather than leaving their labels one keypress behind.
	f.build()
}

// setPool is All and None: the excluded list is every mission or none of them,
// and the rows are made again, because each one captured its own tick.
func (f *settingsForm) setPool(inPool bool) tea.Cmd {
	excluded := make([]string, 0)
	if !inPool {
		for _, mission := range gamedata.PlayableMissions() {
			excluded = append(excluded, mission.PopFile)
		}
	} else {
		visible := runshape.VisibleMissions(f.communityAvailable)
		for _, mission := range gamedata.PlayableMissions() {
			if gamedata.MissionPack(mission.ID) != "" && !slices.ContainsFunc(visible, func(candidate gamedata.Mission) bool {
				return candidate.ID == mission.ID
			}) {
				excluded = append(excluded, mission.PopFile)
			}
		}
	}
	f.edited.MvmExcludedMissions = excluded
	f.build()

	if inPool {
		return func() tea.Msg { return noticeMsg("every mission is in the pool") }
	}
	return func() tea.Msg { return noticeMsg("every mission is left out: tick the ones this run may draw") }
}

// poolField is one mission's place in the pool. The setting is the missions
// left out, so the row reads the other way round from what it writes.
func (f *settingsForm) poolField(mission gamedata.Mission) field {
	played, _ := gamedata.MapByID(mission.Map)
	source := "Valve"
	switch gamedata.MissionPack(mission.ID) {
	case "archive-assets.zip":
		source = "Potato Archive"
	case "mlarchive-assets.zip":
		source = "Moonlight Archive"
	}
	inPool := !slices.Contains(f.edited.MvmExcludedMissions, mission.PopFile)
	held := inPool

	return &poolToggle{
		label:   fmt.Sprintf("[%s] %s (%s)", source, mission.Name, played.Name),
		help:    fmt.Sprintf("%s, %d waves. Off means the seed never draws it.", mission.Difficulty.String(), mission.Waves),
		value:   &held,
		on:      "in the pool",
		off:     "left out",
		popFile: mission.PopFile,
		form:    f,
		held:    &held,
	}
}

// poolToggle keeps the excluded list in step with the tick.
type poolToggle struct {
	toggleField
	popFile string
	form    *settingsForm
	held    *bool
}

type unavailablePoolField struct {
	label  string
	help   string
	reason string
}

func unavailableMissionField(mission gamedata.Mission) field {
	played, _ := gamedata.MapByID(mission.Map)
	requirement := gamedata.MissionRequirement(mission.ID)
	help := "The asset pack has this map's BSP but no bot navigation mesh. It cannot be enabled in a seed."
	if gamedata.MissionServerMod(mission.ID) != "" {
		help = "This mission needs a server mod this launcher does not install. It cannot be enabled in a seed here."
	}
	return &unavailablePoolField{
		label:  fmt.Sprintf("[Potato Archive] %s (%s)", mission.Name, played.Name),
		help:   help,
		reason: strings.ToLower(gamedata.RequirementLabel(requirement)),
	}
}

func (f *unavailablePoolField) Label() string { return f.label }
func (f *unavailablePoolField) Help() string  { return f.help }
func (f *unavailablePoolField) Value() string {
	return styleStopped.Render(f.reason + " — unavailable")
}
func (f *unavailablePoolField) Handle(tea.KeyMsg) bool { return false }

func (p *poolToggle) Handle(msg tea.KeyMsg) bool {
	if !p.toggleField.Handle(msg) {
		return false
	}
	excluded := p.form.edited.MvmExcludedMissions
	excluded = slices.DeleteFunc(excluded, func(popFile string) bool { return popFile == p.popFile })
	if !*p.held {
		excluded = append(excluded, p.popFile)
	}
	p.form.edited.MvmExcludedMissions = excluded
	return true
}

func (f *settingsForm) roomFields() []field {
	return []field{
		&toggleField{
			label: "Test mode",
			help:  "Play without Archipelago at all: the launcher serves a multiworld of one and simulates the other players.",
			value: &f.edited.TestMode, on: "no room needed", off: "use a real room",
		},
		&textField{
			label:       "Room address",
			help:        "The line from your room page on archipelago.gg: host and port.",
			value:       &f.room,
			placeholder: "archipelago.gg:12345",
		},
		&textField{
			label:       "Room password",
			help:        "Only if the room asks for one.",
			value:       &f.edited.APPassword,
			placeholder: "none",
			hidden:      true,
		},
		&textField{
			label:       "Slot name",
			help:        "The name this server plays under in the multiworld. It has to match the name in tf2.yaml.",
			value:       &f.edited.APSlotName,
			placeholder: "tf2",
		},
	}
}

func (f *settingsForm) serverFields() []field {
	return []field{
		&textField{
			label:       "Server name",
			help:        "What the server calls itself in the player list.",
			value:       &f.edited.SrcdsHostname,
			placeholder: "Mann vs Archipelago",
		},
		&textField{
			label:       "Server password",
			help:        "What your friends type before connect. Blank means anybody with the address can join.",
			value:       &f.edited.SrcdsPw,
			placeholder: "none",
			hidden:      true,
		},
		&numberField{
			label: "Game port",
			help:  "UDP and TCP, 27015 by default. Who can reach it is on the last tab.",
			value: &f.edited.SrcdsPort, low: 1024, high: 65535,
		},
		&textField{
			label:       "Admins by Steam id",
			help:        "Who may run the admin commands, separated by commas. The 17 digit id or SourceMod's STEAM_0:1:26975537.",
			value:       &f.edited.SrcdsAdminSteamIDs,
			placeholder: "none",
		},
		&actionField{
			label: "Debug logs",
			help:  "Put the logs, the settings without their passwords and the player file in one zip, for sending to whoever is helping you.",
			hint:  "enter",
			run:   f.debugBundle,
		},
		&confirmField{
			label:   "Repair",
			help:    "Throw SteamCMD and the mods away and fetch them again. Keeps the game files and the run.",
			hint:    "enter",
			run:     f.runRepair,
			warning: "this stops the server, then removes SteamCMD, the mods and Steam's record of the download. No 14 GB again, no lost checks.",
		},
		&confirmField{
			label:   "Reset settings",
			help:    "Put every setting back to what a fresh install has. Keeps the game files and where they are.",
			hint:    "enter",
			run:     f.runReset,
			warning: "this puts the room, the passwords, the missions, the bots and who can join back to their defaults.",
		},
	}
}

func (f *settingsForm) botFields() []field {
	fields := []field{
		&toggleField{
			label: "Fill RED with bots",
			help:  "Valve balances every wave for six players. Off leaves the seats empty until an admin runs sm_addbots.",
			value: &f.edited.SrcdsBots, on: "bots on", off: "bots off",
		},
		&numberField{
			label: "Fill RED to",
			help:  "How many players the server fills RED to, humans included. Lower is harder.",
			value: &f.edited.SrcdsBotTeamSize, low: 1, high: botSeats,
		},
		&toggleField{
			label: "Say what they buy",
			help:  "Write every bot purchase at the upgrade station to the chat. It is a lot of chat.",
			value: &f.edited.BotUpgradesChat, on: "in the chat", off: "quiet",
		},
	}

	// A team is worth naming once. The window has a menu and two buttons for
	// this; here it is a list to load from and a name to save under.
	fields = append(fields, f.loadTeamField(), f.saveTeamField(), f.removeTeamField(), &textField{
		label:       "Team name",
		help:        "The name Save this team as keeps it under.",
		value:       &f.teamName,
		placeholder: "two engineers, one medic",
	})

	// The seats, in the order they fill, each with what it plays and what it
	// carries. Two engineers are only worth naming separately if they can hold
	// different weapons.
	for seat := range botSeats {
		fields = append(fields, f.seatField(seat), f.seatLoadoutField(seat))
	}

	for _, class := range botloadout.Classes {
		fields = append(fields, f.classField(class), f.loadoutField(class))
	}

	// Last, because none of it changes a wave.
	return append(fields,
		&toggleField{
			label: "Cosmetic items",
			help:  "A random cosmetic item on every bot, hat or not, drawn from the ones its class can wear. It changes nothing about how they play.",
			value: &f.edited.SrcdsBotHats, on: "one each", off: "stock looks",
		},
		&toggleField{
			label: "Unusual effects",
			help:  "A random unusual effect on that cosmetic item. Six particle effects on screen for the whole wave.",
			value: &f.edited.SrcdsBotHatEffects, on: "and an effect on it", off: "no effects",
		},
	)
}

func (f *settingsForm) seatField(seat int) field {
	options := []string{"the mod decides"}
	for _, class := range botloadout.Classes {
		options = append(options, class.Name)
	}
	index := 0
	if seat < len(f.edited.SrcdsBotTeamComp) {
		if at := slices.IndexFunc(botloadout.Classes, func(c botloadout.Class) bool {
			return c.Key == f.edited.SrcdsBotTeamComp[seat]
		}); at >= 0 {
			index = at + 1
		}
	}

	return &choiceField{
		label:   fmt.Sprintf("Seat %d", seat+1),
		help:    "The classes the bots fill RED with, in this order. The first seats are the ones that always get filled.",
		options: options,
		index:   index,
		apply:   func(i int) { f.setSeat(seat, i) },
	}
}

// setSeat rewrites the team from the seats. The loadouts come from the same
// array. Otherwise the two lists stop lining up when a middle seat goes back on
// the draw.
func (f *settingsForm) setSeat(seat, index int) {
	seats := f.seats()
	if index == 0 {
		seats[seat] = botloadout.Seat{}
	} else {
		class := botloadout.Classes[index-1]
		// The weapons follow the class: a loadout for a class this seat no
		// longer plays is not a choice anybody made.
		if seats[seat].Class != class.Key {
			seats[seat] = botloadout.Seat{Class: class.Key}
		}
	}
	f.setSeats(seats)
}

// seats is the team as an array of six, whatever the compacted lists hold.
func (f *settingsForm) seats() []botloadout.Seat {
	seats := make([]botloadout.Seat, botSeats)
	copy(seats, botloadout.Seats(f.edited.SrcdsBotTeamComp, f.edited.SrcdsBotSeatLoadouts))
	return seats
}

// setSeats writes the array back as the two lists the mod reads. A seat left to
// the mod is an empty entry, because the mod counts seats by their place in the
// list. It drops the trailing draws, which carry no seat number.
func (f *settingsForm) setSeats(seats []botloadout.Seat) {
	last := -1
	for index, seat := range seats {
		if seat.Class != "" {
			last = index
		}
	}
	f.edited.SrcdsBotTeamComp = nil
	f.edited.SrcdsBotSeatLoadouts = nil
	for _, seat := range seats[:last+1] {
		f.edited.SrcdsBotTeamComp = append(f.edited.SrcdsBotTeamComp, seat.Class)
		f.edited.SrcdsBotSeatLoadouts = append(f.edited.SrcdsBotSeatLoadouts, seat.Loadout)
	}
}

// seatLoadoutField is what one seat carries. A seat with no class of its own
// has nothing to choose: the mod draws the class and the class holds its own.
func (f *settingsForm) seatLoadoutField(seat int) field {
	seats := f.seats()
	class, found := botloadout.ClassByKey(seats[seat].Class)
	if !found {
		return &choiceField{
			label:   fmt.Sprintf("  Seat %d holds", seat+1),
			help:    "Pick a class for this seat first. A seat on the draw holds whatever its class holds.",
			options: []string{"follows the class"},
			apply:   func(int) {},
		}
	}

	choices := f.library().Choices(class)
	options := make([]string, 0, len(choices))
	for _, loadout := range choices {
		options = append(options, loadout.Label())
	}
	index := slices.IndexFunc(choices, func(l botloadout.Loadout) bool {
		return l.Key == seats[seat].Loadout
	})
	return &choiceField{
		label:   fmt.Sprintf("  Seat %d holds", seat+1),
		help:    "The weapons this seat carries, which is what lets two engineers hold different things. Loadouts you built for this class are at the bottom.",
		options: options,
		index:   max(index, 0),
		apply: func(i int) {
			seats := f.seats()
			seats[seat].Loadout = choices[i].Key
			f.setSeats(seats)
		},
	}
}

// library is the loadouts this form can offer, built from what is edited now
// rather than from what was saved: a loadout built on the Loadouts page is
// pickable on the Bots page without leaving the settings.
func (f *settingsForm) library() botloadout.Library {
	return botloadout.Library{Built: f.edited.SrcdsBotCustomLoadouts}
}

// loadTeamField brings back a saved team.
func (f *settingsForm) loadTeamField() field {
	names := settings.BotTeamNames(f.edited)
	options := append([]string{"keep the team below"}, names...)

	return &choiceField{
		label:   "Load a team",
		help:    "A team is the seats, their weapons and the classes the mod may draw from. Saved teams are listed here.",
		options: options,
		apply: func(i int) {
			if i == 0 || i > len(names) {
				return
			}
			f.edited = settings.WithBotTeam(f.edited, f.edited.SrcdsBotTeamPresets[names[i-1]])
			f.build()
		},
	}
}

// removeTeamField throws a saved team away.
//
// By the name in the box, because that is the only thing here somebody has
// typed on purpose: a list that deletes whatever it is scrolled past is a list
// that deletes the wrong team.
func (f *settingsForm) removeTeamField() field {
	return &actionField{
		label: "Remove the team named",
		help:  "Type the name of a saved team in the box below, then press enter here to throw it away.",
		hint:  "enter",
		run: func() tea.Cmd {
			name := strings.TrimSpace(f.teamName)
			if name == "" {
				return func() tea.Msg { return noticeMsg("name the team to remove first") }
			}
			if _, found := f.edited.SrcdsBotTeamPresets[name]; !found {
				return func() tea.Msg { return noticeMsg("no team saved as " + name) }
			}
			delete(f.edited.SrcdsBotTeamPresets, name)
			f.teamName = ""
			f.build()
			return func() tea.Msg { return noticeMsg("removed the team " + name) }
		},
	}
}

// saveTeamField keeps the team below under a name.
func (f *settingsForm) saveTeamField() field {
	return &actionField{
		label: "Save this team as",
		help:  "Type a name in the box below it, then press enter here to keep the seats, their weapons and the class ticks under it.",
		hint:  "enter",
		run: func() tea.Cmd {
			name := strings.TrimSpace(f.teamName)
			if name == "" {
				return func() tea.Msg { return noticeMsg("name the team first") }
			}
			if f.edited.SrcdsBotTeamPresets == nil {
				f.edited.SrcdsBotTeamPresets = map[string]settings.BotTeam{}
			}
			f.edited.SrcdsBotTeamPresets[name] = settings.BotTeamOf(f.edited)
			f.teamName = ""
			f.build()
			return func() tea.Msg { return noticeMsg("saved the team as " + name) }
		},
	}
}

func (f *settingsForm) classField(class botloadout.Class) field {
	allowed := !slices.Contains(f.edited.SrcdsBotClassBlacklist, class.Key)
	held := allowed

	return &classToggle{
		label: class.Name,
		help:  "Off means the bots never play it. A class named in a seat above beats this.",
		value: &held, on: "they play it", off: "never",
		key:  class.Key,
		form: f,
		held: &held,
	}
}

type classToggle struct {
	toggleField
	key  string
	form *settingsForm
	held *bool
}

func (c *classToggle) Handle(msg tea.KeyMsg) bool {
	if !c.toggleField.Handle(msg) {
		return false
	}
	list := c.form.edited.SrcdsBotClassBlacklist
	list = slices.DeleteFunc(list, func(key string) bool { return key == c.key })
	if !*c.held {
		list = append(list, c.key)
	}
	c.form.edited.SrcdsBotClassBlacklist = list
	return true
}

func (f *settingsForm) loadoutField(class botloadout.Class) field {
	choices := f.library().Choices(class)
	options := make([]string, 0, len(choices))
	for _, loadout := range choices {
		options = append(options, loadout.Label())
	}
	current := f.edited.SrcdsBotLoadouts[class.Key]
	index := max(slices.IndexFunc(choices, func(l botloadout.Loadout) bool { return l.Key == current }), 0)

	return &choiceField{
		label:   "  " + class.Name + " loadout",
		help:    "What a bot of this class spawns with. Stock is the game's own, and loadouts you built for this class are at the bottom.",
		options: options,
		index:   index,
		apply: func(i int) {
			if f.edited.SrcdsBotLoadouts == nil {
				f.edited.SrcdsBotLoadouts = map[string]string{}
			}
			pick := choices[i]
			if pick.Key == botloadout.StockKey {
				delete(f.edited.SrcdsBotLoadouts, class.Key)
				return
			}
			f.edited.SrcdsBotLoadouts[class.Key] = pick.Key
		},
	}
}

func (f *settingsForm) reachFields() []field {
	reaches := settings.Reaches()
	labels := make([]string, 0, len(reaches))
	for _, reach := range reaches {
		labels = append(labels, reach.Label())
	}

	return []field{
		&choiceField{
			label:   "Who can reach it",
			help:    "Where the server takes connections from. Without a login token it stays on the local network whatever this says.",
			options: labels,
			index:   max(slices.Index(reaches, f.edited.SrcdsReach), 0),
			apply:   func(i int) { f.edited.SrcdsReach = reaches[i] },
		},
		&textField{
			label:       "Login token",
			help:        "A Game Server Login Token for app id 440, from steamcommunity.com/dev/managegameservers.",
			value:       &f.edited.SrcdsToken,
			placeholder: "0",
		},
		&toggleField{
			label: "Tailscale FastDL",
			help:  "Publish maps through Tailscale Funnel. Only the server needs Tailscale; players use its public HTTPS URL. This does not change the game address. Failure falls back to the game server download.",
			value: &f.edited.TailscaleFastDL,
			on:    "use Funnel",
			off:   "use launcher",
		},
		&actionField{
			label: "Set up / check Funnel",
			help:  "Check Tailscale now. If Funnel needs tailnet approval, this opens the approval page in your browser; approve it, then run this check again.",
			hint:  "enter",
			run:   f.checkTailscaleFunnel,
		},
	}
}

func (f *settingsForm) checkTailscaleFunnel() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()
		result, err := tailscalefastdl.Authorize(ctx)
		if err != nil {
			return noticeMsg("Tailscale Funnel: " + err.Error())
		}
		if result.ApprovalURL != "" {
			if err := winproc.OpenURL(result.ApprovalURL); err != nil {
				return noticeMsg("approve Funnel at " + result.ApprovalURL)
			}
			return noticeMsg("approve Funnel in the browser, then run Set up / check Funnel again")
		}
		return noticeMsg("Tailscale Funnel is ready for this tailnet")
	}
}

// save parses what cannot be typed wrong twice, and hands the settings back.
func (f *settingsForm) save() tea.Cmd {
	room, err := settings.ParseRoom(f.room)
	if err != nil && !f.edited.TestMode {
		f.warn = err.Error()
		return nil
	}
	f.edited.APHost, f.edited.APPort, f.edited.APTls = room.Host, room.Port, room.TLS

	if f.edited.SrcdsReach.NeedsToken() && !settings.HasToken(f.edited.SrcdsToken) {
		f.warn = "that reach needs a login token, or the server stays on the local network"
	}
	if _, err := settings.CheckRunSelection(f.edited); err != nil {
		f.warn = err.Error()
		return nil
	}
	f.closed = true
	return f.saved(f.edited)
}

func (f *settingsForm) generateSeed() tea.Cmd {
	return func() tea.Msg {
		if _, err := settings.CheckRunSelection(f.edited); err != nil {
			return noticeMsg(err.Error())
		}
		if _, err := generate.FindApp(f.edited.ArchipelagoDir); err != nil {
			return noticeMsg("the Archipelago app was not found in " +
				strings.Join(generate.SearchPath(f.edited.ArchipelagoDir), ", "))
		}
		result, err := generate.Run(context.Background(), generate.Options{
			Settings:           f.edited,
			AppDir:             f.edited.ArchipelagoDir,
			Apworld:            assets.Apworld(),
			ArchipelagoVersion: assets.ArchipelagoVersion,
		})
		if err != nil {
			return noticeMsg("generate: " + err.Error())
		}
		_ = winproc.Open(filepath.Dir(result.Archive))
		return noticeMsg("wrote " + result.Archive + ": upload it at archipelago.gg/uploads")
	}
}

func (f *settingsForm) openPlayerFile() tea.Cmd {
	return func() tea.Msg {
		path, err := settings.WritePlayerFile(f.edited, assets.ArchipelagoVersion)
		if err != nil {
			return noticeMsg(err.Error())
		}
		_ = winproc.Open(path)
		return noticeMsg("wrote " + path)
	}
}

func (f *settingsForm) openInstallRoot() tea.Cmd {
	return func() tea.Msg {
		if err := winproc.Open(f.edited.InstallRoot); err != nil {
			return noticeMsg("cannot open " + f.edited.InstallRoot + ": " + err.Error())
		}
		return noticeMsg("opened " + f.edited.InstallRoot)
	}
}

// runRepair stops everything the launcher started and removes what the next
// start can fetch again. It blocks the screen for as long as that takes, which
// is the same wait the window's message box covers.
func (f *settingsForm) runRepair() tea.Cmd {
	return func() tea.Msg {
		removed, err := f.repair()
		switch {
		case err != nil:
			return noticeMsg("repair: " + err.Error())
		case len(removed) == 0:
			return noticeMsg("repair: nothing to remove")
		default:
			return noticeMsg("repair removed " + strings.Join(removed, ", ") + ". Press s when you are ready.")
		}
	}
}

// runReset takes the defaults back into the form as well as onto disk. The
// window closes its dialog instead, because every control on screen still held
// the old answer and Save would have written them straight back.
func (f *settingsForm) runReset() tea.Cmd {
	fresh, err := f.reset()
	if err != nil {
		return func() tea.Msg { return noticeMsg("reset: " + err.Error()) }
	}
	f.edited = fresh
	f.room = settings.Room{Host: fresh.APHost, Port: fresh.APPort, TLS: fresh.APTls}.String()
	f.warn = ""
	f.build()
	return func() tea.Msg { return noticeMsg("every setting is back to its default") }
}

func (f *settingsForm) debugBundle() tea.Cmd {
	return func() tea.Msg {
		path, err := debugbundle.Write(f.edited, assets.Versions(), time.Now())
		if err != nil {
			return noticeMsg(err.Error())
		}
		return noticeMsg("wrote " + path + ", with no passwords in it")
	}
}

// noticeMsg is a line for the log: what an action did, or why it did not.
type noticeMsg string

func defaultAppDir() string {
	if dirs := generate.SearchPath(""); len(dirs) > 0 {
		return dirs[0]
	}
	return ""
}

// startClass is the class name the run starts with, where the first choice is
// the seed's own pick rather than a class.
func startClass(classes []string, index int) string {
	if index <= 0 || index >= len(classes) {
		return ""
	}
	return classes[index]
}

// showTab opens on the tab with that title, and stays where it is for a title
// no tab carries.
func (f *settingsForm) showTab(title string) {
	if title == "" {
		return
	}
	for i, tab := range f.tabs {
		if tab.title == title {
			f.tab = i
			return
		}
	}
}

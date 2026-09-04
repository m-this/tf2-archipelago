//go:build windows

package gui

import (
	"context"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/lxn/walk"
	declarative "github.com/lxn/walk/declarative"
	"github.com/lxn/win"

	"github.com/m-this/tf2-archipelago/gamedata"
	"github.com/m-this/tf2-archipelago/launcher/internal/assets"
	"github.com/m-this/tf2-archipelago/launcher/internal/botloadout"
	"github.com/m-this/tf2-archipelago/launcher/internal/debugbundle"
	"github.com/m-this/tf2-archipelago/launcher/internal/generate"
	"github.com/m-this/tf2-archipelago/launcher/internal/installer"
	"github.com/m-this/tf2-archipelago/launcher/internal/runshape"
	apruntime "github.com/m-this/tf2-archipelago/launcher/internal/runtime"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
	"github.com/m-this/tf2-archipelago/launcher/internal/winproc"
)

// labelWidth keeps every tab's label column the same width, so the fields do
// not jump when the player switches tab.
const labelWidth = 150

// sentenceWidth caps the paragraphs on the Bots tab.
//
// A Label does not wrap, so a sentence of two hundred characters is one row two
// hundred characters wide, and every other row on the tab is stretched to match
// it. A TextLabel wraps, but only inside a width somebody gives it: this is the
// dialog's, less the label column and the margins.
const sentenceWidth = 980

// tokenPageURL issues the Game Server Login Token the server logs in with.
const tokenPageURL = "https://steamcommunity.com/dev/managegameservers"

// runSettingsDialog asks for the values worth changing between evenings, in
// six tabs: what the run is, which missions it may draw, where the room is,
// how the game server behaves, what the bots play, and who can join. Every row
// carries a tooltip, because a name alone does not say what a difficulty floor
// or a login token is.
//
// It returns the edited settings and whether the player accepted them.
func runSettingsDialog(
	owner walk.Form, s settings.Settings,
	repair func() ([]string, error), reset func() error,
	say func(format string, args ...any),
	openOn string,
) (settings.Settings, bool, error) {
	var (
		dialog *walk.Dialog
		accept *walk.PushButton
		cancel *walk.PushButton

		testBox  *walk.CheckBox
		roomEdit *walk.LineEdit
		roomWarn *walk.Label
		roomPass *walk.LineEdit
		slotEdit *walk.LineEdit

		tierBox          *walk.ComboBox
		missions         *walk.NumberEdit
		goalBox          *walk.ComboBox
		sanityPct        *walk.NumberEdit
		deathLink        *walk.CheckBox
		ticketImportance *walk.ComboBox
		classImportance  *walk.ComboBox
		slotImportance   *walk.ComboBox
		buffImportance   *walk.ComboBox
		cashRewards      *walk.CheckBox
		buffPct          *walk.NumberEdit
		buffStack        *walk.NumberEdit
		bluHealth        *walk.NumberEdit

		startBox      *walk.ComboBox
		startClassBox *walk.ComboBox
		poolView      *walk.TableView
		contentEdit   *walk.LineEdit
		potatoBox     *walk.CheckBox
		moonlightBox  *walk.CheckBox
		fetchPacks    *walk.PushButton
		fetchStatus   *walk.Label

		appEdit *walk.LineEdit

		nameEdit   *walk.LineEdit
		passEdit   *walk.LineEdit
		portEdit   *walk.NumberEdit
		adminEdit  *walk.LineEdit
		botsBox    *walk.CheckBox
		botsSize   *walk.NumberEdit
		buysBox    *walk.CheckBox
		looksBox   cosmeticBoxes
		botsHost   *walk.Composite
		botsBuilt  bool
		tabs       *walk.TabWidget
		presetBox  *walk.ComboBox
		presetName *walk.LineEdit
		botTeam    = &botTeamEditor{
			seatBox:    make([]*walk.ComboBox, botSeats),
			seatLoadBx: make([]*walk.ComboBox, botSeats),
			classBox:   make([]*walk.CheckBox, len(botloadout.Classes)),
			loadoutBx:  make([]*walk.ComboBox, len(botloadout.Classes)),
			presetBox:  &presetBox,
			presetName: &presetName,
			keep:       keepBotTeams,
			saved:      maps.Clone(s.SrcdsBotTeamPresets),
		}
		botLoadout = newBotLoadoutEditor(s.SrcdsBotCustomLoadouts, keepBotLoadouts)
	)
	// The team's menus list what the Loadouts page holds, so it is wired after
	// both exist.
	botTeam.library = func() botloadout.Library {
		return botloadout.Library{Built: botLoadout.built}
	}
	// Saving or removing a loadout puts it in and out of those menus at once,
	// rather than at the next open of the dialog.
	botLoadout.changed = botTeam.refreshLoadouts
	var (
		reachLan   *walk.RadioButton
		reachSteam *walk.RadioButton
		reachPort  *walk.RadioButton
		reachHelp  *walk.TextLabel
		tokenEdit  *walk.LineEdit
		tokenWarn  *walk.Label
		tokenLink  *walk.LinkLabel
	)

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
	availablePacks := communityPackNames(installer.AvailableCommunityArchives(settings.KnownCommunityArchives(s.CommunityContentDir)))
	choices := runshape.StartMissionChoicesForPacks(availablePacks)
	choiceLabels := make([]string, 0, len(choices))
	startMissionValue := runshape.AnyLabel
	for _, choice := range choices {
		choiceLabels = append(choiceLabels, choice.Label)
		if choice.PopFile == s.MvmStartMission {
			startMissionValue = choice.Label
		}
	}
	classLabels := runshape.StartClassChoices()
	potatoSelected := slices.Contains(s.CommunityPacks, settings.CommunityPackPotato)
	moonlightSelected := slices.Contains(s.CommunityPacks, settings.CommunityPackMoonlight)
	pool := newPoolModel(s.MvmExcludedMissions, s.CommunityPacks, availablePacks)
	importanceLabels := []string{"Useful", "Required for progression"}
	importanceLabel := func(value string) string {
		if value == "progression" {
			return importanceLabels[1]
		}
		return importanceLabels[0]
	}

	current := settings.Room{Host: s.APHost, Port: s.APPort}
	edited := s

	label := func(text, help string) declarative.Label {
		return declarative.Label{
			Text:        text,
			MinSize:     declarative.Size{Width: labelWidth},
			ToolTipText: help,
		}
	}

	refreshCommunityChoices := func(packs []string) {
		selected := ""
		if startBox != nil && startBox.CurrentIndex() >= 0 && startBox.CurrentIndex() < len(choices) {
			selected = choices[startBox.CurrentIndex()].PopFile
		}
		availablePacks = slices.Clone(packs)
		pool.setAvailablePacks(availablePacks)
		choices = runshape.StartMissionChoicesForPacks(availablePacks)
		choiceLabels = choiceLabels[:0]
		selectedIndex := 0
		for i, choice := range choices {
			choiceLabels = append(choiceLabels, choice.Label)
			if choice.PopFile == selected {
				selectedIndex = i
			}
		}
		if startBox != nil {
			_ = startBox.SetModel(choiceLabels)
			_ = startBox.SetCurrentIndex(selectedIndex)
		}
	}

	// collect reads the widgets into a copy of the settings. Save uses it, and
	// so does the button that writes the player file: both have to work from
	// what is on screen rather than from what was saved last time.
	collect := func() (settings.Settings, error) {
		next := s
		next.TestMode = testBox.Checked()
		room, err := settings.ParseRoom(roomEdit.Text())
		if err != nil {
			// Test mode never dials a real room, so it does not need one.
			if !next.TestMode {
				return next, err
			}
			room = settings.Room{}
		}
		next.APHost, next.APPort, next.APTls = room.Host, room.Port, room.TLS
		next.APPassword = roomPass.Text()
		next.APSlotName = strings.TrimSpace(slotEdit.Text())

		next.MvmDifficulty = tiers[max(tierBox.CurrentIndex(), 0)].Key
		next.MvmMissionCount = int(missions.Value())
		next.MvmGoal = goals[max(goalBox.CurrentIndex(), 0)].Key
		next.MvmMissionsanityPct = int(sanityPct.Value())
		next.MvmDeathLink = deathLink.Checked()
		next.CommunityContentDir = strings.TrimSpace(contentEdit.Text())
		next.CommunityPacks = []string{}
		if potatoBox.Checked() {
			next.CommunityPacks = append(next.CommunityPacks, settings.CommunityPackPotato)
		}
		if moonlightBox.Checked() {
			next.CommunityPacks = append(next.CommunityPacks, settings.CommunityPackMoonlight)
		}
		importanceValue := func(box *walk.ComboBox) string {
			if box.CurrentIndex() == 1 {
				return "progression"
			}
			return "useful"
		}
		next.MvmMissionTicketImportance = importanceValue(ticketImportance)
		next.MvmClassUnlockImportance = importanceValue(classImportance)
		next.MvmWeaponSlotImportance = importanceValue(slotImportance)
		next.MvmWeaponBuffImportance = importanceValue(buffImportance)
		next.MvmCashRewards = cashRewards.Checked()
		next.MvmWeaponBuffPct = int(buffPct.Value())
		next.MvmWeaponBuffStackChance = int(buffStack.Value())
		next.SrcdsBluHealthPct = int(bluHealth.Value())
		next.MvmExcludedMissions = pool.excludedMissions()
		next.ArchipelagoDir = strings.TrimSpace(appEdit.Text())

		// One control for both: the seed starts the run on this mission and the
		// server boots on it, which is the only way they cannot disagree. The
		// draw leaves the server's own default alone, since srcds must boot on
		// something and the plugin moves to the run's mission on its own.
		next.MvmStartMission = choices[max(startBox.CurrentIndex(), 0)].PopFile
		if next.MvmStartMission != "" {
			next.SrcdsStartMission = next.MvmStartMission
			next.MvmExcludedMissions = slices.DeleteFunc(next.MvmExcludedMissions, func(popFile string) bool {
				return popFile == next.MvmStartMission
			})
			if mission, ok := gamedata.MissionByPopFile(next.MvmStartMission); ok {
				pack := gamedata.MissionPack(mission.ID)
				if pack != "" && !slices.Contains(next.CommunityPacks, pack) {
					next.CommunityPacks = append(next.CommunityPacks, pack)
				}
			}
		}
		for _, pack := range []string{settings.CommunityPackPotato, settings.CommunityPackMoonlight} {
			if !slices.Contains(next.CommunityPacks, pack) {
				for _, mission := range gamedata.PlayableMissions() {
					if gamedata.MissionPack(mission.ID) == pack && !slices.Contains(next.MvmExcludedMissions, mission.PopFile) {
						next.MvmExcludedMissions = append(next.MvmExcludedMissions, mission.PopFile)
					}
				}
			}
		}
		next.MvmStartClass = ""
		if index := startClassBox.CurrentIndex(); index > 0 {
			next.MvmStartClass = classLabels[index]
		}

		next.SrcdsHostname = strings.TrimSpace(nameEdit.Text())
		next.SrcdsPw = strings.TrimSpace(passEdit.Text())
		next.SrcdsPort = int(portEdit.Value())
		next.SrcdsAdminSteamIDs = strings.TrimSpace(adminEdit.Text())
		next.SrcdsReach = checkedReach(reachSteam, reachPort)
		next.SrcdsToken = strings.TrimSpace(tokenEdit.Text())
		// The Bots tab is built when somebody opens it, so on a visit that
		// never did there is nothing to read: next keeps what the settings
		// came in with, which is what those fields still say.
		if botsBuilt {
			next.SrcdsBots = botsBox.Checked()
			next.SrcdsBotTeamSize = int(botsSize.Value())
			next.BotUpgradesChat = buysBox.Checked()
			next.SrcdsBotHats = looksBox.hats.Checked()
			next.SrcdsBotHatEffects = looksBox.effects.Checked()
			// One description of what the seats and ticks mean, shared with
			// the Save button. A seat left on the draw contributes nothing, so
			// the list is the picked seats in seat order: all six on the draw
			// leaves it empty, which is the mod deciding, as it always did.
			next = settings.WithBotTeam(next, botTeam.team())
		}
		// Saved teams are written when Save is pressed on the tab, so they are
		// on disk already; this keeps them in the settings the dialog returns.
		next.SrcdsBotTeamPresets = botTeam.saved
		// Built loadouts are written when Save is pressed on the page, so they
		// are on disk already; this keeps them in the settings returned.
		next.SrcdsBotCustomLoadouts = botLoadout.built
		if _, err := settings.CheckRunSelection(next); err != nil {
			return next, err
		}
		return next, nil
	}

	// Beta, because the relay half of this tab has never been proved end to
	// end: a server needs a login token before Valve hands out a relayed
	// address, and no run has yet gone from that address to a Team Fortress 2
	// client that joined. The local network and a forwarded port both have.
	extraPages := []declarative.TabPage{
		{
			Title:  "Networking",
			Layout: declarative.Grid{Columns: 2},
			Children: []declarative.Widget{
				label("Who can reach it", "Where the server takes connections from. The local network is the default, because that is what a server with no login token gets anyway: without one it stays there whatever this says."),
				// The three buttons are consecutive children of one Composite
				// on purpose. walk groups a radio button with the sibling
				// before it, so a label in between would leave three groups of
				// one, all tickable.
				declarative.Composite{
					// Near, or a VBox centres each button on its own text and
					// the stack ends up ragged.
					Layout: declarative.VBox{MarginsZero: true, SpacingZero: true, Alignment: declarative.AlignHNearVCenter},
					Children: []declarative.Widget{
						declarative.RadioButton{AssignTo: &reachLan, Text: settings.ReachLan.Label()},
						declarative.RadioButton{AssignTo: &reachSteam, Text: settings.ReachSteam.Label()},
						declarative.RadioButton{AssignTo: &reachPort, Text: settings.ReachPort.Label()},
					},
				},
				declarative.Label{Text: ""},
				declarative.TextLabel{
					AssignTo: &reachHelp,
					Text:     s.SrcdsReach.Help(),
					MinSize:  declarative.Size{Width: 330},
				},
				label("Login token", "A Game Server Login Token for app id 440, from steamcommunity.com/dev/managegameservers. The server logs in to Steam with it. Every reach that leaves your network needs a real one."),
				declarative.LineEdit{AssignTo: &tokenEdit, Text: s.SrcdsToken, CueBanner: "0"},
				declarative.Label{Text: ""},
				declarative.Label{AssignTo: &tokenWarn, Text: "", MaxSize: declarative.Size{Height: 18}},
				// Steam is the only place a token comes from, and the page that
				// issues one asks for two things a player has no way to guess:
				// 440, which is Team Fortress 2, and a memo, which is theirs to
				// choose and is never seen again.
				declarative.LinkLabel{
					AssignTo: &tokenLink,
					// A LinkLabel with no width of its own takes none, and wraps
					// its text one character to a line: the dialog grew to twice
					// the height of the screen with a column of single letters
					// down the middle of it.
					ColumnSpan: 2,
					MaxSize:    declarative.Size{Width: 470},
					Text:       `<a href="` + tokenPageURL + `">Get one from Steam</a>: app id 440, and any memo you like, such as "TF2 Archipelago".`,
					OnLinkActivated: func(link *walk.LinkLabelLink) {
						if err := winproc.OpenURL(link.URL()); err != nil {
							tokenWarn.SetText(err.Error())
						}
					},
				},
				declarative.TextLabel{
					Text: "A forwarded port is the port above, opened on the router to this machine. " +
						"Your friends join on your public address and that port; the local network " +
						"still reaches the server the way it did. Over Steam, no port is opened and " +
						"the server prints its address in the log at every start, in the form " +
						"connect 169.254.13.42:20232. That one is a new address every time, so send " +
						"the line from the log rather than one you wrote down.",
					ColumnSpan: 2,
					MinSize:    declarative.Size{Width: 470},
				},
			},
		},
	}

	built := time.Now()
	err := declarative.Dialog{
		AssignTo:     &dialog,
		Title:        "Settings",
		CancelButton: &cancel,
		// Wider than it was: a seat is a class and the weapons it carries on
		// one line, and the class pool is three ticks to a line with the same
		// beside each of them.
		Size:    declarative.Size{Width: 1180, Height: 720},
		MinSize: declarative.Size{Width: 900, Height: 560},
		Layout:  declarative.VBox{},
		Children: []declarative.Widget{
			declarative.TabWidget{
				AssignTo: &tabs,
				Pages: append([]declarative.TabPage{
					{
						Title:  "Player options",
						Layout: declarative.Grid{Columns: 2},
						Children: []declarative.Widget{
							label("Easiest tier", "The easiest tier a mission may come from. Harder tiers are always in as well, so the pool shrinks as this rises. Expert leaves four, because Valve made only three expert missions and one haunted one."),
							declarative.ComboBox{
								AssignTo:    &tierBox,
								Model:       tierLabels,
								Value:       tierLabel(tiers, s.MvmDifficulty),
								ToolTipText: "The number is the pool to draw from, not the length of a run.",
							},
							label("Missions used", "How many missions this run uses, out of the pool above. Eight is about fifty waves, which is one evening for a team that knows the mode."),
							declarative.NumberEdit{
								AssignTo:    &missions,
								Value:       float64(s.MvmMissionCount),
								MinValue:    1,
								MaxValue:    float64(runshape.MissionsInPool(s.MvmDifficulty)),
								Decimals:    0,
								ToolTipText: "Asking for more than the pool holds gives you the whole pool.",
							},
							label("Goal", "What ends the run. Final Boss marks the hardest mission the run drew, and clearing it wins. Missionsanity asks for a share of the missions instead, in any order."),
							declarative.ComboBox{AssignTo: &goalBox, Model: goalLabels, Value: goalLabel(goals, s.MvmGoal)},
							label("Missionsanity share", "How much of the run Missionsanity asks for, in percent. It rounds up, and the Final Boss goal ignores it."),
							declarative.NumberEdit{
								AssignTo: &sanityPct, Value: float64(s.MvmMissionsanityPct),
								MinValue: 10, MaxValue: 100, Decimals: 0,
							},
							label("Death Link", "A lost wave kills every other player in the multiworld who has Death Link on, and their deaths wipe your team."),
							declarative.CheckBox{AssignTo: &deathLink, Text: "share deaths", Checked: s.MvmDeathLink},
							label("Archipelago app", "Where the Archipelago app is installed. Leave it blank and the launcher looks where the installer puts it. Set it when the app is on another drive, or in a folder of your own."),
							declarative.Composite{
								Layout: declarative.HBox{MarginsZero: true},
								Children: []declarative.Widget{
									declarative.LineEdit{
										AssignTo:  &appEdit,
										Text:      s.ArchipelagoDir,
										CueBanner: defaultAppDir(),
									},
									declarative.PushButton{
										Text:        "Browse",
										MinSize:     declarative.Size{Width: 70},
										ToolTipText: "Pick the folder holding ArchipelagoGenerate.exe.",
										OnClicked:   func() { browseForApp(dialog, appEdit) },
									},
								},
							},
							declarative.Label{
								Text:        "These are the options the Archipelago website calls player options. They go in tf2.yaml, which the seed is generated from. The Missions tab picks which missions the run may draw.",
								ColumnSpan:  2,
								ToolTipText: "Change them here, then generate again for a new seed. The current run keeps the shape it was generated with.",
							},
							declarative.Composite{
								Layout:     declarative.HBox{MarginsZero: true},
								ColumnSpan: 2,
								MaxSize:    declarative.Size{Height: 32},
								Children: []declarative.Widget{
									declarative.PushButton{
										Text:        "Generate seed",
										ToolTipText: "Make the seed with the Archipelago app installed on this machine: the launcher installs the world file into it, writes the player file, runs the generator and opens the folder with the archive. Upload that archive at archipelago.gg/uploads to open a room.",
										OnClicked:   func() { generateSeed(dialog, collect) },
									},
									declarative.PushButton{
										Text:        "Open tf2.yaml",
										ToolTipText: "Write the player file from what is on screen, then open it. Copy it into the Archipelago app's Players folder to generate the seed.",
										OnClicked:   func() { openPlayerFile(dialog, collect) },
									},
									declarative.PushButton{
										Text:        "Open the folder",
										ToolTipText: "The install root: the game files, the player file, the log and the run's state.",
										OnClicked:   func() { openFolder(dialog, s.InstallRoot) },
									},
									declarative.HSpacer{},
								},
							},
						},
					},
					{
						Title:  "Rewards",
						Layout: declarative.Grid{Columns: 2},
						Children: []declarative.Widget{
							label("Mission tickets", "Required tickets gate deployment to each mission. Useful tickets leave all missions drawn by the seed available."),
							declarative.ComboBox{AssignTo: &ticketImportance, Model: importanceLabels, Value: importanceLabel(s.MvmMissionTicketImportance)},
							label("Class unlocks", "Required classes satisfy mission-tier roster checks. Useful classes only expand the playable roster."),
							declarative.ComboBox{AssignTo: &classImportance, Model: importanceLabels, Value: importanceLabel(s.MvmClassUnlockImportance)},
							label("Weapon slots", "Required slots satisfy mission-tier loadout checks. Useful slots only expand the available loadouts."),
							declarative.ComboBox{AssignTo: &slotImportance, Model: importanceLabels, Value: importanceLabel(s.MvmWeaponSlotImportance)},
							label("Weapon buffs", "Required buffs gate increasingly difficult tiers by total buff count. Useful buffs remain optional power-ups."),
							declarative.ComboBox{AssignTo: &buffImportance, Model: importanceLabels, Value: importanceLabel(s.MvmWeaponBuffImportance)},
							label("Cash rewards", "Include temporary MvM credits in spare checks. Off makes every spare check a persistent weapon buff."),
							declarative.CheckBox{AssignTo: &cashRewards, Text: "include cash filler", Checked: s.MvmCashRewards},
							label("Buff share", "Percent of spare checks that award buffs when cash rewards are enabled. The remainder award cash."),
							declarative.NumberEdit{AssignTo: &buffPct, Value: float64(s.MvmWeaponBuffPct), MinValue: 0, MaxValue: 100, Decimals: 0},
							label("Buff stack chance", "Chance that another buff reward adds a level to a numeric buff already in the seed. Toggle effects never repeat."),
							declarative.NumberEdit{AssignTo: &buffStack, Value: float64(s.MvmWeaponBuffStackChance), MinValue: 0, MaxValue: 100, Decimals: 0},
						},
					},
					{
						/* Balancing is what the robots are worth, and rewards
						are what the run hands out. They were one page and the
						page had to be read twice to find either. */
						Title:  "Balancing",
						Layout: declarative.Grid{Columns: 2},
						Children: []declarative.Widget{
							declarative.TextLabel{
								Text: "Valve tunes every wave for six defenders. This takes the robots down for a team that is short of them, " +
									"and applies equally regardless of how many humans are playing.",
								ColumnSpan: 2,
								MaxSize:    declarative.Size{Width: sentenceWidth},
							},
							label("Robot health (%)", "Direct health multiplier for every robot, from 10% to 1000%."),
							declarative.NumberEdit{AssignTo: &bluHealth, Value: float64(s.SrcdsBluHealthPct), MinValue: settings.RobotHealthPercentMin, MaxValue: settings.RobotHealthPercentMax, Decimals: 0},
						},
					},
					{
						Title: "Missions",
						// A grid, not a row of boxes per line: a composite lays
						// itself out on its own, so the start mission menu began
						// at one place and the start class menu at another.
						Layout: declarative.Grid{Columns: 2},
						Children: []declarative.Widget{
							label("Asset pack folder", "Folder containing archive-assets.zip and/or mlarchive-assets.zip. Start never downloads community content."),
							declarative.Composite{
								Layout: declarative.HBox{MarginsZero: true},
								Children: []declarative.Widget{
									declarative.LineEdit{AssignTo: &contentEdit, Text: s.CommunityContentDir},
									declarative.PushButton{Text: "Browse", OnClicked: func() { browseForCommunity(dialog, contentEdit) }},
									declarative.PushButton{
										AssignTo:    &fetchPacks,
										Text:        "Download Selected Community Assets",
										ToolTipText: "Download only the checked full-with-maps community packs. Start never downloads community content.",
										OnClicked: func() {
											downloadSelectedCommunityAssets(owner, dialog, contentEdit, potatoBox, moonlightBox, fetchPacks, fetchStatus, say, refreshCommunityChoices)
										},
									},
									declarative.PushButton{
										Text:        "Use Local Community Assets",
										ToolTipText: "Choose a folder containing existing archive-assets.zip and/or mlarchive-assets.zip files, validate them, and enable the packs found there.",
										OnClicked: func() {
											useLocalCommunityAssets(dialog, contentEdit, potatoBox, moonlightBox, fetchStatus, refreshCommunityChoices)
										},
									},
								},
							},
							declarative.Label{Text: ""},
							declarative.Label{AssignTo: &fetchStatus, Text: "Community maps appear below only after a valid downloaded or local ZIP is available.", TextColor: colorMuted},
							label("Community packs", "Select which locally available packs Start installs and which packs the explicit download button fetches."),
							declarative.Composite{
								Layout: declarative.HBox{MarginsZero: true},
								Children: []declarative.Widget{
									declarative.CheckBox{AssignTo: &potatoBox, Text: "Potato Archive", Checked: potatoSelected, OnCheckedChanged: func() { pool.setPack(settings.CommunityPackPotato, potatoBox.Checked()) }},
									declarative.CheckBox{AssignTo: &moonlightBox, Text: "Moonlight Archive", Checked: moonlightSelected, OnCheckedChanged: func() { pool.setPack(settings.CommunityPackMoonlight, moonlightBox.Checked()) }},
								},
							},
							declarative.Label{Text: "Archipelago capacity", ToolTipText: "Check that the eligible mission pool has enough locations for all mission, class, and weapon-slot unlocks."},
							declarative.PushButton{
								Text: "Check Run Selection",
								OnClicked: func() {
									next, err := collect()
									if err != nil {
										walk.MsgBox(dialog, "Archipelago run selection", err.Error(), walk.MsgBoxIconWarning)
										return
									}
									result, _ := settings.CheckRunSelection(next)
									walk.MsgBox(dialog, "Archipelago run selection", result.Summary(), walk.MsgBoxIconInformation)
								},
							},
							label("Start mission", "Where the run begins, as map - mission. The seed starts there and the server boots there. Any lets the seed draw the easiest mission it took."),
							declarative.ComboBox{
								AssignTo:      &startBox,
								Model:         choiceLabels,
								Value:         startMissionValue,
								StretchFactor: 1,
							},
							label("Start class", "The mercenary the run starts with. The tier of the start mission decides how many classes it starts with, and this names one of them."),
							declarative.ComboBox{
								AssignTo:      &startClassBox,
								Model:         classLabels,
								Value:         runshape.StartClassLabel(s.MvmStartClass),
								StretchFactor: 1,
							},
							declarative.Label{
								Text:        "Missions the run may draw. Untick one to keep it out of every seed generated from here: Caliginous Caper is one wave of 666 robots and an hour on its own. The tier above still applies.",
								ColumnSpan:  2,
								ToolTipText: "This is the excluded_missions option in tf2.yaml.",
							},
							declarative.TableView{
								AssignTo:         &poolView,
								Model:            pool,
								CheckBoxes:       true,
								AlternatingRowBG: true,
								ColumnSpan:       2,
								StretchFactor:    1,
								Columns: []declarative.TableViewColumn{
									{Title: "Mission", Width: 200},
									{Title: "Map", Width: 130},
									{Title: "Source", Width: 110},
									{Title: "Tier", Width: 90},
									{Title: "Waves", Width: 50},
									{Title: "Compatibility", Width: 130},
								},
							},
							declarative.Composite{
								Layout:     declarative.HBox{MarginsZero: true},
								ColumnSpan: 2,
								MaxSize:    declarative.Size{Height: 30},
								Children: []declarative.Widget{
									declarative.PushButton{Text: "All", OnClicked: func() { pool.setAll(true) }},
									declarative.PushButton{Text: "None", OnClicked: func() { pool.setAll(false) }},
									declarative.HSpacer{},
								},
							},
						},
					},
					{
						Title:  "Archipelago room",
						Layout: declarative.Grid{Columns: 2},
						Children: []declarative.Widget{
							label("Test mode", "Play without Archipelago at all. The launcher serves a multiworld of one, makes up a seed from your run options, and hands out an unlock for every wave you clear. Other players are simulated: they find things, send you things and die."),
							declarative.CheckBox{AssignTo: &testBox, Text: "no room, no seed, just play", Checked: s.TestMode},
							label("Room address", "The line from your room page on archipelago.gg: host and port. A room on your own machine works too, as localhost:38281."),
							declarative.LineEdit{AssignTo: &roomEdit, Text: current.String(), CueBanner: "archipelago.gg:12345"},
							declarative.Label{Text: ""},
							declarative.Label{AssignTo: &roomWarn, Text: "", MaxSize: declarative.Size{Height: 18}},
							label("Room password", "Only if the room asks for one. Leave it blank otherwise."),
							declarative.LineEdit{AssignTo: &roomPass, Text: s.APPassword, PasswordMode: true, CueBanner: "optional"},
							label("Slot name", "The name this server plays under in the multiworld. It has to match the name in tf2.yaml, and the launcher keeps the two in step."),
							declarative.LineEdit{AssignTo: &slotEdit, Text: s.APSlotName},
							declarative.Label{
								Text:        "One slot covers the whole game server: everybody playing here shares its unlocks.",
								ColumnSpan:  2,
								ToolTipText: "Another player who wants a slot of their own plays another game in the same room, from the Archipelago app.",
							},
							// Somewhere for the slack to go. Without it the grid
							// shares it between the rows, which put a hand's
							// width of nothing between every field and left the
							// warning floating in the middle of the tab.
							declarative.VSpacer{ColumnSpan: 2},
						},
					},
					{
						Title:  "Game server",
						Layout: declarative.Grid{Columns: 2},
						Children: []declarative.Widget{
							label("Server name", "What the server calls itself in the player list."),
							declarative.LineEdit{AssignTo: &nameEdit, Text: s.SrcdsHostname},
							label("Server password", "What your friends type before connect. Blank means anybody with the address can join."),
							declarative.LineEdit{AssignTo: &passEdit, Text: s.SrcdsPw, CueBanner: "optional, blank for none"},
							label("Game port", "UDP and TCP, 27015 by default. Who can reach it is on the Steam Networking tab."),
							declarative.NumberEdit{
								AssignTo: &portEdit, Value: float64(s.SrcdsPort),
								MinValue: 1024, MaxValue: 65535, Decimals: 0,
							},
							label("Admins by Steam id", "Who may run the admin commands, separated by commas. Either form works: the 17 digit id from a profile URL, or SourceMod's STEAM_0:1:26975537."),
							declarative.LineEdit{AssignTo: &adminEdit, Text: s.SrcdsAdminSteamIDs, CueBanner: "76561198014216803, ..."},
						},
					},
					{
						Title:  "Bots",
						Layout: declarative.VBox{MarginsZero: true},
						/* Built when the tab is first opened, not when the dialog is.
						 *
						 * This one tab is two thirds of the time the window takes to
						 * come up: 569ms with it, 245ms without, and 529ms with its
						 * scroll views taken out, so it is the twenty-two menus on it.
						 * walk already suspends the dialog while it builds, which is
						 * what stops Windows redrawing and laying out per control, so
						 * what is left is the cost of the controls themselves. The
						 * remedy for that is not to make them until somebody looks.
						 *
						 * The dialog opens on Player options, so most visits never pay
						 * for this tab at all.
						 */
						Children: []declarative.Widget{
							declarative.Composite{
								AssignTo: &botsHost,
								Layout:   declarative.VBox{MarginsZero: true},
							},
						},
					},
				}, extraPages...),
			},
			declarative.Composite{
				Layout:  declarative.HBox{},
				MaxSize: declarative.Size{Height: 34},
				Children: []declarative.Widget{
					declarative.PushButton{
						Text:        "Debug logs",
						ToolTipText: "Put the logs, the settings without their passwords, and the player file in one zip, for sending to whoever is helping you.",
						OnClicked:   func() { saveDebugBundle(dialog, s) },
					},
					declarative.PushButton{
						Text:        "Repair",
						ToolTipText: "Throw SteamCMD and the mods away and fetch them again. Keeps the game files and the run.",
						OnClicked:   func() { runRepair(dialog, repair) },
					},
					declarative.PushButton{
						Text:        "Reset settings",
						ToolTipText: "Put every setting back to what a fresh install has. Keeps the game files and where they are.",
						OnClicked:   func() { runReset(dialog, reset) },
					},
					declarative.HSpacer{},
					declarative.PushButton{AssignTo: &accept, Text: "Save", OnClicked: func() {
						// Read every field before the dialog closes: closing it
						// destroys its children, and a destroyed LineEdit reads
						// back empty.
						//
						// The address is checked here rather than as the player
						// types. A disabled button with no explanation is a dead
						// end, and a paste with the mouse sends no keystroke to
						// check on.
						next, err := collect()
						if err != nil {
							roomWarn.SetText(err.Error())
							return
						}
						// A reach that leaves the network with no token is a
						// server every client is refused from, and nothing on
						// screen would say why. Refuse the save instead.
						if complaint := tokenComplaint(next.SrcdsReach, next.SrcdsToken); complaint != "" {
							tokenWarn.SetText(complaint)
							return
						}
						edited = next
						dialog.Accept()
					}},
					declarative.PushButton{AssignTo: &cancel, Text: "Cancel", OnClicked: func() { dialog.Cancel() }},
				},
			},
		},
	}.Create(owner)
	if err != nil {
		return s, false, err
	}
	// How long the window took to build, in the log a debug bundle carries. A
	// settings dialog that takes a second to appear is a complaint, and a
	// complaint without a number is a guess about which of sixty widgets is
	// the expensive one.
	say("the settings window took %s to build", time.Since(built).Round(time.Millisecond))

	// The Bots tab, the first time somebody looks at it.
	tabs.CurrentIndexChanged().Attach(func() {
		// By title, not by index: the tabs a beta flag adds move the numbers.
		page := tabs.Pages().At(tabs.CurrentIndex())
		if botsBuilt || page == nil || page.Title() != "Bots" {
			return
		}
		botsBuilt = true
		started := time.Now()
		if err := buildBotsTab(botsHost, s, label, &botsBox, &botsSize, &buysBox, &looksBox, botTeam, botLoadout); err != nil {
			say("the bots tab did not build: %v", err)
			return
		}
		// botsSize is made here, long after the call above, so it is aligned
		// here or not at all.
		leftAlign(botsSize)
		say("the bots tab took %s to build", time.Since(started).Round(time.Millisecond))
	})

	/* The tab the caller asked for, once the handler above exists.
	 *
	 * After the attach, never before: the Bots tab builds itself the first time
	 * it is selected, and selecting it with nothing listening leaves an empty
	 * page. Change the team on the Bot Switcher opens straight onto it, which
	 * is what its tooltip has always claimed.
	 */
	selectTab(tabs, openOn)

	// Numbers read from the left, like every other field in the dialog.
	leftAlign(missions, sanityPct, buffPct, buffStack, bluHealth, portEdit)

	// The help under the buttons, and the complaint about a missing token. Both
	// follow the selection, because a reach the player cannot use yet has to
	// say so here rather than in the server log twenty minutes later.
	explainReach := func() {
		reach := checkedReach(reachSteam, reachPort)
		reachHelp.SetText(reach.Help())
		tokenWarn.SetText(tokenComplaint(reach, tokenEdit.Text()))
		// Only where a token is worth anything: on the local network the server
		// never logs in, and a link to go and fetch one is an errand for nothing.
		tokenLink.SetVisible(reach.NeedsToken())
	}
	for _, button := range []*walk.RadioButton{reachLan, reachSteam, reachPort} {
		button.CheckedChanged().Attach(explainReach)
	}
	tokenEdit.TextChanged().Attach(explainReach)
	switch s.SrcdsReach {
	case settings.ReachSteam:
		reachSteam.SetChecked(true)
	case settings.ReachPort:
		reachPort.SetChecked(true)
	default:
		reachLan.SetChecked(true)
	}
	explainReach()

	// A harder floor leaves fewer missions to draw from, so the count a run can
	// ask for follows the tier.
	tierBox.CurrentIndexChanged().Attach(func() {
		pool := tiers[max(tierBox.CurrentIndex(), 0)].Missions
		_ = missions.SetRange(1, float64(pool))
		if missions.Value() > float64(pool) {
			_ = missions.SetValue(float64(pool))
		}
	})

	// The complaint under the address: what is missing, or that test mode
	// makes it optional. Cleared as soon as the address looks right.
	explain := func() {
		switch {
		case testBox.Checked():
			roomWarn.SetText("not needed in test mode: the launcher serves its own room")
		case roomEdit.Text() == "":
			roomWarn.SetText("paste the address from your Archipelago room page")
		default:
			if _, err := settings.ParseRoom(roomEdit.Text()); err == nil {
				roomWarn.SetText("")
			}
		}
	}
	roomEdit.TextChanged().Attach(explain)
	testBox.CheckedChanged().Attach(explain)
	explain()

	if dialog.Run() != walk.DlgCmdOK {
		return s, false, nil
	}
	return edited, true, nil
}

/* Put a number field's text against its left edge.
 *
 * walk builds a NumberEdit out of an edit control it creates with ES_RIGHT and
 * does not expose, so there is no way to ask for this through the API. A right
 * aligned number is right for a column of figures and wrong for one field in a
 * form: the value sits a hand's width from its label, against the far edge of
 * the box, and on the Bots tab the scrollbar was over the top of it.
 *
 * The style is changed on the child window rather than by rebuilding anything.
 * Only a child that says it is an edit control: a NumberEdit with its spin
 * buttons on has a second child, and ES_RIGHT is UDS_HORZ to that one.
 */
func leftAlign(edits ...*walk.NumberEdit) {
	for _, edit := range edits {
		if edit == nil {
			continue
		}
		for child := win.GetWindow(edit.Handle(), win.GW_CHILD); child != 0; child = win.GetWindow(child, win.GW_HWNDNEXT) {
			var class [8]uint16
			length, err := win.GetClassName(child, &class[0], len(class))
			if err != nil || length == 0 {
				continue
			}
			if !strings.EqualFold(syscall.UTF16ToString(class[:length]), "edit") {
				continue
			}
			style := win.GetWindowLong(child, win.GWL_STYLE)
			win.SetWindowLong(child, win.GWL_STYLE, style&^win.ES_RIGHT|win.ES_LEFT)
			win.InvalidateRect(child, nil, true)
		}
	}
}

// botsRows is the Bots tab: whether the bots play, how many, which classes
// they may pick, and what each class holds.
/* teamRows is the Team page: three columns of label, class and weapons.
 *
 * One grid for the whole page rather than a row of boxes per seat. A composite
 * per seat lays itself out on its own, so seat 4's class menu ended up a
 * different width from seat 3's and nothing lined up down the page. A grid is
 * what makes a column a column.
 */
func teamRows(
	s settings.Settings, label func(text, help string) declarative.Label,
	botsBox **walk.CheckBox, botsSize **walk.NumberEdit, buysBox **walk.CheckBox,
	team *botTeamEditor,
) []declarative.Widget {
	rows := []declarative.Widget{
		label("Defender bots", "Fill the RED team with bots that play, so a wave balanced for six is winnable by fewer. They pick classes, fight and buy their own upgrades. A bot steps aside when a friend joins."),
		declarative.CheckBox{AssignTo: botsBox, Text: "fill the RED team", Checked: s.SrcdsBots, ColumnSpan: 2},
		label("Fill RED to", "How many players RED holds, humans included. Lower it for a harder run."),
		declarative.NumberEdit{
			AssignTo: botsSize, Value: float64(s.SrcdsBotTeamSize),
			MinValue: 1, MaxValue: 6, Decimals: 0,
			ColumnSpan: 2,
		},
		label("Purchases in chat", "Write what the bots buy at the upgrade station to the chat, since the game no longer lets you inspect a teammate's upgrades. One line per purchase, so it is off by default."),
		declarative.CheckBox{AssignTo: buysBox, Text: "say what the bots buy", Checked: s.BotUpgradesChat, ColumnSpan: 2},
		declarative.TextLabel{
			Text: "The bot team, in order: what each seat plays and what it holds. The first seats are the ones " +
				"filled when RED is short, so put the classes you cannot do without first. A seat left on the " +
				"draw is one the mod picks, from the classes ticked on the next page.",
			ColumnSpan: 3,
			MaxSize:    declarative.Size{Width: sentenceWidth},
		},
	}
	rows = append(rows, presetRow(s, label, team)...)

	choices := make([]string, 0, len(botloadout.Classes)+1)
	choices = append(choices, drawSeatLabel)
	for _, class := range botloadout.Classes {
		choices = append(choices, class.Name)
	}
	for seat := range team.seatBox {
		rows = append(rows,
			label(fmt.Sprintf("Seat %d", seat+1), "What this seat plays and what it carries. Humans take the seats first, so the last ones are rarely filled."),
			declarative.ComboBox{
				AssignTo:              &team.seatBox[seat],
				Model:                 choices,
				Value:                 seatValue(s.SrcdsBotTeamComp, seat),
				MinSize:               declarative.Size{Width: 150},
				OnCurrentIndexChanged: seatClassChanged(seat, team),
			},
			declarative.ComboBox{
				AssignTo:      &team.seatLoadBx[seat],
				Model:         seatLoadoutChoices(team.lib(), seatClassKey(s.SrcdsBotTeamComp, seat)),
				Value:         seatLoadoutValue(team.lib(), s, seat),
				ToolTipText:   "The weapons this seat carries. Follows the class, so a seat on the draw has nothing to hold.",
				StretchFactor: 1,
			},
		)
	}
	return rows
}

// classRows is the Classes page: two pairs of a tick and its weapons to a line,
// in four columns, so the ticks line up down the page and the menus have room.
func classRows(s settings.Settings, team *botTeamEditor) []declarative.Widget {
	rows := []declarative.Widget{declarative.TextLabel{
		Text: "The classes the mod may draw a seat left on the draw from. Bots are poor snipers and spies; " +
			"untick one and they never pick it. The weapons beside a class are what it holds when the seat " +
			"it fills does not say otherwise.",
		ColumnSpan: 4,
		MaxSize:    declarative.Size{Width: sentenceWidth},
	}}
	return append(rows, classCells(s, team)...)
}

type cosmeticBoxes struct {
	hats    *walk.CheckBox
	effects *walk.CheckBox
	skins   *walk.CheckBox
}

// cosmeticRows is what the bots look like. Last on the tab because none of it
// changes a wave: a run everybody is watching is more fun when the six of them
// do not look like the same mercenary six times.
func cosmeticRows(s settings.Settings, looksBox *cosmeticBoxes) []declarative.Widget {
	return []declarative.Widget{
		declarative.TextLabel{
			Text:       "How the bots look. None of this changes how they play.",
			ColumnSpan: 2,
			MaxSize:    declarative.Size{Width: sentenceWidth},
		},
		// The tick says the whole thing, so there is nothing for a label
		// beside it to add: a second "Cosmetic items" would be the same words twice.
		declarative.CheckBox{
			AssignTo:    &looksBox.hats,
			Text:        "Assign a random cosmetic item to each bot",
			Checked:     s.SrcdsBotHats,
			ColumnSpan:  2,
			ToolTipText: "Drawn from the cosmetic items that bot's class can wear, hat or not. It keeps the one it drew for the whole mission, which is how you tell one Heavy from another.",
		},
		declarative.CheckBox{
			AssignTo:    &looksBox.effects,
			Text:        "Assign a random unusual effect to each cosmetic item",
			Checked:     s.SrcdsBotHatEffects,
			ColumnSpan:  2,
			ToolTipText: "Six particle effects on screen for the whole wave.",
		},
	}
}

/* botTeamEditor is the Bots tab's team, as the widgets hold it.
 *
 * Load and Save are two buttons that read and write every seat, every class
 * tick and every weapons menu at once, so they need all of them in one place.
 * Passing nine slices to two callbacks is what this is instead of.
 */
type botTeamEditor struct {
	seatBox    []*walk.ComboBox
	seatLoadBx []*walk.ComboBox
	classBox   []*walk.CheckBox
	loadoutBx  []*walk.ComboBox
	presetBox  **walk.ComboBox
	presetName **walk.LineEdit

	// saved is the teams as they stand. Save and Load write and read this, and
	// keep is what puts it on disk: a saved team outlives the dialog whatever
	// happens to the rest of the fields, because saving a team and then
	// pressing Cancel because you changed your mind about a port is not asking
	// for the team to be forgotten.
	saved map[string]settings.BotTeam
	keep  func(map[string]settings.BotTeam)

	// library is asked rather than held, because the loadouts it offers are
	// the ones the Loadouts page has right now: build one there and the menus
	// here list it without leaving the dialog.
	library func() botloadout.Library
}

// choices is what one class's loadout menu holds, and the list every index
// read back out of that menu means.
func (e *botTeamEditor) choices(class botloadout.Class) []botloadout.Loadout {
	if e.library == nil {
		return botloadout.Library{}.Choices(class)
	}
	return e.library().Choices(class)
}

// team is what the widgets currently describe. The widgets are read into
// numbers and botTeamFrom says what they mean, because that half has rules
// worth testing and this half needs a window.
func (e *botTeamEditor) team() settings.BotTeam {
	picks := botTeamPicks{
		SeatClass:    make([]int, len(e.seatBox)),
		SeatLoadout:  make([]int, len(e.seatBox)),
		Ticked:       make([]bool, len(botloadout.Classes)),
		ClassLoadout: make([]int, len(botloadout.Classes)),
	}
	for seat, box := range e.seatBox {
		picks.SeatClass[seat] = box.CurrentIndex()
		picks.SeatLoadout[seat] = e.seatLoadBx[seat].CurrentIndex()
	}
	for i := range botloadout.Classes {
		picks.Ticked[i] = e.classBox[i].Checked()
		picks.ClassLoadout[i] = e.loadoutBx[i].CurrentIndex()
	}
	return botTeamFrom(picks, e.lib())
}

/* refreshLoadouts refills every loadout menu on the Team and Classes pages,
 * leaving each on the loadout it already names.
 *
 * The Loadouts page changes what these menus can offer, and a walk ComboBox
 * keeps the model it was built with. Without this, a loadout only reached the
 * menus the next time the dialog opened: assigning one to a bot meant saving
 * every other setting first, which is not what saving a loadout asked for.
 */
func (e *botTeamEditor) refreshLoadouts() {
	library := e.lib()
	for i, class := range botloadout.Classes {
		refillLoadouts(e.loadoutBx[i], library.Choices(class))
	}
	for seat, box := range e.seatBox {
		if box == nil || seat >= len(e.seatLoadBx) {
			continue
		}
		// A seat on the draw holds the one entry that says so, and no class to
		// list loadouts for.
		class, found := botloadout.ClassByKey(classKeyOfLabel(box.Text()))
		if !found {
			continue
		}
		refillLoadouts(e.seatLoadBx[seat], library.Choices(class))
	}
}

// refillLoadouts puts a new list into one menu and puts the selection back.
func refillLoadouts(box *walk.ComboBox, choices []botloadout.Loadout) {
	if box == nil {
		return
	}
	was := box.Text()
	labels := make([]string, 0, len(choices))
	for _, loadout := range choices {
		labels = append(labels, loadout.Label())
	}
	_ = box.SetModel(labels)
	// By index, not by text: SetText on a drop-down list is a no-op, and a
	// menu with nothing selected draws empty.
	_ = box.SetCurrentIndex(reselectLoadout(was, choices))
}

// lib is the loadouts this editor can offer, and the built-in presets alone
// when nothing wired one in.
func (e *botTeamEditor) lib() botloadout.Library {
	if e.library == nil {
		return botloadout.Library{}
	}
	return e.library()
}

// show puts a team into the widgets.
func (e *botTeamEditor) show(team settings.BotTeam) {
	for seat, box := range e.seatBox {
		name, loadout := drawSeatLabel, drawSeatLoadoutLabel
		if seat < len(team.Comp) {
			if class, found := botloadout.ClassByKey(team.Comp[seat]); found {
				name = class.Name
				key := ""
				if seat < len(team.SeatLoadouts) {
					key = team.SeatLoadouts[seat]
				}
				loadout = e.lib().Loadout(class, key).Label()
			}
		}
		selectInCombo(box, name)
		_ = e.seatLoadBx[seat].SetModel(seatLoadoutChoices(e.lib(), classKeyOfLabel(name)))
		selectInCombo(e.seatLoadBx[seat], loadout)
	}
	for i, class := range botloadout.Classes {
		e.classBox[i].SetChecked(!slices.Contains(team.Blacklist, class.Key))
		selectInCombo(e.loadoutBx[i], e.lib().Loadout(class, team.ClassLoadouts[class.Key]).Label())
	}
}

/* selectInCombo picks an entry in a menu by what it says.
 *
 * SetText is the obvious way and it does nothing at all here. These menus are
 * drop-down lists, which have no edit field, and SetWindowText on one of those
 * is a documented no-op: Load appeared to do nothing because every widget it
 * wrote to ignored it.
 *
 * The selection is CB_SETCURSEL, which walk spells SetCurrentIndex, and an
 * index is what a list has instead of text.
 */
func selectInCombo(box *walk.ComboBox, value string) {
	if box == nil {
		return
	}
	model, ok := box.Model().([]string)
	if !ok {
		return
	}
	if index := slices.Index(model, value); index >= 0 {
		_ = box.SetCurrentIndex(index)
	}
}

// comboValue is what a menu is showing, by its model rather than its window
// text: the same drop-down list that ignores SetText is one whose text is the
// selection, and reading the model says so without depending on that.
func comboValue(box *walk.ComboBox) string {
	if box == nil {
		return ""
	}
	model, ok := box.Model().([]string)
	index := box.CurrentIndex()
	if !ok || index < 0 || index >= len(model) {
		return ""
	}
	return model[index]
}

// load brings back a saved team, or says there is none by that name.
func (e *botTeamEditor) load() {
	name := strings.TrimSpace(comboValue(*e.presetBox))
	team, found := e.saved[name]
	if !found {
		return
	}
	e.show(team)
}

// save keeps the team under the name in the box, and puts the name in the menu
// so the next Load can find it.
func (e *botTeamEditor) save() {
	// The name box first: it is where a new name is typed. Blank means save
	// over whichever team is picked in the menu, which is what somebody
	// editing one they loaded is doing.
	name := strings.TrimSpace((*e.presetName).Text())
	if name == "" {
		name = strings.TrimSpace(comboValue(*e.presetBox))
	}
	if name == "" {
		return
	}
	if e.saved == nil {
		e.saved = map[string]settings.BotTeam{}
	}
	e.saved[name] = e.team()
	_ = (*e.presetName).SetText("")
	e.refreshNames(name)
}

/* remove throws away the team the menu is showing.
 *
 * The menu and not the name box: removing what somebody has typed but not
 * saved would be removing nothing, and removing a team by typing its name
 * exactly is a way to delete the wrong one.
 */
func (e *botTeamEditor) remove() {
	name := strings.TrimSpace(comboValue(*e.presetBox))
	if name == "" {
		return
	}
	delete(e.saved, name)
	e.refreshNames("")
}

// refreshNames puts the saved teams back in the menu, selects one, and writes
// them to disk. Shared by Save and Remove: both change the same three things,
// and a menu that disagrees with what is saved is worse than either.
func (e *botTeamEditor) refreshNames(selected string) {
	names := make([]string, 0, len(e.saved))
	for saved := range e.saved {
		names = append(names, saved)
	}
	slices.Sort(names)

	_ = (*e.presetBox).SetModel(names)
	switch {
	case len(names) == 0:
		// Nothing left to show. SetModel has already emptied the menu.
	case slices.Contains(names, selected):
		_ = (*e.presetBox).SetCurrentIndex(slices.Index(names, selected))
	default:
		// The one that was showing is gone, so the menu opens on the first of
		// what is left rather than on nothing.
		_ = (*e.presetBox).SetCurrentIndex(0)
	}
	if e.keep != nil {
		e.keep(e.saved)
	}
}

/* botsScrollPage is one page of the Bots tab: a grid of columns wide, in a
 * view that scrolls when the rows outgrow the window.
 *
 * The vertical scrollbar sits over the right edge of the rows rather than
 * beside them, so the last characters of every row went under it: the team
 * size read as nothing at all, because the number in a NumberEdit is against
 * its right edge. The margin is the room the scrollbar takes, asked of the
 * system rather than guessed, because it is a different number on a display
 * that scales.
 */
func botsScrollPage(title string, columns int, rows []declarative.Widget) declarative.TabPage {
	return declarative.TabPage{
		Title:  title,
		Layout: declarative.VBox{MarginsZero: true},
		Children: []declarative.Widget{
			declarative.ScrollView{
				// Not HorizontalFixed. Fixed, the view sizes itself to its
				// content and then holds it against the scrollbar, with a
				// hand's width of nothing down the left.
				HorizontalFixed: false,
				Layout: declarative.Grid{
					Columns: columns,
					Margins: declarative.Margins{
						Left:   9,
						Top:    9,
						Right:  9 + int(win.GetSystemMetrics(win.SM_CXVSCROLL)),
						Bottom: 9,
					},
				},
				// The slack goes at the bottom. Without somewhere to put it a
				// grid hands it to the rows themselves, and a page of three
				// rows had its two settings against the bottom of the window
				// with the sentence explaining them at the top.
				Children: append(rows, declarative.VSpacer{ColumnSpan: columns}),
			},
		},
	}
}

// firstName is what a menu of saved teams opens on, and "" for none saved.
func firstName(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return names[0]
}

// classCells is the class pool: every class as a tick and the weapons it holds.
func classCells(s settings.Settings, team *botTeamEditor) []declarative.Widget {
	var cells []declarative.Widget
	for i := range botloadout.Classes {
		class := botloadout.Classes[i]
		choices := team.choices(class)
		labels := make([]string, 0, len(choices))
		for _, loadout := range choices {
			labels = append(labels, loadout.Label())
		}
		cells = append(cells,
			declarative.CheckBox{
				AssignTo:    &team.classBox[i],
				Text:        class.Name,
				Checked:     !slices.Contains(s.SrcdsBotClassBlacklist, class.Key),
				MinSize:     declarative.Size{Width: 86},
				ToolTipText: "Unticked, the mod never draws " + class.Name + ".",
			},
			declarative.ComboBox{
				AssignTo:      &team.loadoutBx[i],
				Model:         labels,
				Value:         team.lib().Loadout(class, s.SrcdsBotLoadouts[class.Key]).Label(),
				ToolTipText:   "What a " + class.Name + " holds when the seat it fills does not say otherwise.",
				MinSize:       declarative.Size{Width: 220},
				StretchFactor: 1,
			},
		)
	}
	return cells
}

// botSeats is how many bots RED can hold, which is the team size the Bots tab
// caps at. A seat past the team size is simply never filled.
const botSeats = 6

// presetRow is the saved teams: pick one and load it, or name one and keep it.
func presetRow(
	s settings.Settings, label func(text, help string) declarative.Label, team *botTeamEditor,
) []declarative.Widget {
	return []declarative.Widget{
		label("Saved teams", "A team is the seats, their weapons and the classes the mod may draw from. Type a name and press Save to keep the team below; pick one and press Load to bring it back."),
		// Both remaining columns: a row that leaves a cell empty puts the next
		// row's first widget in it, which walked every seat label one column
		// to the right and wrapped its menus onto the line below.
		declarative.Composite{
			ColumnSpan: 2,
			Layout:     declarative.HBox{MarginsZero: true},
			Children: []declarative.Widget{
				declarative.ComboBox{
					AssignTo: team.presetBox,
					Model:    settings.BotTeamNames(s),
					// Opened on the first saved team rather than on nothing: a
					// menu showing an empty box next to a Load button reads as
					// having nothing saved at all.
					Value:         firstName(settings.BotTeamNames(s)),
					StretchFactor: 1,
					ToolTipText:   "The teams you have saved. Pick one and press Load.",
				},
				declarative.LineEdit{
					AssignTo:      team.presetName,
					CueBanner:     "name this team",
					StretchFactor: 1,
					ToolTipText:   "The name Save keeps this team under. Blank saves over the one picked on the left.",
				},
				declarative.PushButton{
					Text:        "Load",
					ToolTipText: "Put the named team into the seats below.",
					MinSize:     declarative.Size{Width: 70},
					OnClicked:   team.load,
				},
				declarative.PushButton{
					Text:        "Save",
					ToolTipText: "Keep the team below under the name in the box.",
					MinSize:     declarative.Size{Width: 70},
					OnClicked:   team.save,
				},
				declarative.PushButton{
					Text:        "Remove",
					ToolTipText: "Throw away the team the menu is showing. The seats below are left as they are.",
					MinSize:     declarative.Size{Width: 70},
					OnClicked:   team.remove,
				},
			},
		},
	}
}

/* The weapons a seat may carry follow the class it plays.
 *
 * A menu of every class's loadouts would offer a Medic the Gunslinger. The
 * choice is thrown away when the class changes, because a loadout for a class
 * this seat no longer plays is not a choice anybody made.
 */
func seatLoadoutChoices(library botloadout.Library, classKey string) []string {
	class, found := botloadout.ClassByKey(classKey)
	if !found {
		return []string{drawSeatLoadoutLabel}
	}
	choices := library.Choices(class)
	labels := make([]string, 0, len(choices))
	for _, loadout := range choices {
		labels = append(labels, loadout.Label())
	}
	return labels
}

// drawSeatLoadoutLabel is what a seat with no class of its own can hold, which
// is nothing to choose: the mod draws the class and the class holds its own.
const drawSeatLoadoutLabel = "follows the class"

func seatClassKey(comp []string, seat int) string {
	if seat >= len(comp) {
		return ""
	}
	return comp[seat]
}

func seatLoadoutValue(library botloadout.Library, s settings.Settings, seat int) string {
	class, found := botloadout.ClassByKey(seatClassKey(s.SrcdsBotTeamComp, seat))
	if !found {
		return drawSeatLoadoutLabel
	}
	key := ""
	if seat < len(s.SrcdsBotSeatLoadouts) {
		key = s.SrcdsBotSeatLoadouts[seat]
	}
	return library.Loadout(class, key).Label()
}

// seatClassChanged refills that seat's weapons menu, because the old one
// belongs to a class this seat no longer plays.
func seatClassChanged(seat int, team *botTeamEditor) walk.EventHandler {
	return func() {
		box, loadout := team.seatBox[seat], team.seatLoadBx[seat]
		if box == nil || loadout == nil {
			return
		}
		_ = loadout.SetModel(seatLoadoutChoices(team.lib(), classKeyOfLabel(box.Text())))
		// By index, not by text. Setting the text of a menu that has a model
		// selects nothing, and nothing shows as an empty box: a seat whose
		// class had just changed looked like it held no weapons at all.
		// Index zero is the class's stock loadout.
		_ = loadout.SetCurrentIndex(0)
	}
}

// classKeyOfLabel is the mod's spelling of a class named in a menu.
func classKeyOfLabel(name string) string {
	for _, class := range botloadout.Classes {
		if class.Name == name {
			return class.Key
		}
	}
	return ""
}

// drawSeatLabel is the first entry of every seat menu: the mod picks.
const drawSeatLabel = "Let the mod pick"

// seatValue is the label the seat menu opens on.
func seatValue(comp []string, seat int) string {
	if seat >= len(comp) {
		return drawSeatLabel
	}
	for _, class := range botloadout.Classes {
		if class.Key == comp[seat] {
			return class.Name
		}
	}
	return drawSeatLabel
}

// poolModel is the Missions tab's table: every mission the tables know, ticked
// when the run may draw it. The unticked ones are the excluded_missions
// option.
type poolModel struct {
	walk.TableModelBase
	missions       []gamedata.Mission
	inPool         []bool
	excluded       []string
	enabledPacks   []string
	availablePacks []string
}

func newPoolModel(excluded, enabledPacks, availablePacks []string) *poolModel {
	model := &poolModel{
		excluded:       slices.Clone(excluded),
		enabledPacks:   slices.Clone(enabledPacks),
		availablePacks: slices.Clone(availablePacks),
	}
	model.rebuild(nil)
	return model
}

func (m *poolModel) rebuild(previous map[string]bool) {
	m.missions = runshape.VisibleMissions(m.availablePacks)
	m.inPool = make([]bool, len(m.missions))
	for i, mission := range m.missions {
		if held, ok := previous[mission.PopFile]; ok {
			m.inPool[i] = held
			continue
		}
		pack := gamedata.MissionPack(mission.ID)
		m.inPool[i] = gamedata.IsPlayableMission(mission.ID) &&
			!slices.Contains(m.excluded, mission.PopFile) &&
			(pack == "" || slices.Contains(m.enabledPacks, pack))
	}
}

func (m *poolModel) RowCount() int { return len(m.missions) }

func (m *poolModel) Value(row, col int) any {
	mission := m.missions[row]
	switch col {
	case 0:
		return mission.Name
	case 1:
		played, _ := gamedata.MapByID(mission.Map)
		return played.Name
	case 2:
		switch gamedata.MissionPack(mission.ID) {
		case settings.CommunityPackPotato:
			return "Potato Archive"
		case settings.CommunityPackMoonlight:
			return "Moonlight Archive"
		}
		return "Valve"
	case 3:
		return mission.Difficulty.String()
	case 4:
		return int(mission.Waves)
	default:
		return gamedata.RequirementLabel(gamedata.MissionRequirement(mission.ID))
	}
}

func (m *poolModel) StyleCell(style *walk.CellStyle) {
	if !gamedata.IsPlayableMission(m.missions[style.Row()].ID) {
		style.TextColor = colorStopped
	}
}

func browseForCommunity(owner walk.Form, edit *walk.LineEdit) bool {
	dialog := walk.FileDialog{Title: "Where are the asset ZIPs?", InitialDirPath: strings.TrimSpace(edit.Text())}
	accepted, err := dialog.ShowBrowseFolder(owner)
	if err != nil {
		walk.MsgBox(owner, "Community content", err.Error(), walk.MsgBoxIconError)
		return false
	}
	if accepted && dialog.FilePath != "" {
		_ = edit.SetText(dialog.FilePath)
		return true
	}
	return false
}

func communityPackNames(paths []string) []string {
	var packs []string
	for _, path := range paths {
		name := filepath.Base(path)
		if name == settings.CommunityPackPotato || name == settings.CommunityPackMoonlight {
			packs = append(packs, name)
		}
	}
	return packs
}

func selectedCommunityArchives(folder string, potato, moonlight bool) []string {
	selected := settings.Settings{CommunityContentDir: folder}
	if potato {
		selected.CommunityPacks = append(selected.CommunityPacks, settings.CommunityPackPotato)
	}
	if moonlight {
		selected.CommunityPacks = append(selected.CommunityPacks, settings.CommunityPackMoonlight)
	}
	return settings.CommunityArchives(selected)
}

func downloadSelectedCommunityAssets(
	sync walk.Form, dialog *walk.Dialog, folderEdit *walk.LineEdit,
	potato, moonlight *walk.CheckBox, button *walk.PushButton, status *walk.Label,
	say func(string, ...any), ready func([]string),
) {
	folder := strings.TrimSpace(folderEdit.Text())
	if folder == "" {
		walk.MsgBox(dialog, "Download community assets", "Choose an asset pack folder first.", walk.MsgBoxIconWarning)
		return
	}
	archives := selectedCommunityArchives(folder, potato.Checked(), moonlight.Checked())
	if len(archives) == 0 {
		walk.MsgBox(dialog, "Download community assets", "Check at least one community pack first.", walk.MsgBoxIconWarning)
		return
	}
	button.SetEnabled(false)
	status.SetTextColor(colorStarting)
	_ = status.SetText("Downloading the selected full-with-maps packs; progress is also in the main log.")

	go apruntime.Guard("a settings task", func(t string) { say("%s", t) }, func() {
		logf := func(format string, args ...any) {
			message := fmt.Sprintf(format, args...)
			say("community assets: %s", message)
			sync.Synchronize(func() {
				if !dialog.IsDisposed() {
					_ = status.SetText(message)
				}
			})
		}
		err := installer.DownloadCommunityArchives(context.Background(), archives, logf)
		sync.Synchronize(func() {
			if dialog.IsDisposed() {
				return
			}
			button.SetEnabled(true)
			if err != nil {
				status.SetTextColor(colorStopped)
				_ = status.SetText("Download failed: " + err.Error())
				walk.MsgBox(dialog, "Download community assets", err.Error(), walk.MsgBoxIconError)
				return
			}
			packs := communityPackNames(installer.AvailableCommunityArchives(settings.KnownCommunityArchives(folder)))
			ready(packs)
			status.SetTextColor(colorRunning)
			_ = status.SetText("Selected community packs are ready in " + folder)
		})
	})
}

func useLocalCommunityAssets(
	dialog *walk.Dialog, folderEdit *walk.LineEdit, potato, moonlight *walk.CheckBox,
	status *walk.Label, ready func([]string),
) {
	if !browseForCommunity(dialog, folderEdit) {
		return
	}
	folder := strings.TrimSpace(folderEdit.Text())
	paths := installer.AvailableCommunityArchives(settings.KnownCommunityArchives(folder))
	packs := communityPackNames(paths)
	if len(packs) == 0 {
		status.SetTextColor(colorStopped)
		_ = status.SetText("No valid archive-assets.zip or mlarchive-assets.zip was found in " + folder)
		return
	}
	potato.SetChecked(slices.Contains(packs, settings.CommunityPackPotato))
	moonlight.SetChecked(slices.Contains(packs, settings.CommunityPackMoonlight))
	ready(packs)
	status.SetTextColor(colorRunning)
	_ = status.SetText("Using local community packs from " + folder)
}

func (m *poolModel) Checked(row int) bool { return m.inPool[row] }

func (m *poolModel) SetChecked(row int, checked bool) error {
	if !gamedata.IsPlayableMission(m.missions[row].ID) {
		m.inPool[row] = false
		m.PublishRowsReset()
		return nil
	}
	m.inPool[row] = checked
	return nil
}

func (m *poolModel) setAll(checked bool) {
	for i := range m.inPool {
		m.inPool[i] = checked && gamedata.IsPlayableMission(m.missions[i].ID)
	}
	m.PublishRowsReset()
}

func (m *poolModel) setPack(pack string, checked bool) {
	m.enabledPacks = slices.DeleteFunc(m.enabledPacks, func(name string) bool { return name == pack })
	if checked {
		m.enabledPacks = append(m.enabledPacks, pack)
	}
	for _, mission := range gamedata.PlayableMissions() {
		if gamedata.MissionPack(mission.ID) != pack {
			continue
		}
		m.excluded = slices.DeleteFunc(m.excluded, func(popFile string) bool { return popFile == mission.PopFile })
		if !checked {
			m.excluded = append(m.excluded, mission.PopFile)
		}
	}
	for i, mission := range m.missions {
		if gamedata.MissionPack(mission.ID) == pack && gamedata.IsPlayableMission(mission.ID) {
			m.inPool[i] = checked
		}
	}
	m.PublishRowsReset()
}

func (m *poolModel) setAvailablePacks(packs []string) {
	previous := make(map[string]bool, len(m.missions))
	for i, mission := range m.missions {
		previous[mission.PopFile] = m.inPool[i]
	}
	m.availablePacks = slices.Clone(packs)
	m.rebuild(previous)
	m.PublishRowsReset()
}

// excluded is the popfiles the player unticked, in table order.
func (m *poolModel) excludedMissions() []string {
	var out []string
	for i, mission := range m.missions {
		if gamedata.IsPlayableMission(mission.ID) && !m.inPool[i] {
			out = append(out, mission.PopFile)
		}
	}
	for _, mission := range gamedata.PlayableMissions() {
		if gamedata.MissionPack(mission.ID) != "" && !slices.ContainsFunc(m.missions, func(visible gamedata.Mission) bool {
			return visible.ID == mission.ID
		}) {
			out = append(out, mission.PopFile)
		}
	}
	return out
}

// generateSeed makes the seed with the Archipelago app and opens the folder the
// archive landed in. It runs off the UI thread and reports through message
// boxes, because the dialog is modal and the log view is behind it.
func generateSeed(owner walk.Form, collect func() (settings.Settings, error)) {
	next, err := collect()
	if err != nil {
		walk.MsgBox(owner, "Generate seed", err.Error(), walk.MsgBoxIconError)
		return
	}
	if _, err := generate.FindApp(next.ArchipelagoDir); err != nil {
		walk.MsgBox(owner, "Generate seed",
			"The Archipelago app was not found.\n\nThe launcher looked in:\n"+
				"    "+strings.Join(generate.SearchPath(next.ArchipelagoDir), "\n    ")+
				"\n\nIf the app is somewhere else, put its folder in Archipelago app "+
				"above and press this again. If it is not installed, get it from "+
				"github.com/ArchipelagoMW/Archipelago/releases.",
			walk.MsgBoxIconWarning)
		return
	}

	var lines []string
	result, err := generate.Run(context.Background(), generate.Options{
		Settings:           next,
		AppDir:             next.ArchipelagoDir,
		Apworld:            assets.Apworld(),
		ArchipelagoVersion: assets.ArchipelagoVersion,
		Log:                func(line string) { lines = append(lines, line) },
	})
	if err != nil {
		tail := lines
		if len(tail) > 12 {
			tail = tail[len(tail)-12:]
		}
		walk.MsgBox(owner, "Generate seed",
			err.Error()+"\n\n"+strings.Join(tail, "\n"), walk.MsgBoxIconError)
		return
	}
	walk.MsgBox(owner, "Generate seed",
		"Seed written to\n"+result.Archive+"\n\nUpload it at archipelago.gg/uploads to open a room, "+
			"then paste the room address into the Archipelago room tab.",
		walk.MsgBoxIconInformation)
	_ = winproc.Open(filepath.Dir(result.Archive))
}

// openPlayerFile writes the player file from what is on screen and opens it.
// Writing first is the point: a player who edits the options then presses this
// wants to see those options, not the ones from the last save.
func openPlayerFile(owner walk.Form, collect func() (settings.Settings, error)) {
	next, err := collect()
	if err != nil {
		walk.MsgBox(owner, "Open tf2.yaml", err.Error(), walk.MsgBoxIconError)
		return
	}
	path, err := settings.WritePlayerFile(next, assets.ArchipelagoVersion)
	if err != nil {
		walk.MsgBox(owner, "Open tf2.yaml", err.Error(), walk.MsgBoxIconError)
		return
	}
	if err := winproc.Open(path); err != nil {
		walk.MsgBox(owner, "Open tf2.yaml", err.Error(), walk.MsgBoxIconError)
	}
}

// browseForApp asks for the app's folder and puts it in the box. The dialog
// starts wherever the box points, so a second try opens where the first one
// left off rather than at the desktop.
func browseForApp(owner walk.Form, edit *walk.LineEdit) {
	dialog := walk.FileDialog{
		Title:          "Where is the Archipelago app?",
		InitialDirPath: strings.TrimSpace(edit.Text()),
	}
	accepted, err := dialog.ShowBrowseFolder(owner)
	if err != nil {
		walk.MsgBox(owner, "Archipelago app", err.Error(), walk.MsgBoxIconError)
		return
	}
	if !accepted || dialog.FilePath == "" {
		return
	}
	_ = edit.SetText(dialog.FilePath)
}

// defaultAppDir is the first place the launcher looks, shown as the box's
// placeholder so a blank field says what it means.
func defaultAppDir() string {
	if dirs := generate.SearchPath(""); len(dirs) > 0 {
		return dirs[0]
	}
	return ""
}

func openFolder(owner walk.Form, path string) {
	if err := winproc.Open(path); err != nil {
		walk.MsgBox(owner, "Open the folder", err.Error(), walk.MsgBoxIconError)
	}
}

// saveDebugBundle writes the zip a play-tester sends on, and opens the folder
// so they can find it.
func saveDebugBundle(owner walk.Form, s settings.Settings) {
	path, err := debugbundle.Write(s, assets.Versions(), time.Now())
	if err != nil {
		walk.MsgBox(owner, "Debug logs", err.Error(), walk.MsgBoxIconError)
		return
	}
	walk.MsgBox(owner, "Debug logs",
		"Wrote "+path+"\n\nIt holds this run's launcher log and the one before "+
			"it, the SourceMod logs, the server console, what the bridge says "+
			"about the run, the player file and the settings. The passwords are "+
			"not in it.",
		walk.MsgBoxIconInformation)
	_ = winproc.Open(filepath.Dir(path))
}

// runRepair throws away SteamCMD, the mods and Steam's download record, for a
// player whose install will not go through. The next Start puts them back.
//
// The caller stops the server and any install first, so the button works on the
// first press rather than the third.
func runRepair(owner walk.Form, repair func() ([]string, error)) {
	answer := walk.MsgBox(owner, "Repair",
		"This stops the server, then removes SteamCMD, the mods and Steam's "+
			"record of the download. The next Start fetches them again.\n\n"+
			"It keeps the game files and the run: no 14 GB download, no lost checks.",
		walk.MsgBoxOKCancel|walk.MsgBoxIconQuestion)
	if answer != walk.DlgCmdOK {
		return
	}
	removed, err := repair()
	switch {
	case err != nil:
		walk.MsgBox(owner, "Repair", err.Error(), walk.MsgBoxIconError)
	case len(removed) == 0:
		walk.MsgBox(owner, "Repair", "Nothing to remove.", walk.MsgBoxIconInformation)
	default:
		walk.MsgBox(owner, "Repair",
			"Removed:\n"+strings.Join(removed, "\n")+"\n\nPress Start when you are ready.",
			walk.MsgBoxIconInformation)
	}
}

// runReset puts every setting back to its default, for a player whose answers
// have drifted somewhere they cannot see and cannot undo.
//
// It closes the dialog rather than redrawing it. Every field on screen still
// holds the old answer, and Save would write all of them straight back over
// the reset.
func runReset(owner walk.Form, reset func() error) {
	answer := walk.MsgBox(owner, "Reset settings",
		"This puts every setting back to what a fresh install has: the room, "+
			"the server name and its passwords, the missions, the bots, who can join.\n\n"+
			"It keeps the game files and where they are, so nothing is downloaded again.",
		walk.MsgBoxOKCancel|walk.MsgBoxIconQuestion)
	if answer != walk.DlgCmdOK {
		return
	}
	if err := reset(); err != nil {
		walk.MsgBox(owner, "Reset settings", err.Error(), walk.MsgBoxIconError)
		return
	}
	walk.MsgBox(owner, "Reset settings",
		"Every setting is back to its default. Open Settings again to go through them.",
		walk.MsgBoxIconInformation)
	owner.(*walk.Dialog).Cancel()
}

// tokenComplaint says what is wrong with a reach and a token together, or ""
// when they go together. One sentence, shown under the token field and checked
// again on Save, so the answer is the same in both places.
func tokenComplaint(reach settings.Reach, token string) string {
	if reach.NeedsToken() && !settings.HasToken(token) {
		return "this one needs a login token, or every player is refused"
	}
	return ""
}

// checkedReach reads the selection back. Nothing checked means the private
// default, which is the answer that cannot open a server by mistake.
func checkedReach(steam, port *walk.RadioButton) settings.Reach {
	switch {
	case steam.Checked():
		return settings.ReachSteam
	case port.Checked():
		return settings.ReachPort
	default:
		return settings.ReachLan
	}
}

func tierLabel(tiers []runshape.Tier, key string) string {
	for _, tier := range tiers {
		if tier.Key == key {
			return tier.Label()
		}
	}
	if len(tiers) == 0 {
		return ""
	}
	return tiers[0].Label()
}

func goalLabel(goals []runshape.Goal, key string) string {
	for _, goal := range goals {
		if goal.Key == key {
			return goal.Label()
		}
	}
	return fmt.Sprintf("%v", goals[0].Label())
}

/* keepBotTeams writes the saved teams to the config file, on their own.
 *
 * Everything else in this dialog is written when it is closed with Save, which
 * is right for a field somebody is still editing and wrong for a team they have
 * just named: saving one and then pressing Cancel, because the port was a typo,
 * threw the team away with the typo.
 *
 * The file is read back and only this key is replaced, so a launcher writing a
 * team does not overwrite whatever else has changed on disk since it opened.
 */
// keepBotLoadouts writes the built loadouts to the config file, on their own,
// for the reason keepBotTeams does the same for a team.
func keepBotLoadouts(built map[string]botloadout.Built) {
	current, err := settings.Load()
	if err != nil {
		return
	}
	current.SrcdsBotCustomLoadouts = built
	_ = settings.Save(current)
}

func keepBotTeams(teams map[string]settings.BotTeam) {
	current, err := settings.Load()
	if err != nil {
		return
	}
	current.SrcdsBotTeamPresets = teams
	_ = settings.Save(current)
}

/* buildBotsTab fills the Bots tab, the first time it is looked at.
 *
 * The same three pages the tab always had, made when they are wanted rather
 * than when the dialog opens. declarative.NewBuilder is what lets a tree be
 * built into a container that already exists, which is the whole trick.
 *
 * Suspended while it happens, for the reason walk suspends the dialog while it
 * builds one: a container that is already on screen lays itself out and repaints
 * per control added otherwise, and this one is on screen by definition.
 */
func buildBotsTab(
	host *walk.Composite, s settings.Settings, label func(text, help string) declarative.Label,
	botsBox **walk.CheckBox, botsSize **walk.NumberEdit, buysBox **walk.CheckBox,
	looksBox *cosmeticBoxes, team *botTeamEditor, built *botLoadoutEditor,
) error {
	if host == nil {
		return fmt.Errorf("the bots tab has nowhere to go")
	}
	host.SetSuspended(true)
	defer host.SetSuspended(false)

	return declarative.TabWidget{
		StretchFactor: 1,
		Pages: []declarative.TabPage{
			botsScrollPage("Team", 3, teamRows(s, label, botsBox, botsSize, buysBox, team)),
			botsScrollPage("Classes", 4, classRows(s, team)),
			botsScrollPage("Loadouts", 3, loadoutRows(label, built)),
			botsScrollPage("Looks", 2, cosmeticRows(s, looksBox)),
		},
	}.Create(declarative.NewBuilder(host))
}

// selectTab shows the tab with that title, and does nothing for a title no tab
// carries: an unknown tab is not a reason to refuse the dialog.
func selectTab(tabs *walk.TabWidget, title string) {
	if tabs == nil || title == "" {
		return
	}
	pages := tabs.Pages()
	for i := 0; i < pages.Len(); i++ {
		if pages.At(i).Title() == title {
			_ = tabs.SetCurrentIndex(i)
			return
		}
	}
}

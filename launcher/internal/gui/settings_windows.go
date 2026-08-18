//go:build windows

package gui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/lxn/walk"
	declarative "github.com/lxn/walk/declarative"

	"github.com/m-this/tf2-archipelago/launcher/internal/assets"
	"github.com/m-this/tf2-archipelago/launcher/internal/debugbundle"
	"github.com/m-this/tf2-archipelago/launcher/internal/generate"
	"github.com/m-this/tf2-archipelago/launcher/internal/runshape"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
	"github.com/m-this/tf2-archipelago/launcher/internal/winproc"
)

// labelWidth keeps every tab's label column the same width, so the fields do
// not jump when the player switches tab.
const labelWidth = 150

// runSettingsDialog asks for the values worth changing between evenings, in
// three tabs: what the run is, where the room is, and how the game server
// behaves. Every row carries a tooltip, because a name alone does not say what
// a difficulty floor or a login token is.
//
// It returns the edited settings and whether the player accepted them.
func runSettingsDialog(owner walk.Form, s settings.Settings, repair func() ([]string, error)) (settings.Settings, bool, error) {
	var (
		dialog *walk.Dialog
		accept *walk.PushButton
		cancel *walk.PushButton

		testBox  *walk.CheckBox
		roomEdit *walk.LineEdit
		roomWarn *walk.Label
		roomPass *walk.LineEdit
		slotEdit *walk.LineEdit

		tierBox   *walk.ComboBox
		missions  *walk.NumberEdit
		goalBox   *walk.ComboBox
		sanityPct *walk.NumberEdit
		deathLink *walk.CheckBox

		nameEdit  *walk.LineEdit
		passEdit  *walk.LineEdit
		portEdit  *walk.NumberEdit
		mapBox    *walk.ComboBox
		adminEdit *walk.LineEdit
		lanBox    *walk.CheckBox
		tokenEdit *walk.LineEdit
		botsBox   *walk.CheckBox
		botsSize  *walk.NumberEdit
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

	current := settings.Room{Host: s.APHost, Port: s.APPort}
	edited := s

	label := func(text, help string) declarative.Label {
		return declarative.Label{
			Text:        text,
			MinSize:     declarative.Size{Width: labelWidth},
			ToolTipText: help,
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
		next.TestMode = testBox.Checked()
		next.APHost, next.APPort, next.APTls = room.Host, room.Port, room.TLS
		next.APPassword = roomPass.Text()
		next.APSlotName = strings.TrimSpace(slotEdit.Text())

		next.MvmDifficulty = tiers[max(tierBox.CurrentIndex(), 0)].Key
		next.MvmMissionCount = int(missions.Value())
		next.MvmGoal = goals[max(goalBox.CurrentIndex(), 0)].Key
		next.MvmMissionsanityPct = int(sanityPct.Value())
		next.MvmDeathLink = deathLink.Checked()

		next.SrcdsHostname = strings.TrimSpace(nameEdit.Text())
		next.SrcdsPw = strings.TrimSpace(passEdit.Text())
		next.SrcdsPort = int(portEdit.Value())
		next.SrcdsStartMap = mapBox.Text()
		next.SrcdsAdminSteamIDs = strings.TrimSpace(adminEdit.Text())
		next.SrcdsLan = lanBox.Checked()
		next.SrcdsToken = strings.TrimSpace(tokenEdit.Text())
		next.SrcdsBots = botsBox.Checked()
		next.SrcdsBotTeamSize = int(botsSize.Value())
		return next, nil
	}

	err := declarative.Dialog{
		AssignTo:     &dialog,
		Title:        "Settings",
		CancelButton: &cancel,
		Size:         declarative.Size{Width: 640, Height: 500},
		MinSize:      declarative.Size{Width: 560, Height: 440},
		Layout:       declarative.VBox{},
		Children: []declarative.Widget{
			declarative.TabWidget{
				Pages: []declarative.TabPage{
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
							declarative.Label{
								Text:        "These are the options the Archipelago website calls player options. They go in tf2.yaml, which the seed is generated from.",
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
							label("Game port", "UDP and TCP, 27015 by default. This is the one to forward on your router for friends outside your network."),
							declarative.NumberEdit{
								AssignTo: &portEdit, Value: float64(s.SrcdsPort),
								MinValue: 1024, MaxValue: 65535, Decimals: 0,
							},
							label("Start map", "The map the server loads first. A map, not a mission: set the run's mission once the server is up."),
							declarative.ComboBox{AssignTo: &mapBox, Model: mapNames(), Value: s.SrcdsStartMap},
							label("Admins by Steam id", "Who may run the admin commands, separated by commas. Either form works: the 17 digit id from a profile URL, or SourceMod's STEAM_0:1:26975537."),
							declarative.LineEdit{AssignTo: &adminEdit, Text: s.SrcdsAdminSteamIDs, CueBanner: "76561198014216803, ..."},
							label("Local network only", "On, the server never logs in to Steam and stays off the public list, which is what makes a game with friends work with no token at all. Off needs a real token below."),
							declarative.CheckBox{AssignTo: &lanBox, Text: "keep it off the internet", Checked: s.SrcdsLan},
							label("Login token", "A Game Server Login Token from steamcommunity.com/dev/managegameservers. Needed only with the box above off. 0 means none."),
							declarative.LineEdit{AssignTo: &tokenEdit, Text: s.SrcdsToken},
							label("Defender bots", "Fill the RED team with bots that play, so a wave balanced for six is winnable by fewer. They pick classes, fight and buy their own upgrades."),
							declarative.CheckBox{AssignTo: &botsBox, Text: "fill the RED team", Checked: s.SrcdsBots},
							label("Fill RED to", "How many players RED holds, humans included. Lower it for a harder run."),
							declarative.NumberEdit{
								AssignTo: &botsSize, Value: float64(s.SrcdsBotTeamSize),
								MinValue: 1, MaxValue: 6, Decimals: 0,
							},
						},
					},
				},
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
			"The Archipelago app was not found.\n\nInstall it from "+
				"github.com/ArchipelagoMW/Archipelago/releases, then press this again. "+
				"The launcher looks in the folder its installer uses.",
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
		"Wrote "+path+"\n\nIt holds the launcher log, the SourceMod logs, the "+
			"server console, the player file and the settings. The passwords are "+
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

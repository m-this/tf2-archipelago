package webapi

import (
	"fmt"
	"net"
	"slices"
	"strings"

	"github.com/m-this/tf2-archipelago/gamedata"
	"github.com/m-this/tf2-archipelago/launcher/internal/assets"
	"github.com/m-this/tf2-archipelago/launcher/internal/botlive"
	"github.com/m-this/tf2-archipelago/launcher/internal/form"
	"github.com/m-this/tf2-archipelago/launcher/internal/runshape"
	apruntime "github.com/m-this/tf2-archipelago/launcher/internal/runtime"
	"github.com/m-this/tf2-archipelago/launcher/internal/saveplan"
	"github.com/m-this/tf2-archipelago/launcher/internal/session"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

// Snapshot is everything the page needs for one draw. Passwords never cross
// the loopback boundary; form already represents them as replacement fields.
type Snapshot struct {
	Title             string           `json:"title"`
	Slot              string           `json:"slot,omitempty"`
	Status            string           `json:"status"`
	Running           bool             `json:"running"`
	Busy              bool             `json:"busy"`
	Activity          string           `json:"activity,omitempty"`
	Room              string           `json:"room"`
	TrackerRoomURL    string           `json:"tracker_room_url,omitempty"`
	Join              string           `json:"join"`
	JoinURL           string           `json:"join_url"`
	Mission           string           `json:"mission"`
	Logs              []apruntime.Line `json:"logs"`
	Session           session.Snapshot `json:"session"`
	SessionError      string           `json:"session_error,omitempty"`
	Bots              []botlive.Seat   `json:"bots"`
	DrawnBots         string           `json:"drawn_bots,omitempty"`
	Form              *form.Model      `json:"form,omitempty"`
	FormPage          string           `json:"form_page,omitempty"`
	Notice            string           `json:"notice,omitempty"`
	NoticeSeq         uint64           `json:"notice_seq,omitempty"`
	ItemServer        string           `json:"item_server,omitempty"`
	MissionPool       []MissionPoolRow `json:"mission_pool,omitempty"`
	RestartNeeded     bool             `json:"restart_needed,omitempty"`
	ManagedExternally bool             `json:"managed_externally,omitempty"`

	GameBuild           string `json:"game_build,omitempty"`
	GameUpdateAvailable bool   `json:"game_update_available,omitempty"`
	GameUpdateError     string `json:"game_update_error,omitempty"`
}

// MissionPoolRow is the domain data behind one dense row in the settings
// table. The form field still owns selection and mutation; this only keeps the
// browser from reverse-engineering facts out of a human-readable label.
type MissionPoolRow struct {
	Field         string `json:"field"`
	Source        string `json:"source"`
	Map           string `json:"map"`
	Name          string `json:"name"`
	Loadout       string `json:"loadout"`
	Waves         string `json:"waves"`
	Compatibility string `json:"compatibility"`
	Mods          string `json:"mods"`
	Tier          string `json:"tier"`
	Disabled      bool   `json:"disabled"`
}

func (a *App) Snapshot() Snapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	running := a.supervisor.Running()
	if a.attached {
		running = a.attachedUp
	}
	status := "stopped"
	if a.busy && !running {
		status = "starting"
	} else if running {
		status = "running"
		if settings.Effective(a.settings.SrcdsReach, a.settings.SrcdsToken) == settings.ReachSteam && a.steamURL == "" {
			status = "starting"
		}
	}
	s := a.settings
	playing := apruntime.StartMap(s)
	if running && a.mission != "" {
		playing = a.mission
	}
	room := "no room set"
	if s.TestMode {
		room = "test mode"
	} else if s.APPort != 0 {
		room = "room " + (settings.Room{Host: s.APHost, Port: s.APPort, TLS: s.APTls}).String()
	}
	if playing != "" {
		room += "   " + playing
	}
	sessionState := a.snapshot
	sessionState.Missions = slices.Clone(sessionState.Missions)
	for i := range sessionState.Missions {
		// The launcher names the archive, not the bridge. The bridge answers
		// with whatever its own state file holds, which for a mission it has
		// never seen is nothing at all; gamedata knows which pack every pop
		// file came from. This used to overwrite the source only for an
		// imported pack, so a Potato mission on a machine that had downloaded
		// it read as whatever the bridge happened to say (kelly-cs, #50).
		if mission, known := gamedata.MissionByPopFile(sessionState.Missions[i].PopFile); known {
			sessionState.Missions[i].Source = missionSource(mission, a.imported)
		}
	}
	screen := a.screenLocked(running)
	result := Snapshot{
		Title:  assets.Title("Mann vs Archipelago"),
		Slot:   s.APSlotName,
		Status: status, Running: running, Busy: a.busy, Activity: a.activity, Room: room,
		TrackerRoomURL: s.APRoomURL,
		Join:           a.joinLineLocked(), JoinURL: a.joinURLLocked(), Mission: playing,
		Logs: slices.Clone(a.logs), Session: sessionState,
		Bots: botlive.Team(s), DrawnBots: botlive.Drawn(s),
		Form: screen.Form, FormPage: screen.Page, Notice: a.notice, NoticeSeq: a.noticeSeq,
		ItemServer: a.itemServer, MissionPool: screen.MissionPool, RestartNeeded: screen.RestartNeeded,
		ManagedExternally: a.attached,
		GameBuild:         a.gameBuild, GameUpdateAvailable: a.gameUpdateAvailable, GameUpdateError: a.gameUpdateError,
	}
	if a.fetchErr != nil {
		result.SessionError = a.fetchErr.Error()
	}
	return result
}

func restartNeeded(running bool, before settings.Settings, draft *form.State) bool {
	return running && draft != nil && saveplan.For(before, draft.Settings).Restart
}

type missionPoolSources struct {
	availablePacks    []string
	importedPacks     []string
	readyMods         []string
	managedExternally bool
}

func missionPoolRows(s form.State, built form.Model, sources missionPoolSources) []MissionPoolRow {
	availablePacks, importedPacks, readyMods := sources.availablePacks, sources.importedPacks, sources.readyMods
	floor, hasFloor := gamedata.DifficultyByKey(s.Settings.MvmDifficulty)
	activeMods := activeReadyServerMods(s.Settings, readyMods)
	missions := runshape.VisibleMissions(availablePacks)
	rows := make([]MissionPoolRow, 0, len(missions))
	for _, mission := range missions {
		played, _ := gamedata.MapByID(mission.Map)
		compatibility := gamedata.RequirementLabel(gamedata.MissionRequirement(mission.ID))
		playable := gamedata.IsMissionPlayableWith(mission.ID, activeMods)
		fieldID := "missions.pool." + mission.PopFile
		field, found := built.Field(fieldID)
		if nav := gamedata.MissingNavigationMesh(mission.ID); nav != "" {
			compatibility = "Missing " + nav
		}
		if key := gamedata.MissionServerMod(mission.ID); key != "" {
			mod, _ := gamedata.ServerModByKey(key)
			switch {
			case !slices.Contains(settings.ServerModKeys(s.Settings), key):
				compatibility = "Turn on " + mod.Name + " above"
			case !slices.Contains(readyMods, key):
				compatibility = form.MissingServerModReason(mod.Name, sources.managedExternally)
			}
		}
		if playable {
			compatibility = "Ready"
			if gamedata.IsCommunityMission(mission.ID) && !s.Settings.MvmCommunityMissions {
				compatibility = "Community missions are off"
			} else if hasFloor && mission.Difficulty < floor {
				compatibility = "Below " + floor.String() + " floor"
			}
		}
		mods := "—"
		if key := gamedata.MissionServerMod(mission.ID); key != "" {
			if mod, ok := gamedata.ServerModByKey(key); ok {
				mods = mod.Name
			} else {
				mods = key
			}
		}
		rows = append(rows, MissionPoolRow{
			Field:         fieldID,
			Source:        missionSource(mission, importedPacks),
			Map:           played.Name,
			Name:          mission.Name,
			Loadout:       runshape.MissionLoadoutLabel(mission),
			Waves:         fmt.Sprintf("1–%d", mission.Waves),
			Tier:          mission.Difficulty.String(),
			Compatibility: compatibility,
			Mods:          mods,
			Disabled:      !found || field.Disabled,
		})
	}
	return rows
}

func missionSource(mission gamedata.Mission, importedPacks []string) string {
	if slices.Contains(importedPacks, gamedata.MissionPack(mission.ID)) {
		return "Imported"
	}
	switch gamedata.MissionPack(mission.ID) {
	case settings.CommunityPackPotato:
		return "Potato Archive"
	case settings.CommunityPackMoonlight:
		return "Moonlight Archive"
	default:
		return "Valve"
	}
}

func (a *App) joinLineLocked() string {
	if a.attached {
		line := a.attachedJoinAddressLocked()
		if settings.Effective(a.settings.SrcdsReach, a.settings.SrcdsToken) == settings.ReachSteam {
			if a.steamURL == "" {
				return "Steam relay address: waiting"
			}
			line = a.steamURL
		}
		if a.settings.SrcdsPw != "" {
			line += "   (password " + a.settings.SrcdsPw + ")"
		}
		return line
	}
	port := fmt.Sprintf("%d", a.settings.SrcdsPort)
	var parts []string
	if settings.Effective(a.settings.SrcdsReach, a.settings.SrcdsToken) == settings.ReachSteam {
		if a.steamURL == "" {
			parts = append(parts, "Steam public IP: waiting")
		} else {
			parts = append(parts, "Steam public IP: "+a.steamURL)
		}
	}
	for _, address := range apruntime.LocalAddresses() {
		parts = append(parts, address+":"+port)
	}
	if len(parts) == 0 {
		parts = append(parts, "127.0.0.1:"+port)
	}
	line := strings.Join(parts, "   ")
	if a.settings.SrcdsPw != "" {
		line += "   (password " + a.settings.SrcdsPw + ")"
	}
	return line
}

func (a *App) joinURLLocked() string {
	if a.attached {
		// SteamConnectURL otherwise discovers this process's LAN address. In a
		// sidecar that is the private container address, which the browser cannot
		// join. Treat the known host-published address like a relay address so it
		// is used verbatim.
		s := a.settings
		s.SrcdsReach = settings.ReachSteam
		address := a.attachedJoinAddressLocked()
		if settings.Effective(a.settings.SrcdsReach, a.settings.SrcdsToken) == settings.ReachSteam {
			if a.steamURL == "" {
				return ""
			}
			address = a.steamURL
		}
		return apruntime.SteamConnectURL(s, address)
	}
	if settings.Effective(a.settings.SrcdsReach, a.settings.SrcdsToken) == settings.ReachSteam && a.steamURL == "" {
		return ""
	}
	return apruntime.SteamConnectURL(a.settings, a.steamURL)
}

func (a *App) attachedJoinAddressLocked() string {
	host := a.settings.SrcdsJoinHost
	if host == "" {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, fmt.Sprintf("%d", a.settings.SrcdsPort))
}

// Screen is the settings part of one draw: the rows, the page they are on, the
// mission table beside them and whether saving would restart a running server.
// A closed settings screen has a nil Form, which is not the same as a Form with
// no tabs.
type Screen struct {
	Form          *form.Model
	Page          string
	MissionPool   []MissionPoolRow
	RestartNeeded bool
}

// Screen answers what the settings screen holds now. Every settings call
// answers with one, because a change can move another row's bounds, add and
// remove rows outright, and move the mission table under them.
func (a *App) Screen() Screen {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.screenLocked(a.supervisor.Running())
}

// screenLocked builds it. Passwords are blanked here rather than at the wire,
// because there is one Screen and every interface reads this one.
func (a *App) screenLocked(running bool) Screen {
	if a.draft == nil {
		return Screen{Page: a.formPage}
	}
	built := form.Build(*a.draft, a.formEnvLocked())
	for page := range built.Tabs {
		for field := range built.Tabs[page].Fields {
			if built.Tabs[page].Fields[field].Kind == form.Password {
				built.Tabs[page].Fields[field].Value = ""
			}
		}
	}
	return Screen{
		Form:          &built,
		Page:          a.formPage,
		MissionPool:   missionPoolRows(*a.draft, built, missionPoolSources{availablePacks: a.community, importedPacks: a.imported, readyMods: a.serverMods, managedExternally: a.attached}),
		RestartNeeded: restartNeeded(running, a.settings, a.draft),
	}
}

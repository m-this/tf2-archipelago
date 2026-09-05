// Package runshape turns the game data into the choices the launcher offers
// for a run: which difficulty tiers a player can draw from, how many missions
// that leaves, and what can end the run. Every number here is counted from
// gamedata rather than written down, so a mission added there shows up in the
// prompts and in the limits without an edit.
package runshape

import (
	"fmt"
	"slices"
	"strings"

	"github.com/m-this/tf2-archipelago/gamedata"
)

// Tier is one choice of difficulty floor: the easiest tier a run may draw, and
// how many missions that leaves in the pool. A choice includes every harder
// tier, so the pool shrinks as the floor rises.
type Tier struct {
	Key      string
	Missions int
	Waves    int
}

// Tiers lists the pools worth offering, easiest first.
//
// Haunted is left out as a useful launcher preset. It holds only Caliginous
// Caper, so it has no mission-ticket progression and commits the run to one
// unusually long 666-robot mission. Hand-authored YAML may still use it.
func Tiers() []Tier {
	tiers := make([]Tier, 0, len(gamedata.Difficulties))
	for _, difficulty := range gamedata.Difficulties {
		if difficulty == gamedata.DifficultyHaunted {
			continue
		}
		tier := Tier{Key: difficulty.Key()}
		for _, mission := range gamedata.PlayableMissions() {
			if mission.Difficulty >= difficulty {
				tier.Missions++
				tier.Waves += int(mission.Waves)
			}
		}
		tiers = append(tiers, tier)
	}
	return tiers
}

// MissionsInPool reports how many missions a difficulty key leaves to draw. An
// unknown key gives the whole pool, which is what the generator does.
func MissionsInPool(key string) int {
	for _, tier := range Tiers() {
		if tier.Key == key {
			return tier.Missions
		}
	}
	return len(gamedata.PlayableMissions())
}

// Label describes a tier in one line, for a menu.
//
// The number is the pool the choice leaves, not the length of a run: picking a
// tier draws that tier and every harder one, and the mission count decides how
// many of them a run uses. Saying "29 missions" against normal and "4" against
// expert reads backwards without that word.
func (t Tier) Label() string {
	return fmt.Sprintf("%-14s draws from %2d missions", t.Key, t.Missions)
}

// Goal is one way for a run to end.
type Goal struct {
	Key         string
	Description string
}

// Goals lists what can end a run. The keys are the apworld's option values.
func Goals() []Goal {
	return []Goal{
		{"final_boss", "clear the hardest mission the run drew"},
		{"missionsanity", "clear a share of the missions, in any order"},
	}
}

// Label describes a goal in one line, for a menu.
func (g Goal) Label() string {
	return fmt.Sprintf("%-14s %s", g.Key, g.Description)
}

// WavesFor estimates how many waves a run of this many missions holds, which
// is the closest thing to "how long is this evening".
func (t Tier) WavesFor(missions int) int {
	if t.Missions == 0 {
		return 0
	}
	return t.Waves * missions / t.Missions
}

// MissionChoice is one mission as a menu offers it: the popfile the server
// loads, and a label that reads map first, since that is how the game groups
// them.
type MissionChoice struct {
	PopFile string
	Label   string
}

// MissionChoices lists every mission the tables know, in table order, which
// is map by map and easiest first.
func MissionChoices() []MissionChoice {
	return MissionChoicesForPacks([]string{"archive-assets.zip", "mlarchive-assets.zip"})
}

// VisibleMissions is the mission table a launcher may populate from the packs
// that exist locally. Valve missions are always present. Community missions
// appear only after their owning archive has been downloaded or supplied
// locally, the locked ones included: a row that says why it is locked beats a
// mission that is not there.
func VisibleMissions(availablePacks []string) []gamedata.Mission {
	var visible []gamedata.Mission
	for _, mission := range gamedata.Missions {
		pack := gamedata.MissionPack(mission.ID)
		switch {
		case pack == "" && gamedata.IsPlayableMission(mission.ID):
			visible = append(visible, mission)
		case pack != "" && slices.Contains(availablePacks, pack):
			visible = append(visible, mission)
		}
	}
	return visible
}

// MissionChoicesForPacks excludes unavailable and locked community missions
// from menus that can actually select a starting mission.
func MissionChoicesForPacks(availablePacks []string) []MissionChoice {
	playable := VisibleMissions(availablePacks)
	choices := make([]MissionChoice, 0, len(playable))
	for _, mission := range playable {
		if !gamedata.IsPlayableMission(mission.ID) {
			continue
		}
		played, _ := gamedata.MapByID(mission.Map)
		loadout := MissionLoadoutLabel(mission)
		if loadout != "" {
			loadout = ", " + loadout
		}
		choices = append(choices, MissionChoice{
			PopFile: mission.PopFile,
			Label: fmt.Sprintf("[%s] %s - %s (%s, %d waves%s)",
				missionSource(mission), played.Name, mission.Name, mission.Difficulty.Key(), mission.Waves, loadout),
		})
	}
	return choices
}

// MissionLoadoutLabel is the player-facing name of a mission's special
// loadout. It deliberately says "loadout" rather than "mode": the catalog
// describes how these missions are designed, not an engine-level map setting.
func MissionLoadoutLabel(mission gamedata.Mission) string {
	if gamedata.MissionLoadout(mission.ID) == "medieval" {
		return "Medieval loadout"
	}
	return ""
}

func missionSource(mission gamedata.Mission) string {
	switch gamedata.MissionPack(mission.ID) {
	case "archive-assets.zip":
		return "Potato Archive"
	case "mlarchive-assets.zip":
		return "Moonlight Archive"
	}
	return "Valve"
}

// MissionLabel is the label of one popfile, or the popfile itself when the
// tables do not know it.
func MissionLabel(popFile string) string {
	for _, choice := range MissionChoices() {
		if choice.PopFile == popFile {
			return choice.Label
		}
	}
	return popFile
}

// AnyLabel is the first entry of the start mission and start class menus: the
// seed draws it. An empty popfile or class name means this.
const AnyLabel = "Any - the run draws it"

// StartMissionChoices is MissionChoices with the draw in front, for the menu
// that decides where a run begins. The empty popfile is the draw.
func StartMissionChoices() []MissionChoice {
	return append([]MissionChoice{{PopFile: "", Label: AnyLabel}}, MissionChoices()...)
}

// StartMissionChoicesForPacks is the availability-aware start menu.
func StartMissionChoicesForPacks(availablePacks []string) []MissionChoice {
	return append([]MissionChoice{{PopFile: "", Label: AnyLabel}}, MissionChoicesForPacks(availablePacks)...)
}

// StartMissionLabel is what StartMissionChoices shows for one popfile.
func StartMissionLabel(popFile string) string {
	if popFile == "" {
		return AnyLabel
	}
	return MissionLabel(popFile)
}

// StartClassChoices is the nine mercenaries with the draw in front. The empty
// name is the draw, and the rest are the names the apworld's option takes.
func StartClassChoices() []string {
	names := make([]string, 0, len(gamedata.Classes)+1)
	names = append(names, AnyLabel)
	for _, class := range gamedata.Classes {
		names = append(names, class.Name)
	}
	return names
}

// StartClassLabel is what StartClassChoices shows for one class name.
func StartClassLabel(name string) string {
	for _, class := range gamedata.Classes {
		if strings.EqualFold(class.Name, name) {
			return class.Name
		}
	}
	return AnyLabel
}

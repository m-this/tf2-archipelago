// Package runshape turns the game data into the choices the launcher offers
// for a run: which difficulty tiers a player can draw from, how many missions
// that leaves, and what can end the run. Every number here is counted from
// gamedata rather than written down, so a mission added there shows up in the
// prompts and in the limits without an edit.
package runshape

import (
	"fmt"

	"github.com/m-this/tf2-archipelago/gamedata"
)

// Tier is one choice of difficulty pool: the easiest tier a run can draw, and
// what drawing from there leaves to draw.
type Tier struct {
	Key      string
	Missions int
	Waves    int
}

// Tiers lists the pools worth offering, easiest first.
//
// Haunted is left out. It holds one mission, and a pool of one gives too few
// locations for the items of a run, so generation stops with an error.
func Tiers() []Tier {
	tiers := make([]Tier, 0, len(gamedata.Difficulties))
	for _, difficulty := range gamedata.Difficulties {
		if difficulty == gamedata.DifficultyHaunted {
			continue
		}
		tier := Tier{Key: difficulty.Key()}
		for _, mission := range gamedata.Missions {
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
	return len(gamedata.Missions)
}

// Label describes a tier in one line, for a menu.
func (t Tier) Label() string {
	return fmt.Sprintf("%-14s %2d missions, %3d waves", t.Key, t.Missions, t.Waves)
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

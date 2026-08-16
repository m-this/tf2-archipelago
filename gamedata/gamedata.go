// Package gamedata is the single source of truth for every Mann vs Machine
// fact this project relies on, and for every Archipelago id derived from one.
//
// Ids are append-only. A seed records numbers, not names, so renumbering an id
// silently hands a player the wrong item in every seed already generated, and
// nothing detects it at play time. Entities therefore carry explicit id
// literals, deletions leave the id reserved, and the tests guard both.
//
// See ADR 0001.
package gamedata

//go:generate go run ./cmd/export ../apworld/tf2_mvm/data

const (
	// GameName is the multiworld's primary key for this game. One spelling
	// exists and this is it.
	GameName = "Team Fortress 2 Mann vs Machine"

	// FormatVersion is the shape of the exported JSON. The apworld refuses to
	// load an export whose version it does not know.
	FormatVersion = 1
)

// Difficulty is a mission's tier. The keys are Valve's own, taken from the
// mvm_maps block of tf/scripts/items/items_game.txt. The TF2 wiki calls
// haunted "Nightmare"; Valve does not, and Valve wins here.
type Difficulty uint8

const (
	DifficultyNormal Difficulty = iota + 1
	DifficultyIntermediate
	DifficultyAdvanced
	DifficultyExpert
	DifficultyHaunted
)

var difficultyKeys = [...]string{
	DifficultyNormal:       "normal",
	DifficultyIntermediate: "intermediate",
	DifficultyAdvanced:     "advanced",
	DifficultyExpert:       "expert",
	DifficultyHaunted:      "haunted",
}

var difficultyNames = [...]string{
	DifficultyNormal:       "Normal",
	DifficultyIntermediate: "Intermediate",
	DifficultyAdvanced:     "Advanced",
	DifficultyExpert:       "Expert",
	DifficultyHaunted:      "Haunted",
}

// Difficulties is every tier, easiest first. The order is the ladder the
// difficulty_pool option walks, which is why haunted sits at the end rather
// than beside expert.
var Difficulties = []Difficulty{
	DifficultyNormal,
	DifficultyIntermediate,
	DifficultyAdvanced,
	DifficultyExpert,
	DifficultyHaunted,
}

func (d Difficulty) Key() string    { return difficultyKeys[d] }
func (d Difficulty) String() string { return difficultyNames[d] }

// Classification is Archipelago's item category. Getting this wrong produces
// seeds that are unwinnable or trivial, so it is data, not a guess made at
// pool-building time.
type Classification uint8

const (
	Progression Classification = iota + 1
	Useful
	Filler
	Trap
)

var classificationKeys = [...]string{
	Progression: "progression",
	Useful:      "useful",
	Filler:      "filler",
	Trap:        "trap",
}

func (c Classification) Key() string { return classificationKeys[c] }

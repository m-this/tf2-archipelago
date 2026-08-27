// Package gamedata is the single source of truth for every Mann vs Machine
// fact this project relies on, and for every Archipelago id derived from one.
//
// Ids are append-only. A seed records numbers, not names, so renumbering an id
// silently hands a player the wrong item in every seed already generated, and
// nothing detects it at play time. Deletions leave the id reserved.
//
// See ADR 0001.
package gamedata

//go:generate go run ./cmd/export ../apworld/tf2_mvm/data

const (
	// GameName is the multiworld's primary key for this game.
	GameName = "Team Fortress 2 Mann vs Machine"

	// FormatVersion is the shape of the exported JSON. The apworld refuses to
	// load an export whose version it does not know.
	FormatVersion = 4
)

// Difficulty is a mission's tier. The keys are Valve's own, from the mvm_maps
// block of tf/scripts/items/items_game.txt, not the wiki's (haunted, not
// "Nightmare").
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

// Difficulties is every tier in the order the difficulty_pool option walks,
// which is why haunted sits at the end rather than beside expert.
var Difficulties = []Difficulty{
	DifficultyNormal,
	DifficultyIntermediate,
	DifficultyAdvanced,
	DifficultyExpert,
	DifficultyHaunted,
}

func (d Difficulty) Key() string    { return difficultyKeys[d] }
func (d Difficulty) String() string { return difficultyNames[d] }

// DifficultyByKey turns the difficulty_pool option's value back into a tier.
// The key comes from a settings file or an environment variable, so an unknown
// one is a typo rather than a broken table: it reads as the whole pool, which
// is what the generator does with a mission count larger than the pool holds.
func DifficultyByKey(key string) (Difficulty, bool) {
	for _, d := range Difficulties {
		if d.Key() == key {
			return d, true
		}
	}
	return DifficultyNormal, false
}

// Classification is Archipelago's item category. A wrong one makes a seed
// unwinnable or trivial, so it is data rather than a guess at pool-build time.
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

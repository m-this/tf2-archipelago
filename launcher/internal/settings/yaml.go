package settings

import (
	"fmt"
	"strings"
)

// Game is the name the apworld registers with Archipelago. It has to match
// apworld/tf2_mvm/data/meta.json, which is what the generator matches the
// YAML against.
const Game = "Team Fortress 2 Mann vs Machine"

// PlayerYAML renders the Archipelago player file for this run's shape. The
// player drops it into the Archipelago app's Players folder and generates
// there, which is the one step the launcher cannot do for them: generation is
// Python, and bundling Python would undo the single-exe promise.
//
// archipelagoVersion goes in `requires`, so a mismatched Archipelago refuses
// the file instead of generating a seed the apworld cannot load.
func PlayerYAML(s Settings, archipelagoVersion string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "name: %s\n", yamlString(s.APSlotName))
	fmt.Fprintf(&b, "game: %s\n", yamlString(Game))
	if archipelagoVersion != "" {
		b.WriteString("requires:\n")
		fmt.Fprintf(&b, "  version: %s\n", archipelagoVersion)
	}
	fmt.Fprintf(&b, "%s:\n", yamlString(Game))
	fmt.Fprintf(&b, "  mission_count: %d\n", s.MvmMissionCount)
	fmt.Fprintf(&b, "  difficulty_pool: %s\n", s.MvmDifficulty)
	fmt.Fprintf(&b, "  goal: %s\n", s.MvmGoal)
	fmt.Fprintf(&b, "  missionsanity_percentage: %d\n", s.MvmMissionsanityPct)
	fmt.Fprintf(&b, "  death_link: %t\n", s.MvmDeathLink)
	return b.String()
}

// yamlString quotes a scalar. The game name holds spaces and the slot name is
// whatever the player typed, so neither can go in bare.
func yamlString(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}

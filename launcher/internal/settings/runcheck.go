package settings

import (
	"fmt"

	"github.com/m-this/tf2-archipelago/launcher/internal/runshape"
)

// CheckRunSelection validates the mission pool before the launcher writes or
// generates a player file. The official generator repeats this check and
// remains the final authority.
func CheckRunSelection(s Settings) (runshape.Preflight, error) {
	report, err := runshape.CheckSelection(runshape.Selection{
		Difficulty:   s.MvmDifficulty,
		MissionCount: s.MvmMissionCount,
		Excluded:     s.MvmExcludedMissions,
		StartMission: s.MvmStartMission,
	})
	if err != nil {
		return report, fmt.Errorf("archipelago run selection: %w", err)
	}
	return report, nil
}

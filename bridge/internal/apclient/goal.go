package apclient

import (
	"fmt"
	"slices"

	"github.com/m-this/tf2-archipelago/gamedata"
)

// validate rejects a seed from an apworld this binary may disagree with about what an id means.
func (s SlotData) validate() error {
	if s.FormatVersion != gamedata.FormatVersion {
		return fmt.Errorf(
			"seed has data format version %d, this bridge reads %d",
			s.FormatVersion, gamedata.FormatVersion,
		)
	}
	if len(s.Missions) == 0 {
		return fmt.Errorf("seed has no missions")
	}
	if s.MissionTicketImportance != "" && s.MissionTicketImportance != "useful" && s.MissionTicketImportance != "progression" {
		return fmt.Errorf("unknown mission ticket importance %q", s.MissionTicketImportance)
	}
	for _, popFile := range s.Missions {
		if _, known := gamedata.MissionByPopFile(popFile); !known {
			return fmt.Errorf("seed uses mission %q, which is not in the tables", popFile)
		}
	}
	// Older seeds carry no start mission, and the server finds its own way to
	// the first unlocked one. An unknown name is a different thing: it means
	// this binary and the seed disagree about the tables.
	if s.StartMission != "" {
		if _, known := gamedata.MissionByPopFile(s.StartMission); !known {
			return fmt.Errorf("start mission %q is not in the tables", s.StartMission)
		}
		if !slices.Contains(s.Missions, s.StartMission) {
			return fmt.Errorf("start mission %q is not one of the run's missions", s.StartMission)
		}
	}
	switch s.Goal {
	case "final_boss":
		if _, known := gamedata.MissionByPopFile(s.GoalMission); !known {
			return fmt.Errorf("goal mission %q is not in the tables", s.GoalMission)
		}
	case "missionsanity":
		if s.MissionsanityTarget < 1 || s.MissionsanityTarget > len(s.Missions) {
			return fmt.Errorf(
				"missionsanity asks for %d of %d missions",
				s.MissionsanityTarget, len(s.Missions),
			)
		}
	default:
		return fmt.Errorf("unknown goal %q", s.Goal)
	}
	return nil
}

// goalReached reads the win off the locations this server checked itself: a
// mission is cleared when its clear location is one of them. The list the
// multiworld holds is a different thing, and it moves when other people finish.
func (s SlotData) goalReached(checks []int64) bool {
	switch s.Goal {
	case "final_boss":
		mission, known := gamedata.MissionByPopFile(s.GoalMission)
		return known && slices.Contains(checks, mission.ClearLocationID())

	case "missionsanity":
		cleared := 0
		for _, popFile := range s.Missions {
			mission, known := gamedata.MissionByPopFile(popFile)
			if known && slices.Contains(checks, mission.ClearLocationID()) {
				cleared++
			}
		}
		return cleared >= s.MissionsanityTarget

	default:
		return false
	}
}

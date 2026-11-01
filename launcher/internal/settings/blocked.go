package settings

import (
	"slices"

	"github.com/m-this/tf2-archipelago/gamedata"
)

// WithoutServerMods is s as the generator should see it when the server has
// refused to load some mods: they are off, and the missions that name them are
// excluded, so the seed does not draw a mission the server cannot run. Nothing
// is saved; the caller passes the result to the seed and the run check only.
func WithoutServerMods(s Settings, blocked []string) Settings {
	if len(blocked) == 0 {
		return s
	}
	s.SrcdsMods = slices.DeleteFunc(slices.Clone(s.SrcdsMods), func(key string) bool {
		return slices.Contains(blocked, key)
	})
	s.MvmExcludedMissions = slices.Clone(s.MvmExcludedMissions)
	for _, mission := range gamedata.Missions {
		if !slices.Contains(blocked, gamedata.MissionServerMod(mission.ID)) {
			continue
		}
		if !slices.Contains(s.MvmExcludedMissions, mission.PopFile) {
			s.MvmExcludedMissions = append(s.MvmExcludedMissions, mission.PopFile)
		}
		if s.MvmStartMission == mission.PopFile {
			s.MvmStartMission = ""
		}
		if s.SrcdsStartMission == mission.PopFile {
			s.SrcdsStartMission = ""
		}
	}
	return s
}

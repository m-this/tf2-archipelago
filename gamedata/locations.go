package gamedata

// ObjectiveKind is the plugin's vocabulary for what Archipelago calls a
// location. The plugin reports an objective in MvM terms and never learns that
// an id was involved; turning one into the other is this package's job.
type ObjectiveKind uint8

const (
	ObjectiveWaveCleared ObjectiveKind = iota + 1
	ObjectiveMissionCleared
)

var objectiveKeys = [...]string{
	ObjectiveWaveCleared:    "wave_cleared",
	ObjectiveMissionCleared: "mission_cleared",
}

// Key is the string on the wire between the plugin and the bridge.
func (k ObjectiveKind) Key() string { return objectiveKeys[k] }

// Location is one check. Wave is zero for a mission clear.
type Location struct {
	ID      int64
	Name    string
	Kind    ObjectiveKind
	Mission MissionID
	Wave    uint8
}

// Locations is every check in the game, mission by mission, wave order within
// a mission, the mission clear last. 176 wave clears and 29 mission clears.
var Locations = buildLocations()

func buildLocations() []Location {
	all := make([]Location, 0, len(Missions)*int(WavesMax))
	for _, m := range Missions {
		for wave := uint8(1); wave <= m.Waves; wave++ {
			all = append(all, Location{
				ID:      m.WaveLocationID(wave),
				Name:    m.WaveLocationName(wave),
				Kind:    ObjectiveWaveCleared,
				Mission: m.ID,
				Wave:    wave,
			})
		}
		all = append(all, Location{
			ID:      m.ClearLocationID(),
			Name:    m.ClearLocationName(),
			Kind:    ObjectiveMissionCleared,
			Mission: m.ID,
		})
	}
	return all
}

var locationsByID = indexLocations()

func indexLocations() map[int64]Location {
	byID := make(map[int64]Location, len(Locations))
	for _, l := range Locations {
		byID[l.ID] = l
	}
	return byID
}

// LocationByID is how the bridge reads a check back: an id on the wire says
// nothing, the location it came from says which mission and which wave.
func LocationByID(id int64) (Location, bool) {
	l, ok := locationsByID[id]
	return l, ok
}

// LocationByObjective resolves what the plugin reported. Wave is ignored for a
// mission clear. This is the whole southbound translation: the bridge holds no
// id table of its own.
func LocationByObjective(kind ObjectiveKind, popFile string, wave uint8) (Location, bool) {
	m, ok := MissionByPopFile(popFile)
	if !ok {
		return Location{}, false
	}
	switch kind {
	case ObjectiveMissionCleared:
		return Location{
			ID:      m.ClearLocationID(),
			Name:    m.ClearLocationName(),
			Kind:    kind,
			Mission: m.ID,
		}, true
	case ObjectiveWaveCleared:
		if wave < 1 || wave > m.Waves {
			return Location{}, false
		}
		return Location{
			ID:      m.WaveLocationID(wave),
			Name:    m.WaveLocationName(wave),
			Kind:    kind,
			Mission: m.ID,
			Wave:    wave,
		}, true
	default:
		return Location{}, false
	}
}

package gamedata

// MapID identifies one MvM map. Explicit literals, append-only, never reused.
type MapID uint8

const (
	MapDecoy      MapID = 1
	MapCoaltown   MapID = 2
	MapMannworks  MapID = 3
	MapBigrock    MapID = 4
	MapMannhattan MapID = 5
	MapRottenburg MapID = 6
	MapGhostTown  MapID = 7
)

// Map is one MvM map. Name is what changelevel takes, not a display name.
type Map struct {
	ID   MapID
	Name string
}

// Maps is every map with at least one Valve mission on it. The seven .bsp
// files shipped in tf/maps.
var Maps = []Map{
	{MapDecoy, "mvm_decoy"},
	{MapCoaltown, "mvm_coaltown"},
	{MapMannworks, "mvm_mannworks"},
	{MapBigrock, "mvm_bigrock"},
	{MapMannhattan, "mvm_mannhattan"},
	{MapRottenburg, "mvm_rottenburg"},
	{MapGhostTown, "mvm_ghost_town"},
}

var mapsByID = indexMaps()

func indexMaps() map[MapID]Map {
	byID := make(map[MapID]Map, len(Maps))
	for _, m := range Maps {
		byID[m.ID] = m
	}
	return byID
}

// MapByID returns the map with that id.
func MapByID(id MapID) (Map, bool) {
	m, ok := mapsByID[id]
	return m, ok
}

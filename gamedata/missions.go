package gamedata

// MissionID identifies one mission. Explicit literals, append-only, never
// reused: every location id in a seed is derived from one of these.
type MissionID uint16

// Mission is one .pop file: an ordered run of waves on one map, at one
// difficulty tier.
type Mission struct {
	ID         MissionID
	PopFile    string
	Name       string
	Map        MapID
	Difficulty Difficulty
	Waves      uint8
}

// Missions is the 29 Valve missions. Pop file names come from
// tf2_misc_dir.vpk; display names, tiers and wave counts from the TF2 wiki.
//
// UNVERIFIED: inside a tier with more than one entry, the pairing of display
// name to pop file is a guess. Settle it against resource/tf_english.txt in the
// VPK before the first seed is played: the name is baked into location names.
//
// Community missions are out of scope for v1: nothing on the host can read a
// wave count out of a .pop file, and a downloaded pack can change under a seed
// that already references it.
var Missions = []Mission{
	{1, "mvm_decoy", "Doe's Drill", MapDecoy, DifficultyNormal, 8},
	{2, "mvm_decoy_intermediate", "Doe's Doom", MapDecoy, DifficultyIntermediate, 7},
	{3, "mvm_decoy_intermediate2", "Day of Wreckening", MapDecoy, DifficultyIntermediate, 6},
	{4, "mvm_decoy_advanced", "Disk Deletion", MapDecoy, DifficultyAdvanced, 8},
	{5, "mvm_decoy_advanced2", "Data Demolition", MapDecoy, DifficultyAdvanced, 6},
	{6, "mvm_decoy_advanced3", "Disintegration", MapDecoy, DifficultyAdvanced, 6},
	{7, "mvm_decoy_expert1", "Desperation", MapDecoy, DifficultyExpert, 7},
	{8, "mvm_coaltown", "Crash Course", MapCoaltown, DifficultyNormal, 6},
	{9, "mvm_coaltown_intermediate", "Cave-in", MapCoaltown, DifficultyIntermediate, 6},
	{10, "mvm_coaltown_intermediate2", "Quarry", MapCoaltown, DifficultyIntermediate, 6},
	{11, "mvm_coaltown_advanced", "Ctrl+Alt+Destruction", MapCoaltown, DifficultyAdvanced, 7},
	{12, "mvm_coaltown_advanced2", "CPU Slaughter", MapCoaltown, DifficultyAdvanced, 6},
	{13, "mvm_coaltown_expert1", "Cataclysm", MapCoaltown, DifficultyExpert, 7},
	{14, "mvm_mannworks", "Mann-euvers", MapMannworks, DifficultyNormal, 7},
	{15, "mvm_mannworks_intermediate", "Mean Machines", MapMannworks, DifficultyIntermediate, 6},
	{16, "mvm_mannworks_intermediate2", "Mannhunt", MapMannworks, DifficultyIntermediate, 6},
	{17, "mvm_mannworks_advanced", "Machine Massacre", MapMannworks, DifficultyAdvanced, 7},
	{18, "mvm_mannworks_ironman", "Mech Mutilation", MapMannworks, DifficultyAdvanced, 3},
	{19, "mvm_mannworks_expert1", "Mannslaughter", MapMannworks, DifficultyExpert, 5},
	{20, "mvm_bigrock", "Benign Infiltration", MapBigrock, DifficultyNormal, 6},
	{21, "mvm_bigrock_advanced1", "Broken Parts", MapBigrock, DifficultyAdvanced, 7},
	{22, "mvm_bigrock_advanced2", "Bone Shaker", MapBigrock, DifficultyAdvanced, 8},
	{23, "mvm_mannhattan", "Big Apple Barricade", MapMannhattan, DifficultyIntermediate, 6},
	{24, "mvm_mannhattan_advanced1", "Empire Escalation", MapMannhattan, DifficultyAdvanced, 6},
	{25, "mvm_mannhattan_advanced2", "Metro Malice", MapMannhattan, DifficultyAdvanced, 6},
	{26, "mvm_rottenburg", "Village Vanguard", MapRottenburg, DifficultyIntermediate, 7},
	{27, "mvm_rottenburg_advanced1", "Hamlet Hostility", MapRottenburg, DifficultyAdvanced, 7},
	{28, "mvm_rottenburg_advanced2", "Bavarian Botbash", MapRottenburg, DifficultyAdvanced, 7},
	{29, "mvm_ghost_town_666", "Caliginous Caper", MapGhostTown, DifficultyHaunted, 1},
}

var (
	missionsByPopFile = indexMissionsByPopFile()
	missionsByID      = indexMissionsByID()
)

func indexMissionsByPopFile() map[string]Mission {
	byPopFile := make(map[string]Mission, len(Missions))
	for _, m := range Missions {
		byPopFile[m.PopFile] = m
	}
	return byPopFile
}

func indexMissionsByID() map[MissionID]Mission {
	byID := make(map[MissionID]Mission, len(Missions))
	for _, m := range Missions {
		byID[m.ID] = m
	}
	return byID
}

// MissionByID returns the mission an item or a location belongs to.
func MissionByID(id MissionID) (Mission, bool) {
	m, ok := missionsByID[id]
	return m, ok
}

// MissionByPopFile returns the mission the plugin means when it reports an
// objective. The pop file name is the only part of a mission the game knows.
func MissionByPopFile(popFile string) (Mission, bool) {
	m, ok := missionsByPopFile[popFile]
	return m, ok
}

// MissionsByDifficulty returns the missions in one tier, in table order.
func MissionsByDifficulty(d Difficulty) []Mission {
	tier := make([]Mission, 0, len(Missions))
	for _, m := range Missions {
		if m.Difficulty == d {
			tier = append(tier, m)
		}
	}
	return tier
}

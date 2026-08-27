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

	// HasTank and HasGiant say the mission holds one, which is what makes that
	// check reachable. Three of Valve's missions have no tank. Every one of
	// them has a giant.
	//
	// A false costs one check. A true on a mission that holds neither costs
	// the seed: the location can never be reached and the run cannot be
	// finished. So both come from the source below rather than from memory.
	HasTank  bool
	HasGiant bool
}

// valveMissions is the 29 Valve missions. Pop file names come from
// tf2_misc_dir.vpk. Display names, maps, tiers, wave counts and what each
// mission holds come from the wiki's own mission list, the source of this
// table:
//
//	https://wiki.teamfortress.com/w/index.php?title=Template:List_of_MVM_Missions&action=edit
//
// That table names the tier haunted "Nightmare"; the key here is Valve's, from
// items_game.txt.
//
// UNVERIFIED, still: inside one map and one tier, which display name goes with
// which numbered pop file. The wiki lists a map's missions in the order the
// pop files are numbered, which agrees with this table, but agreeing is not
// the same as proving. Settle it against resource/tf_english.txt in the VPK:
// the name is baked into every location name.
//
// Community missions live in the versioned manifest instead. Their wave and
// objective metadata is explicit because a seed must not depend on parsing a
// mutable population file at runtime.
var valveMissions = []Mission{
	{1, "mvm_decoy", "Doe's Drill", MapDecoy, DifficultyNormal, 8, true, true},
	{2, "mvm_decoy_intermediate", "Doe's Doom", MapDecoy, DifficultyIntermediate, 7, true, true},
	{3, "mvm_decoy_intermediate2", "Day of Wreckening", MapDecoy, DifficultyIntermediate, 6, true, true},
	{4, "mvm_decoy_advanced", "Disk Deletion", MapDecoy, DifficultyAdvanced, 8, true, true},
	{5, "mvm_decoy_advanced2", "Data Demolition", MapDecoy, DifficultyAdvanced, 6, true, true},
	{6, "mvm_decoy_advanced3", "Disintegration", MapDecoy, DifficultyAdvanced, 6, true, true},
	{7, "mvm_decoy_expert1", "Desperation", MapDecoy, DifficultyExpert, 7, true, true},
	{8, "mvm_coaltown", "Crash Course", MapCoaltown, DifficultyNormal, 6, true, true},
	{9, "mvm_coaltown_intermediate", "Cave-in", MapCoaltown, DifficultyIntermediate, 6, true, true},
	{10, "mvm_coaltown_intermediate2", "Quarry", MapCoaltown, DifficultyIntermediate, 6, true, true},
	{11, "mvm_coaltown_advanced", "Ctrl+Alt+Destruction", MapCoaltown, DifficultyAdvanced, 7, true, true},
	{12, "mvm_coaltown_advanced2", "CPU Slaughter", MapCoaltown, DifficultyAdvanced, 6, true, true},
	{13, "mvm_coaltown_expert1", "Cataclysm", MapCoaltown, DifficultyExpert, 7, true, true},
	{14, "mvm_mannworks", "Mann-euvers", MapMannworks, DifficultyNormal, 7, true, true},
	{15, "mvm_mannworks_intermediate", "Mean Machines", MapMannworks, DifficultyIntermediate, 6, true, true},
	{16, "mvm_mannworks_intermediate2", "Mannhunt", MapMannworks, DifficultyIntermediate, 6, true, true},
	{17, "mvm_mannworks_advanced", "Machine Massacre", MapMannworks, DifficultyAdvanced, 7, true, true},
	{18, "mvm_mannworks_ironman", "Mech Mutilation", MapMannworks, DifficultyAdvanced, 3, true, true},
	{19, "mvm_mannworks_expert1", "Mannslaughter", MapMannworks, DifficultyExpert, 5, true, true},
	{20, "mvm_bigrock", "Benign Infiltration", MapBigrock, DifficultyNormal, 6, true, true},
	{21, "mvm_bigrock_advanced1", "Broken Parts", MapBigrock, DifficultyAdvanced, 7, true, true},
	{22, "mvm_bigrock_advanced2", "Bone Shaker", MapBigrock, DifficultyAdvanced, 8, true, true},
	{23, "mvm_mannhattan", "Big Apple Barricade", MapMannhattan, DifficultyIntermediate, 6, false, true},
	{24, "mvm_mannhattan_advanced1", "Empire Escalation", MapMannhattan, DifficultyAdvanced, 6, false, true},
	{25, "mvm_mannhattan_advanced2", "Metro Malice", MapMannhattan, DifficultyAdvanced, 6, false, true},
	{26, "mvm_rottenburg", "Village Vanguard", MapRottenburg, DifficultyIntermediate, 7, true, true},
	{27, "mvm_rottenburg_advanced1", "Hamlet Hostility", MapRottenburg, DifficultyAdvanced, 7, true, true},
	{28, "mvm_rottenburg_advanced2", "Bavarian Botbash", MapRottenburg, DifficultyAdvanced, 7, true, true},
	{29, "mvm_ghost_town_666", "Caliginous Caper", MapGhostTown, DifficultyHaunted, 1, true, true},
}

// Missions contains Valve's missions followed by the entries in community.json.
var Missions = append(append([]Mission(nil), valveMissions...), communityMissions...)

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

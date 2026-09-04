package gamedata

import "slices"

// ServerMod is a server-side mod a community mission can name in its
// `requires` field. The server that plays the mission must have it loaded.
// Keys are permanent: a seed generated with server_mods naming one must mean
// the same mod for as long as that seed is played.
type ServerMod struct {
	Key  string
	Name string
	// Linux and Windows say which dedicated servers upstream ships a build
	// for. The Windows launcher cannot offer a mission whose mod has no
	// Windows build, however the operator asks.
	Linux   bool
	Windows bool
}

// ServerMods is the catalog. Versions and checksums live in
// deploy/env/versions.env, which is where every pin of this project lives.
var ServerMods = []ServerMod{
	{Key: "sigsegv-mvm", Name: "SigMod", Linux: true, Windows: false},
}

// noNavRequirement marks a mission whose map ships no bot navigation mesh.
// It is a fact about the pack, not a mod, and nothing can enable it.
const noNavRequirement = "no_nav"

// ServerModByKey finds a mod by its manifest key.
func ServerModByKey(key string) (ServerMod, bool) {
	for _, mod := range ServerMods {
		if mod.Key == key {
			return mod, true
		}
	}
	return ServerMod{}, false
}

// ServerModKeys lists the catalog's keys in catalog order.
func ServerModKeys() []string {
	keys := make([]string, 0, len(ServerMods))
	for _, mod := range ServerMods {
		keys = append(keys, mod.Key)
	}
	return keys
}

func knownRequirement(requirement string) bool {
	if requirement == "" || requirement == noNavRequirement {
		return true
	}
	_, known := ServerModByKey(requirement)
	return known
}

// MissionServerMod is the mod a mission needs, or blank when it needs none
// or when what it lacks is not a mod at all.
func MissionServerMod(id MissionID) string {
	requirement := MissionRequirement(id)
	if _, isMod := ServerModByKey(requirement); isMod {
		return requirement
	}
	return ""
}

// IsMissionPlayableWith reports whether a server that loads mods can put the
// mission in a seed. A mission with no requirement always can; one that
// names a mod can when the mod is among those; a no_nav mission never can.
func IsMissionPlayableWith(id MissionID, mods []string) bool {
	if IsPlayableMission(id) {
		return true
	}
	mod := MissionServerMod(id)
	return mod != "" && slices.Contains(mods, mod)
}

// MissionsPlayableWith is PlayableMissions for a server that loads mods.
func MissionsPlayableWith(mods []string) []Mission {
	out := make([]Mission, 0, len(Missions))
	for _, mission := range Missions {
		if IsMissionPlayableWith(mission.ID, mods) {
			out = append(out, mission)
		}
	}
	return out
}

// RequirementLabel says in one line why a mission is not offered, for a
// mission table.
func RequirementLabel(requirement string) string {
	switch requirement {
	case "":
		return "Ready"
	case noNavRequirement:
		return "Missing bot .nav"
	}
	mod, known := ServerModByKey(requirement)
	if !known {
		return "Needs " + requirement
	}
	if !mod.Windows {
		return "Needs " + mod.Name + " (Linux server only)"
	}
	return "Needs " + mod.Name
}

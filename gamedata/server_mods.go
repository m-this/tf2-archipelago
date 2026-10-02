package gamedata

import "slices"

// ServerMod is a server-side mod a community mission can name in its
// `requires` field. The server that plays the mission must have it loaded.
// Keys are permanent: a seed generated with server_mods naming one must mean
// the same mod for as long as that seed is played.
type ServerMod struct {
	Key  string
	Name string
	// Linux and Windows say which dedicated servers there is a build for. The
	// launcher cannot offer a mission whose mod has no build for the platform
	// it runs on, however the operator asks.
	Linux   bool
	Windows bool
}

/*
ServerMods is the catalog. Versions and checksums live in
deploy/env/versions.env, which is where every pin of this project lives.

Both SigMod builds come from m-this/sigsegv-mvm-win: Linux from its master,
which is upstream rebuilt against the current TF2 SDK, and Windows from the port.

Windows is true, and what makes that safe is the loading setting rather than
this flag. v1.17.0 shipped Windows with no such setting: installing SigMod was
loading it, on every map, and a host whose server died with
STATUS_HEAP_CORRUPTION could not turn it off from the launcher. The default now
loads a mod only while the pool holds a mission that names it, so a run that
never asks for SigMod never has it in the process.

The Windows port is this project's own, from m-this/sigsegv-mvm-win: upstream
publishes Linux only. It runs a community mission under Wine with the whole
stack loaded, and it crashed one player's real Windows server. What is left to
verify is apw-5g4.12 and apw-5g4.13.
*/
var ServerMods = []ServerMod{
	{Key: "sigsegv-mvm", Name: "SigMod", Linux: true, Windows: true},
}

// noNavRequirement marks a mission whose map ships no bot navigation mesh.
// It is a fact about the pack, not a mod, and nothing can enable it.
const noNavRequirement = "no_nav"

// BuildsOn reports whether there is a dedicated server build of this mod for
// a platform, named the way Go names it. A mod with no build there cannot be
// installed, cannot be selected, and cannot hold a mission in the pool.
func (m ServerMod) BuildsOn(goos string) bool {
	switch goos {
	case "windows":
		return m.Windows
	case "linux":
		return m.Linux
	}
	return false
}

// ServerModBuildsOn is BuildsOn for a key the caller has not looked up. An
// unknown key builds nowhere.
func ServerModBuildsOn(key, goos string) bool {
	mod, known := ServerModByKey(key)
	return known && mod.BuildsOn(goos)
}

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

// MissingNavigationMesh names the exact asset a no-nav mission lacks. The
// mission table uses the path rather than a generic "missing .nav" warning so
// two missions on different maps do not become indistinguishable.
func MissingNavigationMesh(id MissionID) string {
	if MissionRequirement(id) != noNavRequirement {
		return ""
	}
	mission, ok := MissionByID(id)
	if !ok {
		return ""
	}
	played, ok := MapByID(mission.Map)
	if !ok {
		return ""
	}
	return "maps/" + played.Name + ".nav"
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
	return "Needs " + mod.Name
}

package availability

import "github.com/m-this/tf2-archipelago/gamedata"

// Reason is why a server cannot play a mission. Playable is the zero value.
type Reason uint8

const (
	Playable Reason = iota
	// NoNavMesh: the map ships without a bot navigation mesh. Nothing on the
	// server can fix that, so the mission is never drawn.
	NoNavMesh
	// PackMissing: the community pack the mission comes from has no valid
	// ZIP on disk.
	PackMissing
	// ModNoBuild: the mission names a server mod with no build for the
	// server's platform.
	ModNoBuild
	// ModOff: the mission names a server mod that is not turned on.
	ModOff
	// ModUnverified: the mod is turned on, and its installation was not found
	// or did not check out.
	ModUnverified
	// ModBlocked: the server refused to load the mod on its TF2 build.
	ModBlocked
	// PackOff: the pack is on disk but not ticked, so Start never installs it.
	PackOff
)

func (r Reason) String() string {
	switch r {
	case Playable:
		return "playable"
	case NoNavMesh:
		return "no navigation mesh"
	case PackMissing:
		return "pack missing"
	case ModNoBuild:
		return "mod has no build"
	case ModOff:
		return "mod off"
	case ModUnverified:
		return "mod unverified"
	case ModBlocked:
		return "mod did not load"
	case PackOff:
		return "pack off"
	}
	return "unknown"
}

// Verdict is the answer for one mission on one server.
type Verdict struct {
	Mission gamedata.Mission
	Reason  Reason

	managed  bool
	platform string
}

// Playable reports whether the server can play the mission.
func (v Verdict) Playable() bool { return v.Reason == Playable }

/*
Why says what stands in the way and what removes it, in the words the Missions
page uses. Blank for a playable mission.

Every interface that explains a mission it will not offer says this sentence:
the row's tick, the table's column and the bridge's refusal. They used to say
it three ways, and the table told a player "Ready" for a mission the bridge
then refused.
*/
func (v Verdict) Why() string {
	pack := gamedata.PackLabel(gamedata.MissionPack(v.Mission.ID))
	mod, _ := gamedata.ServerModByKey(gamedata.MissionServerMod(v.Mission.ID))
	switch v.Reason {
	case Playable:
		return ""
	case NoNavMesh:
		return "missing " + gamedata.MissingNavigationMesh(v.Mission.ID)
	case PackMissing:
		return pack + " is not on this server: download or import it on the Missions page"
	case ModNoBuild:
		return mod.Name + " has no " + v.platform + " server build, so this mission cannot run here"
	case ModOff:
		if v.managed {
			return "turn on " + mod.Name + ", then save and recreate the containers"
		}
		return "turn on " + mod.Name + ", then press Download / set up selected server mods"
	case ModUnverified:
		if v.managed {
			return mod.Name + " files are missing or incomplete in the server volume; recreate the container"
		}
		return mod.Name + " is selected but its installation is missing or incomplete; press Download / set up selected server mods"
	case ModBlocked:
		return mod.Name + " did not load on this TF2 build; update the launcher"
	case PackOff:
		return "tick " + pack
	}
	return v.Reason.String()
}

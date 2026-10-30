/*
Package availability answers one question: can this server play this mission,
and if not, why not.

The catalog in gamedata says what a mission needs: the community pack it comes
from, the server mod it names, a navigation mesh its map ships without. Server
says what one game server has. Check puts the two together, and every place
that offers, counts, draws or refuses a mission asks it rather than assembling
the answer from the catalog's predicates on its own. Nine places used to, with
different inputs, and they disagreed: the mission table counted mods that were
verified and packs that were ticked, the pool ceiling counted neither, and the
bridge looked at the disk.

What a server has is passed in, never looked up. The launcher reads it off its
settings and its install, the bridge is handed it by the launcher that runs it,
and a test writes it down.
*/
package availability

import (
	"slices"

	"github.com/m-this/tf2-archipelago/gamedata"
)

/*
Server is what one game server holds, as far as missions are concerned.

Selected is what the settings ask for; present and verified are what is on the
disk. They are separate because the fix is: a pack on disk that is not ticked
is never installed into the server, and a mod that is ticked but not verified
has to be set up before it loads.
*/
type Server struct {
	// Platform is the GOOS the dedicated server runs on. Blank rules no mod
	// out for want of a build, which is what the apworld does: it cannot know.
	Platform string

	// PacksSelected are the community archives the server installs.
	// PacksPresent are those with a valid ZIP on disk.
	PacksSelected []string
	PacksPresent  []string

	// ModsSelected are the server mods turned on. ModsVerified are those whose
	// pinned installation was found and checked.
	ModsSelected []string
	ModsVerified []string

	// ManagedExternally means Docker Compose owns the install, so the fix for a
	// missing mod is recreating a container rather than pressing a button.
	ManagedExternally bool
}

// Planned is the server a set of settings describes once Start has set it up:
// every ticked pack in place and every ticked mod installed. It is what the
// apworld assumes when it reads server_mods, so the pool the launcher counts
// and the pool the generator draws from are the same.
func Planned(packs, mods []string) Server {
	return Server{PacksSelected: packs, PacksPresent: packs, ModsSelected: mods, ModsVerified: mods}
}

// Check is the verdict on one mission. A pack that is on disk and not ticked
// is the last thing it says, because ticking the pack is all that is left to
// do: a mission that also needs a mod the server cannot load says so first.
func (s Server) Check(mission gamedata.Mission) Verdict {
	verdict := Verdict{
		Mission: mission, managed: s.ManagedExternally, platform: s.Platform,
		Reason: s.reason(mission),
	}
	return verdict
}

func (s Server) reason(mission gamedata.Mission) Reason {
	pack := gamedata.MissionPack(mission.ID)
	if pack != "" && !slices.Contains(s.PacksPresent, pack) {
		return PackMissing
	}
	if gamedata.MissionRequirement(mission.ID) == noNavRequirement {
		return NoNavMesh
	}
	if key := gamedata.MissionServerMod(mission.ID); key != "" {
		mod, _ := gamedata.ServerModByKey(key)
		switch {
		case s.Platform != "" && !mod.BuildsOn(s.Platform):
			return ModNoBuild
		case !slices.Contains(s.ModsSelected, key):
			return ModOff
		case !slices.Contains(s.ModsVerified, key):
			return ModUnverified
		}
	}
	if pack != "" && !slices.Contains(s.PacksSelected, pack) {
		return PackOff
	}
	return Playable
}

// Playable is every mission Check passes, in table order.
func (s Server) Playable() []gamedata.Mission {
	var playable []gamedata.Mission
	for _, mission := range gamedata.Missions {
		if s.Check(mission).Playable() {
			playable = append(playable, mission)
		}
	}
	return playable
}

// Shown is the verdict on every mission a mission table lists, in table
// order: all of them but those whose pack is not on disk. A row that says why
// it is locked beats a mission that is not there, and a pack nobody has
// downloaded would be twenty rows of the same reason.
func (s Server) Shown() []Verdict {
	var shown []Verdict
	for _, mission := range gamedata.Missions {
		if verdict := s.Check(mission); verdict.Reason != PackMissing {
			shown = append(shown, verdict)
		}
	}
	return shown
}

// noNavRequirement is the catalog's word for a map that ships without a bot
// navigation mesh.
const noNavRequirement = "no_nav"

// Source is the column a mission table puts first: the pack a mission came
// from, Valve for the game's own, and Imported for a pack the player supplied
// from their own disk rather than downloaded.
func Source(mission gamedata.Mission, imported []string) string {
	pack := gamedata.MissionPack(mission.ID)
	if pack == "" {
		return "Valve"
	}
	if slices.Contains(imported, pack) {
		return "Imported"
	}
	if label := gamedata.PackLabel(pack); label != "" {
		return label
	}
	return pack
}

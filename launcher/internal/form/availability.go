package form

import (
	"github.com/m-this/tf2-archipelago/gamedata/availability"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

// Availability is the server the Missions page describes: the packs and mods
// the settings tick, and what the launcher found on disk. The rows, the
// mission table and the pool buttons all ask it, so they cannot disagree about
// a mission.
func Availability(s State, env Env) availability.Server {
	return availability.Server{
		Platform:          env.Platform,
		PacksSelected:     s.Settings.CommunityPacks,
		PacksPresent:      env.CommunityAvailable,
		ModsSelected:      settings.ServerModKeys(s.Settings),
		ModsVerified:      env.ServerModsReady,
		ModsBlocked:       env.ServerModsBlocked,
		ManagedExternally: env.ManagedExternally,
	}
}

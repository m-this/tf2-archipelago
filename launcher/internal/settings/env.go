package settings

import (
	"os"
	"strings"

	"github.com/m-this/tf2-archipelago/launcher/internal/botloadout"
)

/*
EnvVar is one variable of the environment: the field it carries, how it is
read and written, and the Compose services that read it.

Services is what deploy/compose.yml and deploy/compose.seed.yml must hand that
variable to, and TestComposePassesEveryVariableToItsReaders holds them to it.
The admin service is never named: it loads the whole of .env, because it is
what writes it. A variable no service names is one only the launcher and the
admin page act on.

The archipelago and seed services both run playeryaml, so both read whatever
the player file renders.
*/
type EnvVar struct {
	Name     string
	Services []string

	read func(*Settings, string)
	// write is nil for a variable the admin page never writes into .env.
	write func(Settings) string
}

var (
	playerFile       = []string{"archipelago", "seed"}
	playerFileBridge = []string{"archipelago", "seed", "bridge"}
	// missionPool is read by the game server too: which missions can come up
	// decides which server mods it loads.
	missionPool = []string{"archipelago", "seed", "bridge", "srcds"}
	gameServer  = []string{"srcds"}
)

/*
EnvVars is every variable ApplyEnv reads, in the order it reads them and the
help prints them. Adding a setting to the environment is a line here: ApplyEnv,
the .env the admin page writes, and the Compose tests all come from it.

Order matters where two variables set one field: the later one wins. The names
are the ones deploy/.env.example already uses, so a compose operator's file
works here unchanged.
*/
var EnvVars = []EnvVar{
	bind("TF2AP_INSTALL_ROOT", nil, text, func(s *Settings) *string { return &s.InstallRoot }),
	bind("TF2AP_COMMUNITY_CONTENT_DIR", nil, text, func(s *Settings) *string { return &s.CommunityContentDir }),
	bind("TF2AP_COMMUNITY_PACKS", playerFile, words, func(s *Settings) *[]string { return &s.CommunityPacks }),
	bind("TF2AP_TEST_MODE", []string{"srcds", "bridge"}, boolean, func(s *Settings) *bool { return &s.TestMode }),
	bind("TF2AP_ARCHIPELAGO_DIR", nil, text, func(s *Settings) *string { return &s.ArchipelagoDir }),

	// The whole address in one variable, which is how the room page gives it.
	// The three parts stay readable for a compose .env, and win.
	alias("AP_ROOM", nil, readRoom),
	bind("AP_HOST", []string{"bridge"}, text, func(s *Settings) *string { return &s.APHost }),
	bind("AP_PORT", []string{"archipelago", "bridge"}, number, func(s *Settings) *int { return &s.APPort }),
	bind("AP_TLS", []string{"bridge"}, boolean, func(s *Settings) *bool { return &s.APTls }),
	readOnly(bind("AP_ROOM_URL", nil, text, func(s *Settings) *string { return &s.APRoomURL })),
	bind("AP_SLOT_NAME", playerFileBridge, text, func(s *Settings) *string { return &s.APSlotName }),
	bind("AP_PASSWORD", []string{"archipelago", "bridge"}, text, func(s *Settings) *string { return &s.APPassword }),

	bind("SRCDS_HOSTNAME", gameServer, text, func(s *Settings) *string { return &s.SrcdsHostname }),
	bind("SRCDS_RCONPW", gameServer, text, func(s *Settings) *string { return &s.SrcdsRconPw }),
	bind("SRCDS_PW", gameServer, text, func(s *Settings) *string { return &s.SrcdsPw }),
	bind("SRCDS_PORT", gameServer, number, func(s *Settings) *int { return &s.SrcdsPort }),
	bind("TF2AP_JOIN_HOST", nil, text, func(s *Settings) *string { return &s.SrcdsJoinHost }),
	bind("SRCDS_MAXPLAYERS", gameServer, number, func(s *Settings) *int { return &s.SrcdsMaxPlayers }),
	// The compose stack names a map; the launcher names a mission. Both work,
	// the mission wins. An empty one names nothing, which is what Compose passes.
	alias("SRCDS_STARTMAP", gameServer, readStartMap),
	bind("SRCDS_START_MISSION", gameServer, named, func(s *Settings) *string { return &s.SrcdsStartMission }),
	bind("SRCDS_TOKEN", gameServer, text, func(s *Settings) *string { return &s.SrcdsToken }),
	// The older spelling, covering two of the three reaches. SRCDS_REACH wins.
	alias("SRCDS_LAN", gameServer, readLan),
	bind("SRCDS_REACH", gameServer, reach, func(s *Settings) *Reach { return &s.SrcdsReach }),
	bind("SRCDS_ADMIN_STEAMIDS", gameServer, text, func(s *Settings) *string { return &s.SrcdsAdminSteamIDs }),
	bind("SRCDS_MODS", []string{"archipelago", "seed", "bridge", "srcds"}, words, func(s *Settings) *[]string { return &s.SrcdsMods }),
	bind("SRCDS_MOD_LOADING", gameServer, modLoading, func(s *Settings) *ModLoading { return &s.SrcdsModLoading }),
	bind("FASTDL_PORT", gameServer, number, func(s *Settings) *int { return &s.FastDLPort }),
	bind("SRCDS_DOWNLOADURL", gameServer, text, func(s *Settings) *string { return &s.SrcdsDownloadURL }),
	bind("TAILSCALE_FASTDL", []string{"srcds", "tailscale-fastdl"}, boolean, func(s *Settings) *bool { return &s.TailscaleFastDL }),

	bind("SRCDS_BOTS", gameServer, boolean, func(s *Settings) *bool { return &s.SrcdsBots }),
	bind("SRCDS_BOT_TEAM_SIZE", gameServer, number, func(s *Settings) *int { return &s.SrcdsBotTeamSize }),
	bind("SRCDS_BOT_CLASS_BLACKLIST", gameServer, words, func(s *Settings) *[]string { return &s.SrcdsBotClassBlacklist }),
	bind("SRCDS_BOT_LOADOUTS", gameServer, pairs, func(s *Settings) *map[string]string { return &s.SrcdsBotLoadouts }),
	bind("SRCDS_BOT_CUSTOM_LOADOUTS", gameServer, table[botloadout.Built](), func(s *Settings) *map[string]botloadout.Built { return &s.SrcdsBotCustomLoadouts }),
	bind("SRCDS_BOT_TEAM_PRESETS", nil, table[BotTeam](), func(s *Settings) *map[string]BotTeam { return &s.SrcdsBotTeamPresets }),
	bind("SRCDS_BOT_NAMES_EXCLUDED", gameServer, words, func(s *Settings) *[]string { return &s.SrcdsBotNamesExcluded }),
	bind("SRCDS_BOT_NAMES_ADDED", gameServer, words, func(s *Settings) *[]string { return &s.SrcdsBotNamesAdded }),
	bind("SRCDS_BOT_SEAT_NAMES", gameServer, seats, func(s *Settings) *[]string { return &s.SrcdsBotSeatNames }),
	bind("SRCDS_BLU_HEALTH_PCT", gameServer, number, func(s *Settings) *int { return &s.SrcdsBluHealthPct }),
	bind("SRCDS_BOT_TEAM_COMP", gameServer, seats, func(s *Settings) *[]string { return &s.SrcdsBotTeamComp }),
	bind("SRCDS_BOT_SEAT_LOADOUTS", gameServer, seats, func(s *Settings) *[]string { return &s.SrcdsBotSeatLoadouts }),
	bind("SRCDS_BOT_CARD_FORMS", gameServer, pairs, func(s *Settings) *map[string]string { return &s.SrcdsBotCardForms }),
	bind("SRCDS_BOT_HATS", gameServer, boolean, func(s *Settings) *bool { return &s.SrcdsBotHats }),
	bind("SRCDS_BOT_BUY_ANYWHERE", gameServer, boolean, func(s *Settings) *bool { return &s.SrcdsBotBuyAnywhere }),
	bind("SRCDS_BOT_HAT_EFFECTS", gameServer, boolean, func(s *Settings) *bool { return &s.SrcdsBotHatEffects }),
	bind("TF2AP_BOT_UPGRADES_CHAT", gameServer, boolean, func(s *Settings) *bool { return &s.BotUpgradesChat }),

	bind("MVM_MISSION_COUNT", playerFileBridge, number, func(s *Settings) *int { return &s.MvmMissionCount }),
	bind("MVM_DIFFICULTY", playerFileBridge, text, func(s *Settings) *string { return &s.MvmDifficulty }),
	bind("MVM_GOAL", playerFileBridge, text, func(s *Settings) *string { return &s.MvmGoal }),
	bind("MVM_MISSIONSANITY_PERCENTAGE", playerFile, number, func(s *Settings) *int { return &s.MvmMissionsanityPct }),
	bind("MVM_MISSION_MODIFIERS", playerFileBridge, boolean, func(s *Settings) *bool { return &s.MvmMissionModifiers }),
	bind("MVM_MINIMUM_MISSION_MODIFIERS", playerFileBridge, number, func(s *Settings) *int { return &s.MvmModifierMin }),
	bind("MVM_MAXIMUM_MISSION_MODIFIERS", playerFileBridge, number, func(s *Settings) *int { return &s.MvmModifierMax }),
	bind("MVM_MEDAL_ON_CLEAR", playerFile, boolean, func(s *Settings) *bool { return &s.MvmMedalOnClear }),
	bind("MVM_VICTORY_CACHES", playerFileBridge, boolean, func(s *Settings) *bool { return &s.MvmVictoryCaches }),
	bind("MVM_MILESTONE_CHECKS", playerFileBridge, boolean, func(s *Settings) *bool { return &s.MvmMilestoneChecks }),
	bind("MVM_CLASS_WEAPON_SLOTS", playerFile, classSlots, func(s *Settings) *ClassSlots { return &s.MvmClassWeaponSlots }),
	bind("MVM_GIANTSANITY", playerFileBridge, boolean, func(s *Settings) *bool { return &s.MvmGiantsanity }),
	bind("MVM_TANKSANITY", playerFileBridge, boolean, func(s *Settings) *bool { return &s.MvmTanksanity }),
	bind("MVM_SERVER_SETTINGS", playerFile, boolean, func(s *Settings) *bool { return &s.MvmServerSettings }),
	bind("MVM_DEATH_LINK", playerFileBridge, boolean, func(s *Settings) *bool { return &s.MvmDeathLink }),
	bind("MVM_EXCLUDED_MISSIONS", missionPool, words, func(s *Settings) *[]string { return &s.MvmExcludedMissions }),
	bind("MVM_START_MISSION", missionPool, text, func(s *Settings) *string { return &s.MvmStartMission }),
	bind("MVM_START_CLASS", playerFileBridge, text, func(s *Settings) *string { return &s.MvmStartClass }),
	bind("MVM_COMMUNITY_MISSIONS", missionPool, boolean, func(s *Settings) *bool { return &s.MvmCommunityMissions }),
	bind("MVM_MISSION_TICKET_IMPORTANCE", playerFile, text, func(s *Settings) *string { return &s.MvmMissionTicketImportance }),
	bind("MVM_CLASS_UNLOCK_IMPORTANCE", playerFile, text, func(s *Settings) *string { return &s.MvmClassUnlockImportance }),
	bind("MVM_WEAPON_SLOT_IMPORTANCE", playerFile, text, func(s *Settings) *string { return &s.MvmWeaponSlotImportance }),
	bind("MVM_WEAPON_BUFF_IMPORTANCE", playerFile, text, func(s *Settings) *string { return &s.MvmWeaponBuffImportance }),
	bind("MVM_CASH_REWARDS", playerFile, boolean, func(s *Settings) *bool { return &s.MvmCashRewards }),
	bind("MVM_BOT_CARDS", playerFile, boolean, func(s *Settings) *bool { return &s.MvmBotCards }),
	bind("MVM_STARTING_BOT_CARDS", playerFile, number, func(s *Settings) *int { return &s.MvmStartingBotCards }),
	bind("MVM_STARTING_BOT_CARD_MODE", playerFile, text, func(s *Settings) *string { return &s.MvmStartingBotCardMode }),
	bind("SRCDS_BOT_CARD_ROLLS", nil, table[string](), func(s *Settings) *map[string]string { return &s.SrcdsBotCardRolls }),
	bind("MVM_WEAPON_BUFF_PERCENTAGE", playerFile, number, func(s *Settings) *int { return &s.MvmWeaponBuffPct }),
	bind("MVM_WEAPON_BUFF_STACK_CHANCE", playerFile, number, func(s *Settings) *int { return &s.MvmWeaponBuffStackChance }),
	bind("MVM_TRAP_PERCENTAGE", playerFile, number, func(s *Settings) *int { return &s.MvmTrapPct }),
	bind("BRIDGE_METRICS_PORT", nil, number, func(s *Settings) *int { return &s.MetricsPort }),
}

// EnvNames is the names of EnvVars, in the same order.
var EnvNames = envNames()

func envNames() []string {
	names := make([]string, len(EnvVars))
	for i, v := range EnvVars {
		names[i] = v.Name
	}
	return names
}

/*
ApplyEnv overlays the environment on top of the saved settings. A variable
that is set wins over the file; one that is unset changes nothing, and so does
one whose value does not parse.

This is what makes the exe scriptable: a shortcut, a .bat or a CI job can
set SRCDS_RCONPW and AP_PORT and start a server with no prompts and no
config file. Values taken from the environment are not written back, so an
override for one run does not become the saved answer.
*/
func ApplyEnv(s Settings) Settings {
	for _, v := range EnvVars {
		if value, ok := os.LookupEnv(v.Name); ok {
			v.read(&s, value)
		}
	}
	return s
}

// EnvValues is what the admin page writes into .env for s, by name. The
// variables that only spell another one differently are not in it.
func EnvValues(s Settings) map[string]string {
	values := make(map[string]string, len(EnvVars))
	for _, v := range EnvVars {
		if v.write != nil {
			values[v.Name] = v.write(s)
		}
	}
	return values
}

func readRoom(s *Settings, value string) {
	if room, err := ParseRoom(value); err == nil {
		s.APHost, s.APPort, s.APTls = room.Host, room.Port, room.TLS
	}
}

func readStartMap(s *Settings, value string) {
	s.SrcdsStartMission = startMissionFor(value, s.SrcdsStartMission)
}

func readLan(s *Settings, value string) {
	lan, ok := boolean.parse(value)
	if !ok {
		return
	}
	s.SrcdsReach = ReachLan
	if !lan {
		s.SrcdsReach = ReachPort
	}
}

// SplitSeats is SplitList that keeps the empties, because a seat nobody named
// is still a seat. It drops the trailing empties, which name no seat.
func SplitSeats(value string) []string {
	var out []string
	last := -1
	for entry := range strings.SplitSeq(value, ",") {
		out = append(out, strings.TrimSpace(entry))
		if out[len(out)-1] != "" {
			last = len(out) - 1
		}
	}
	return out[:last+1]
}

// SplitList splits on commas and drops blanks, the way every list in an .env
// file is written.
func SplitList(value string) []string {
	var out []string
	for entry := range strings.SplitSeq(value, ",") {
		if entry = strings.TrimSpace(entry); entry != "" {
			out = append(out, entry)
		}
	}
	return out
}

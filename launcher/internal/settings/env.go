package settings

import (
	"os"
	"strconv"
	"strings"
)

// EnvNames lists every variable ApplyEnv reads, in the order the help prints
// them. The names are the ones deploy/.env.example already uses, so a compose
// operator's file works here unchanged.
var EnvNames = []string{
	"TF2AP_INSTALL_ROOT",
	"TF2AP_COMMUNITY_CONTENT_DIR", "TF2AP_COMMUNITY_PACKS",
	"TF2AP_TEST_MODE", "TF2AP_ARCHIPELAGO_DIR",
	"AP_ROOM", "AP_HOST", "AP_PORT", "AP_TLS", "AP_SLOT_NAME", "AP_PASSWORD",
	"SRCDS_HOSTNAME", "SRCDS_RCONPW", "SRCDS_PW", "SRCDS_PORT",
	"SRCDS_MAXPLAYERS", "SRCDS_START_MISSION", "SRCDS_STARTMAP", "SRCDS_TOKEN",
	"SRCDS_LAN", "SRCDS_REACH", "SRCDS_ADMIN_STEAMIDS", "SRCDS_MODS",
	"SRCDS_LAN", "SRCDS_REACH", "SRCDS_ADMIN_STEAMIDS",
	"FASTDL_PORT", "SRCDS_DOWNLOADURL", "TAILSCALE_FASTDL",
	"SRCDS_BOTS", "SRCDS_BOT_TEAM_SIZE", "SRCDS_BOT_CLASS_BLACKLIST", "SRCDS_BOT_LOADOUTS",
	"SRCDS_BLU_HEALTH_PCT",
	"SRCDS_BOT_TEAM_COMP",
	"SRCDS_BOT_SEAT_LOADOUTS",
	"SRCDS_BOT_HATS",
	"SRCDS_BOT_HAT_EFFECTS",
	"TF2AP_BOT_UPGRADES_CHAT",
	"MVM_MISSION_COUNT", "MVM_DIFFICULTY", "MVM_GOAL",
	"MVM_MISSIONSANITY_PERCENTAGE", "MVM_DEATH_LINK", "MVM_EXCLUDED_MISSIONS",
	"MVM_START_MISSION", "MVM_START_CLASS",
	"MVM_MISSION_TICKET_IMPORTANCE", "MVM_CLASS_UNLOCK_IMPORTANCE",
	"MVM_WEAPON_SLOT_IMPORTANCE", "MVM_WEAPON_BUFF_IMPORTANCE",
	"MVM_CASH_REWARDS", "MVM_WEAPON_BUFF_PERCENTAGE", "MVM_WEAPON_BUFF_STACK_CHANCE",
	"MVM_TRAP_PERCENTAGE",
	"BRIDGE_METRICS_PORT",
}

// ApplyEnv overlays the environment on top of the saved settings. A variable
// that is set wins over the file; one that is unset changes nothing.
//
// This is what makes the exe scriptable: a shortcut, a .bat or a CI job can
// set SRCDS_RCONPW and AP_PORT and start a server with no prompts and no
// config file. Values taken from the environment are not written back, so an
// override for one run does not become the saved answer.
// applyBotEnv is the bots' half: who fills RED, what they carry, and how far
// the robots are taken down for a short team. Split out because ApplyEnv reads
// as one list and a list has a length limit.
func applyBotEnv(s Settings) Settings {
	boolean(&s.SrcdsBots, "SRCDS_BOTS")
	num(&s.SrcdsBotTeamSize, "SRCDS_BOT_TEAM_SIZE")
	num(&s.SrcdsBluHealthPct, "SRCDS_BLU_HEALTH_PCT")
	list(&s.SrcdsBotClassBlacklist, "SRCDS_BOT_CLASS_BLACKLIST")
	pairs(&s.SrcdsBotLoadouts, "SRCDS_BOT_LOADOUTS")
	seatList(&s.SrcdsBotTeamComp, "SRCDS_BOT_TEAM_COMP")
	seatList(&s.SrcdsBotSeatLoadouts, "SRCDS_BOT_SEAT_LOADOUTS")
	boolean(&s.SrcdsBotHats, "SRCDS_BOT_HATS")
	boolean(&s.SrcdsBotHatEffects, "SRCDS_BOT_HAT_EFFECTS")
	boolean(&s.BotUpgradesChat, "TF2AP_BOT_UPGRADES_CHAT")
	return s
}

func ApplyEnv(s Settings) Settings {
	str(&s.InstallRoot, "TF2AP_INSTALL_ROOT")
	str(&s.CommunityContentDir, "TF2AP_COMMUNITY_CONTENT_DIR")
	list(&s.CommunityPacks, "TF2AP_COMMUNITY_PACKS")
	boolean(&s.TestMode, "TF2AP_TEST_MODE")
	str(&s.ArchipelagoDir, "TF2AP_ARCHIPELAGO_DIR")

	// AP_ROOM is the whole address in one variable, which is how the room page
	// gives it. The three parts stay readable for a compose .env.
	if value, ok := os.LookupEnv("AP_ROOM"); ok {
		if room, err := ParseRoom(value); err == nil {
			s.APHost, s.APPort, s.APTls = room.Host, room.Port, room.TLS
		}
	}
	str(&s.APHost, "AP_HOST")
	num(&s.APPort, "AP_PORT")
	boolean(&s.APTls, "AP_TLS")
	str(&s.APSlotName, "AP_SLOT_NAME")
	str(&s.APPassword, "AP_PASSWORD")

	str(&s.SrcdsHostname, "SRCDS_HOSTNAME")
	str(&s.SrcdsRconPw, "SRCDS_RCONPW")
	str(&s.SrcdsPw, "SRCDS_PW")
	num(&s.SrcdsPort, "SRCDS_PORT")
	num(&s.SrcdsMaxPlayers, "SRCDS_MAXPLAYERS")
	// The compose stack names a map; the launcher names a mission. Both work,
	// the mission wins.
	if value, ok := os.LookupEnv("SRCDS_STARTMAP"); ok {
		s.SrcdsStartMission = startMissionFor(value, s.SrcdsStartMission)
	}
	str(&s.SrcdsStartMission, "SRCDS_START_MISSION")
	str(&s.SrcdsToken, "SRCDS_TOKEN")

	// SRCDS_LAN is the older spelling and covers two of the three reaches, so
	// SRCDS_REACH is read after it and wins when both are set.
	if lan, ok := lookupBool("SRCDS_LAN"); ok {
		s.SrcdsReach = ReachLan
		if !lan {
			s.SrcdsReach = ReachPort
		}
	}
	if value, ok := os.LookupEnv("SRCDS_REACH"); ok {
		if reach, ok := ParseReach(strings.ToLower(strings.TrimSpace(value))); ok {
			s.SrcdsReach = reach
		}
	}
	str(&s.SrcdsAdminSteamIDs, "SRCDS_ADMIN_STEAMIDS")
	list(&s.SrcdsMods, "SRCDS_MODS")
	num(&s.FastDLPort, "FASTDL_PORT")
	str(&s.SrcdsDownloadURL, "SRCDS_DOWNLOADURL")
	boolean(&s.TailscaleFastDL, "TAILSCALE_FASTDL")
	s = applyBotEnv(s)

	num(&s.MvmMissionCount, "MVM_MISSION_COUNT")
	str(&s.MvmDifficulty, "MVM_DIFFICULTY")
	str(&s.MvmGoal, "MVM_GOAL")
	num(&s.MvmMissionsanityPct, "MVM_MISSIONSANITY_PERCENTAGE")
	boolean(&s.MvmDeathLink, "MVM_DEATH_LINK")
	list(&s.MvmExcludedMissions, "MVM_EXCLUDED_MISSIONS")
	str(&s.MvmStartMission, "MVM_START_MISSION")
	str(&s.MvmStartClass, "MVM_START_CLASS")
	applyRewardEnv(&s)

	num(&s.MetricsPort, "BRIDGE_METRICS_PORT")
	return s
}

func applyRewardEnv(s *Settings) {
	str(&s.MvmMissionTicketImportance, "MVM_MISSION_TICKET_IMPORTANCE")
	str(&s.MvmClassUnlockImportance, "MVM_CLASS_UNLOCK_IMPORTANCE")
	str(&s.MvmWeaponSlotImportance, "MVM_WEAPON_SLOT_IMPORTANCE")
	str(&s.MvmWeaponBuffImportance, "MVM_WEAPON_BUFF_IMPORTANCE")
	boolean(&s.MvmCashRewards, "MVM_CASH_REWARDS")
	num(&s.MvmWeaponBuffPct, "MVM_WEAPON_BUFF_PERCENTAGE")
	num(&s.MvmWeaponBuffStackChance, "MVM_WEAPON_BUFF_STACK_CHANCE")
	num(&s.MvmTrapPct, "MVM_TRAP_PERCENTAGE")
}

// FromEnv reports whether name is set, so the UI can skip a prompt whose
// answer the environment already gave.
func FromEnv(name string) bool {
	_, ok := os.LookupEnv(name)
	return ok
}

func str(target *string, name string) {
	if value, ok := os.LookupEnv(name); ok {
		*target = value
	}
}

func num(target *int, name string) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return
	}
	*target = parsed
}

// boolean accepts the spellings an .env file and a shell both produce.
func boolean(target *bool, name string) {
	if value, ok := lookupBool(name); ok {
		*target = value
	}
}

// Truthy reads one of the spellings an .env file and a shell both produce.
// Anything else, including an empty string, is false: a flag nobody set stays
// off, and so does one somebody spelled wrong.
func Truthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// lookupBool is boolean for a caller that does not have a *bool to write into.
// The second result is false both when the variable is unset and when it holds
// something that is not a boolean at all.
func lookupBool(name string) (bool, bool) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return false, false
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true, true
	case "0", "false", "no", "off":
		return false, true
	}
	return false, false
}

// list reads a comma-separated value. An empty value empties the list, which
// is how a shortcut clears a saved one.
func list(target *[]string, name string) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return
	}
	*target = SplitList(value)
}

// seatList reads a list where entry n is seat n of RED. Drop the empties the
// way list does and every seat after a draw moves up one.
func seatList(target *[]string, name string) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return
	}
	*target = SplitSeats(value)
}

// pairs reads "key=value,key=value".
func pairs(target *map[string]string, name string) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return
	}
	parsed := make(map[string]string)
	for _, entry := range SplitList(value) {
		key, item, found := strings.Cut(entry, "=")
		if found && key != "" {
			parsed[strings.TrimSpace(key)] = strings.TrimSpace(item)
		}
	}
	*target = parsed
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

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
	"AP_HOST", "AP_PORT", "AP_TLS", "AP_SLOT_NAME", "AP_PASSWORD",
	"SRCDS_HOSTNAME", "SRCDS_RCONPW", "SRCDS_PW", "SRCDS_PORT",
	"SRCDS_MAXPLAYERS", "SRCDS_STARTMAP", "SRCDS_TOKEN", "SRCDS_LAN",
	"SRCDS_ADMIN_STEAMIDS", "SRCDS_BOTS", "SRCDS_BOT_TEAM_SIZE",
	"MVM_MISSION_COUNT", "MVM_DIFFICULTY", "MVM_GOAL",
	"MVM_MISSIONSANITY_PERCENTAGE", "MVM_DEATH_LINK",
	"BRIDGE_METRICS_PORT",
}

// ApplyEnv overlays the environment on top of the saved settings. A variable
// that is set wins over the file; one that is unset changes nothing.
//
// This is what makes the exe scriptable: a shortcut, a .bat or a CI job can
// set SRCDS_RCONPW and AP_PORT and start a server with no prompts and no
// config file. Values taken from the environment are not written back, so an
// override for one run does not become the saved answer.
func ApplyEnv(s Settings) Settings {
	str(&s.InstallRoot, "TF2AP_INSTALL_ROOT")

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
	str(&s.SrcdsStartMap, "SRCDS_STARTMAP")
	str(&s.SrcdsToken, "SRCDS_TOKEN")
	boolean(&s.SrcdsLan, "SRCDS_LAN")
	str(&s.SrcdsAdminSteamIDs, "SRCDS_ADMIN_STEAMIDS")
	boolean(&s.SrcdsBots, "SRCDS_BOTS")
	num(&s.SrcdsBotTeamSize, "SRCDS_BOT_TEAM_SIZE")

	num(&s.MvmMissionCount, "MVM_MISSION_COUNT")
	str(&s.MvmDifficulty, "MVM_DIFFICULTY")
	str(&s.MvmGoal, "MVM_GOAL")
	num(&s.MvmMissionsanityPct, "MVM_MISSIONSANITY_PERCENTAGE")
	boolean(&s.MvmDeathLink, "MVM_DEATH_LINK")

	num(&s.MetricsPort, "BRIDGE_METRICS_PORT")
	return s
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
	value, ok := os.LookupEnv(name)
	if !ok {
		return
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		*target = true
	case "0", "false", "no", "off":
		*target = false
	}
}

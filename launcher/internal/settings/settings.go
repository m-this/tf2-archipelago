// Package settings is the launcher's persisted user configuration. One JSON
// file under the OS's user config dir (on Windows, %APPDATA%\tf2ap\config.json),
// written on every successful start, restored on the next launch. It is the
// "last used parameters" memory the prompt asked for.
package settings

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/m-this/tf2-archipelago/gamedata"
	"github.com/m-this/tf2-archipelago/launcher/internal/botloadout"
)

// Settings is everything the launcher asks for and remembers. Fields are the
// .env values from the compose stack, renamed to the launcher's vocabulary.
type Settings struct {
	// InstallRoot is where the 14 GB of game files, SourceMod and steamcmd
	// live. Default is <home>/tf2-archipelago.
	InstallRoot string `json:"install_root"`

	// ArchipelagoDir is where the Archipelago app is, when it is not where its
	// installer puts it. Empty means look in the usual places.
	ArchipelagoDir string `json:"archipelago_dir,omitempty"`

	// CommunityContentDir holds downloaded or user-supplied community archives.
	// CommunityPacks names the archives the launcher installs into SRCDS.
	CommunityContentDir string   `json:"community_content_dir,omitempty"`
	CommunityPacks      []string `json:"community_packs"`

	// TestMode plays without Archipelago: the launcher serves a multiworld of
	// one on loopback, makes up a seed, and hands out unlocks as waves are
	// cleared. For trying the server out, and for play-testing.
	TestMode bool `json:"test_mode"`

	// Archipelago room.
	APHost     string `json:"ap_host"`
	APPort     int    `json:"ap_port"`
	APTls      bool   `json:"ap_tls"`
	APSlotName string `json:"ap_slot_name"`
	APPassword string `json:"ap_password,omitempty"`

	// Game server.
	SrcdsHostname      string `json:"srcds_hostname"`
	SrcdsRconPw        string `json:"srcds_rcon_pw,omitempty"`
	SrcdsPw            string `json:"srcds_pw,omitempty"`
	SrcdsPort          int    `json:"srcds_port"`
	SrcdsMaxPlayers    int    `json:"srcds_max_players"`
	SrcdsToken         string `json:"srcds_token"`
	SrcdsReach         Reach  `json:"srcds_reach"`
	SrcdsAdminSteamIDs string `json:"srcds_admin_steamids,omitempty"`

	// SrcdsMods are the server mods the game server loads, by the keys
	// gamedata catalogs. A community mission that needs one is offered only
	// when the mod is here, and the player file names the same mods so the
	// seed and the server agree. The Windows launcher installs none of them
	// yet: it only carries the choice through for a server that has one.
	SrcdsMods []string `json:"srcds_mods"`
	/*
		FastDLPort is where the launcher serves the game's maps and other
		content over HTTP, on this machine, so a joining client downloads
		a community map from here rather than through the game server. 0
		turns the server off.

		SrcdsDownloadURL is the whole sv_downloadurl value, for an
		operator who knows an address this machine cannot work out. The
		two are separate on purpose: this one says where a client is told
		to look, the port above says whether this machine answers. A
		server reached over a forwarded port needs both, because the
		address friends use is the router's and nothing here can see it.
	*/
	FastDLPort       int    `json:"fastdl_port"`
	SrcdsDownloadURL string `json:"srcds_download_url,omitempty"`
	// TailscaleFastDL asks Tailscale Funnel to publish only the downloadable
	// content directories and gives its public HTTPS URL to SRCDS. It changes
	// no game-server address or reach setting. Only the server runs Tailscale.
	TailscaleFastDL bool `json:"tailscale_fastdl,omitempty"`

	// SrcdsStartMission is the popfile the server loads first. The map comes
	// with it: gamedata knows which map a mission runs on.
	SrcdsStartMission string `json:"srcds_start_mission"`

	// SrcdsStartMap is what older files hold. Read for the migration to
	// SrcdsStartMission and never written again.
	SrcdsStartMap string `json:"srcds_start_map,omitempty"`

	// SrcdsLanLegacy is the srcds_lan boolean SrcdsReach replaced. It is read
	// once, to pick the reach a config file written before 1.3 meant, and
	// never written back. Drop it when no such file is left.
	SrcdsLanLegacy *bool `json:"srcds_lan,omitempty"`

	// Defender bots. RED is filled to SrcdsBotTeamSize when a wave begins.
	SrcdsBots        bool `json:"srcds_bots"`
	SrcdsBotTeamSize int  `json:"srcds_bot_team_size"`

	// SrcdsBotClassBlacklist names the classes the bots never play, in the
	// mod's spelling (heavyweapons, not heavy). SrcdsBotLoadouts picks a
	// loadout preset per class, keyed the same way; a class not in it plays
	// stock.
	SrcdsBotClassBlacklist []string          `json:"srcds_bot_class_blacklist,omitempty"`
	SrcdsBotLoadouts       map[string]string `json:"srcds_bot_loadouts,omitempty"`

	// SrcdsBotTeamComp names the classes the bots fill RED with, in order,
	// keyed the same way. Empty leaves the mod to draw its own team, which is
	// what gave a play-test three Spies and two Scouts on an Advanced mission.
	// A team named here beats the blacklist.
	SrcdsBotTeamComp []string `json:"srcds_bot_team_comp,omitempty"`

	// SrcdsBotTeamPresets are teams somebody named and kept: the seats, their
	// loadouts, and the classes the mod may draw from. Naming a team is the
	// point of the Bots tab, and naming it twice because the last one was
	// overwritten is not.
	SrcdsBotTeamPresets map[string]BotTeam `json:"srcds_bot_team_presets,omitempty"`

	// SrcdsBotSeatLoadouts is the loadout each seat carries, in the same order
	// and keyed by loadout preset. It is what lets one engineer hold the
	// Wrangler and the next one hold something else, which the per-class map
	// above cannot say.
	//
	// A seat with no entry falls back to the class's own pick.
	SrcdsBotSeatLoadouts []string `json:"srcds_bot_seat_loadouts,omitempty"`

	/* SrcdsBotCustomLoadouts is the loadouts the player has built, keyed by the
	 * name they gave. A seat or a class names one with the custom: prefix, so a
	 * built loadout is a loadout key like any other and nothing downstream has
	 * to know the difference.
	 *
	 * A team naming one that has since been deleted plays stock, which is the
	 * same rule an unknown preset key already follows.
	 */
	SrcdsBotCustomLoadouts map[string]botloadout.Built `json:"srcds_bot_custom_loadouts,omitempty"`

	// BotUpgradesChat writes what the bots buy at the upgrade station to the
	// chat. Off by default: it is a line per purchase.
	BotUpgradesChat bool `json:"bot_upgrades_chat"`

	// What the bots look like, which changes nothing about how they play. A
	// hat each is on and so is the unusual effect on it: six mercenaries in the
	// same stock hat is what a bot team looks like otherwise, and this is the
	// part of the mod nobody has to think about.
	//
	// War paints were here and are gone. They painted the weapon entities the
	// upgrade station replaces, and the server died the moment two engineers
	// finished shopping.
	//
	// A config file written before these fields existed does not mention them,
	// and a bool nobody wrote reads back as false, so the whole lot arrived
	// switched off on every install that had ever saved a setting. Load tells
	// "the file said no" from "the file said nothing", which is what
	// SrcdsLanLegacy does a few fields down for the same reason.
	SrcdsBotHats       bool `json:"srcds_bot_hats"`
	SrcdsBotHatEffects bool `json:"srcds_bot_hat_effects"`

	// Run shape, for seed generation guidance (the launcher does not generate
	// seeds itself, but it can write a starter YAML for the Archipelago app).
	MvmMissionCount     int      `json:"mvm_mission_count"`
	MvmDifficulty       string   `json:"mvm_difficulty"`
	MvmGoal             string   `json:"mvm_goal"`
	MvmMissionsanityPct int      `json:"mvm_missionsanity_percentage"`
	MvmDeathLink        bool     `json:"mvm_death_link"`
	MvmExcludedMissions []string `json:"mvm_excluded_missions,omitempty"`

	// MvmStartMission is the popfile the run starts on and MvmStartClass the
	// mercenary it starts with, both empty for the seed's own random draw.
	// Popfile and not display name, the same as MvmExcludedMissions: every
	// other part of this project names a mission by its popfile, and yaml.go
	// does the one translation the apworld's options need.
	MvmStartMission string `json:"mvm_start_mission,omitempty"`
	MvmStartClass   string `json:"mvm_start_class,omitempty"`

	MvmMissionTicketImportance string `json:"mvm_mission_ticket_importance"`
	MvmClassUnlockImportance   string `json:"mvm_class_unlock_importance"`
	MvmWeaponSlotImportance    string `json:"mvm_weapon_slot_importance"`
	MvmWeaponBuffImportance    string `json:"mvm_weapon_buff_importance"`
	MvmCashRewards             bool   `json:"mvm_cash_rewards"`
	MvmWeaponBuffPct           int    `json:"mvm_weapon_buff_percentage"`
	MvmWeaponBuffStackChance   int    `json:"mvm_weapon_buff_stack_chance"`

	// MvmTrapPct is how much of the run's spare space is traps. Zero is off and
	// zero is the default, so it needs no entry in withDefaults. A config file
	// that predates it reads back as a run that asked for no traps.
	MvmTrapPct int `json:"mvm_trap_percentage"`

	/* A direct multiplier for every robot. 100 percent is neutral.
	 *
	 * Percentages rather than the mod's floats, because a settings page with
	 * 0.7 in a box asks the player to know what the 1.0 end means.
	 */
	SrcdsBluHealthPct int `json:"srcds_blu_health_pct"`

	// Whether to enable the metrics listener and on what port.
	MetricsPort int `json:"metrics_port"`
}

// FastDLPortDefault matches FASTDL_PORT in deploy/.env.example. Off the game
// port, whose TCP side srcds already holds for rcon.
const FastDLPortDefault = 27080

// Defaults returns the factory settings, matching deploy/.env.example.
func Defaults() Settings {
	return Settings{
		InstallRoot:         defaultInstallRoot(),
		CommunityContentDir: defaultCommunityContentDir(),
		APHost:              "archipelago.gg",
		APPort:              0,
		APTls:               true,
		APSlotName:          "tf2",
		SrcdsHostname:       "Mann vs Archipelago",
		SrcdsPort:           27015,
		SrcdsMaxPlayers:     32,
		SrcdsStartMission:   "mvm_decoy",
		SrcdsToken:          "0",
		// Lan, to match SrcdsToken above. Port reach with no token is the one
		// combination that cannot work: the server never logs in to Steam, so
		// it answers the query and then refuses the join.
		SrcdsReach:                 ReachLan,
		FastDLPort:                 FastDLPortDefault,
		SrcdsBots:                  true,
		SrcdsBotTeamSize:           6,
		SrcdsBotHats:               true,
		SrcdsBotHatEffects:         true,
		MvmMissionCount:            8,
		MvmDifficulty:              "intermediate",
		MvmGoal:                    "final_boss",
		MvmMissionsanityPct:        80,
		MvmExcludedMissions:        defaultExcludedMissions(),
		MvmMissionTicketImportance: "progression",
		MvmClassUnlockImportance:   "progression",
		MvmWeaponSlotImportance:    "progression",
		MvmWeaponBuffImportance:    "useful",
		MvmWeaponBuffPct:           75,
		MvmWeaponBuffStackChance:   25,
		SrcdsBluHealthPct:          RobotHealthPercentNeutral,
		MetricsPort:                24681,
	}
}

const (
	RobotHealthPercentMin     = 10
	RobotHealthPercentNeutral = 100
	RobotHealthPercentMax     = 1000
)

// BotTeam is one saved team: what each seat plays and holds, and which classes
// the mod may draw the rest from. The same three things the Bots tab edits, so
// loading one is a copy in and saving one is a copy out.
type BotTeam struct {
	Comp          []string          `json:"comp,omitempty"`
	SeatLoadouts  []string          `json:"seat_loadouts,omitempty"`
	ClassLoadouts map[string]string `json:"class_loadouts,omitempty"`
	Blacklist     []string          `json:"blacklist,omitempty"`
}

// BotTeamOf is the team the settings currently describe.
func BotTeamOf(s Settings) BotTeam {
	return BotTeam{
		Comp:          slices.Clone(s.SrcdsBotTeamComp),
		SeatLoadouts:  slices.Clone(s.SrcdsBotSeatLoadouts),
		ClassLoadouts: maps.Clone(s.SrcdsBotLoadouts),
		Blacklist:     slices.Clone(s.SrcdsBotClassBlacklist),
	}
}

// WithBotTeam is the settings with that team in place of the current one.
func WithBotTeam(s Settings, team BotTeam) Settings {
	s.SrcdsBotTeamComp = slices.Clone(team.Comp)
	s.SrcdsBotSeatLoadouts = slices.Clone(team.SeatLoadouts)
	s.SrcdsBotLoadouts = maps.Clone(team.ClassLoadouts)
	s.SrcdsBotClassBlacklist = slices.Clone(team.Blacklist)
	return s
}

// BotTeamNames are the saved teams, in the order a menu should show them.
func BotTeamNames(s Settings) []string {
	names := make([]string, 0, len(s.SrcdsBotTeamPresets))
	for name := range s.SrcdsBotTeamPresets {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// EffectiveReach is where the players can come from with the token this server
// has, which is not always the reach that was asked for. See Effective.
func (s Settings) EffectiveReach() Reach { return Effective(s.SrcdsReach, s.SrcdsToken) }

// Path returns the config file location. On Windows this is
// %APPDATA%\tf2ap\config.json; on Linux ~/.config/tf2ap/config.json.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot find the user config directory: %w", err)
	}
	return filepath.Join(dir, "tf2ap", "config.json"), nil
}

// Load reads the config file, applying defaults for anything unset. A missing
// file is not an error: it returns Defaults.
func Load() (Settings, error) {
	path, err := Path()
	if err != nil {
		return Settings{}, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Defaults(), nil
	}
	if err != nil {
		return Settings{}, fmt.Errorf("cannot read %s: %w", path, err)
	}
	s, err := parse(data)
	if err != nil {
		return Settings{}, fmt.Errorf("cannot parse %s: %w", path, err)
	}
	return s, nil
}

// parse is the config file's bytes as settings, defaults filled in. Separate
// from Load because the order matters and there is only one right one: the
// file, then the defaults for what it left out, then the three switches that
// have to tell "said no" from "said nothing".
func parse(data []byte) (Settings, error) {
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return Settings{}, err
	}
	return s.withDefaults().withAppearanceDefaults(data), nil
}

// withAppearanceDefaults fills the two appearance switches a file that
// predates them does not mention. Called by parse, after the file's own values
// are in.
//
// It reads the same bytes a second time rather than making the fields
// pointers: a *bool that every caller has to dereference is a worse trade than
// one function that knows why these three are different from the rest.
func (s Settings) withAppearanceDefaults(data []byte) Settings {
	var said struct {
		Hats                  *bool `json:"srcds_bot_hats"`
		Effects               *bool `json:"srcds_bot_hat_effects"`
		WeaponBuffPct         *int  `json:"mvm_weapon_buff_percentage"`
		WeaponBuffStackChance *int  `json:"mvm_weapon_buff_stack_chance"`
	}
	// A file that parsed once parses again; anything else has already been
	// reported by the caller.
	if err := json.Unmarshal(data, &said); err != nil {
		return s
	}
	d := Defaults()
	if said.Hats == nil {
		s.SrcdsBotHats = d.SrcdsBotHats
	}
	if said.Effects == nil {
		s.SrcdsBotHatEffects = d.SrcdsBotHatEffects
	}
	if said.WeaponBuffPct == nil {
		s.MvmWeaponBuffPct = d.MvmWeaponBuffPct
	}
	if said.WeaponBuffStackChance == nil {
		s.MvmWeaponBuffStackChance = d.MvmWeaponBuffStackChance
	}
	return s
}

// Render returns the settings as the JSON that Save writes. The debug bundle
// uses it to ship a copy with the passwords taken out.
func Render(s Settings) (string, error) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}

// Save writes the config file, creating the directory.
func Save(s Settings) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("cannot create %s: %w", filepath.Dir(path), err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("cannot write %s: %w", path, err)
	}
	return nil
}

// withDefaults fills zero values with the defaults, so an old config file that
// predates a field still works.
/* withRobotScaleDefaults fills the three robot scales a file that predates them
does not mention.

Nothing said is not nought per cent: zero would be robots at a tenth of their
damage for anybody who upgraded.

The window is what cannot survive it. Its number boxes for these take a minimum
of ten, walk refuses a value below the minimum, and a refused widget takes the
whole settings dialog with it. Reported as the settings window no longer opening
at all.
*/
func (s Settings) withRobotScaleDefaults(d Settings) Settings {
	if s.SrcdsBluHealthPct == 0 {
		s.SrcdsBluHealthPct = d.SrcdsBluHealthPct
	}
	return s
}

func (s Settings) withDefaults() Settings {
	d := Defaults()
	s = withCommunityDefaults(s, d)
	if s.InstallRoot == "" {
		s.InstallRoot = d.InstallRoot
	}
	if s.APSlotName == "" {
		s.APSlotName = d.APSlotName
	}
	if s.SrcdsHostname == "" {
		s.SrcdsHostname = d.SrcdsHostname
	}
	if s.SrcdsPort == 0 {
		s.SrcdsPort = d.SrcdsPort
	}
	if s.SrcdsMaxPlayers == 0 {
		s.SrcdsMaxPlayers = d.SrcdsMaxPlayers
	}
	if s.SrcdsBotTeamSize == 0 {
		s.SrcdsBotTeamSize = d.SrcdsBotTeamSize
	}
	s = s.withRobotScaleDefaults(d)
	if s.SrcdsStartMission == "" {
		s.SrcdsStartMission = startMissionFor(s.SrcdsStartMap, d.SrcdsStartMission)
	}
	if mission, known := gamedata.MissionByPopFile(s.SrcdsStartMission); known && !gamedata.IsPlayableMission(mission.ID) {
		s.SrcdsStartMission = d.SrcdsStartMission
	}
	s.SrcdsStartMap = ""
	if s.SrcdsToken == "" {
		s.SrcdsToken = d.SrcdsToken
	}
	if !s.SrcdsReach.Valid() {
		// A file that predates SrcdsReach says only whether sv_lan was on, and
		// that answer still stands: on meant the local network. A file that
		// says neither, or says something nobody recognizes, takes the default,
		// which EffectiveReach keeps local until there is a token to leave with.
		s.SrcdsReach = d.SrcdsReach
		if s.SrcdsLanLegacy != nil {
			if *s.SrcdsLanLegacy {
				s.SrcdsReach = ReachLan
			} else {
				s.SrcdsReach = ReachPort
			}
		}
	}
	s.SrcdsLanLegacy = nil
	if s.MvmMissionCount == 0 {
		s.MvmMissionCount = d.MvmMissionCount
	}
	if s.MvmDifficulty == "" {
		s.MvmDifficulty = d.MvmDifficulty
	}
	if s.MvmGoal == "" {
		s.MvmGoal = d.MvmGoal
	}
	if s.MvmMissionsanityPct == 0 {
		s.MvmMissionsanityPct = d.MvmMissionsanityPct
	}
	if s.MvmMissionTicketImportance == "" {
		s.MvmMissionTicketImportance = d.MvmMissionTicketImportance
	}
	if s.MvmClassUnlockImportance == "" {
		s.MvmClassUnlockImportance = d.MvmClassUnlockImportance
	}
	if s.MvmWeaponSlotImportance == "" {
		s.MvmWeaponSlotImportance = d.MvmWeaponSlotImportance
	}
	if s.MvmWeaponBuffImportance == "" {
		s.MvmWeaponBuffImportance = d.MvmWeaponBuffImportance
	}
	return withListenerDefaults(s, d)
}

// withListenerDefaults fills the two ports the launcher itself listens on.
func withListenerDefaults(s, d Settings) Settings {
	if s.MetricsPort == 0 {
		s.MetricsPort = d.MetricsPort
	}
	if s.FastDLPort == 0 {
		s.FastDLPort = d.FastDLPort
	}
	return s
}

func withCommunityDefaults(s, defaults Settings) Settings {
	if s.CommunityContentDir == "" {
		s.CommunityContentDir = defaults.CommunityContentDir
	}
	if s.MvmExcludedMissions == nil {
		s.MvmExcludedMissions = slices.Clone(defaults.MvmExcludedMissions)
	}
	if mission, known := gamedata.MissionByPopFile(s.MvmStartMission); known && !gamedata.IsPlayableMission(mission.ID) {
		s.MvmStartMission = ""
	}
	return s
}

func defaultExcludedMissions() []string {
	var excluded []string
	for _, mission := range gamedata.Missions {
		if gamedata.IsCommunityMission(mission.ID) {
			excluded = append(excluded, mission.PopFile)
		}
	}
	return excluded
}

const (
	CommunityPackPotato    = "archive-assets.zip"
	CommunityPackMoonlight = "mlarchive-assets.zip"
)

// CommunityArchives resolves selected pack names under their configured folder.
func CommunityArchives(s Settings) []string {
	paths := make([]string, 0, len(s.CommunityPacks))
	for _, name := range s.CommunityPacks {
		if name == CommunityPackPotato || name == CommunityPackMoonlight {
			paths = append(paths, filepath.Join(s.CommunityContentDir, name))
		}
	}
	return paths
}

// KnownCommunityArchives resolves every supported pack under dir, whether or
// not it is selected. The UIs use this to discover already-downloaded or local
// files before deciding which community missions may be shown.
func KnownCommunityArchives(dir string) []string {
	return []string{
		filepath.Join(dir, CommunityPackPotato),
		filepath.Join(dir, CommunityPackMoonlight),
	}
}

func defaultCommunityContentDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "tf2"
	}
	return filepath.Join(home, "tf2")
}

func defaultInstallRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "tf2-archipelago"
	}
	return filepath.Join(home, "tf2-archipelago")
}

// startMissionFor is the migration from a start map to a start mission: the
// first mission the tables list on that map, which is its normal tier.
func startMissionFor(mapName, fallback string) string {
	for _, mission := range gamedata.PlayableMissions() {
		if played, ok := gamedata.MapByID(mission.Map); ok && played.Name == mapName {
			return mission.PopFile
		}
	}
	return fallback
}

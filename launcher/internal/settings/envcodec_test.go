package settings

import (
	"flag"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/m-this/tf2-archipelago/launcher/internal/botloadout"
)

/*
everyField is a Settings with every field set to something no default holds,
so a field the codec drops or misreads changes the result.

TestEveryFieldIsInTheFixture keeps it whole: a field added to Settings and not
here fails, and once it is here the round trip fails until it has a variable.
*/
func everyField() Settings {
	lan := true
	return Settings{
		InstallRoot:         "/srv/tf2",
		ArchipelagoDir:      "/opt/archipelago",
		CommunityContentDir: "/srv/overlay",
		CommunityPacks:      []string{CommunityPackPotato, CommunityPackMoonlight},
		TestMode:            true,

		APHost:     "archipelago.gg",
		APPort:     38281,
		APTls:      true,
		APRoomURL:  "https://archipelago.gg/room/abc",
		APSlotName: "Mann Co",
		APPassword: "it's $ecret",

		SrcdsHostname:      "Mann's $erver",
		SrcdsRconPw:        "rcon",
		SrcdsPw:            "join",
		SrcdsPort:          27115,
		SrcdsJoinHost:      "tf2.example.com",
		SrcdsMaxPlayers:    12,
		SrcdsToken:         "ABCDEF",
		SrcdsReach:         ReachSteam,
		SrcdsAdminSteamIDs: "STEAM_0:1:1,STEAM_0:1:2",
		SrcdsMods:          []string{"sigsegv", "rafmod"},
		SrcdsModLoading:    ModLoadingAlways,
		FastDLPort:         27081,
		SrcdsDownloadURL:   "http://fastdl.example.com/tf",
		TailscaleFastDL:    true,
		SrcdsStartMission:  "mvm_coaltown_advanced",
		SrcdsStartMap:      "mvm_coaltown",
		SrcdsLanLegacy:     &lan,

		SrcdsBots:              true,
		SrcdsBotTeamSize:       4,
		SrcdsBotClassBlacklist: []string{"sniper", "spy"},
		SrcdsBotLoadouts:       map[string]string{"scout": "milk", "soldier": "banner"},
		SrcdsBotTeamComp:       []string{"", "engineer", "", "heavyweapons"},
		SrcdsBotNamesExcluded:  []string{"Ivan"},
		SrcdsBotNamesAdded:     []string{"Kelly", "Zoe"},
		SrcdsBotTeamPresets: map[string]BotTeam{"tanky": {
			Comp:          []string{"heavyweapons", "medic"},
			SeatLoadouts:  []string{"", "quickfix"},
			SeatNames:     []string{"Big", "Doc"},
			CardForms:     map[string]string{"medic": "vaccinator"},
			ClassLoadouts: map[string]string{"heavyweapons": "brass"},
			Blacklist:     []string{"spy"},
		}},
		SrcdsBotSeatLoadouts: []string{"ranger", "", "brass"},
		SrcdsBluHealthPct:    150,
		SrcdsBotSeatNames:    []string{"", "Kelly"},
		SrcdsBotCardForms:    map[string]string{"medic": "kritz"},
		SrcdsBotCustomLoadouts: map[string]botloadout.Built{
			"rocket": {Class: "soldier", Primary: 228, Second: 10, Melee: 6, PDA2: 1},
		},
		BotUpgradesChat:     true,
		SrcdsBotBuyAnywhere: true,
		SrcdsBotHats:        true,
		SrcdsBotHatEffects:  true,

		MvmMissionCount:            7,
		MvmDifficulty:              "expert",
		MvmGoal:                    "all_missions",
		MvmMissionsanityPct:        40,
		MvmMissionModifiers:        true,
		MvmModifierMin:             1,
		MvmModifierMax:             3,
		MvmMedalOnClear:            true,
		MvmVictoryCaches:           true,
		MvmMilestoneChecks:         true,
		MvmClassWeaponSlots:        ClassSlotsAnyOrder,
		MvmGiantsanity:             true,
		MvmTanksanity:              true,
		MvmServerSettings:          true,
		MvmDeathLink:               true,
		MvmExcludedMissions:        []string{"mvm_ghost_town_666"},
		MvmStartMission:            "mvm_decoy_advanced",
		MvmStartClass:              "medic",
		MvmCommunityMissions:       true,
		MvmMissionTicketImportance: "high",
		MvmClassUnlockImportance:   "low",
		MvmWeaponSlotImportance:    "none",
		MvmWeaponBuffImportance:    "normal",
		MvmCashRewards:             true,
		MvmBotCards:                true,
		MvmStartingBotCards:        2,
		MvmStartingBotCardMode:     "pick",
		SrcdsBotCardRolls:          map[string]string{"medic": "roll"},
		MvmWeaponBuffPct:           25,
		MvmWeaponBuffStackChance:   30,
		MvmTrapPct:                 15,
		MvmTraps:                   []string{"team_gas", "team_bleed"},
		MetricsPort:                9101,
	}
}

/*
notInTheEnvironment is every field the round trip through .env does not carry,
with the reason. An entry here is a decision, not a way to quieten the test.
*/
var notInTheEnvironment = map[string]string{
	"SrcdsLanLegacy": "read from an older config file and never saved",
	"SrcdsStartMap":  "read from an older config file; SRCDS_STARTMAP sets SrcdsStartMission",
	"APRoomURL":      "ApplyEnv reads AP_ROOM_URL, the admin page has never written it",
}

func withoutUncarried(s Settings) Settings {
	s.SrcdsLanLegacy = nil
	s.SrcdsStartMap = ""
	s.APRoomURL = ""
	return s
}

func TestEveryFieldIsInTheFixture(t *testing.T) {
	fixture := reflect.ValueOf(everyField())
	for field := range reflect.TypeFor[Settings]().Fields() {
		if fixture.FieldByIndex(field.Index).IsZero() {
			t.Errorf("everyField leaves %s at its zero value; give it one no default holds", field.Name)
		}
	}
	for field := range notInTheEnvironment {
		if _, ok := reflect.TypeFor[Settings]().FieldByName(field); !ok {
			t.Errorf("notInTheEnvironment names %s, which Settings no longer has", field)
		}
	}
}

var update = flag.Bool("update", false, "rewrite testdata/*.env from EnvValues")

// readEnvFile reads a golden file in the shape composeenv writes:
// NAME='value', with \' for a quote.
func readEnvFile(t *testing.T, name string) map[string]string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]string{}
	for line := range strings.Lines(string(body)) {
		key, value, ok := strings.Cut(strings.TrimSuffix(line, "\n"), "=")
		if !ok {
			t.Fatalf("%s: %q is not NAME=value", name, line)
		}
		value = strings.TrimSuffix(strings.TrimPrefix(value, "'"), "'")
		values[key] = strings.ReplaceAll(value, `\'`, "'")
	}
	return values
}

// The .env the admin page writes for everyField reads back as everyField.
func TestApplyEnvReadsTheGoldenFile(t *testing.T) {
	for name, value := range readEnvFile(t, "every_field.env") {
		t.Setenv(name, value)
	}
	got, want := ApplyEnv(Settings{}), withoutUncarried(everyField())
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ApplyEnv(every_field.env):\n got %+v\nwant %+v", got, want)
	}
}

/*
What ApplyEnv does with each kind of value, at the edges.

Each case starts from everyField, so a variable that touches more than the
field it names shows up as a difference too.
*/
func TestApplyEnvEdges(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want func(*Settings)
	}{
		{"an empty string empties a string", map[string]string{"SRCDS_PW": ""}, func(s *Settings) { s.SrcdsPw = "" }},
		{"a number is trimmed", map[string]string{"AP_PORT": " 12345 "}, func(s *Settings) { s.APPort = 12345 }},
		{"a number that is not one is ignored", map[string]string{"AP_PORT": "port"}, func(*Settings) {}},
		{"an empty number is ignored", map[string]string{"SRCDS_PORT": ""}, func(*Settings) {}},
		{"a boolean that is not one is ignored", map[string]string{"SRCDS_BOTS": "maybe"}, func(*Settings) {}},
		{"a boolean in capitals", map[string]string{"SRCDS_BOTS": " OFF "}, func(s *Settings) { s.SrcdsBots = false }},
		{"an empty list empties the list", map[string]string{"SRCDS_MODS": ""}, func(s *Settings) { s.SrcdsMods = nil }},
		{"a list drops its blanks", map[string]string{"SRCDS_MODS": " a, ,b,"}, func(s *Settings) { s.SrcdsMods = []string{"a", "b"} }},
		{"a seat list keeps its holes", map[string]string{"SRCDS_BOT_SEAT_NAMES": ", x ,,"}, func(s *Settings) { s.SrcdsBotSeatNames = []string{"", "x"} }},
		{"pairs drop what is not a pair", map[string]string{"SRCDS_BOT_LOADOUTS": "scout = milk, =x, spy"}, func(s *Settings) { s.SrcdsBotLoadouts = map[string]string{"scout": "milk"} }},
		{"empty pairs are an empty map", map[string]string{"SRCDS_BOT_CARD_FORMS": ""}, func(s *Settings) { s.SrcdsBotCardForms = map[string]string{} }},
		{"an empty table empties it", map[string]string{"SRCDS_BOT_CARD_ROLLS": " "}, func(s *Settings) { s.SrcdsBotCardRolls = nil }},
		{"a table that will not parse is kept", map[string]string{"SRCDS_BOT_TEAM_PRESETS": "{bad"}, func(*Settings) {}},
		{"a table", map[string]string{"SRCDS_BOT_CARD_ROLLS": `{"spy":"x"}`}, func(s *Settings) { s.SrcdsBotCardRolls = map[string]string{"spy": "x"} }},
		{"AP_ROOM sets the three parts", map[string]string{"AP_ROOM": "localhost:12345"}, func(s *Settings) { s.APHost, s.APPort, s.APTls = "localhost", 12345, false }},
		{"AP_HOST wins over AP_ROOM", map[string]string{"AP_ROOM": "localhost:12345", "AP_HOST": "other"}, func(s *Settings) { s.APHost, s.APPort, s.APTls = "other", 12345, false }},
		{"an AP_ROOM that is no address is ignored", map[string]string{"AP_ROOM": " "}, func(*Settings) {}},
		{"AP_ROOM_URL", map[string]string{"AP_ROOM_URL": "https://x"}, func(s *Settings) { s.APRoomURL = "https://x" }},
		{"SRCDS_START_MISSION wins over SRCDS_STARTMAP", map[string]string{"SRCDS_STARTMAP": "mvm_bigrock", "SRCDS_START_MISSION": "mine"}, func(s *Settings) { s.SrcdsStartMission = "mine" }},
		{"an unknown SRCDS_STARTMAP keeps the mission", map[string]string{"SRCDS_STARTMAP": "ctf_2fort"}, func(*Settings) {}},
		{"SRCDS_LAN=0 is the forwarded port", map[string]string{"SRCDS_LAN": "0"}, func(s *Settings) { s.SrcdsReach = ReachPort }},
		{"SRCDS_LAN that is not a boolean is ignored", map[string]string{"SRCDS_LAN": "x"}, func(*Settings) {}},
		{"SRCDS_REACH wins over SRCDS_LAN", map[string]string{"SRCDS_LAN": "1", "SRCDS_REACH": " PORT "}, func(s *Settings) { s.SrcdsReach = ReachPort }},
		{"an unknown reach is ignored", map[string]string{"SRCDS_REACH": "moon"}, func(*Settings) {}},
		{"mod loading in capitals", map[string]string{"SRCDS_MOD_LOADING": " REQUIRED "}, func(s *Settings) { s.SrcdsModLoading = ModLoadingRequired }},
		{"an unknown mod loading is ignored", map[string]string{"SRCDS_MOD_LOADING": "never"}, func(*Settings) {}},
		{"class slots from the old boolean", map[string]string{"MVM_CLASS_WEAPON_SLOTS": "yes"}, func(s *Settings) { s.MvmClassWeaponSlots = ClassSlotsProgressive }},
		{"empty class slots are off", map[string]string{"MVM_CLASS_WEAPON_SLOTS": ""}, func(s *Settings) { s.MvmClassWeaponSlots = ClassSlotsOff }},
		{"unknown class slots are off", map[string]string{"MVM_CLASS_WEAPON_SLOTS": "sometimes"}, func(s *Settings) { s.MvmClassWeaponSlots = ClassSlotsOff }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for name, value := range c.env {
				t.Setenv(name, value)
			}
			want := everyField()
			c.want(&want)
			if got := ApplyEnv(everyField()); !reflect.DeepEqual(got, want) {
				t.Errorf("\n got %+v\nwant %+v", got, want)
			}
		})
	}
}

/*
EnvValues writes what composeenv wrote before the table existed, for every
field and for none. -update rewrites the files, and the diff is the review.
*/
func TestEnvValuesMatchesTheGoldenFiles(t *testing.T) {
	for file, s := range map[string]Settings{"every_field.env": everyField(), "zero.env": {}} {
		t.Run(file, func(t *testing.T) {
			got := EnvValues(s)
			if *update {
				var out strings.Builder
				for _, name := range slices.Sorted(maps.Keys(got)) {
					out.WriteString(name + "='" + strings.ReplaceAll(got[name], "'", `\'`) + "'\n")
				}
				if err := os.WriteFile(filepath.Join("testdata", file), []byte(out.String()), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if want := readEnvFile(t, file); !maps.Equal(got, want) {
				t.Errorf("EnvValues differs from %s:\n got %v\nwant %v", file, got, want)
			}
		})
	}
}

/*
Every field the admin page writes reads back as it was.

This is what keeps the two directions from drifting: a variable read and not
written, or written in a spelling the reader does not take, fails here.
*/
func TestEnvValuesRoundTrip(t *testing.T) {
	want := withoutUncarried(everyField())
	for name, value := range EnvValues(want) {
		t.Setenv(name, value)
	}
	if got := ApplyEnv(Settings{}); !reflect.DeepEqual(got, want) {
		t.Errorf("\n got %+v\nwant %+v", got, want)
	}
}

// Every variable is read, and none is listed twice.
func TestEveryVariableIsDeclaredOnce(t *testing.T) {
	seen := map[string]bool{}
	for _, v := range EnvVars {
		if seen[v.Name] {
			t.Errorf("%s is declared twice", v.Name)
		}
		seen[v.Name] = true
		if v.read == nil {
			t.Errorf("%s is declared but nothing reads it", v.Name)
		}
	}
}

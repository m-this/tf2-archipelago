package availability

import (
	"slices"
	"testing"

	"github.com/m-this/tf2-archipelago/gamedata"
)

const (
	potato    = "archive-assets.zip"
	moonlight = "mlarchive-assets.zip"
	sigmod    = "sigsegv-mvm"

	decoy           = "mvm_decoy"
	homestead       = "mvm_kelly_rc1b_adv_homestead_happenings"
	binaryBroadcast = "mvm_transmission_rc7a_int_binary_broadcast"
	pointOfImpact   = "mvm_bronx_rc2_adv_point_of_impact"
	swampFever      = "mvm_bogland_rc12_adv_swamp_fever"
	troubleInTown   = "mvm_coaltown_int_trouble_in_mann_town"
)

func mission(t *testing.T, popFile string) gamedata.Mission {
	t.Helper()
	found, ok := gamedata.MissionByPopFile(popFile)
	if !ok {
		t.Fatalf("%s is not in the catalog", popFile)
	}
	return found
}

func TestCheck(t *testing.T) {
	everything := Server{
		Platform:      "linux",
		PacksSelected: []string{potato, moonlight},
		PacksPresent:  []string{potato, moonlight},
		ModsSelected:  []string{sigmod},
		ModsVerified:  []string{sigmod},
	}
	tests := []struct {
		name    string
		server  Server
		popFile string
		want    Reason
		why     string
	}{
		{
			name:    "a Valve mission needs nothing",
			popFile: decoy,
			want:    Playable,
		},
		{
			name:    "a Valve mission while SigMod is ticked and not verified",
			server:  Server{Platform: "linux", ModsSelected: []string{sigmod}},
			popFile: decoy,
			want:    Playable,
		},
		{
			name:    "a Valve mission on a server that loads SigMod",
			server:  everything,
			popFile: decoy,
			want:    Playable,
		},
		{
			name:    "a community mission whose pack is ticked and on disk",
			server:  Server{PacksSelected: []string{potato}, PacksPresent: []string{potato}},
			popFile: homestead,
			want:    Playable,
		},
		{
			name:    "a community mission whose pack is not on disk",
			server:  Server{PacksSelected: []string{potato}},
			popFile: binaryBroadcast,
			want:    PackMissing,
			why:     "Potato Archive is not on this server: download or import it on the Missions page",
		},
		{
			name:    "a community mission whose pack is on disk and not ticked",
			server:  Server{PacksPresent: []string{potato}},
			popFile: binaryBroadcast,
			want:    PackOff,
			why:     "tick Potato Archive",
		},
		{
			name:    "a Moonlight mission on a Valve map is still the pack's",
			server:  Server{PacksSelected: []string{potato}, PacksPresent: []string{potato}},
			popFile: troubleInTown,
			want:    PackMissing,
			why:     "Moonlight Archive is not on this server: download or import it on the Missions page",
		},
		{
			name:    "a map with embedded navigation is playable",
			server:  everything,
			popFile: swampFever,
			want:    Playable,
		},
		{
			name:    "a SigMod mission on a server that loads SigMod",
			server:  everything,
			popFile: pointOfImpact,
			want:    Playable,
		},
		{
			name:    "a SigMod mission with SigMod off",
			server:  Server{Platform: "linux", PacksSelected: []string{potato}, PacksPresent: []string{potato}},
			popFile: pointOfImpact,
			want:    ModOff,
			why:     "turn on SigMod, then press Download / set up selected server mods",
		},
		{
			name: "a SigMod mission with SigMod off, in Docker",
			server: Server{
				Platform: "linux", PacksSelected: []string{potato}, PacksPresent: []string{potato},
				ManagedExternally: true,
			},
			popFile: pointOfImpact,
			want:    ModOff,
			why:     "turn on SigMod, then save and recreate the containers",
		},
		{
			name: "a SigMod mission with SigMod ticked and not verified",
			server: Server{
				Platform: "linux", PacksSelected: []string{potato}, PacksPresent: []string{potato},
				ModsSelected: []string{sigmod},
			},
			popFile: pointOfImpact,
			want:    ModUnverified,
			why:     "SigMod is selected but its installation is missing or incomplete; press Download / set up selected server mods",
		},
		{
			name: "a SigMod mission with SigMod ticked and not verified, in Docker",
			server: Server{
				Platform: "linux", PacksSelected: []string{potato}, PacksPresent: []string{potato},
				ModsSelected: []string{sigmod}, ManagedExternally: true,
			},
			popFile: pointOfImpact,
			want:    ModUnverified,
			why:     "SigMod files are missing or incomplete in the server volume; recreate the container",
		},
		{
			name: "a SigMod mission verified but not ticked does not load",
			server: Server{
				Platform: "linux", PacksSelected: []string{potato}, PacksPresent: []string{potato},
				ModsVerified: []string{sigmod},
			},
			popFile: pointOfImpact,
			want:    ModOff,
			why:     "turn on SigMod, then press Download / set up selected server mods",
		},
		{
			name: "a SigMod mission on a platform with no SigMod build",
			server: Server{
				Platform: "plan9", PacksSelected: []string{potato}, PacksPresent: []string{potato},
				ModsSelected: []string{sigmod},
			},
			popFile: pointOfImpact,
			want:    ModNoBuild,
			why:     "SigMod has no plan9 server build, so this mission cannot run here",
		},
		{
			name:    "an unticked pack is said after the mod its mission needs",
			server:  Server{Platform: "linux", PacksPresent: []string{potato}},
			popFile: pointOfImpact,
			want:    ModOff,
			why:     "turn on SigMod, then press Download / set up selected server mods",
		},
		{
			name:    "a missing pack is said before the mod its mission needs",
			server:  Server{Platform: "linux"},
			popFile: pointOfImpact,
			want:    PackMissing,
			why:     "Potato Archive is not on this server: download or import it on the Missions page",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			verdict := test.server.Check(mission(t, test.popFile))
			if verdict.Reason != test.want {
				t.Errorf("reason = %v, want %v", verdict.Reason, test.want)
			}
			if verdict.Playable() != (test.want == Playable) {
				t.Errorf("Playable() = %v with reason %v", verdict.Playable(), verdict.Reason)
			}
			if got := verdict.Why(); got != test.why {
				t.Errorf("Why() = %q, want %q", got, test.why)
			}
		})
	}
}

func TestPlayableIsCheckOverTheCatalog(t *testing.T) {
	server := Server{Platform: "linux", PacksSelected: []string{potato}, PacksPresent: []string{potato, moonlight}}
	playable := server.Playable()
	var want []gamedata.Mission
	for _, one := range gamedata.Missions {
		if server.Check(one).Playable() {
			want = append(want, one)
		}
	}
	if !slices.Equal(playable, want) {
		t.Fatalf("Playable() holds %d missions, Check passes %d", len(playable), len(want))
	}
	for _, popFile := range []string{decoy, homestead, swampFever} {
		if !slices.ContainsFunc(playable, func(m gamedata.Mission) bool { return m.PopFile == popFile }) {
			t.Errorf("%s is missing from the playable missions", popFile)
		}
	}
	for _, popFile := range []string{troubleInTown, pointOfImpact} {
		if slices.ContainsFunc(playable, func(m gamedata.Mission) bool { return m.PopFile == popFile }) {
			t.Errorf("%s is playable on a server that cannot play it", popFile)
		}
	}
}

func TestShownLeavesOutOnlyMissingPacks(t *testing.T) {
	server := Server{Platform: "linux", PacksPresent: []string{potato}}
	shown := server.Shown()
	listed := func(popFile string) bool {
		return slices.ContainsFunc(shown, func(v Verdict) bool { return v.Mission.PopFile == popFile })
	}
	for _, popFile := range []string{decoy, homestead, swampFever, pointOfImpact} {
		if !listed(popFile) {
			t.Errorf("%s is not shown, and its pack is on disk", popFile)
		}
	}
	if listed(troubleInTown) {
		t.Errorf("%s is shown without its pack", troubleInTown)
	}
	if valveOnly := (Server{}).Shown(); slices.ContainsFunc(valveOnly, func(v Verdict) bool { return v.Mission.PopFile == swampFever }) {
		t.Errorf("%s is shown without its pack", swampFever)
	}
}

func TestSource(t *testing.T) {
	tests := []struct {
		popFile  string
		imported []string
		want     string
	}{
		{popFile: decoy, want: "Valve"},
		{popFile: decoy, imported: []string{potato}, want: "Valve"},
		{popFile: homestead, want: "Potato Archive"},
		{popFile: homestead, imported: []string{potato}, want: "Imported"},
		{popFile: troubleInTown, imported: []string{potato}, want: "Moonlight Archive"},
	}
	for _, test := range tests {
		t.Run(test.popFile+" "+test.want, func(t *testing.T) {
			if got := Source(mission(t, test.popFile), test.imported); got != test.want {
				t.Errorf("Source = %q, want %q", got, test.want)
			}
		})
	}
}

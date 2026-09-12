package runtime

import (
	"slices"
	"testing"
)

type fakeRCONExecutor struct {
	commands []string
	replies  map[string]string
}

func (f *fakeRCONExecutor) Exec(command string) (string, error) {
	f.commands = append(f.commands, command)
	return f.replies[command], nil
}

func TestMVMMapChangeRequestsNavigationRefresh(t *testing.T) {
	for _, tc := range []struct {
		line Line
		want bool
	}{
		{Line{Source: "srcds", Text: "L 09/12/2026: -------- Mapchange to mvm_bloodlust_b6 --------"}, true},
		{Line{Source: "srcds", Text: "-------- Mapchange to cp_badlands --------"}, false},
		{Line{Source: "bridge", Text: "Mapchange to mvm_decoy"}, false},
	} {
		if got := isMVMMapChange(tc.line); got != tc.want {
			t.Errorf("isMVMMapChange(%q, %q) = %t, want %t", tc.line.Source, tc.line.Text, got, tc.want)
		}
	}
}

func TestNavigationRefreshReloadsTheDependentPluginAroundCBaseNPC(t *testing.T) {
	client := &fakeRCONExecutor{replies: map[string]string{
		"sm_dump_nest": "0 nav areas on this map\n0 buildings standing",
		"sm exts list": "[08] Actions (3.9.2)\n[12] CBaseNPC (1.15.4.126)\n[13] DHooks",
	}}
	refreshed, err := refreshDefenderNavigation(client)
	if err != nil {
		t.Fatal(err)
	}
	if !refreshed {
		t.Fatal("empty navigation cache was not refreshed")
	}
	want := []string{
		"sm_dump_nest",
		"sm exts list",
		"sm plugins unload tf2_defenderbots",
		"sm exts reload 12",
		"sm plugins load tf2_defenderbots",
	}
	if !slices.Equal(client.commands, want) {
		t.Fatalf("commands = %q, want %q", client.commands, want)
	}
}

func TestNavigationRefreshLeavesAPopulatedCacheAlone(t *testing.T) {
	client := &fakeRCONExecutor{replies: map[string]string{
		"sm_dump_nest": "1480 nav areas on this map\n0 buildings standing",
	}}
	refreshed, err := refreshDefenderNavigation(client)
	if err != nil {
		t.Fatal(err)
	}
	if refreshed {
		t.Fatal("populated navigation cache was refreshed")
	}
	want := []string{"sm_dump_nest"}
	if !slices.Equal(client.commands, want) {
		t.Fatalf("commands = %q, want %q", client.commands, want)
	}
}

package webapi

import (
	"slices"
	"strings"
	"testing"
	"time"

	apruntime "github.com/m-this/tf2-archipelago/launcher/internal/runtime"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

func TestSigmodRefusalLeavesItsMissionsOutOfTheSeed(t *testing.T) {
	keys := `server.dll is TimeDateStamp 0x6ad30000 SizeOfImage 0xcb7000, the address table for ServerVersion 11191711 was made from 0x6ad26f14 0xcb6000, and the server is 11200000`
	version := `this SigMod build is for TF2 ServerVersion 11191711, and the server is 11200000`
	tests := []struct {
		name        string
		source      string
		text        string
		wantBlocked bool
		wantNotice  string
	}{
		{"with Keys, from srcds", "srcds", keys, true, "server.dll"},
		{"without Keys, from the SourceMod error log", "sourcemod", version, true, "11191711"},
		{"the launcher quoting the line is not the server saying it", "launcher", keys, false, ""},
		{"an unrelated server line", "srcds", "Map: mvm_decoy", false, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := settings.Defaults()
			s.SrcdsMods = []string{"sigsegv-mvm"}
			s.MvmStartMission = "mvm_bronx_rc2_adv_point_of_impact"
			app := New(s, nil)

			app.append(apruntime.Line{At: time.Now(), Source: test.source, Text: test.text})

			if got := slices.Contains(app.modsBlocked, "sigsegv-mvm"); got != test.wantBlocked {
				t.Fatalf("blocked = %v, want %v", got, test.wantBlocked)
			}
			seed := app.seedSettings(s)
			if test.wantBlocked {
				if !strings.Contains(app.Snapshot().Notice, test.wantNotice) {
					t.Errorf("notice = %q, want it to mention %q", app.Snapshot().Notice, test.wantNotice)
				}
				if len(seed.SrcdsMods) != 0 || seed.MvmStartMission != "" {
					t.Errorf("seed still asks for SigMod: mods %v, start %q", seed.SrcdsMods, seed.MvmStartMission)
				}
			} else if !slices.Equal(seed.SrcdsMods, s.SrcdsMods) {
				t.Errorf("seed lost its mods: %v", seed.SrcdsMods)
			}
		})
	}
}

func TestSigmodRefusalIsNoticedOnce(t *testing.T) {
	app := New(settings.Defaults(), nil)
	line := apruntime.Line{At: time.Now(), Source: "srcds", Text: "this SigMod build is for TF2 ServerVersion 1, and the server is 2"}
	app.append(line)
	first := app.Snapshot().NoticeSeq
	app.append(line)
	if got := app.Snapshot().NoticeSeq; got != first {
		t.Errorf("a repeated refusal raised a second notice (%d, then %d)", first, got)
	}
}

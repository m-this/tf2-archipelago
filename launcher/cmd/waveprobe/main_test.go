package main

import (
	"errors"
	"testing"

	"github.com/m-this/tf2-archipelago/gamedata"
)

func TestParseStatus(t *testing.T) {
	got, err := parseStatus("[SM] WAVEPROBE state=passed reason=none map=mvm_decoy pop=mvm_decoy_advanced3 max=6 gamewave=2 expected=1 observed=1 botspawns=45 tankspawns=1 bots=42 tanks=1 attempts=46 alive=0 remaining=0 initial=45 progress=100.0 defender=3 defteam=2 defclass=1 playerteam=2 enemyteam=3 elapsed=70.5")
	if err != nil {
		t.Fatal(err)
	}
	if got.State != "passed" || got.Pop != "mvm_decoy_advanced3" || got.Max != 6 ||
		got.GameWave != 2 || got.Expected != 1 || got.Observed != 1 || got.Bots != 42 || got.Tanks != 1 ||
		got.BotSpawns != 45 || got.TankSpawns != 1 || got.Attempts != 46 || got.Alive != 0 ||
		got.Initial != 45 || got.Progress != 100 || got.DefClass != 1 ||
		got.DefTeam != 2 || got.PlayerTeam != 2 || got.EnemyTeam != 3 {
		t.Fatalf("status = %+v", got)
	}
	if _, err := parseStatus("Unknown command sm_waveprobe_status"); err == nil {
		t.Fatal("missing test plugin looked healthy")
	}
}

func TestClassifyWave(t *testing.T) {
	tests := []struct {
		name   string
		status probeStatus
		err    error
		want   string
	}{
		{"pass", probeStatus{State: "passed", BotSpawns: 1}, nil, "passed"},
		{"single tank boss", probeStatus{State: "passed", TankSpawns: 1}, nil, "passed"},
		{"game loss", probeStatus{State: "failed", Reason: "wave_failed", BotSpawns: 1}, errors.New("lost"), "wave failed"},
		{"wave zero", probeStatus{State: "running", GameWave: 0, BotSpawns: 1}, errWaveZero, "wave 0"},
		{"empty wave", probeStatus{State: "passed"}, nil, "no enemies spawned"},
		{"empty timeout", probeStatus{State: "running", GameWave: 1}, errWallTimeout, "no enemies spawned"},
		{"active timeout", probeStatus{State: "running", GameWave: 1, BotSpawns: 1}, errWallTimeout, "wave timed out"},
		{"reset at deadline", probeStatus{State: "idle", GameWave: 1}, errWallTimeout, "probe error"},
		{"probe failure", probeStatus{State: "failed", Reason: "defender_missing"}, errors.New("missing"), "probe error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyWave(tt.status, tt.err); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoadFailureRequiresObservedOldMap(t *testing.T) {
	before := probeStatus{Map: "mvm_decoy", Pop: "mvm_decoy_advanced"}
	after := probeStatus{Map: "mvm_decoy", Pop: "mvm_decoy_advanced", State: "idle"}
	change := classifyMapChange("mvm_deathpour_rc1", before, after,
		"map change was rejected", nil, errors.New("timed out"))
	row := loadFailure("mvm_deathpour_rc1_int_technical_terror", "mvm_deathpour_rc1", "normal", change)
	if row.Outcome != "changelevel failure" || row.Changelevel == nil ||
		row.Changelevel.AfterMap != "mvm_decoy" || row.Changelevel.CommandReply != "map change was rejected" {
		t.Fatalf("changelevel evidence lost: %+v", row)
	}
	blocked := loadFailure("mission", "mvm_deathpour_rc1", "normal", errors.New("RCON unavailable"))
	if blocked.Outcome != "load blocked" || blocked.Changelevel != nil {
		t.Fatalf("unobserved map change called a changelevel failure: %+v", blocked)
	}
	after.Map = "mvm_deathpour_rc1"
	if err := classifyMapChange("mvm_deathpour_rc1", before, after, "", nil, errors.New("late status")); err == nil {
		t.Fatal("late status error disappeared")
	} else if row := loadFailure("mission", "mvm_deathpour_rc1", "normal", err); row.Outcome != "load blocked" {
		t.Fatalf("arrived map called a changelevel failure: %+v", row)
	}
}

func TestProbeClassName(t *testing.T) {
	medic, ok := gamedata.MissionByPopFile("mvm_chateau_rc3_adv_remedic")
	if !ok || probeClassName(medic) != "medic" {
		t.Fatalf("Remedic probe class: found=%v class=%q", ok, probeClassName(medic))
	}
	recalled, ok := gamedata.MissionByPopFile("mvm_villa_b13f_adv_recalled_to_life")
	if !ok || probeClassName(recalled) != "medic" {
		t.Fatalf("Recalled to Life probe class: found=%v class=%q", ok, probeClassName(recalled))
	}
	regular, ok := gamedata.MissionByPopFile("mvm_decoy_advanced3")
	if !ok || probeClassName(regular) != "scout" {
		t.Fatalf("ordinary probe class: found=%v class=%q", ok, probeClassName(regular))
	}
}

func TestTimescaleMatches(t *testing.T) {
	for _, reply := range []string{`host_timescale = "10" ( def. "1" )`, `host_timescale = "10.000000"`} {
		if !timescaleMatches(reply, 10) {
			t.Errorf("did not accept %q", reply)
		}
	}
	if timescaleMatches(`host_timescale = "1" ( def. "1" )`, 10) {
		t.Fatal("accepted unchanged clock")
	}
}

func TestShardsPartitionCatalog(t *testing.T) {
	for _, shards := range []int{1, 2, 6, 17} {
		seen := make(map[string]int)
		for shard := range shards {
			missions, err := selectMissions(options{mission: "all", shards: shards, shard: shard, includeSig: true})
			if err != nil {
				t.Fatal(err)
			}
			for _, mission := range missions {
				seen[mission.PopFile]++
			}
		}
		for _, mission := range gamedata.Missions {
			want := 1
			if gamedata.MissionRequirement(mission.ID) == "no_nav" {
				want = 0
			}
			if seen[mission.PopFile] != want {
				t.Errorf("%d shards: %s occurs %d times, want %d", shards, mission.PopFile, seen[mission.PopFile], want)
			}
		}
	}
}

func TestParseDefendersKeepsNamesWithSpaces(t *testing.T) {
	reply := "WAVEPROBE_DEF client=3 class=9 alive=1 seen=80.0 left=-1.0 inspawn=1 stillmax=61.5 stillspawn=1 stillnow=61.5 still=10,-20,30 idlemax=0.0 idle=0,0,0 at=11,-21,31 hatchmin=2400 hatchnow=2410 teleports=0 lives=2 leftmax=61.5 spawnnow=61.5 name=One-Man Cheeseburger\n" +
		"WAVEPROBE_DEF client=4 class=3 alive=1 seen=80.0 left=4.2 inspawn=0 stillmax=13.0 stillspawn=0 stillnow=0.0 still=1,2,3 idlemax=12.5 idle=7,8,9 at=4,5,6 hatchmin=300 hatchnow=900 teleports=1 lives=1 leftmax=4.2 spawnnow=0.0 name=THEM\n" +
		"WAVEPROBE_DEF_END hatch=1\n"
	rows, err := parseDefenders(reply)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	stuck := rows[0]
	if stuck.Name != "One-Man Cheeseburger" || stuck.Left != -1 || !stuck.InSpawn || !stuck.StillInSpawn ||
		stuck.StillAt != [3]float64{10, -20, 30} || stuck.Class != 9 {
		t.Errorf("stuck engineer parsed as %+v", stuck)
	}
	if rows[1].Teleports != 1 || rows[1].Left != 4.2 || rows[1].InSpawn ||
		rows[1].IdleMax != 12.5 || rows[1].IdleAt != [3]float64{7, 8, 9} {
		t.Errorf("second bot parsed as %+v", rows[1])
	}
}

func TestParseDefendersRefusesATruncatedRecord(t *testing.T) {
	if _, err := parseDefenders("WAVEPROBE_DEF client=3 class=9"); err == nil {
		t.Fatal("a record without its end line was accepted")
	}
}

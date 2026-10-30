package main

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/m-this/tf2-archipelago/gamedata"
)

func TestParseStatus(t *testing.T) {
	reply := "[SM] WAVEPROBE state=passed reason=none map=mvm_decoy pop=mvm_decoy_advanced3 max=6 gamewave=2 expected=1 observed=1 botspawns=45 tankspawns=1 bots=42 tanks=1 attempts=46 alive=0 remaining=0 initial=45 progress=100.0 defender=3 defteam=2 defclass=1 playerteam=2 enemyteam=3 elapsed=70.5"
	got, err := parseStatus(reply)
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
	if withObjective, err := parseStatus(reply + " objectives=3 pickups=2 captures=1 configsready=1"); err != nil || withObjective.ObjectiveVisits != 3 || withObjective.FlagPickups != 2 || withObjective.FlagCaptures != 1 || withObjective.ConfigsReady != 1 {
		t.Fatalf("objective evidence: %+v, %v", withObjective, err)
	}
	if _, err := parseStatus(reply + " captures=broken"); err == nil {
		t.Fatal("invalid capture evidence accepted")
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
		if !cvarMatches(reply, "host_timescale", 10) {
			t.Errorf("did not accept %q", reply)
		}
	}
	if cvarMatches(`host_timescale = "1" ( def. "1" )`, "host_timescale", 10) {
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

func TestOnlySigModSelectsTheSigModMissionsAndNothingElse(t *testing.T) {
	missions, err := selectMissions(options{mission: "all", shards: 1, includeSig: true, onlySig: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(missions) == 0 {
		t.Fatal("no SigMod missions selected")
	}
	for _, mission := range missions {
		if requirement := gamedata.MissionRequirement(mission.ID); requirement != "sigsegv-mvm" {
			t.Errorf("%s requires %q, not SigMod", mission.PopFile, requirement)
		}
	}
}

func TestLastWave(t *testing.T) {
	mission := gamedata.Mission{Waves: 7}
	tests := []struct {
		name string
		opt  options
		want int
	}{
		{"every wave", options{startWave: 1}, 7},
		{"first waves", options{startWave: 1, waves: 3}, 3},
		{"more than the mission has", options{startWave: 1, waves: 9}, 7},
		{"end wave", options{startWave: 2, endWave: 4}, 4},
		{"waves after the start", options{startWave: 5, waves: 2}, 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lastWave(mission, tt.opt); got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestHeadlessWakeReadback(t *testing.T) {
	if !cvarMatches(`"tf_allow_server_hibernation" = "0" ( def. "1" )`, "tf_allow_server_hibernation", 0) {
		t.Fatal("did not recognize an awake headless server")
	}
	for _, reply := range []string{
		`"tf_allow_server_hibernation" = "1" ( def. "0" )`,
		`Unknown command "tf_allow_server_hibernation"`,
		`"another_cvar" = "0"`,
	} {
		if cvarMatches(reply, "tf_allow_server_hibernation", 0) {
			t.Fatalf("accepted unverified wake-up: %q", reply)
		}
	}
}

func TestPartitionManifestOverridesHashAndRefusesIncompleteCoverage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "partition.json")
	assignments := make(map[string]int)
	for i, mission := range gamedata.Missions {
		assignments[mission.PopFile] = i % 3
	}
	write := func(shards int) {
		t.Helper()
		body, err := json.Marshal(map[string]any{"shards": shards, "missions": assignments})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(3)
	seen := make(map[string]bool)
	for shard := range 3 {
		selected, err := selectMissions(options{mission: "all", includeSig: true, shards: 3, shard: shard, partition: path})
		if err != nil {
			t.Fatal(err)
		}
		for _, mission := range selected {
			if seen[mission.PopFile] || assignments[mission.PopFile] != shard {
				t.Fatalf("invalid assignment for %s", mission.PopFile)
			}
			seen[mission.PopFile] = true
		}
	}
	for _, mission := range gamedata.Missions {
		if gamedata.MissionRequirement(mission.ID) != "no_nav" && !seen[mission.PopFile] {
			t.Fatalf("missing %s", mission.PopFile)
		}
	}
	write(4)
	if _, err := selectMissions(options{mission: "all", shards: 3, partition: path}); err == nil {
		t.Fatal("accepted wrong shard count")
	}
	clear(assignments)
	write(3)
	if _, err := selectMissions(options{mission: "all", shards: 3, partition: path}); err == nil {
		t.Fatal("accepted a missing mission")
	}
}

func TestReverseCompletionNeedsNativeFlagCaptureEvidence(t *testing.T) {
	status := probeStatus{State: "passed", PlayerTeam: 3, BotSpawns: 4}
	if got := classifyWave(status, nil); got != "probe error" {
		t.Fatalf("unwitnessed completion classified %s", got)
	}
	status.FlagCaptures = 1
	if got := classifyWave(status, nil); got != "passed" {
		t.Fatalf("native capture classified %s", got)
	}
}

func TestReverseSelectionIsExplicitAndKeepsUnsupportedCoverageByDefault(t *testing.T) {
	if err := run(options{reverseOnly: true}); err == nil || !strings.Contains(err.Error(), "requires -reverse-bombs") {
		t.Fatalf("reverse-only without opt-in: %v", err)
	}
	opt := options{mission: "all", shards: 1, includeSig: true, reverseOnly: true, startWave: 1}
	missions, err := selectMissions(opt)
	if err != nil || len(missions) == 0 {
		t.Fatalf("reverse catalog: %v, %d missions", err, len(missions))
	}
	for _, m := range missions {
		if !strings.Contains(m.PopFile, "_rev_") || gamedata.MissionRequirement(m.ID) == "no_nav" {
			t.Fatalf("unexpected reverse mission: %s", m.PopFile)
		}
	}
	for _, enabled := range []bool{false, true} {
		opt.reverse = enabled
		file, err := os.CreateTemp(t.TempDir(), "plan")
		if err != nil {
			t.Fatal(err)
		}
		previous := os.Stdout
		os.Stdout = file
		err = writePlan(missions, []string{"normal", "surge"}, opt)
		os.Stdout = previous
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Seek(0, 0); err != nil {
			t.Fatal(err)
		}
		decoder := json.NewDecoder(file)
		cases := 0
		for {
			var row result
			err := decoder.Decode(&row)
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			want := "unsupported_reverse"
			if enabled {
				want = "planned"
			}
			if row.State != want {
				t.Errorf("%s state=%s want=%s", row.Mission, row.State, want)
			}
			cases++
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		if cases != 122 {
			t.Errorf("reverse cases=%d want=122", cases)
		}
	}
}

func TestOnlyCompletedReverseMissionsNeedTransitionSynchronization(t *testing.T) {
	final := probeStatus{State: "passed", Pop: "mvm_rottenburg_rev_int_breaking_walls", Expected: 6, Max: 6}
	if !reverseMissionTransitionPending(final) {
		t.Fatal("final reverse wave did not wait for its natural transition")
	}
	ordinary := final
	ordinary.Pop = "mvm_decoy_expert1"
	earlier := final
	earlier.Expected = 5
	failed := final
	failed.State = "failed"
	empty := final
	empty.Max = 0
	empty.Expected = 0
	for _, status := range []probeStatus{ordinary, earlier, failed, empty} {
		if reverseMissionTransitionPending(status) {
			t.Fatalf("unnecessary transition wait: %+v", status)
		}
	}
}

func TestDailyReverseBombProfileIncludesOnlyValidatedMissions(t *testing.T) {
	opt := options{mission: "all", shards: 1, includeSig: true, startWave: 1, reverseBombs: true}
	missions, err := selectMissions(opt)
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.CreateTemp(t.TempDir(), "daily-plan")
	if err != nil {
		t.Fatal(err)
	}
	previous := os.Stdout
	os.Stdout = file
	err = writePlan(missions, []string{"normal", "surge"}, opt)
	os.Stdout = previous
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := file.Close(); err != nil {
			t.Error(err)
		}
	})
	decoder := json.NewDecoder(file)
	counts := map[string]int{}
	reverseCases := 0
	for {
		var row result
		err := decoder.Decode(&row)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		counts[row.State]++
		if row.State == "planned" && strings.Contains(row.Mission, "_rev_") {
			reverseCases++
		}
		if row.Mission == "mvm_teien_rc6_rev_adv_shogun_slaughter" && row.State != "unsupported_reverse" {
			t.Fatal("payload mission was enabled by the bomb profile")
		}
	}
	if reverseCases != 34 || counts["planned"] != 2540 || counts["unsupported_reverse"] != 88 {
		t.Fatalf("reverse=%d, plan=%v", reverseCases, counts)
	}
}

func TestAuthoredReverseCombatWaveRequiresItsTargetDeathWitness(t *testing.T) {
	if reverseObjectiveKind("mvm_coaltown_rev_adv_computer_crash", 6) != "combat" || reverseObjectiveKind("mvm_coaltown_rev_adv_computer_crash", 5) != "bomb" {
		t.Fatal("authored final-wave objective differs from bomb waves")
	}
	status := probeStatus{State: "passed", PlayerTeam: 3, ObjectiveKind: "combat", BotSpawns: 12, Bots: 7}
	if got := classifyWave(status, nil); got != "probe error" {
		t.Fatalf("unwitnessed boss clear: %s", got)
	}
	status.TargetKills = 1
	if got := classifyWave(status, nil); got != "passed" {
		t.Fatalf("witnessed boss clear: %s", got)
	}
}

func TestFullReverseProfileHasNoExcludedPlayableMissions(t *testing.T) {
	opt := options{mission: "all", includeSig: true, reverseObjectives: true, shards: 1, startWave: 1}
	missions, err := selectMissions(opt)
	if err != nil {
		t.Fatal(err)
	}
	total, reverse := 0, 0
	for _, mission := range missions {
		cases := int(mission.Waves) * 2
		total += cases
		if strings.Contains(mission.PopFile, "_rev_") {
			reverse += cases
			if !reverseEnabled(mission.PopFile, opt) {
				t.Fatalf("excluded %s", mission.PopFile)
			}
		}
	}
	if total != 2628 || reverse != 122 {
		t.Fatalf("coverage changed: %d total, %d reverse; inspect the playable catalog", total, reverse)
	}
}

func TestSpecialReverseWitnessesCannotPassWithoutNativeEvidence(t *testing.T) {
	for _, kind := range []string{"repair", "combat-victory", "route"} {
		st := probeStatus{State: "passed", PlayerTeam: 3, ObjectiveKind: kind, BotSpawns: 10, Bots: 10}
		if got := classifyWave(st, nil); got == "passed" {
			t.Fatalf("%s accepted without objective evidence", kind)
		}
	}
	st := probeStatus{State: "passed", PlayerTeam: 3, ObjectiveKind: "repair", BotSpawns: 10, Bots: 10, Repairs: 5, BluWon: 1}
	if got := classifyWave(st, nil); got != "passed" {
		t.Fatalf("native repair completion refused: %s", got)
	}
	st.Repairs = 4
	if got := classifyWave(st, nil); got == "passed" {
		t.Fatal("unfinished repair accepted")
	}
}

func TestNativeReverseObjectivesCannotHideMissingEnemySpawns(t *testing.T) {
	st := probeStatus{State: "passed", PlayerTeam: 3, ObjectiveKind: "repair", Repairs: 5, BluWon: 1}
	if got := classifyWave(st, nil); got != "no enemies spawned" {
		t.Fatalf("empty population passed: %s", got)
	}
}

func TestAuthoredRepairCompletionSurvivesBuildingCleanup(t *testing.T) {
	st := probeStatus{State: "passed", PlayerTeam: 3, ObjectiveKind: "repair", Repairs: 4, RepairComplete: 1, BluWon: 1, BotSpawns: 2, Bots: 2}
	if got := classifyWave(st, nil); got != "passed" {
		t.Fatalf("native repair completion lost after cleanup: %s", got)
	}
	st.RepairComplete = 0
	if got := classifyWave(st, nil); got == "passed" {
		t.Fatal("four repairs without completion accepted")
	}
}

func TestReadNativeRepairCompletionAfterWorldCleanup(t *testing.T) {
	reply := "WAVEPROBE state=passed reason=none map=mvm_robotfactory_b30 pop=mvm_robotfactory_b30_rev_adv_short_circuit_execution max=3 gamewave=2 expected=1 observed=1 botspawns=157 tankspawns=0 bots=112 tanks=0 attempts=114 alive=0 remaining=319 initial=1125 progress=71.6 defender=10 defteam=3 defclass=9 playerteam=3 enemyteam=2 elapsed=716.4 objectives=5 pickups=0 captures=0 configsready=1 objectivekind=repair targetkills=0 pointcaps=0 repairs=4 tankdeliveries=0 bluwon=1 escortstages=0 repaircomplete=1"
	status, err := parseStatus(reply)
	if err != nil {
		t.Fatal(err)
	}
	if status.RepairComplete != 1 || status.Repairs != 4 || status.BluWon != 1 {
		t.Fatalf("native completion evidence lost: %+v", status)
	}
	if got := classifyWave(status, nil); got != "passed" {
		t.Fatalf("native completion after cleanup classified %s", got)
	}
}

func TestArcticArrangementUsesAuthoredPointAndTankObjectives(t *testing.T) {
	for wave := 1; wave <= 5; wave++ {
		want := "route"
		if wave == 4 {
			want = "combat-victory"
		}
		if got := reverseObjectiveKind("mvm_cyberia_rc6a_rev_arctic_arrangement", wave); got != want {
			t.Errorf("wave %d kind=%q want=%q", wave, got, want)
		}
	}
}

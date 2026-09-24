// Command waveprobe drives the disposable SRCDS wave smoke test over RCON.
// It never advances a wave itself: the test plugin waits for the game's
// mvm_wave_complete event while defeating each spawn after a seeded delay.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"hash/crc32"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/m-this/tf2-archipelago/gamedata"
	"github.com/m-this/tf2-archipelago/launcher/internal/rcon"
)

type options struct {
	address    string
	mission    string
	mode       string
	seed       int
	speed      int
	timeout    time.Duration
	loadWait   time.Duration
	shard      int
	shards     int
	startWave  int
	endWave    int
	failFast   bool
	includeSig bool
	onlySig    bool
	plan       bool
}

type probeStatus struct {
	State      string
	Reason     string
	Map        string
	Pop        string
	Max        int
	GameWave   int
	Expected   int
	Observed   int
	Bots       int
	Tanks      int
	BotSpawns  int
	TankSpawns int
	Attempts   int
	Alive      int
	Remaining  int
	Initial    int
	DefClass   int
	DefTeam    int
	PlayerTeam int
	EnemyTeam  int
	Elapsed    float64
	Progress   float64
}

type sample struct {
	WallSeconds float64 `json:"wall_seconds"`
	GameSeconds float64 `json:"game_seconds"`
	Spawned     int     `json:"spawned"`
	Killed      int     `json:"killed"`
	Alive       int     `json:"alive"`
	Remaining   int     `json:"remaining"`
	Progress    float64 `json:"progress_percent"`
}

type result struct {
	Mission     string               `json:"mission"`
	Map         string               `json:"map"`
	Mode        string               `json:"mode"`
	Wave        int                  `json:"wave"`
	Seed        int                  `json:"seed"`
	State       string               `json:"state"`
	Outcome     string               `json:"outcome,omitempty"`
	Bots        int                  `json:"bots"`
	Tanks       int                  `json:"tanks"`
	BotSpawns   int                  `json:"bot_spawns"`
	TankSpawns  int                  `json:"tank_spawns"`
	Attempts    int                  `json:"kill_attempts"`
	Alive       int                  `json:"alive_at_end"`
	Remaining   int                  `json:"remaining_at_end"`
	Progress    float64              `json:"progress_percent"`
	Seconds     float64              `json:"wall_seconds"`
	GameSeconds float64              `json:"game_seconds,omitempty"`
	Error       string               `json:"error,omitempty"`
	Debug       string               `json:"debug_snapshot,omitempty"`
	Timeline    []sample             `json:"timeline,omitempty"`
	Changelevel *changelevelEvidence `json:"changelevel,omitempty"`
}

type changelevelEvidence struct {
	RequestedMap string `json:"requested_map"`
	BeforeMap    string `json:"before_map"`
	BeforePop    string `json:"before_pop"`
	AfterMap     string `json:"after_map"`
	AfterPop     string `json:"after_pop"`
	AfterState   string `json:"after_state"`
	CommandReply string `json:"command_reply,omitempty"`
	CommandError string `json:"command_error,omitempty"`
}

type changelevelError struct {
	evidence changelevelEvidence
	cause    error
}

func (e *changelevelError) Error() string {
	return fmt.Sprintf("changelevel did not reach %s: before %s/%s, after %s/%s (%s); reply %q; command error %q; %v",
		e.evidence.RequestedMap, e.evidence.BeforeMap, e.evidence.BeforePop,
		e.evidence.AfterMap, e.evidence.AfterPop, e.evidence.AfterState,
		e.evidence.CommandReply, e.evidence.CommandError, e.cause)
}

func classifyMapChange(target string, before, after probeStatus, reply string, commandErr, cause error) error {
	if after.Map == target {
		return cause
	}
	evidence := changelevelEvidence{
		RequestedMap: target, BeforeMap: before.Map, BeforePop: before.Pop,
		AfterMap: after.Map, AfterPop: after.Pop, AfterState: after.State,
		CommandReply: strings.TrimSpace(reply),
	}
	if commandErr != nil {
		evidence.CommandError = commandErr.Error()
	}
	return &changelevelError{evidence: evidence, cause: cause}
}

func loadFailure(mission, mapName, mode string, err error) result {
	row := result{
		Mission: mission, Map: mapName, Mode: mode,
		State: "load_failed", Outcome: "load blocked", Error: err.Error(),
	}
	if errors.Is(err, errWaveZero) {
		row.Outcome = "wave 0"
	}
	if changeErr, ok := errors.AsType[*changelevelError](err); ok {
		row.Outcome = "changelevel failure"
		row.Changelevel = &changeErr.evidence
	}
	return row
}

func main() {
	var opt options
	flag.StringVar(&opt.address, "rcon", "127.0.0.1:27035", "isolated server RCON address")
	flag.StringVar(&opt.mission, "mission", "all", "population file name or all")
	flag.StringVar(&opt.mode, "mode", "both", "normal, surge, or both")
	flag.IntVar(&opt.seed, "seed", 1, "seed for each spawn's 15-25 game-second lifetime")
	flag.IntVar(&opt.speed, "speed", 20, "isolated server host_timescale")
	flag.DurationVar(&opt.timeout, "timeout", 15*time.Minute, "wall-clock timeout after the wave starts")
	flag.DurationVar(&opt.loadWait, "load-timeout", 90*time.Second, "wall-clock timeout for a map or mission load")
	flag.IntVar(&opt.shard, "shard", 0, "zero-based mission shard")
	flag.IntVar(&opt.shards, "shards", 1, "number of mission shards")
	flag.IntVar(&opt.startWave, "start-wave", 1, "first wave to test (single mission only)")
	flag.IntVar(&opt.endWave, "end-wave", 0, "last wave to test (0 means mission end)")
	flag.BoolVar(&opt.failFast, "fail-fast", false, "stop after the first failed wave")
	flag.BoolVar(&opt.includeSig, "include-sigmod", true, "test SigMod missions")
	flag.BoolVar(&opt.onlySig, "only-sigmod", false, "test SigMod missions and nothing else")
	flag.BoolVar(&opt.plan, "plan", false, "print planned wave tests without connecting to a server")
	flag.Parse()
	if err := run(opt); err != nil {
		fmt.Fprintln(os.Stderr, "waveprobe:", err)
		os.Exit(1)
	}
}

func run(opt options) error {
	if opt.shards < 1 || opt.shard < 0 || opt.shard >= opt.shards {
		return errors.New("shard must be in [0, shards)")
	}
	if opt.speed < 1 || opt.speed > 20 {
		return errors.New("speed must be between 1 and 20")
	}
	if opt.timeout <= 0 || opt.loadWait <= 0 {
		return errors.New("time limits must be positive")
	}
	if opt.mission == "all" && (opt.startWave != 1 || opt.endWave != 0) {
		return errors.New("wave range requires one named mission")
	}
	modes := []string{opt.mode}
	if opt.mode == "both" {
		modes = []string{"normal", "surge"}
	} else if opt.mode != "normal" && opt.mode != "surge" {
		return errors.New("mode must be normal, surge, or both")
	}
	missions, err := selectMissions(opt)
	if err != nil {
		return err
	}
	if opt.plan {
		return writePlan(missions, modes)
	}
	if opt.mission != "all" && strings.Contains(opt.mission, "_rev_") {
		return errors.New("reverse MvM needs a BLU objective simulator; kill-only waveprobe cannot validate it")
	}
	password := os.Getenv("WAVEPROBE_RCONPW")
	if password == "" {
		return errors.New("WAVEPROBE_RCONPW is empty")
	}
	server := &server{address: opt.address, password: password}
	defer server.close()
	if err := server.prepare(opt); err != nil {
		return err
	}
	return runMissions(server, opt, missions, modes)
}

func writePlan(missions []gamedata.Mission, modes []string) error {
	for _, mission := range missions {
		played, ok := gamedata.MapByID(mission.Map)
		if !ok {
			return fmt.Errorf("unknown map for %s", mission.PopFile)
		}
		for _, mode := range modes {
			for wave := 1; wave <= int(mission.Waves); wave++ {
				state := "planned"
				if strings.Contains(mission.PopFile, "_rev_") {
					state = "unsupported_reverse"
				}
				writeResult(result{
					Mission: mission.PopFile, Map: played.Name,
					Mode: mode, Wave: wave, State: state,
				})
			}
		}
	}
	return nil
}

func (s *server) prepare(opt options) error {
	if err := s.await(3*time.Minute, func(probeStatus) bool { return true }); err != nil {
		return fmt.Errorf("test plugin is unavailable at %s: %w", opt.address, err)
	}
	if _, err := s.exec("sv_cheats 1"); err != nil {
		return err
	}
	// A bot reaching the hatch would test an undefended loss, not population
	// progression. Valve provides this cvar specifically for bot testing.
	if _, err := s.exec("tf_bot_flag_kill_on_touch 1"); err != nil {
		return err
	}
	if _, err := s.exec(fmt.Sprintf("host_timescale %d", opt.speed)); err != nil {
		return err
	}
	if reply, err := s.exec("host_timescale"); err != nil {
		return fmt.Errorf("read host_timescale: %w", err)
	} else if !timescaleMatches(reply, opt.speed) {
		return fmt.Errorf("host_timescale %d did not stick: %q", opt.speed, reply)
	}
	return nil
}

func runMissions(s *server, opt options, missions []gamedata.Mission, modes []string) error {
	failures := 0
	for _, mission := range missions {
		if strings.Contains(mission.PopFile, "_rev_") {
			continue
		}
		played, ok := gamedata.MapByID(mission.Map)
		if !ok {
			return fmt.Errorf("unknown map for %s", mission.PopFile)
		}
		for _, mode := range modes {
			if err := s.load(played.Name, mission, mode, opt.loadWait, false); err != nil {
				writeResult(loadFailure(mission.PopFile, played.Name, mode, err))
				failures++
				if opt.failFast {
					return fmt.Errorf("%d wave tests failed", failures)
				}
				continue
			}
			count, err := s.runWaves(opt, played.Name, mission, mode)
			failures += count
			if err != nil {
				return err
			}
			if failures > 0 && opt.failFast {
				return fmt.Errorf("%d wave tests failed", failures)
			}
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d wave tests failed", failures)
	}
	return nil
}

func (s *server) runWaves(opt options, mapName string, mission gamedata.Mission, mode string) (int, error) {
	first := opt.startWave
	sequential := mission.PopFile == "mvm_villa_b13f_adv_recalled_to_life"
	if sequential && first > 1 {
		// Villa's hunt wave needs the map's earlier room setup. Jumping
		// directly to wave 5 starts its timer bot but never spawns the room
		// groups, which produces a false wave timeout.
		first = 1
	}
	last := int(mission.Waves)
	if opt.endWave > 0 {
		last = opt.endWave
	}
	if first > 1 {
		if _, err := s.exec(fmt.Sprintf("tf_mvm_jump_to_wave %d 1", first)); err != nil {
			return 0, err
		}
	}
	failures := 0
	for wave := first; wave <= last; wave++ {
		if err := s.recordWave(opt, mapName, mission, mode, wave); err == nil {
			continue
		}
		failures++
		if opt.failFast || wave == last || sequential {
			break
		}
		// The failed wave may still be running. Reload the mission and jump
		// ahead so later waves are tested independently too.
		if err := s.load(mapName, mission, mode, opt.loadWait, true); err != nil {
			writeResult(loadFailure(mission.PopFile, mapName, mode, err))
			failures++
			break
		}
		if _, err := s.exec(fmt.Sprintf("tf_mvm_jump_to_wave %d 1", wave+1)); err != nil {
			writeResult(result{
				Mission: mission.PopFile, Map: mapName, Mode: mode,
				State: "load_failed", Outcome: "load blocked", Error: err.Error(),
			})
			failures++
			break
		}
	}
	return failures, nil
}

func (s *server) recordWave(opt options, mapName string, mission gamedata.Mission, mode string, wave int) error {
	started := time.Now()
	status, timeline, err := s.testWave(mission, wave, opt.seed, opt.timeout, opt.loadWait)
	row := result{
		Mission: mission.PopFile, Map: mapName, Mode: mode,
		Wave: wave, Seed: opt.seed, State: status.State, Bots: status.Bots,
		Tanks: status.Tanks, BotSpawns: status.BotSpawns, TankSpawns: status.TankSpawns,
		Attempts: status.Attempts, Alive: status.Alive, Remaining: status.Remaining,
		Progress: status.Progress, Seconds: time.Since(started).Seconds(), GameSeconds: status.Elapsed,
	}
	row.Outcome = classifyWave(status, err)
	if row.Outcome != "passed" {
		row.State = "failed"
		if err != nil {
			row.Error = err.Error()
		} else {
			row.Error = "the wave completed without an observed enemy spawn"
		}
		row.Timeline = timeline
		if snapshot, debugErr := s.exec("sm_waveprobe_debug"); debugErr == nil {
			row.Debug = strings.TrimSpace(snapshot)
		} else {
			row.Debug = "debug snapshot unavailable: " + debugErr.Error()
		}
	}
	writeResult(row)
	if row.Outcome == "passed" {
		return nil
	}
	if errors.Is(err, errWallTimeout) {
		s.stopWave(mission)
	}
	return fmt.Errorf("%s: %s", row.Outcome, row.Error)
}

func classifyWave(status probeStatus, err error) string {
	switch {
	case errors.Is(err, errWaveZero) || (status.State == "running" && status.GameWave == 0):
		return "wave 0"
	case status.Reason == "wave_failed":
		return "wave failed"
	case status.State == "passed" && err == nil && status.BotSpawns+status.TankSpawns > 0:
		return "passed"
	case status.State == "idle" || status.State == "armed":
		return "probe error"
	case (status.State == "passed" || (status.State == "running" && errors.Is(err, errWallTimeout))) &&
		status.BotSpawns+status.TankSpawns == 0:
		return "no enemies spawned"
	case errors.Is(err, errWallTimeout):
		return "wave timed out"
	default:
		return "probe error"
	}
}

func selectMissions(opt options) ([]gamedata.Mission, error) {
	if opt.mission != "all" {
		mission, ok := gamedata.MissionByPopFile(opt.mission)
		if !ok {
			return nil, fmt.Errorf("unknown mission %q", opt.mission)
		}
		if opt.startWave < 1 || opt.startWave > int(mission.Waves) ||
			(opt.endWave > 0 && (opt.endWave < opt.startWave || opt.endWave > int(mission.Waves))) {
			return nil, errors.New("wave range is outside the mission")
		}
		return []gamedata.Mission{mission}, nil
	}
	var selected []gamedata.Mission
	for _, mission := range gamedata.Missions {
		requirement := gamedata.MissionRequirement(mission.ID)
		if requirement == "no_nav" || (requirement == "sigsegv-mvm" && !opt.includeSig) ||
			(requirement != "sigsegv-mvm" && opt.onlySig) {
			continue
		}
		if crc32.ChecksumIEEE([]byte(mission.PopFile))%uint32(opt.shards) != uint32(opt.shard) {
			continue
		}
		selected = append(selected, mission)
	}
	sort.Slice(selected, func(i, j int) bool {
		left, _ := gamedata.MapByID(selected[i].Map)
		right, _ := gamedata.MapByID(selected[j].Map)
		if left.Name == right.Name {
			return selected[i].PopFile < selected[j].PopFile
		}
		return left.Name < right.Name
	})
	return selected, nil
}

func writeResult(row result) {
	body, err := json.Marshal(row)
	if err != nil {
		fmt.Fprintf(os.Stderr, "waveprobe: encode result: %v\n", err)
		return
	}
	fmt.Println(string(body))
}

type server struct {
	address  string
	password string
	client   *rcon.Client
}

func (s *server) close() {
	if s.client != nil {
		_ = s.client.Close()
		s.client = nil
	}
}

func (s *server) exec(command string) (string, error) {
	if s.client == nil {
		client, err := rcon.Dial(s.address, s.password)
		if err != nil {
			return "", err
		}
		s.client = client
	}
	reply, err := s.client.Exec(command)
	if err != nil {
		// changelevel can close the connection; the next command reconnects.
		s.close()
	}
	return reply, err
}

func (s *server) status() (probeStatus, error) {
	reply, err := s.exec("sm_waveprobe_status")
	if err != nil {
		return probeStatus{}, err
	}
	return parseStatus(reply)
}

func parseStatus(reply string) (probeStatus, error) {
	position := strings.Index(reply, "WAVEPROBE state=")
	if position < 0 {
		return probeStatus{}, fmt.Errorf("unexpected probe reply: %q", reply)
	}
	fields := map[string]string{}
	for token := range strings.FieldsSeq(reply[position+len("WAVEPROBE "):]) {
		key, value, ok := strings.Cut(token, "=")
		if ok {
			fields[key] = value
		}
	}
	var status probeStatus
	status.State = fields["state"]
	status.Reason = fields["reason"]
	status.Map = fields["map"]
	status.Pop = fields["pop"]
	var err error
	for _, field := range []struct {
		name string
		dest *int
	}{
		{"max", &status.Max},
		{"gamewave", &status.GameWave},
		{"expected", &status.Expected},
		{"observed", &status.Observed},
		{"bots", &status.Bots},
		{"tanks", &status.Tanks},
		{"botspawns", &status.BotSpawns},
		{"tankspawns", &status.TankSpawns},
		{"attempts", &status.Attempts},
		{"alive", &status.Alive},
		{"remaining", &status.Remaining},
		{"initial", &status.Initial},
		{"defteam", &status.DefTeam},
		{"defclass", &status.DefClass},
		{"playerteam", &status.PlayerTeam},
		{"enemyteam", &status.EnemyTeam},
	} {
		*field.dest, err = strconv.Atoi(fields[field.name])
		if err != nil {
			return probeStatus{}, fmt.Errorf("invalid %s in probe reply %q: %w", field.name, reply, err)
		}
	}
	status.Elapsed, err = strconv.ParseFloat(fields["elapsed"], 64)
	if err != nil {
		return probeStatus{}, fmt.Errorf("invalid elapsed in probe reply %q: %w", reply, err)
	}
	status.Progress, err = strconv.ParseFloat(fields["progress"], 64)
	if err != nil {
		return probeStatus{}, fmt.Errorf("invalid progress in probe reply %q: %w", reply, err)
	}
	if status.State == "" || status.Map == "" || status.Pop == "" {
		return probeStatus{}, fmt.Errorf("incomplete probe reply: %q", reply)
	}
	return status, nil
}

func timescaleMatches(reply string, speed int) bool {
	equals := strings.IndexByte(reply, '=')
	if equals < 0 {
		return false
	}
	start := strings.IndexByte(reply[equals+1:], '"')
	if start < 0 {
		return false
	}
	start += equals + 1
	end := strings.IndexByte(reply[start+1:], '"')
	if end < 0 {
		return false
	}
	value, err := strconv.ParseFloat(reply[start+1:start+1+end], 64)
	return err == nil && value == float64(speed)
}

func (s *server) await(timeout time.Duration, predicate func(probeStatus) bool) error {
	deadline := time.Now().Add(timeout)
	var last probeStatus
	var lastErr error
	for time.Now().Before(deadline) {
		last, lastErr = s.status()
		if lastErr == nil && predicate(last) {
			return nil
		}
		time.Sleep(time.Second)
	}
	if lastErr != nil {
		return fmt.Errorf("timed out after %s (last status %+v): %w", timeout, last, lastErr)
	}
	return fmt.Errorf("timed out after %s (last status %+v)", timeout, last)
}

func (s *server) load(mapName string, mission gamedata.Mission, mode string, timeout time.Duration, forceMapReload bool) error {
	status, err := s.status()
	if err != nil {
		return err
	}
	if forceMapReload || status.Map != mapName {
		// The server may drop the RCON connection as changelevel runs. The
		// status poll, rather than that connection, determines success.
		changeReply, changeErr := s.exec("changelevel " + mapName)
		if forceMapReload {
			// A same-map status check can otherwise succeed before changelevel
			// has actually restarted the population manager.
			time.Sleep(2 * time.Second)
		}
		if err := s.await(timeout, func(st probeStatus) bool { return st.Map == mapName }); err != nil {
			// A responsive server still on the old map is direct evidence that
			// changelevel did not complete. Preserve the command reply and both
			// observed maps; a lost RCON connection alone is not that evidence.
			after, statusErr := s.status()
			if statusErr == nil && after.Map != mapName {
				return classifyMapChange(mapName, status, after, changeReply, changeErr, err)
			}
			if statusErr != nil {
				return fmt.Errorf("changelevel %s from %s/%s: reply %q, command error %s, final status error %w: %w",
					mapName, status.Map, status.Pop, strings.TrimSpace(changeReply), fmt.Sprint(changeErr), statusErr, err)
			}
			return s.classifyLoadError(mapName, mission, err)
		}
		// Some missions initialize only after a player joins. configureMode
		// creates that player before we require a nonzero wave count.
	}
	teamName, playerTeam, playerClass, err := s.configureMode(mission, mode)
	if err != nil {
		return err
	}
	// The modifier query in configureMode confirms the chosen mode before
	// explicitly loading the full population file.
	var popReply string
	popDeadline := time.Now().Add(min(timeout, 15*time.Second))
	for {
		popReply, err = s.exec("tf_mvm_popfile " + mission.PopFile)
		if err == nil && !strings.Contains(popReply, "Could not find a valid population file") {
			break
		}
		if time.Now().After(popDeadline) {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		return err
	}
	if strings.Contains(popReply, "Could not find a valid population file") {
		return fmt.Errorf("population file %s was rejected: %s", mission.PopFile, strings.TrimSpace(popReply))
	}
	loaded := func(st probeStatus) bool {
		return st.Map == mapName && st.Pop == mission.PopFile &&
			st.Max == int(mission.Waves) && st.GameWave == 1 &&
			st.DefTeam == playerTeam && st.DefClass == playerClass && st.PlayerTeam == playerTeam &&
			st.EnemyTeam == 5-playerTeam
	}
	if err := s.await(5*time.Second, loaded); err != nil {
		// A popfile reload can drop the fake player after the first wake.
		// Recreate it once the population manager is initialized.
		if _, wakeErr := s.exec("sm_waveprobe_wake " + teamName + " " + probeClassName(mission)); wakeErr != nil {
			return wakeErr
		}
		if retryErr := s.await(timeout, loaded); retryErr != nil {
			return s.classifyLoadError(mapName, mission,
				fmt.Errorf("mission %s did not load: %w", mission.PopFile, retryErr))
		}
	}
	return nil
}

func (s *server) classifyLoadError(mapName string, mission gamedata.Mission, cause error) error {
	status, err := s.status()
	if err == nil && status.Map == mapName && status.GameWave == 0 &&
		(status.Pop == mission.PopFile || status.Pop == "unknown") {
		return fmt.Errorf("%w: %s (max=%d): %w", errWaveZero, mission.PopFile, status.Max, cause)
	}
	return cause
}

func probeClassName(mission gamedata.Mission) string {
	if gamedata.MissionLoadout(mission.ID) == "medic_only" {
		return "medic"
	}
	return "scout"
}

func (s *server) configureMode(mission gamedata.Mission, mode string) (string, int, int, error) {
	if _, err := s.exec("sm_waveprobe_reset"); err != nil {
		return "", 0, 0, err
	}
	// No human is connected to these disposable servers. server.cfg resets
	// this value on map changes.
	if _, err := s.exec("tf_mvm_min_players_to_start 0"); err != nil {
		return "", 0, 0, err
	}
	// Bot Surge prints a line per rewritten spawner at debug level 1. That
	// overflows one Source RCON packet on some missions and drowns the probe.
	if _, err := s.exec("tf2ap_debug 0"); err != nil {
		return "", 0, 0, err
	}
	if _, err := s.exec("tf2ap_bots_wait_for_players 0"); err != nil {
		return "", 0, 0, err
	}
	teamName := "red"
	playerTeam := 2
	if strings.Contains(mission.PopFile, "_rev_") {
		teamName = "blue"
		playerTeam = 3
	}
	className := probeClassName(mission)
	playerClass := 1 // TFClass_Scout
	if className == "medic" {
		playerClass = 5 // TFClass_Medic
	}
	if _, err := s.exec("sm_waveprobe_wake " + teamName + " " + className); err != nil {
		return "", 0, 0, err
	}
	if _, err := s.exec("sm_ap_modifier clear"); err != nil {
		return "", 0, 0, err
	}
	if mode == "surge" {
		if _, err := s.exec("sm_ap_modifier on bot_surge"); err != nil {
			return "", 0, 0, err
		}
	}
	modifiers, err := s.exec("sm_ap_modifiers")
	if err != nil {
		return "", 0, 0, err
	}
	active := strings.Contains(modifiers, "Mission modifiers: Bot Surge")
	if active != (mode == "surge") {
		return "", 0, 0, fmt.Errorf("expected mode %s, got %q", mode, strings.TrimSpace(modifiers))
	}
	return teamName, playerTeam, playerClass, nil
}

var (
	errWallTimeout = errors.New("wave exceeded wall-clock time limit")
	errWaveZero    = errors.New("population manager remained at wave 0")
)

func waveSample(status probeStatus, elapsed time.Duration) sample {
	return sample{
		WallSeconds: elapsed.Seconds(), GameSeconds: status.Elapsed,
		Spawned: status.BotSpawns + status.TankSpawns,
		Killed:  status.Bots + status.Tanks, Alive: status.Alive,
		Remaining: status.Remaining, Progress: status.Progress,
	}
}

func (s *server) testWave(mission gamedata.Mission, wave, seed int, timeout, loadWait time.Duration) (probeStatus, []sample, error) {
	if err := s.await(loadWait, func(st probeStatus) bool {
		return st.Pop == mission.PopFile && st.GameWave == wave
	}); err != nil {
		status, _ := s.status()
		if status.Pop == mission.PopFile && status.GameWave == 0 {
			return status, nil, fmt.Errorf("%w: %s wave %d", errWaveZero, mission.PopFile, wave)
		}
		return status, nil, fmt.Errorf("wave %d was not initialized: %w", wave, err)
	}
	if _, err := s.exec(fmt.Sprintf("sm_waveprobe_arm %d %d", wave, seed)); err != nil {
		return probeStatus{}, nil, err
	}
	if _, err := s.exec("mp_restartgame 1"); err != nil {
		return probeStatus{}, nil, err
	}
	last, err := s.waitWaveStart(mission, wave, loadWait)
	if err != nil {
		return last, nil, err
	}
	if last.State == "passed" {
		return last, []sample{waveSample(last, 0)}, nil
	}
	// The 900-second budget starts when the wave actually runs. Map changes,
	// jump-to-wave, and the one-second restart are covered by loadWait instead.
	started := time.Now()
	deadline := started.Add(timeout)
	timeline := []sample{waveSample(last, 0)}
	lastSample := started
	for time.Now().Before(deadline) {
		status, err := s.status()
		if err == nil {
			if status.State != "running" && status.State != "passed" && status.State != "failed" ||
				status.Pop != mission.PopFile {
				timeline = append(timeline, waveSample(status, time.Since(started)))
				return last, timeline, fmt.Errorf("probe state reset during wave %d: state=%s pop=%s gamewave=%d",
					wave, status.State, status.Pop, status.GameWave)
			}
			last = status
			now := time.Now()
			if now.Sub(lastSample) >= 10*time.Second || status.State != "running" {
				timeline = append(timeline, waveSample(status, now.Sub(started)))
				lastSample = now
			}
			if status.State == "passed" && status.Expected == wave && status.Observed == wave {
				return status, timeline, nil
			}
			if status.State == "failed" {
				return status, timeline, fmt.Errorf("game or probe failed wave %d: %s", wave, status.Reason)
			}
			if status.State == "running" && status.GameWave == 0 {
				return status, timeline, fmt.Errorf("%w: %s wave %d reset during play", errWaveZero, mission.PopFile, wave)
			}
		}
		time.Sleep(time.Second)
	}
	status, err := s.status()
	if err == nil {
		last = status
	}
	timeline = append(timeline, waveSample(last, time.Since(started)))
	if last.State == "passed" && last.Expected == wave && last.Observed == wave {
		return last, timeline, nil
	}
	if last.State == "failed" {
		return last, timeline, fmt.Errorf("game or probe failed wave %d: %s", wave, last.Reason)
	}
	return last, timeline, fmt.Errorf("%w: wave %d still active after %s (%.1f game seconds)",
		errWallTimeout, wave, timeout, last.Elapsed)
}

func (s *server) waitWaveStart(mission gamedata.Mission, wave int, loadWait time.Duration) (probeStatus, error) {
	startDeadline := time.Now().Add(loadWait)
	var last probeStatus
	for time.Now().Before(startDeadline) {
		status, err := s.status()
		if err == nil {
			last = status
			if status.State == "passed" && status.Expected == wave && status.Observed == wave {
				return status, nil
			}
			if status.State == "failed" {
				return status, fmt.Errorf("game or probe failed before wave %d ran: %s", wave, status.Reason)
			}
			if status.State == "running" && status.Expected == wave && status.Observed == wave {
				return status, nil
			}
		}
		time.Sleep(time.Second)
	}
	if last.GameWave == 0 && last.Pop == mission.PopFile {
		return last, fmt.Errorf("%w: %s wave %d never started", errWaveZero, mission.PopFile, wave)
	}
	return last, fmt.Errorf("wave %d did not start within %s: %+v", wave, loadWait, last)
}

func (s *server) stopWave(mission gamedata.Mission) {
	// An active Bot Surge wave can keep creating bots and projectiles during
	// a queued map change. Reset the population manager before leaving it.
	_, _ = s.exec("tf_mvm_popfile " + mission.PopFile)
	_, _ = s.exec("sm_waveprobe_reset")
}

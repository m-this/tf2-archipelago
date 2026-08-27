package httpapi

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/m-this/tf2-archipelago/bridge/internal/apclient"
	"github.com/m-this/tf2-archipelago/bridge/internal/chat"
	"github.com/m-this/tf2-archipelago/bridge/internal/deathlink"
	"github.com/m-this/tf2-archipelago/bridge/internal/state"
	"github.com/m-this/tf2-archipelago/gamedata"
)

func newTestServer(t *testing.T, pollTimeout time.Duration) (*state.Store, http.Handler) {
	t.Helper()
	store, err := state.Open(filepath.Join(t.TempDir(), "bridge.json"))
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.DiscardHandler)
	messages := chat.New(8)
	deaths := deathlink.New(8)
	client := apclient.New(apclient.Options{
		SlotName: "tf2", Store: store, Chat: messages, Deaths: deaths, Logger: logger,
	})
	return store, New(store, client, messages, deaths, pollTimeout, logger).Handler()
}

func post(t *testing.T, handler http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/objective", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func get(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	return recorder
}

func TestObjectiveRecordsACheck(t *testing.T) {
	store, handler := newTestServer(t, time.Second)

	got := post(t, handler, `{"kind":"wave_cleared","popfile":"mvm_coaltown","wave":3}`)
	if got.Code != http.StatusNoContent {
		t.Fatalf("code = %d, body = %s", got.Code, got.Body)
	}
	mission, _ := gamedata.MissionByPopFile("mvm_coaltown")
	held := store.Checks()
	if len(held) != 1 || held[0] != mission.WaveLocationID(3) {
		t.Fatalf("checks = %v", held)
	}
}

func TestObjectiveIsIdempotent(t *testing.T) {
	store, handler := newTestServer(t, time.Second)
	body := `{"kind":"mission_cleared","popfile":"mvm_coaltown"}`

	for range 3 {
		if got := post(t, handler, body); got.Code != http.StatusNoContent {
			t.Fatalf("code = %d", got.Code)
		}
	}
	if held := store.Checks(); len(held) != 1 {
		t.Fatalf("%d checks held after three reports", len(held))
	}
}

func TestObjectiveRejectsWhatItCannotResolve(t *testing.T) {
	_, handler := newTestServer(t, time.Second)
	tests := map[string]string{
		// Deliberately not a kind that could become one: this case is about a
		// plugin newer than the bridge, not about the next objective added.
		"unknown kind":      `{"kind":"nonsense","popfile":"mvm_coaltown","wave":1}`,
		"unknown pop file":  `{"kind":"wave_cleared","popfile":"mvm_potato","wave":1}`,
		"wave past the end": `{"kind":"wave_cleared","popfile":"mvm_coaltown","wave":99}`,
		"wave zero":         `{"kind":"wave_cleared","popfile":"mvm_coaltown","wave":0}`,
		// Mannhattan runs on gates and has no tank, so no seed holds that
		// check. The plugin reports what the game gives it; this is where a
		// report with no location behind it stops.
		"tank on a mission with none": `{"kind":"tank_destroyed","popfile":"mvm_mannhattan","wave":1}`,
		"not json":                    `{`,
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			if got := post(t, handler, body); got.Code != http.StatusBadRequest {
				t.Fatalf("code = %d, want 400", got.Code)
			}
		})
	}
}

func TestUnlocksReportsWhatHasBeenGranted(t *testing.T) {
	store, handler := newTestServer(t, time.Second)
	mission, _ := gamedata.MissionByPopFile("mvm_coaltown")
	if err := store.ApplyItems(0, []int64{
		mission.TicketItemID(),
		gamedata.Classes[0].ItemID(),
	}); err != nil {
		t.Fatal(err)
	}

	var unlocks state.Unlocks
	decode(t, get(t, handler, "/unlocks"), &unlocks)
	// The cursor the plugin resumes from is the acknowledged one, not the
	// length of the item list. Nothing has been acknowledged here.
	if unlocks.ResumeFrom != 0 {
		t.Errorf("resume_from = %d", unlocks.ResumeFrom)
	}
	missions := unlocks.Of(gamedata.ItemMissionTicket)
	if len(missions) != 1 || missions[0] != "mvm_coaltown" {
		t.Errorf("missions = %v", missions)
	}
	classes := unlocks.Of(gamedata.ItemClass)
	if len(classes) != 1 || classes[0] != gamedata.Classes[0].Key {
		t.Errorf("classes = %v", classes)
	}
	// Every state kind is present even when the run holds none of it, so the
	// shape a plugin parses does not depend on what it happens to have.
	if _, listed := unlocks.ByKind[gamedata.ItemWeaponSlot.Key()]; !listed {
		t.Errorf("the unlock set has no entry for weapon slots: %+v", unlocks.ByKind)
	}
	if _, listed := unlocks.ByKind[gamedata.ItemCredits.Key()]; listed {
		t.Errorf("credits are an effect and must not be in the unlock set: %+v", unlocks.ByKind)
	}
}

func TestMissionsNameTheMapAndWhatIsUnlocked(t *testing.T) {
	// The run is what the seed drew, in that order: gamedata knows all 29
	// missions, a run holds a handful of them.
	drawn := []string{"mvm_ghost_town_666", "mvm_coaltown_intermediate"}
	coaltown, _ := gamedata.MissionByPopFile("mvm_coaltown_intermediate")
	// Checked by the room but not played here, which is the case another
	// world's !collect produces.
	missions, unknown := missionsFor(
		drawn, []string{"mvm_ghost_town_666"}, []int64{coaltown.ClearLocationID()}, nil, false)

	if len(unknown) != 0 {
		t.Fatalf("the tables did not know %v", unknown)
	}
	if len(missions) != 2 {
		t.Fatalf("missions = %+v", missions)
	}

	// The case that defeats guessing a map by trimming the popfile name, which
	// is the reason the bridge serves the map at all.
	haunted := missions[0]
	if haunted.PopFile != "mvm_ghost_town_666" || haunted.Map != "mvm_ghost_town" {
		t.Errorf("the haunted mission came back as %+v", haunted)
	}
	if haunted.Name != "Caliginous Caper" || haunted.Waves != 1 {
		t.Errorf("the haunted mission came back as %+v", haunted)
	}
	if !haunted.Unlocked {
		t.Error("the mission whose ticket the run holds is not marked unlocked")
	}
	if missions[1].Unlocked {
		t.Error("a mission with no ticket is marked unlocked")
	}
	if missions[1].PopFile != "mvm_coaltown_intermediate" {
		t.Errorf("the run came back in a different order: %+v", missions)
	}
	if haunted.Cleared || !missions[1].Cleared {
		t.Errorf("cleared marks the wrong mission: %+v", missions)
	}
}

func TestMissionsSkipWhatTheTablesDoNotKnow(t *testing.T) {
	missions, unknown := missionsFor([]string{"mvm_potato", "mvm_coaltown"}, nil, nil, nil, false)
	if len(missions) != 1 || missions[0].PopFile != "mvm_coaltown" {
		t.Fatalf("missions = %+v", missions)
	}
	if len(unknown) != 1 || unknown[0] != "mvm_potato" {
		t.Fatalf("unknown = %v", unknown)
	}
}

func TestUsefulTicketsLeaveEveryDrawnMissionUnlocked(t *testing.T) {
	drawn := []string{"mvm_coaltown", "mvm_coaltown_intermediate"}
	missions, unknown := missionsFor(drawn, nil, nil, nil, true)
	if len(unknown) != 0 || len(missions) != 2 {
		t.Fatalf("missions = %+v, unknown = %v", missions, unknown)
	}
	for _, mission := range missions {
		if !mission.Unlocked {
			t.Errorf("useful ticket mode left %s locked", mission.PopFile)
		}
	}
}

// A run whose session has not handshaked yet has no missions, and the shape has
// to hold: the plugin parses the same object either way.
func TestMissionsAreAnEmptyListWithoutASession(t *testing.T) {
	_, handler := newTestServer(t, time.Second)
	var response missionsResponse
	decode(t, get(t, handler, "/missions"), &response)
	if len(response.Missions) != 0 {
		t.Fatalf("missions = %+v", response.Missions)
	}
}

func TestGrantsReturnWhatIsAlreadyThere(t *testing.T) {
	store, handler := newTestServer(t, time.Second)
	if err := store.ApplyItems(0, []int64{gamedata.Classes[0].ItemID()}); err != nil {
		t.Fatal(err)
	}

	var response grantsResponse
	decode(t, get(t, handler, "/grants?since=0"), &response)
	if len(response.Grants) != 1 || response.Grants[0].Seq != 1 {
		t.Fatalf("grants = %+v", response.Grants)
	}
}

func TestGrantsTimeOutWithNothingNew(t *testing.T) {
	_, handler := newTestServer(t, 20*time.Millisecond)

	var response grantsResponse
	decode(t, get(t, handler, "/grants?since=0"), &response)
	if len(response.Grants) != 0 || response.Seq != 0 {
		t.Fatalf("timed out with %+v", response)
	}
}

func TestGrantsTellAPluginThatIsAhead(t *testing.T) {
	store, handler := newTestServer(t, 5*time.Second)
	if err := store.ApplyItems(0, []int64{gamedata.Classes[0].ItemID()}); err != nil {
		t.Fatal(err)
	}

	// Only a restarted run can put a plugin ahead, and then it must be told at once.
	done := make(chan *httptest.ResponseRecorder, 1)
	go func() { done <- get(t, handler, "/grants?since=9") }()
	select {
	case got := <-done:
		var response grantsResponse
		decode(t, got, &response)
		if response.Seq != 1 || len(response.Grants) != 0 {
			t.Fatalf("response = %+v", response)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("a plugin ahead of the bridge was left waiting")
	}
}

func TestGrantsWakeOnANewItem(t *testing.T) {
	store, handler := newTestServer(t, 5*time.Second)
	done := make(chan *httptest.ResponseRecorder, 1)
	go func() { done <- get(t, handler, "/grants?since=0") }()

	// The poll may not have registered yet; the watch channel is taken before the first read.
	time.Sleep(20 * time.Millisecond)
	if err := store.ApplyItems(0, []int64{gamedata.Classes[2].ItemID()}); err != nil {
		t.Fatal(err)
	}

	select {
	case got := <-done:
		var response grantsResponse
		decode(t, got, &response)
		if len(response.Grants) != 1 {
			t.Fatalf("grants = %+v", response.Grants)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the long poll was not woken")
	}
}

func TestGrantsKeepWaitingThroughAnUnrelatedChange(t *testing.T) {
	store, handler := newTestServer(t, 300*time.Millisecond)
	mission, _ := gamedata.MissionByPopFile("mvm_decoy")
	done := make(chan *httptest.ResponseRecorder, 1)
	go func() { done <- get(t, handler, "/grants?since=0") }()

	time.Sleep(20 * time.Millisecond)
	if _, err := store.AddCheck(mission.WaveLocationID(1)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	if err := store.ApplyItems(0, []int64{gamedata.Classes[1].ItemID()}); err != nil {
		t.Fatal(err)
	}

	select {
	case got := <-done:
		var response grantsResponse
		decode(t, got, &response)
		if len(response.Grants) != 1 {
			t.Fatalf("grants = %+v", response.Grants)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the long poll never answered")
	}
}

func TestGrantsRejectAMissingSequence(t *testing.T) {
	_, handler := newTestServer(t, time.Second)
	for _, path := range []string{"/grants", "/grants?since=", "/grants?since=-1", "/grants?since=x"} {
		if got := get(t, handler, path); got.Code != http.StatusBadRequest {
			t.Errorf("%s: code = %d, want 400", path, got.Code)
		}
	}
}

// A lost wave used to leave no trace when the seed had DeathLink off, so
// "which waves did we lose" had no answer at all. That question is the whole
// of tuning the team size for fewer than six players.
func TestHealthCountsTheWavesTheTeamLost(t *testing.T) {
	_, handler := newTestServer(t, time.Second)
	postTo := func(body string) {
		t.Helper()
		request := httptest.NewRequest(http.MethodPost, "/death", strings.NewReader(body))
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		// No session here, so the forward fails; the count is taken first.
		if recorder.Code != http.StatusNoContent && recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("code = %d", recorder.Code)
		}
	}
	postTo(`{"popfile":"mvm_coaltown","wave":3}`)
	postTo(`{"popfile":"mvm_coaltown","wave":3}`)
	postTo(`{"popfile":"mvm_decoy","wave":1}`)

	var health healthResponse
	decode(t, get(t, handler, "/healthz"), &health)
	want := []waveFailure{
		{PopFile: "mvm_coaltown", Wave: 3, Lost: 2},
		{PopFile: "mvm_decoy", Wave: 1, Lost: 1},
	}
	// Worst first: the wave that cost the evening is the first line.
	if !slices.Equal(health.WaveFailures, want) {
		t.Errorf("wave failures = %v, want %v", health.WaveFailures, want)
	}
}

func TestHealthReportsTheSession(t *testing.T) {
	_, handler := newTestServer(t, time.Second)
	var health healthResponse
	decode(t, get(t, handler, "/healthz"), &health)
	if health.Connected {
		t.Error("reported as connected without a session")
	}
	if health.Slot != "tf2" {
		t.Errorf("slot = %q", health.Slot)
	}
	if health.APIVersion != APIVersion {
		t.Errorf("api version = %d, want %d", health.APIVersion, APIVersion)
	}
}

func TestHealthReportsTheRun(t *testing.T) {
	store, handler := newTestServer(t, time.Second)
	if got := post(t, handler, `{"kind":"wave_cleared","popfile":"mvm_coaltown","wave":2}`); got.Code != http.StatusNoContent {
		t.Fatalf("code = %d", got.Code)
	}
	if err := store.ApplyItems(0, []int64{gamedata.Classes[0].ItemID()}); err != nil {
		t.Fatal(err)
	}

	var health healthResponse
	decode(t, get(t, handler, "/healthz"), &health)
	if health.Checks != 1 || health.Items != 1 {
		t.Errorf("checks = %d, items = %d", health.Checks, health.Items)
	}
	// "Did my wave count" is a question about the last five minutes, and the
	// bridge's side of it is invisible without this.
	if health.LastCheck != "Crash Course Wave 2" {
		t.Errorf("last check = %q", health.LastCheck)
	}
	if health.LastCheckAt == nil {
		t.Error("the last check has no time")
	}
}

func TestTheGameDisagreeingAboutAMissionLengthIsReported(t *testing.T) {
	_, handler := newTestServer(t, time.Second)
	mission, _ := gamedata.MissionByPopFile("mvm_coaltown")

	// Every wave count in the tables comes from the wiki. A wrong one on the
	// goal mission makes a seed unwinnable, and this is the only thing that
	// would ever say so.
	body := fmt.Sprintf(
		`{"kind":"wave_cleared","popfile":"mvm_coaltown","wave":1,"waves_total":%d}`,
		mission.Waves+2,
	)
	if got := post(t, handler, body); got.Code != http.StatusNoContent {
		t.Fatalf("the check was refused over a table disagreement: %d", got.Code)
	}

	var health healthResponse
	decode(t, get(t, handler, "/healthz"), &health)
	if len(health.WaveDrift) != 1 {
		t.Fatalf("wave drift = %+v", health.WaveDrift)
	}
	got := health.WaveDrift[0]
	if got.PopFile != "mvm_coaltown" || got.Tables != int(mission.Waves) || got.Observed != int(mission.Waves)+2 {
		t.Fatalf("wave drift = %+v", got)
	}
}

func TestAMissionLongerThanTheIDSchemeAllowsIsStillReported(t *testing.T) {
	_, handler := newTestServer(t, time.Second)

	// Past what gamedata can number, and therefore the most useful thing this
	// could ever report: every location id for that mission would be wrong.
	body := fmt.Sprintf(
		`{"kind":"wave_cleared","popfile":"mvm_coaltown","wave":1,"waves_total":%d}`,
		int(gamedata.WavesMax)+1)
	if got := post(t, handler, body); got.Code != http.StatusNoContent {
		t.Fatalf("code = %d", got.Code)
	}
	var health healthResponse
	decode(t, get(t, handler, "/healthz"), &health)
	if len(health.WaveDrift) != 1 || health.WaveDrift[0].Observed != int(gamedata.WavesMax)+1 {
		t.Fatalf("wave drift = %+v", health.WaveDrift)
	}
}

func TestAWaveCountThatCannotBeReadIsIgnored(t *testing.T) {
	_, handler := newTestServer(t, time.Second)

	// The property behind this has never been seen answer. Whatever it returns,
	// it must not cost the wave the team just cleared.
	for _, observed := range []int{-1, wavesObservedMax + 1, 1 << 30} {
		body := fmt.Sprintf(
			`{"kind":"wave_cleared","popfile":"mvm_coaltown","wave":1,"waves_total":%d}`, observed)
		if got := post(t, handler, body); got.Code != http.StatusNoContent {
			t.Fatalf("waves_total %d answered %d, and the check was lost", observed, got.Code)
		}
	}
	var health healthResponse
	decode(t, get(t, handler, "/healthz"), &health)
	if len(health.WaveDrift) != 0 {
		t.Fatalf("a bad read was reported as drift: %+v", health.WaveDrift)
	}
}

func TestAMissionLengthThatAgreesIsNotReported(t *testing.T) {
	_, handler := newTestServer(t, time.Second)
	mission, _ := gamedata.MissionByPopFile("mvm_coaltown")

	for _, observed := range []uint8{mission.Waves, 0} {
		body := fmt.Sprintf(
			`{"kind":"wave_cleared","popfile":"mvm_coaltown","wave":1,"waves_total":%d}`, observed)
		if got := post(t, handler, body); got.Code != http.StatusNoContent {
			t.Fatalf("code = %d", got.Code)
		}
	}
	var health healthResponse
	decode(t, get(t, handler, "/healthz"), &health)
	if len(health.WaveDrift) != 0 {
		t.Fatalf("wave drift = %+v", health.WaveDrift)
	}
}

func TestAnAcknowledgementStopsAnEffectComingBack(t *testing.T) {
	store, handler := newTestServer(t, 50*time.Millisecond)
	cash := cashBundleID(t)
	if err := store.ApplyItems(0, []int64{gamedata.Classes[0].ItemID(), cash}); err != nil {
		t.Fatal(err)
	}

	var response grantsResponse
	decode(t, get(t, handler, "/grants?since=0"), &response)
	if len(response.Grants) != 2 {
		t.Fatalf("grants = %+v", response.Grants)
	}

	if got := postTo(t, handler, "/grants/ack", `{"seq":2}`); got.Code != http.StatusNoContent {
		t.Fatalf("the acknowledgement answered %d: %s", got.Code, got.Body)
	}

	// A plugin that reloaded asks from zero again. The class comes back, the
	// cash does not: paying it twice is money nobody earned.
	decode(t, get(t, handler, "/grants?since=0"), &response)
	if len(response.Grants) != 1 || response.Grants[0].Kind != gamedata.ItemClass.Key() {
		t.Fatalf("after the acknowledgement: %+v", response.Grants)
	}
}

func TestAnAcknowledgementPastTheEndIsRefused(t *testing.T) {
	_, handler := newTestServer(t, time.Second)
	for name, body := range map[string]string{
		"past the items that exist": `{"seq":5}`,
		"negative":                  `{"seq":-1}`,
		"not json":                  `{`,
	} {
		t.Run(name, func(t *testing.T) {
			if got := postTo(t, handler, "/grants/ack", body); got.Code != http.StatusBadRequest {
				t.Fatalf("code = %d, want 400", got.Code)
			}
		})
	}
}

func TestSayRefusesACommandThatCanEndTheRun(t *testing.T) {
	_, handler := newTestServer(t, time.Second)
	// There is no session, so a line that got through would answer 503. This
	// has to be refused before that, and told apart from it: the plugin says
	// something different to the player for each.
	if got := postTo(t, handler, "/say", `{"text":"!release"}`); got.Code != http.StatusForbidden {
		t.Fatalf("code = %d, want 403", got.Code)
	}
}

func TestSayGuardsAgainstAFlood(t *testing.T) {
	_, handler := newTestServer(t, time.Second)
	refused := 0
	for range 20 {
		if postTo(t, handler, "/say", `{"text":"spam"}`).Code == http.StatusTooManyRequests {
			refused++
		}
	}
	if refused == 0 {
		t.Fatal("a player pasting a wall of text reached the multiworld unchecked")
	}
}

func postTo(t *testing.T, handler http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func cashBundleID(t *testing.T) int64 {
	t.Helper()
	for _, item := range gamedata.Items {
		if item.Kind == gamedata.ItemCredits {
			return item.ID
		}
	}
	t.Fatal("no credits item in the tables")
	return 0
}

func decode(t *testing.T, recorder *httptest.ResponseRecorder, into any) {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("code = %d, body = %s", recorder.Code, recorder.Body)
	}
	body, err := io.ReadAll(recorder.Body)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, into); err != nil {
		t.Fatalf("%s: %v", body, err)
	}
}

func TestSayNeedsAMultiworld(t *testing.T) {
	_, handler := newTestServer(t, time.Second)
	request := httptest.NewRequest(http.MethodPost, "/say", strings.NewReader(`{"text":"!hint"}`))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("code = %d, want 503", recorder.Code)
	}
}

func TestSayRejectsAnEmptyLine(t *testing.T) {
	_, handler := newTestServer(t, time.Second)
	request := httptest.NewRequest(http.MethodPost, "/say", strings.NewReader(`{"text":""}`))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", recorder.Code)
	}
}

func TestMessagesStartFromNow(t *testing.T) {
	_, handler := newTestServer(t, time.Second)

	var response messagesResponse
	decode(t, get(t, handler, "/messages?since=-1"), &response)
	if len(response.Messages) != 0 {
		t.Fatalf("a negative sequence returned %d message(s)", len(response.Messages))
	}
}

/*
Cleared is what the room holds; played is what this server did.

Another world running !collect on its goal sends every check it still holds,
which marked missions here as cleared that nobody on this server had played.
Reported by Peppy: "it does make it harder to keep track of what you've
actually completed or not".
*/
func TestAMissionCheckedByTheRoomIsNotPlayedHere(t *testing.T) {
	drawn := []string{"mvm_coaltown_intermediate", "mvm_ghost_town_666"}
	coaltown, _ := gamedata.MissionByPopFile("mvm_coaltown_intermediate")
	ghost, _ := gamedata.MissionByPopFile("mvm_ghost_town_666")

	missions, _ := missionsFor(drawn, drawn,
		[]int64{coaltown.ClearLocationID(), ghost.ClearLocationID()},
		[]int64{ghost.ClearLocationID()}, false)

	if len(missions) != 2 {
		t.Fatalf("missions = %+v", missions)
	}
	if !missions[0].Cleared || missions[0].Played {
		t.Errorf("the collected mission = cleared %v, played %v; want cleared, not played",
			missions[0].Cleared, missions[0].Played)
	}
	if !missions[1].Cleared || !missions[1].Played {
		t.Errorf("the played mission = cleared %v, played %v; want both",
			missions[1].Cleared, missions[1].Played)
	}
}

/*
	The missions answer carries where the team was, so a restarted server can put

them back without a second round trip.

Absent until there is something to say. A fresh run and a mission just started
both have nothing, and a plugin that reads an empty record as "resume at wave
zero" would be worse than one that reads nothing at all.
*/
func TestTheMissionsAnswerCarriesWhereTheRunWas(t *testing.T) {
	store, handler := newTestServer(t, time.Second)

	body := get(t, handler, "/missions").Body.String()
	if strings.Contains(body, "resume") {
		t.Errorf("a fresh run offered somewhere to resume:\n%s", body)
	}

	if err := store.NoteProgress("mvm_decoy_advanced", 3); err != nil {
		t.Fatal(err)
	}
	body = get(t, handler, "/missions").Body.String()
	for _, want := range []string{`"resume"`, `"mvm_decoy_advanced"`, `"wave":3`} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q in:\n%s", want, body)
		}
	}

	if err := store.ClearProgress(); err != nil {
		t.Fatal(err)
	}
	if body = get(t, handler, "/missions").Body.String(); strings.Contains(body, "resume") {
		t.Errorf("a finished mission still offered a resume:\n%s", body)
	}
}

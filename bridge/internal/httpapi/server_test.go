package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git-ssh.croque.top/mathis/tf2-archipelago/bridge/internal/apclient"
	"git-ssh.croque.top/mathis/tf2-archipelago/bridge/internal/chat"
	"git-ssh.croque.top/mathis/tf2-archipelago/bridge/internal/state"
	"git-ssh.croque.top/mathis/tf2-archipelago/gamedata"
)

func newTestServer(t *testing.T, pollTimeout time.Duration) (*state.Store, http.Handler) {
	t.Helper()
	store, err := state.Open(filepath.Join(t.TempDir(), "bridge.json"))
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.DiscardHandler)
	messages := chat.New(8)
	client := apclient.New(apclient.Options{
		SlotName: "tf2", Store: store, Chat: messages, Logger: logger,
	})
	return store, New(store, client, messages, pollTimeout, logger).Handler()
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
		"unknown kind":      `{"kind":"tank_destroyed","popfile":"mvm_coaltown","wave":1}`,
		"unknown pop file":  `{"kind":"wave_cleared","popfile":"mvm_potato","wave":1}`,
		"wave past the end": `{"kind":"wave_cleared","popfile":"mvm_coaltown","wave":99}`,
		"wave zero":         `{"kind":"wave_cleared","popfile":"mvm_coaltown","wave":0}`,
		"not json":          `{`,
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
	if unlocks.Seq != 2 {
		t.Errorf("seq = %d", unlocks.Seq)
	}
	if len(unlocks.Missions) != 1 || unlocks.Missions[0] != "mvm_coaltown" {
		t.Errorf("missions = %v", unlocks.Missions)
	}
	if len(unlocks.Classes) != 1 || unlocks.Classes[0] != gamedata.Classes[0].Key {
		t.Errorf("classes = %v", unlocks.Classes)
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
	got := get(t, handler, "/grants?since=0")
	if got.Code != http.StatusNoContent {
		t.Fatalf("code = %d, want 204", got.Code)
	}
}

func TestGrantsWakeOnANewItem(t *testing.T) {
	store, handler := newTestServer(t, 5*time.Second)
	done := make(chan *httptest.ResponseRecorder, 1)
	go func() { done <- get(t, handler, "/grants?since=0") }()

	// The long poll may not have registered yet; the store's watch channel is
	// taken before the first read, so a slightly late write still wakes it.
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

func TestHealthReportsTheSession(t *testing.T) {
	_, handler := newTestServer(t, time.Second)
	var health apclient.Health
	decode(t, get(t, handler, "/healthz"), &health)
	if health.Connected {
		t.Error("reported as connected without a session")
	}
	if health.Slot != "tf2" {
		t.Errorf("slot = %q", health.Slot)
	}
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

	// Nothing is connected in this test, and a line with nowhere to go is
	// refused rather than queued.
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
	store, handler := newTestServer(t, time.Second)
	_ = store

	var response messagesResponse
	decode(t, get(t, handler, "/messages?since=-1"), &response)
	if len(response.Messages) != 0 {
		t.Fatalf("a negative sequence returned %d message(s)", len(response.Messages))
	}
}

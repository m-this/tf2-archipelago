package httpapi

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/m-this/tf2-archipelago/bridge/internal/apclient"
	"github.com/m-this/tf2-archipelago/bridge/internal/chat"
	"github.com/m-this/tf2-archipelago/bridge/internal/deathlink"
	"github.com/m-this/tf2-archipelago/bridge/internal/state"
)

func newTestServerWithDeaths(t *testing.T) (*deathlink.Feed, http.Handler) {
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
	return deaths, New(store, client, messages, deaths, time.Second, logger).Handler()
}

func TestDeathNeedsAMultiworld(t *testing.T) {
	_, handler := newTestServerWithDeaths(t)
	request := httptest.NewRequest(http.MethodPost, "/death", strings.NewReader(`{"popfile":"mvm_decoy","wave":2}`))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("code = %d, want 503", recorder.Code)
	}
}

func TestDeathRejectsWhatItCannotRead(t *testing.T) {
	_, handler := newTestServerWithDeaths(t)
	request := httptest.NewRequest(http.MethodPost, "/death", strings.NewReader(`nope`))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", recorder.Code)
	}
}

func TestDeathsStartFromNow(t *testing.T) {
	deaths, handler := newTestServerWithDeaths(t)
	deaths.Append("Ana", "fell")

	var response deathsResponse
	decode(t, get(t, handler, "/deaths?since=-1"), &response)
	if response.Seq != 1 || len(response.Deaths) != 0 {
		t.Fatalf("response = %+v", response)
	}
}

func TestDeathsReturnWhatIsPastTheSequence(t *testing.T) {
	deaths, handler := newTestServerWithDeaths(t)
	deaths.Append("Ana", "fell")
	deaths.Append("Bram", "")

	var response deathsResponse
	decode(t, get(t, handler, "/deaths?since=1"), &response)
	if response.Seq != 2 || len(response.Deaths) != 1 || response.Deaths[0].Source != "Bram" {
		t.Fatalf("response = %+v", response)
	}
	if response.DeathLink {
		t.Fatal("death_link is on with no session")
	}
}

func TestDeathsTimeOutWithNothingNew(t *testing.T) {
	_, handler := newTestServerWithDeaths(t)
	var response deathsResponse
	decode(t, get(t, handler, "/deaths?since=0"), &response)
	if response.Seq != 0 || response.Deaths == nil || len(response.Deaths) != 0 {
		t.Fatalf("response = %+v", response)
	}
}

func TestDeathCauseNamesTheMission(t *testing.T) {
	got := deathCause("tf2", deathRequest{PopFile: "mvm_decoy", Wave: 3})
	if got != "tf2 lost wave 3 of Doe's Drill" {
		t.Fatalf("cause = %q", got)
	}
	got = deathCause("tf2", deathRequest{PopFile: "mvm_nowhere"})
	if got != "tf2 lost a wave of mvm_nowhere" {
		t.Fatalf("cause = %q", got)
	}
}

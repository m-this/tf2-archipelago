package httpapi

import (
	"fmt"
	"log/slog"
	"net"
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
	"github.com/m-this/tf2-archipelago/gamedata"
)

// newTestMetrics returns both handlers over one server: the plugin's, to drive
// state through it, and the metrics one, to read the result.
func newTestMetrics(t *testing.T) (http.Handler, http.Handler) {
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
	server := New(store, client, messages, deathlink.New(1), time.Second, logger)
	return server.Handler(), server.MetricsHandler("")
}

// newTestMetricsWithGame returns the metrics handler alone, wired to a game
// server to query. Nothing here drives the plugin's side.
func newTestMetricsWithGame(t *testing.T, gameAddr string) http.Handler {
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
	server := New(store, client, messages, deathlink.New(1), time.Second, logger)
	return server.MetricsHandler(gameAddr)
}

func scrape(t *testing.T, handler http.Handler) string {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("code = %d, body = %s", recorder.Code, recorder.Body)
	}
	return recorder.Body.String()
}

func requireLine(t *testing.T, body, want string) {
	t.Helper()
	for line := range strings.SplitSeq(body, "\n") {
		if line == want {
			return
		}
	}
	t.Fatalf("missing %q in:\n%s", want, body)
}

func TestMetricsReportsAnIdleRun(t *testing.T) {
	_, metrics := newTestMetrics(t)

	body := scrape(t, metrics)
	requireLine(t, body, "tf2ap_up 1")
	requireLine(t, body, "tf2ap_session_connected 0")
	requireLine(t, body, "tf2ap_run_checks_total 0")
	requireLine(t, body, "tf2ap_run_goal_sent 0")
	requireLine(t, body, `tf2ap_run_info{seed="",slot="tf2"} 1`)

	// A run that has recorded nothing has no timestamp to report, and 0 there
	// would read as 1970 on every dashboard that plots it.
	if strings.Contains(body, "tf2ap_run_last_check_timestamp_seconds") {
		t.Fatalf("idle run reported a last-check timestamp:\n%s", body)
	}
	// Nor does it have drift: the metric is declared, with no series.
	if strings.Contains(body, "tf2ap_mission_wave_drift{") {
		t.Fatalf("idle run reported wave drift:\n%s", body)
	}
}

func TestMetricsCountsAChecked(t *testing.T) {
	plugin, metrics := newTestMetrics(t)

	got := post(t, plugin, `{"kind":"wave_cleared","popfile":"mvm_coaltown","wave":3}`)
	if got.Code != http.StatusNoContent {
		t.Fatalf("code = %d, body = %s", got.Code, got.Body)
	}

	body := scrape(t, metrics)
	requireLine(t, body, "tf2ap_run_checks_total 1")
	if !strings.Contains(body, "tf2ap_run_last_check_timestamp_seconds ") {
		t.Fatalf("a recorded check left no timestamp:\n%s", body)
	}
}

func TestMetricsReportsWaveDrift(t *testing.T) {
	plugin, metrics := newTestMetrics(t)
	mission, _ := gamedata.MissionByPopFile("mvm_coaltown")

	// The game saying a mission is longer than the tables claim means every
	// location id for it is wrong, which is the one thing here worth an alert.
	body := fmt.Sprintf(
		`{"kind":"wave_cleared","popfile":"mvm_coaltown","wave":1,"waves_total":%d}`,
		mission.Waves+2)
	if got := post(t, plugin, body); got.Code != http.StatusNoContent {
		t.Fatalf("code = %d, body = %s", got.Code, got.Body)
	}

	requireLine(t, scrape(t, metrics), `tf2ap_mission_wave_drift{mission="mvm_coaltown"} 2`)
}

// fakeGameServer answers one A2S_INFO with the counts given and returns its
// address. Mann vs Machine reports its robot wave as bots, so 7 players with 5
// bots is two people playing.
func fakeGameServer(t *testing.T, players, maxPlayers, bots byte) string {
	t.Helper()
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	reply := []byte("\xff\xff\xff\xffI\x11")
	for _, field := range []string{"Mann vs Archipelago", "mvm_decoy", "tf", "Team Fortress"} {
		reply = append(reply, append([]byte(field), 0)...)
	}
	reply = append(reply, 0xb8, 0x01, players, maxPlayers, bots)

	go func() {
		buffer := make([]byte, 1024)
		_, from, err := conn.ReadFrom(buffer)
		if err != nil {
			return
		}
		_, _ = conn.WriteTo(reply, from)
	}()
	return conn.LocalAddr().String()
}

func TestMetricsCountsWhoIsConnected(t *testing.T) {
	metrics := newTestMetricsWithGame(t, fakeGameServer(t, 7, 6, 5))

	body := scrape(t, metrics)
	requireLine(t, body, "tf2ap_game_up 1")
	requireLine(t, body, "tf2ap_game_players 7")
	requireLine(t, body, "tf2ap_game_players_human 2")
	requireLine(t, body, "tf2ap_game_bots 5")
	requireLine(t, body, "tf2ap_game_players_max 6")
	requireLine(t, body, `tf2ap_game_map{map="mvm_decoy"} 1`)
}

func TestMetricsSeparatesAnEmptyServerFromAnAbsentOne(t *testing.T) {
	// Nothing listening: srcds restarting, or the port wrong. Reporting zero
	// players there would draw an empty server rather than a missing one.
	metrics := newTestMetricsWithGame(t, "127.0.0.1:1")

	body := scrape(t, metrics)
	requireLine(t, body, "tf2ap_game_up 0")
	if strings.Contains(body, "tf2ap_game_players ") {
		t.Fatalf("a server that did not answer still reported players:\n%s", body)
	}
}

func TestMetricsLeavesGameOutWhenNoAddressIsGiven(t *testing.T) {
	_, metrics := newTestMetrics(t)

	if body := scrape(t, metrics); strings.Contains(body, "tf2ap_game_") {
		t.Fatalf("game metrics appeared without an address:\n%s", body)
	}
}

func TestEscapeLabelEscapesWhatTheFormatReserves(t *testing.T) {
	got := escapeLabel("a\\b\"c\nd")
	want := `a\\b\"c\nd`
	if got != want {
		t.Fatalf("escapeLabel = %q, want %q", got, want)
	}
}

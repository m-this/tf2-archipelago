package httpapi

import (
	"fmt"
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
	server := New(store, client, messages, time.Second, logger)
	return server.Handler(), server.MetricsHandler()
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

func TestEscapeLabelEscapesWhatTheFormatReserves(t *testing.T) {
	got := escapeLabel("a\\b\"c\nd")
	want := `a\\b\"c\nd`
	if got != want {
		t.Fatalf("escapeLabel = %q, want %q", got, want)
	}
}

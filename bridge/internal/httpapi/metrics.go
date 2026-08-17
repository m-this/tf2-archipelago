package httpapi

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// The metrics endpoint is a second view of what /healthz already reports, in
// Prometheus exposition format, so a run can be watched over time instead of
// polled by hand. It answers on its own listener: /healthz and the rest are the
// plugin's, on loopback inside the game server's network namespace, and a
// scraper lives on another machine.
//
// Written by hand rather than through a client library. There are nine numbers
// here, all read off state that is already in memory; a registry, a collector
// interface and a dependency would be more machinery than the thing measured.

// MetricsHandler serves the run as Prometheus exposition text.
func (s *Server) MetricsHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /metrics", s.getMetrics)
	return mux
}

func (s *Server) getMetrics(w http.ResponseWriter, r *http.Request) {
	session, run := s.client.Health(), s.store.Stats()

	var out strings.Builder
	metric(&out, "tf2ap_up", "gauge", "1 while the bridge is serving.", 1)
	metric(&out, "tf2ap_api_version", "gauge",
		"Version of the API the plugin talks to.", float64(APIVersion))

	metric(&out, "tf2ap_session_connected", "gauge",
		"1 while the Archipelago session is up. A run survives this being 0; checks queue.",
		boolValue(session.Connected))
	metric(&out, "tf2ap_session_missions", "gauge",
		"Missions the seed drew for this slot.", float64(len(session.Missions)))

	metric(&out, "tf2ap_run_checks_total", "counter",
		"Locations checked and recorded by the bridge.", float64(run.Checks))
	metric(&out, "tf2ap_run_items_total", "counter",
		"Items received from the multiworld.", float64(run.Items))
	metric(&out, "tf2ap_run_acked_seq", "gauge",
		"Highest grant sequence the plugin acknowledged. Stalling behind the item count is the game server not applying them.",
		float64(run.AckedSeq))
	metric(&out, "tf2ap_run_goal_sent", "gauge",
		"1 once the goal has been reported, which is the run being finished.",
		boolValue(run.GoalSent))
	if run.LastCheckAt != nil {
		metric(&out, "tf2ap_run_last_check_timestamp_seconds", "gauge",
			"When the last check was recorded.", float64(run.LastCheckAt.Unix()))
	}

	// One series per mission the game and the tables disagree about, valued at
	// the difference. Anything non-zero is a seed with wrong wave counts, which
	// is the failure that cannot be repaired once a run is under way.
	header(&out, "tf2ap_mission_wave_drift", "gauge",
		"Waves the game reported minus waves the tables claim, per mission. No series is the healthy case.")
	for _, drift := range s.waveDrift() {
		sample(&out, "tf2ap_mission_wave_drift",
			labels("mission", drift.PopFile), float64(drift.Observed-drift.Tables))
	}

	// The seed and the slot identify the run the numbers above belong to, and
	// they are strings: labels on a constant is the only shape Prometheus has
	// for that.
	header(&out, "tf2ap_run_info", "gauge", "The run these metrics belong to.")
	sample(&out, "tf2ap_run_info", labels("seed", run.Seed, "slot", session.Slot), 1)

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	if _, err := io.WriteString(w, out.String()); err != nil {
		s.logger.ErrorContext(r.Context(), "cannot write the metrics response", "error", err)
	}
}

// metric writes a whole single-sample metric: both comment lines and the value.
func metric(out *strings.Builder, name, kind, help string, value float64) {
	header(out, name, kind, help)
	sample(out, name, "", value)
}

func header(out *strings.Builder, name, kind, help string) {
	fmt.Fprintf(out, "# HELP %s %s\n# TYPE %s %s\n", name, help, name, kind)
}

func sample(out *strings.Builder, name, labelSet string, value float64) {
	fmt.Fprintf(out, "%s%s %g\n", name, labelSet, value)
}

// labels renders key/value pairs as a label set. The values are a seed name and
// mission file names, so they are escaped rather than trusted.
func labels(pairs ...string) string {
	if len(pairs) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteByte('{')
	for i := 0; i+1 < len(pairs); i += 2 {
		if i > 0 {
			out.WriteByte(',')
		}
		fmt.Fprintf(&out, `%s="%s"`, pairs[i], escapeLabel(pairs[i+1]))
	}
	out.WriteByte('}')
	return out.String()
}

// escapeLabel escapes the three characters the exposition format reserves in a
// label value.
func escapeLabel(value string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`).Replace(value)
}

func boolValue(set bool) float64 {
	if set {
		return 1
	}
	return 0
}

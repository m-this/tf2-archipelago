// Package httpapi is the plugin-facing side of the bridge.
//
// The wire format is the whole contract with the SourceMod plugin, and it is in
// Mann vs Machine's vocabulary: the plugin reports objectives and applies
// grants, never Archipelago ids.
//
// Loopback only. Nothing here authenticates, because nothing off the machine
// can reach it.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"git-ssh.croque.top/mathis/tf2-archipelago/bridge/internal/apclient"
	"git-ssh.croque.top/mathis/tf2-archipelago/bridge/internal/chat"
	"git-ssh.croque.top/mathis/tf2-archipelago/bridge/internal/state"
	"git-ssh.croque.top/mathis/tf2-archipelago/gamedata"
)

// APIVersion is the contract with the plugin. The plugin reads it at startup
// and says so in chat when it does not match: the two ship in one compose file,
// so a mismatch means one image was updated and the other was not.
const APIVersion = 1

// wavesObservedMax bounds what the game is believed about a mission's length.
// The property behind it has never been seen answer, so anything past what a
// byte holds is a bad read rather than a long mission.
const wavesObservedMax = 255

// objectiveRequest is what the plugin posts. Wave is ignored for a mission clear.
type objectiveRequest struct {
	Kind    string `json:"kind"`
	PopFile string `json:"popfile"`
	Wave    uint8  `json:"wave"`

	// WavesTotal is how many waves the game says the mission has, zero when it
	// would not say. Every wave count in gamedata comes from the wiki and none
	// has been checked against a running server, so this is the one chance to
	// notice a wrong row before it makes a seed unwinnable.
	//
	// Deliberately a wide, lenient int. It is read from a network property this
	// project has never seen answer, and a diagnostic field that fails to parse
	// would take the whole check down with it: the plugin reads that 400 as
	// "this objective does not exist" and drops a cleared wave.
	WavesTotal int `json:"waves_total"`
}

// ackRequest is the plugin confirming it applied everything up to a sequence.
type ackRequest struct {
	Seq int `json:"seq"`
}

// grantsResponse always carries the sequence the bridge is at, even with
// nothing new to say. That is how a plugin discovers it is ahead, which only
// happens when the run restarted under it and it has to resync.
//
// Grants is never null on the wire. SourcePawn reads a JSON null where it
// expects an array and errors out mid-callback, which stops the poll loop for
// the rest of the map.
type grantsResponse struct {
	Seq    int           `json:"seq"`
	Grants []state.Grant `json:"grants"`
}

// sayRequest is a player talking to the multiworld; a leading ! is a command there.
type sayRequest struct {
	Text string `json:"text"`
}

type messagesResponse struct {
	Seq      int            `json:"seq"`
	Messages []chat.Message `json:"messages"`
}

// waveDrift is a mission whose wave count in the tables is not the one the game
// reported. It is served rather than only logged, because a wrong count on the
// goal mission makes a seed unwinnable and nothing else would ever say so.
type waveDrift struct {
	PopFile  string `json:"popfile"`
	Tables   int    `json:"tables"`
	Observed int    `json:"observed"`
}

// healthResponse is the operator's one window into the bridge.
//
// The fields are spelled out rather than embedded from apclient and state. Two
// embedded structs that ever grow the same json tag lose both fields silently,
// with nothing to fail a build or a test.
type healthResponse struct {
	APIVersion int `json:"api_version"`

	Connected bool     `json:"connected"`
	Slot      string   `json:"slot"`
	Missions  []string `json:"missions"`
	LastError string   `json:"last_error,omitempty"`

	Seed        string     `json:"seed"`
	Checks      int        `json:"checks"`
	Items       int        `json:"items"`
	AckedSeq    int        `json:"acked_seq"`
	GoalSent    bool       `json:"goal_sent"`
	LastCheck   string     `json:"last_check,omitempty"`
	LastCheckAt *time.Time `json:"last_check_at,omitempty"`

	WaveDrift []waveDrift `json:"wave_drift,omitempty"`
}

// Server serves the plugin.
type Server struct {
	store       *state.Store
	client      *apclient.Client
	chat        *chat.Log
	pollTimeout time.Duration
	logger      *slog.Logger

	// drift is what the game said about missions the tables disagree with. It
	// is observation, not state: it belongs to this process, not to the run.
	driftMu sync.Mutex
	drift   map[string]int
}

func New(
	store *state.Store, client *apclient.Client, messages *chat.Log,
	pollTimeout time.Duration, logger *slog.Logger,
) *Server {
	return &Server{
		store:       store,
		client:      client,
		chat:        messages,
		pollTimeout: pollTimeout,
		logger:      logger,
		drift:       make(map[string]int),
	}
}

// Handler wires the routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /objective", s.postObjective)
	mux.HandleFunc("GET /unlocks", s.getUnlocks)
	mux.HandleFunc("GET /grants", s.getGrants)
	mux.HandleFunc("POST /grants/ack", s.postGrantsAck)
	mux.HandleFunc("GET /messages", s.getMessages)
	mux.HandleFunc("POST /say", s.postSay)
	mux.HandleFunc("GET /healthz", s.getHealth)
	return mux
}

// postObjective records a check and answers 204 once it is on disk, before
// anything goes upstream: the plugin is free the moment the check is durable,
// so Archipelago can be down for an hour without costing a wave.
func (s *Server) postObjective(w http.ResponseWriter, r *http.Request) {
	var request objectiveRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&request); err != nil {
		http.Error(w, "cannot read the body", http.StatusBadRequest)
		return
	}
	kind, known := objectiveKind(request.Kind)
	if !known {
		http.Error(w, "unknown objective kind "+request.Kind, http.StatusBadRequest)
		return
	}
	location, resolved := gamedata.LocationByObjective(kind, request.PopFile, request.Wave)
	if !resolved {
		http.Error(w, "no such objective", http.StatusBadRequest)
		return
	}
	s.noteWaveDrift(r.Context(), request.PopFile, request.WavesTotal)

	fresh, err := s.store.AddCheck(location.ID)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "cannot record a check",
			"location", location.Name, "error", err)
		http.Error(w, "cannot record the check", http.StatusInternalServerError)
		return
	}
	if fresh {
		s.logger.InfoContext(r.Context(), "check recorded", "location", location.Name)
	}
	w.WriteHeader(http.StatusNoContent)
}

// noteWaveDrift records the game disagreeing with the tables about a mission's
// length, and says it once per mission rather than once per wave.
//
// It never refuses the check. The wave the team just cleared happened, and the
// table being wrong is the operator's problem to fix between runs, not a reason
// to throw away ten minutes of play.
func (s *Server) noteWaveDrift(ctx context.Context, popFile string, observed int) {
	// Garbage is dropped, a disagreement is not. The bound is what an unsigned
	// byte can hold rather than what the id scheme allows: a mission the game
	// says is longer than gamedata can even number is the single most useful
	// thing this could ever report.
	if observed < 1 || observed > wavesObservedMax {
		return
	}
	mission, known := gamedata.MissionByPopFile(popFile)
	if !known || int(mission.Waves) == observed {
		return
	}
	s.driftMu.Lock()
	_, said := s.drift[popFile]
	s.drift[popFile] = observed
	s.driftMu.Unlock()
	if said {
		return
	}
	s.logger.ErrorContext(ctx,
		"the game says this mission is not the length the tables say, so its checks are wrong",
		"mission", popFile, "tables", mission.Waves, "game", observed)
}

func (s *Server) waveDrift() []waveDrift {
	s.driftMu.Lock()
	defer s.driftMu.Unlock()
	drifted := make([]waveDrift, 0, len(s.drift))
	for popFile, observed := range s.drift {
		tables, known := gamedata.MissionByPopFile(popFile)
		if !known {
			continue
		}
		drifted = append(drifted, waveDrift{
			PopFile:  popFile,
			Tables:   int(tables.Waves),
			Observed: observed,
		})
	}
	slices.SortFunc(drifted, func(a, b waveDrift) int { return strings.Compare(a.PopFile, b.PopFile) })
	return drifted
}

// getUnlocks serves everything that should be true right now. The plugin asks
// on load and on every map change rather than remembering it.
func (s *Server) getUnlocks(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.logger, s.store.Unlocks())
}

// getGrants long-polls: the plugin passes the sequence it last applied and the
// request is held open until there is something past it, so nothing has to
// connect into srcds.
func (s *Server) getGrants(w http.ResponseWriter, r *http.Request) {
	since, err := strconv.Atoi(r.URL.Query().Get("since"))
	if err != nil || since < 0 {
		http.Error(w, "since must be a sequence number", http.StatusBadRequest)
		return
	}

	timeout := time.NewTimer(s.pollTimeout)
	defer timeout.Stop()

	// A check is a change too, so a wake-up with no new grant goes back to waiting.
	for {
		changed := s.store.Watch()
		grants, latest := s.store.GrantsSince(since)
		if len(grants) > 0 || since > latest {
			writeJSON(w, s.logger, grantsResponse{Seq: latest, Grants: grants})
			return
		}
		select {
		case <-changed:
		case <-timeout.C:
			writeJSON(w, s.logger, grantsResponse{Seq: latest, Grants: []state.Grant{}})
			return
		case <-r.Context().Done():
			return
		}
	}
}

// postGrantsAck is the plugin saying it applied everything up to a sequence.
//
// Only effects need it. A class granted twice is still one class, but credits
// paid twice are credits nobody earned, and a plugin reload asks from sequence
// zero. Recording the acknowledgement here rather than in the plugin is what
// makes it survive the reload: the plugin holds no state on purpose.
func (s *Server) postGrantsAck(w http.ResponseWriter, r *http.Request) {
	var request ackRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&request); err != nil {
		http.Error(w, "cannot read the body", http.StatusBadRequest)
		return
	}
	if request.Seq < 0 {
		http.Error(w, "seq must be a sequence number", http.StatusBadRequest)
		return
	}
	if err := s.store.Ack(request.Seq); err != nil {
		s.logger.ErrorContext(r.Context(), "cannot record an acknowledgement",
			"seq", request.Seq, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// getMessages long-polls the multiworld's chat. A negative sequence asks only
// where the conversation is, so a server joining late does not dump the backlog
// into everyone's chat.
func (s *Server) getMessages(w http.ResponseWriter, r *http.Request) {
	since, err := strconv.Atoi(r.URL.Query().Get("since"))
	if err != nil {
		http.Error(w, "since must be a sequence number", http.StatusBadRequest)
		return
	}

	timeout := time.NewTimer(s.pollTimeout)
	defer timeout.Stop()
	for {
		changed := s.chat.Watch()
		messages, latest := s.chat.Since(since)
		if messages == nil {
			messages = []chat.Message{}
		}
		if len(messages) > 0 || since < 0 {
			writeJSON(w, s.logger, messagesResponse{Seq: latest, Messages: messages})
			return
		}
		select {
		case <-changed:
		case <-timeout.C:
			writeJSON(w, s.logger, messagesResponse{Seq: latest, Messages: []chat.Message{}})
			return
		case <-r.Context().Done():
			return
		}
	}
}

// postSay passes a line to the multiworld. Unlike a check it is refused rather
// than queued, and the player is standing right there to be told.
func (s *Server) postSay(w http.ResponseWriter, r *http.Request) {
	var request sayRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&request); err != nil {
		http.Error(w, "cannot read the body", http.StatusBadRequest)
		return
	}
	if request.Text == "" {
		http.Error(w, "nothing to say", http.StatusBadRequest)
		return
	}
	if err := s.client.Say(r.Context(), request.Text); err != nil {
		http.Error(w, err.Error(), sayStatus(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// sayStatus separates the three ways a line is refused, because the plugin says
// something different to the player for each one.
func sayStatus(err error) int {
	switch {
	case errors.Is(err, apclient.ErrCommandRefused):
		return http.StatusForbidden
	case errors.Is(err, apclient.ErrSaidTooMuch):
		return http.StatusTooManyRequests
	case errors.Is(err, apclient.ErrSayTooLong):
		return http.StatusRequestEntityTooLarge
	default:
		return http.StatusServiceUnavailable
	}
}

func (s *Server) getHealth(w http.ResponseWriter, r *http.Request) {
	session, run := s.client.Health(), s.store.Stats()
	writeJSON(w, s.logger, healthResponse{
		APIVersion: APIVersion,

		Connected: session.Connected,
		Slot:      session.Slot,
		Missions:  session.Missions,
		LastError: session.LastError,

		Seed:        run.Seed,
		Checks:      run.Checks,
		Items:       run.Items,
		AckedSeq:    run.AckedSeq,
		GoalSent:    run.GoalSent,
		LastCheck:   run.LastCheck,
		LastCheckAt: run.LastCheckAt,

		WaveDrift: s.waveDrift(),
	})
}

func objectiveKind(key string) (gamedata.ObjectiveKind, bool) {
	for _, kind := range []gamedata.ObjectiveKind{
		gamedata.ObjectiveWaveCleared,
		gamedata.ObjectiveMissionCleared,
	} {
		if kind.Key() == key {
			return kind, true
		}
	}
	return 0, false
}

func writeJSON(w http.ResponseWriter, logger *slog.Logger, body any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(body); err != nil {
		logger.Error("cannot write a response", "error", err)
	}
}

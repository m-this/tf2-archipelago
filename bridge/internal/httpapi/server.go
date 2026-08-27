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

	"github.com/m-this/tf2-archipelago/bridge/internal/apclient"
	"github.com/m-this/tf2-archipelago/bridge/internal/chat"
	"github.com/m-this/tf2-archipelago/bridge/internal/deathlink"
	"github.com/m-this/tf2-archipelago/bridge/internal/state"
	"github.com/m-this/tf2-archipelago/gamedata"
)

// APIVersion is the contract with the plugin. The plugin reads it at startup
// and says so in chat when it does not match: the two ship in one compose file,
// so a mismatch means one image was updated and the other was not.
const APIVersion = 3

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

// deathRequest is the plugin reporting the team lost a wave, in the game's
// terms. The bridge words the cause: the plugin does not know what a slot is.
type deathRequest struct {
	PopFile string `json:"popfile"`
	Wave    uint8  `json:"wave"`
}

// deathsResponse carries whether the seed has DeathLink on at all, so the
// plugin can say so without asking anything else. Deaths is never null on the
// wire, for the same reason grants is not.
type deathsResponse struct {
	Seq       int               `json:"seq"`
	DeathLink bool              `json:"death_link"`
	Deaths    []deathlink.Death `json:"deaths"`
}

// mission is one mission of the run, in the terms a mission switcher needs: a
// popfile to load, the map that popfile runs on, and something to show a
// player. The plugin cannot work the map out for itself, because a popfile
// name does not contain one that can be relied on (mvm_ghost_town_666 runs on
// mvm_ghost_town), and gamedata is where that fact already lives.
type mission struct {
	PopFile  string `json:"popfile"`
	Name     string `json:"name"`
	Map      string `json:"map"`
	Waves    int    `json:"waves"`
	Unlocked bool   `json:"unlocked"`

	// Cleared is the mission clear check being on the bridge's disk. The plugin
	// chains from a cleared mission to the next unlocked one that is not, so
	// it has to know both.
	Cleared bool `json:"cleared"`

	/* Played is this server having actually cleared it.
	 *
	 * A check reaches the disk when anybody in the room sends it, and another
	 * world running !collect on its goal sends every one it still holds. That
	 * marked missions here as cleared that nobody on this server had played,
	 * and the run list stopped saying what this player had done. Reported by
	 * Peppy after apw-o75 pointed the goal at played and left the display
	 * reading checks.
	 *
	 * Cleared stays what it was, because the plugin chains on it and the room's
	 * answer is the right one for "is this check already spent". The two are
	 * shown apart rather than merged. */
	Played bool `json:"played"`
}

type missionsResponse struct {
	Missions []mission `json:"missions"`
}

// waveDrift is a mission whose wave count in the tables is not the one the game
// reported. It is served rather than only logged, because a wrong count on the
// goal mission makes a seed unwinnable and nothing else would ever say so.
type waveDrift struct {
	PopFile  string `json:"popfile"`
	Tables   int    `json:"tables"`
	Observed int    `json:"observed"`
}

// waveFailure is a wave the team lost, and how often. Valve tunes every wave
// for six defenders and the bots exist so fewer can win, but nobody has a
// number for how well they do. This is that number: the waves an evening
// actually lost, per mission, so tuning the team size or the bots argues with
// a record rather than with a memory.
//
// Observation, like waveDrift: it belongs to this process, not to the run. A
// restart starts the count again, and nothing about the run depends on it.
type waveFailure struct {
	PopFile string `json:"popfile"`
	Wave    int    `json:"wave"`
	Lost    int    `json:"lost"`
}

// healthResponse is the operator's one window into the bridge.
//
// The fields are spelled out rather than embedded from apclient and state. Two
// embedded structs that ever grow the same json tag lose both fields silently,
// with nothing to fail a build or a test.
type healthResponse struct {
	APIVersion int `json:"api_version"`

	Connected    bool     `json:"connected"`
	Slot         string   `json:"slot"`
	Missions     []string `json:"missions"`
	StartMission string   `json:"start_mission,omitempty"`
	DeathLink    bool     `json:"death_link"`
	LastError    string   `json:"last_error,omitempty"`

	Seed        string     `json:"seed"`
	Checks      int        `json:"checks"`
	Items       int        `json:"items"`
	AckedSeq    int        `json:"acked_seq"`
	GoalSent    bool       `json:"goal_sent"`
	LastCheck   string     `json:"last_check,omitempty"`
	LastCheckAt *time.Time `json:"last_check_at,omitempty"`

	WaveDrift    []waveDrift   `json:"wave_drift,omitempty"`
	WaveFailures []waveFailure `json:"wave_failures,omitempty"`
}

// Server serves the plugin.
type Server struct {
	store       *state.Store
	client      *apclient.Client
	chat        *chat.Log
	deaths      *deathlink.Feed
	pollTimeout time.Duration
	logger      *slog.Logger

	// drift is what the game said about missions the tables disagree with. It
	// is observation, not state: it belongs to this process, not to the run.
	driftMu sync.Mutex
	drift   map[string]int

	// failures counts the waves the team lost, keyed by mission and wave. Same
	// nature as drift, and the reason it exists is that a lost wave used to
	// leave no trace at all when the seed had DeathLink off.
	failMu   sync.Mutex
	failures map[waveKey]int
}

// waveKey is one wave of one mission, the key failures counts against.
type waveKey struct {
	PopFile string
	Wave    int
}

func New(
	store *state.Store, client *apclient.Client, messages *chat.Log, deaths *deathlink.Feed,
	pollTimeout time.Duration, logger *slog.Logger,
) *Server {
	return &Server{
		store:       store,
		client:      client,
		chat:        messages,
		deaths:      deaths,
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
	mux.HandleFunc("GET /missions", s.getMissions)
	mux.HandleFunc("GET /grants", s.getGrants)
	mux.HandleFunc("POST /grants/ack", s.postGrantsAck)
	mux.HandleFunc("GET /messages", s.getMessages)
	mux.HandleFunc("POST /say", s.postSay)
	mux.HandleFunc("POST /death", s.postDeath)
	mux.HandleFunc("GET /deaths", s.getDeaths)
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

// recordFailure counts one lost wave. A mission the tables do not know is
// still counted: it is the operator's record, not a check, so refusing it
// would hide exactly the run that most needs looking at.
func (s *Server) recordFailure(popFile string, wave int) {
	if popFile == "" {
		return
	}
	s.failMu.Lock()
	defer s.failMu.Unlock()
	if s.failures == nil {
		s.failures = make(map[waveKey]int)
	}
	s.failures[waveKey{popFile, wave}]++
}

// waveFailures is the record, worst first, so the wave that cost the evening
// is the first line of the answer.
func (s *Server) waveFailures() []waveFailure {
	s.failMu.Lock()
	defer s.failMu.Unlock()
	lost := make([]waveFailure, 0, len(s.failures))
	for key, count := range s.failures {
		lost = append(lost, waveFailure{PopFile: key.PopFile, Wave: key.Wave, Lost: count})
	}
	slices.SortFunc(lost, func(a, b waveFailure) int {
		if a.Lost != b.Lost {
			return b.Lost - a.Lost
		}
		if a.PopFile != b.PopFile {
			return strings.Compare(a.PopFile, b.PopFile)
		}
		return a.Wave - b.Wave
	})
	return lost
}

// getUnlocks serves everything that should be true right now. The plugin asks
// on load and on every map change rather than remembering it.
func (s *Server) getUnlocks(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.logger, s.store.Unlocks())
}

// getMissions serves the run's missions, in the order the seed drew them, each
// with the map its popfile runs on. It is what a mission switcher needs and
// what the unlock set deliberately does not carry: the unlock set answers "may
// we play this", this answers "what is there and how do I load it".
func (s *Server) getMissions(w http.ResponseWriter, r *http.Request) {
	health := s.client.Health()
	missions, unknown := missionsFor(
		health.Missions,
		s.store.Unlocks().Of(gamedata.ItemMissionTicket),
		s.store.Checks(),
		s.store.Played(),
		health.MissionTicketImportance == "useful",
	)
	for _, popFile := range unknown {
		s.logger.WarnContext(r.Context(), "the seed holds a mission the tables do not",
			"mission", popFile)
	}
	writeJSON(w, s.logger, missionsResponse{Missions: missions})
}

// missionsFor turns the popfiles the seed drew into what a switcher can act on,
// and reports the ones the tables do not know. A seed from a newer gamedata is
// the only way that happens, and skipping such a mission beats serving a name
// and a map this binary would be guessing at.
func missionsFor(drawn, unlocked []string, checks, own []int64, unlockAll bool) ([]mission, []string) {
	missions := make([]mission, 0, len(drawn))
	var unknown []string
	for _, popFile := range drawn {
		known, ok := gamedata.MissionByPopFile(popFile)
		if !ok {
			unknown = append(unknown, popFile)
			continue
		}
		played, _ := gamedata.MapByID(known.Map)
		missions = append(missions, mission{
			PopFile:  known.PopFile,
			Name:     known.Name,
			Map:      played.Name,
			Waves:    int(known.Waves),
			Unlocked: unlockAll || slices.Contains(unlocked, known.PopFile),
			Cleared:  slices.Contains(checks, known.ClearLocationID()),
			Played:   slices.Contains(own, known.ClearLocationID()),
		})
	}
	return missions, unknown
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

// postDeath is the team losing a wave, passed on to every DeathLink player in
// the multiworld. Never queued: like a chat line, one that lands after the
// reconnect is a different death. A seed without DeathLink answers as if it had
// been sent, because the plugin has nothing to do differently either way.
func (s *Server) postDeath(w http.ResponseWriter, r *http.Request) {
	var request deathRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&request); err != nil {
		http.Error(w, "cannot read the body", http.StatusBadRequest)
		return
	}
	// Counted before it is forwarded, and whatever the forward does with it: a
	// seed with DeathLink off drops the death, and the wave was still lost.
	s.recordFailure(request.PopFile, int(request.Wave))

	err := s.client.Die(r.Context(), deathCause(s.client.Health().Slot, request))
	switch {
	case errors.Is(err, apclient.ErrDeathLinkOff), errors.Is(err, apclient.ErrDiedTooMuch):
		s.logger.DebugContext(r.Context(), "death not sent", "reason", err)
	case errors.Is(err, apclient.ErrNotConnected):
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	case err != nil:
		s.logger.ErrorContext(r.Context(), "cannot send a death", "error", err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// deathCause words the death for the other players, who know nothing about
// Mann vs Machine: the mission's name from the tables rather than its popfile.
func deathCause(slot string, request deathRequest) string {
	name := request.PopFile
	if mission, known := gamedata.MissionByPopFile(request.PopFile); known {
		name = mission.Name
	}
	if request.Wave == 0 {
		return slot + " lost a wave of " + name
	}
	return slot + " lost wave " + strconv.Itoa(int(request.Wave)) + " of " + name
}

// getDeaths long-polls the multiworld's deaths, the way getMessages does chat.
// A negative sequence asks only where the feed is: a death from before the
// plugin was listening is not one it should apply.
func (s *Server) getDeaths(w http.ResponseWriter, r *http.Request) {
	since, err := strconv.Atoi(r.URL.Query().Get("since"))
	if err != nil {
		http.Error(w, "since must be a sequence number", http.StatusBadRequest)
		return
	}

	timeout := time.NewTimer(s.pollTimeout)
	defer timeout.Stop()
	for {
		changed := s.deaths.Watch()
		deaths, latest := s.deaths.Since(since)
		if deaths == nil {
			deaths = []deathlink.Death{}
		}
		if len(deaths) > 0 || since < 0 {
			writeJSON(w, s.logger, s.deathsResponse(latest, deaths))
			return
		}
		select {
		case <-changed:
		case <-timeout.C:
			writeJSON(w, s.logger, s.deathsResponse(latest, deaths))
			return
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) deathsResponse(latest int, deaths []deathlink.Death) deathsResponse {
	return deathsResponse{Seq: latest, DeathLink: s.client.Health().DeathLink, Deaths: deaths}
}

func (s *Server) getHealth(w http.ResponseWriter, r *http.Request) {
	session, run := s.client.Health(), s.store.Stats()
	writeJSON(w, s.logger, healthResponse{
		APIVersion: APIVersion,

		Connected:    session.Connected,
		Slot:         session.Slot,
		Missions:     session.Missions,
		StartMission: session.StartMission,
		DeathLink:    session.DeathLink,
		LastError:    session.LastError,

		Seed:        run.Seed,
		Checks:      run.Checks,
		Items:       run.Items,
		AckedSeq:    run.AckedSeq,
		GoalSent:    run.GoalSent,
		LastCheck:   run.LastCheck,
		LastCheckAt: run.LastCheckAt,

		WaveDrift:    s.waveDrift(),
		WaveFailures: s.waveFailures(),
	})
}

func objectiveKind(key string) (gamedata.ObjectiveKind, bool) {
	for _, kind := range gamedata.ObjectiveKinds {
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

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
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"git-ssh.croque.top/mathis/tf2-archipelago/bridge/internal/apclient"
	"git-ssh.croque.top/mathis/tf2-archipelago/bridge/internal/chat"
	"git-ssh.croque.top/mathis/tf2-archipelago/bridge/internal/state"
	"git-ssh.croque.top/mathis/tf2-archipelago/gamedata"
)

// objectiveRequest is what the plugin posts. Wave is ignored for a mission clear.
type objectiveRequest struct {
	Kind    string `json:"kind"`
	PopFile string `json:"popfile"`
	Wave    uint8  `json:"wave"`
}

// grantsResponse always carries the sequence the bridge is at, even with
// nothing new to say. That is how a plugin discovers it is ahead, which only
// happens when the run restarted under it and it has to resync.
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

// Server serves the plugin.
type Server struct {
	store       *state.Store
	client      *apclient.Client
	chat        *chat.Log
	pollTimeout time.Duration
	logger      *slog.Logger
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
	}
}

// Handler wires the routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /objective", s.postObjective)
	mux.HandleFunc("GET /unlocks", s.getUnlocks)
	mux.HandleFunc("GET /grants", s.getGrants)
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
		grants := s.store.GrantsSince(since)
		latest := s.store.Unlocks().Seq
		if len(grants) > 0 || since > latest {
			writeJSON(w, s.logger, grantsResponse{Seq: latest, Grants: grants})
			return
		}
		select {
		case <-changed:
		case <-timeout.C:
			writeJSON(w, s.logger, grantsResponse{Seq: latest})
			return
		case <-r.Context().Done():
			return
		}
	}
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
		if len(messages) > 0 || since < 0 {
			writeJSON(w, s.logger, messagesResponse{Seq: latest, Messages: messages})
			return
		}
		select {
		case <-changed:
		case <-timeout.C:
			writeJSON(w, s.logger, messagesResponse{Seq: latest, Messages: nil})
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
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.logger, s.client.Health())
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

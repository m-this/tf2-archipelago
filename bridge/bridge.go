// Package bridge runs the Archipelago session for one Team Fortress 2 server
// and serves the SourceMod plugin over loopback HTTP.
//
// This is the importable form of what bridge/cmd/bridge runs as a binary. The
// launcher embeds it so the bridge runs in-process alongside the srcds
// subprocess it supervises. The binary stays as a thin wrapper for the
// container image and for anyone running it standalone.
//
// See docs/en/adr/0002 for the invariants and the split.
package bridge

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/m-this/tf2-archipelago/bridge/config"
	"github.com/m-this/tf2-archipelago/bridge/internal/apclient"
	"github.com/m-this/tf2-archipelago/bridge/internal/chat"
	"github.com/m-this/tf2-archipelago/bridge/internal/httpapi"
	"github.com/m-this/tf2-archipelago/bridge/internal/state"
	"github.com/m-this/tf2-archipelago/gamedata"
)

const (
	// ShutdownGrace is how long in-flight long-polls get to finish after the
	// context is cancelled.
	ShutdownGrace = 5 * time.Second

	// ChatHistory is how much a reconnecting plugin can still catch up on; chat
	// is not state.
	ChatHistory = 200

	// HealthTimeout bounds the health check; it talks to loopback, so slower
	// means a stall.
	HealthTimeout = 3 * time.Second
)

// CheckHealth is what the container health check runs: the image has no shell
// and no curl, so the binary answers for itself. It reports the process being
// up, not the Archipelago session, which may legitimately be down.
func CheckHealth(listen string) error {
	ctx, cancel := context.WithTimeout(context.Background(), HealthTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+listen+"/healthz", nil)
	if err != nil {
		return err
	}
	response, err := (&http.Client{}).Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("healthz answered %d", response.StatusCode)
	}
	return nil
}

// servers builds the plugin's listener and, when one was asked for, the metrics
// listener. They are separate servers because they are reachable from different
// places and can hold a request open for different reasons: the plugin's long
// poll must not have a write timeout, and the scraper's must.
func servers(cfg config.Config, api *httpapi.Server) (plugin, metrics *http.Server) {
	plugin = &http.Server{
		Addr:              cfg.Listen,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,

		// No write timeout: GET /grants is a long poll and holding it open is the point.
	}
	if cfg.MetricsListen == "" {
		return plugin, nil
	}
	return plugin, &http.Server{
		Addr:              cfg.MetricsListen,
		Handler:           api.MetricsHandler(cfg.GameQueryAddr),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
	}
}

// serveAll starts every listener and reports the first one to stop. Any of them
// stopping ends the process: neither is optional once it has been asked for.
func serveAll(listeners ...*http.Server) <-chan error {
	stopped := make(chan error, len(listeners))
	for _, listener := range listeners {
		go func() {
			err := listener.ListenAndServe()
			if errors.Is(err, http.ErrServerClosed) {
				err = nil
			}
			stopped <- err
		}()
	}
	return stopped
}

// shutdownAll gives every listener the same grace period and keeps the first
// failure. Every one of them is asked to stop even if an earlier one refused.
func shutdownAll(ctx context.Context, listeners ...*http.Server) error {
	var first error
	for _, listener := range listeners {
		if err := listener.Shutdown(ctx); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// Run loads the configuration, opens the state store, starts the Archipelago
// session and the HTTP listeners, and blocks until the context is cancelled or
// something stops. It is what both the bridge binary and the launcher run.
//
// Cancel the context to shut down. The returned error is nil for a clean
// shutdown, non-nil otherwise.
func Run(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	store, err := state.Open(cfg.StatePath)
	if err != nil {
		return err
	}

	messages := chat.New(ChatHistory)

	client := apclient.New(apclient.Options{
		URL:      cfg.ArchipelagoURL,
		SlotName: cfg.SlotName,
		Password: cfg.Password,
		Store:    store,
		Chat:     messages,
		Logger:   logger,
	})
	api := httpapi.New(store, client, messages, cfg.PollTimeout, logger)
	server, metrics := servers(cfg, api)

	logger.InfoContext(ctx, "bridge starting",
		"game", gamedata.GameName,
		"archipelago", cfg.ArchipelagoURL,
		"slot", cfg.SlotName,
		"listen", cfg.Listen,
		"metrics", cfg.MetricsListen,
		"state", cfg.StatePath,
	)

	listeners := []*http.Server{server}
	if metrics != nil {
		listeners = append(listeners, metrics)
	}
	served := serveAll(listeners...)

	sessionEnded := make(chan error, 1)
	go func() { sessionEnded <- client.Run(ctx) }()

	var runErr error
	select {
	case <-ctx.Done():
		logger.InfoContext(ctx, "bridge stopping")
	case runErr = <-served:
	case runErr = <-sessionEnded:
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), ShutdownGrace)
	defer cancel()
	// shutdownCtx derives from Background on purpose: ctx is already cancelled
	// when we reach here, so deriving from it would give an already-done context
	// and the listeners would get no grace period.
	if closeErr := shutdownAll(shutdownCtx, listeners...); closeErr != nil && runErr == nil { //nolint:contextcheck // see above
		runErr = closeErr
	}
	return runErr
}

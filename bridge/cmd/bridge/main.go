// Command bridge holds the Archipelago session for one Team Fortress 2 server
// and serves the SourceMod plugin over loopback HTTP.
//
// It is the only component that knows the Archipelago protocol and the only
// one that knows the id mapping. See docs/en/adr/0002.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/m-this/tf2-archipelago/bridge/internal/apclient"
	"github.com/m-this/tf2-archipelago/bridge/internal/chat"
	"github.com/m-this/tf2-archipelago/bridge/internal/config"
	"github.com/m-this/tf2-archipelago/bridge/internal/httpapi"
	"github.com/m-this/tf2-archipelago/bridge/internal/state"
	"github.com/m-this/tf2-archipelago/gamedata"
)

const (
	// shutdownGrace is how long in-flight long-polls get to finish after a signal.
	shutdownGrace = 5 * time.Second

	// chatHistory is how much a reconnecting plugin can still catch up on; chat is not state.
	chatHistory = 200

	// healthTimeout bounds the health check; it talks to loopback, so slower means a stall.
	healthTimeout = 3 * time.Second
)

// checkHealth is what the container health check runs: the image has no shell
// and no curl, so the binary answers for itself. It reports the process being
// up, not the Archipelago session, which may legitimately be down.
func checkHealth(listen string) error {
	ctx, cancel := context.WithTimeout(context.Background(), healthTimeout)
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

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := run(logger); err != nil {
		logger.Error("bridge stopped", "error", err)
		os.Exit(1)
	}
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

func run(logger *slog.Logger) error {
	health := flag.Bool("health", false,
		"ask the running bridge for its health and exit; this is the container health check")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if *health {
		return checkHealth(cfg.Listen)
	}
	store, err := state.Open(cfg.StatePath)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	messages := chat.New(chatHistory)

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

	logger.Info("bridge starting",
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

	select {
	case <-ctx.Done():
		logger.Info("signal received, stopping")
	case err = <-served:
	case err = <-sessionEnded:
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
	defer cancel()
	if closeErr := shutdownAll(shutdownCtx, listeners...); closeErr != nil && err == nil {
		err = closeErr
	}
	return err
}

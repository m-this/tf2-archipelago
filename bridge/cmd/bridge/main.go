// Command bridge holds the Archipelago session for one Team Fortress 2 server
// and serves the SourceMod plugin over loopback HTTP.
//
// It is the only component that knows the Archipelago protocol and the only
// one that knows the id mapping. See docs/adr/0002.
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

	"git-ssh.croque.top/mathis/tf2-archipelago/bridge/internal/apclient"
	"git-ssh.croque.top/mathis/tf2-archipelago/bridge/internal/chat"
	"git-ssh.croque.top/mathis/tf2-archipelago/bridge/internal/config"
	"git-ssh.croque.top/mathis/tf2-archipelago/bridge/internal/httpapi"
	"git-ssh.croque.top/mathis/tf2-archipelago/bridge/internal/state"
	"git-ssh.croque.top/mathis/tf2-archipelago/gamedata"
)

const (
	// shutdownGrace is how long in-flight long-polls get to finish after a
	// signal.
	shutdownGrace = 5 * time.Second

	// chatHistory is how many multiworld messages are kept for a plugin that
	// reconnects. Chat is not state: what falls off the end is gone.
	chatHistory = 200

	// healthTimeout bounds the health check. It talks to loopback, so anything
	// slower than this is a bridge that has stopped answering.
	healthTimeout = 3 * time.Second
)

// checkHealth is what the container health check runs. The image has no shell
// and no curl, so the binary answers for itself.
//
// It reports the bridge process being up, not the Archipelago session: a
// bridge with no multiworld to talk to is still doing its job, queueing checks
// until one comes back.
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

	// The multiworld says more in an evening than anyone reads, so the log
	// keeps the last few hundred lines and no more.
	messages := chat.New(chatHistory)

	client := apclient.New(apclient.Options{
		URL:      cfg.ArchipelagoURL,
		SlotName: cfg.SlotName,
		Password: cfg.Password,
		Store:    store,
		Chat:     messages,
		Logger:   logger,
	})
	server := &http.Server{
		Addr:              cfg.Listen,
		Handler:           httpapi.New(store, client, messages, cfg.PollTimeout, logger).Handler(),
		ReadHeaderTimeout: 5 * time.Second,

		// No write timeout: GET /grants is a long poll and holding it open is
		// the point.
	}

	logger.Info("bridge starting",
		"game", gamedata.GameName,
		"archipelago", cfg.ArchipelagoURL,
		"slot", cfg.SlotName,
		"listen", cfg.Listen,
		"state", cfg.StatePath,
	)

	served := make(chan error, 1)
	go func() {
		err := server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		served <- err
	}()

	sessionEnded := make(chan error, 1)
	go func() { sessionEnded <- client.Run(ctx) }()

	select {
	case <-ctx.Done():
		logger.Info("signal received, shutting down")
	case err = <-served:
	case err = <-sessionEnded:
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
	defer cancel()
	if closeErr := server.Shutdown(shutdownCtx); closeErr != nil && err == nil {
		err = closeErr
	}
	return err
}

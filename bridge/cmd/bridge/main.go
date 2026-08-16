// Command bridge holds the Archipelago session for one Team Fortress 2 server
// and serves the SourceMod plugin over loopback HTTP.
//
// It is the only component that knows the Archipelago protocol and the only
// one that knows the id mapping. See docs/adr/0002.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git-ssh.croque.top/mathis/tf2-archipelago/bridge/internal/apclient"
	"git-ssh.croque.top/mathis/tf2-archipelago/bridge/internal/config"
	"git-ssh.croque.top/mathis/tf2-archipelago/bridge/internal/httpapi"
	"git-ssh.croque.top/mathis/tf2-archipelago/bridge/internal/state"
	"git-ssh.croque.top/mathis/tf2-archipelago/gamedata"
)

// shutdownGrace is how long in-flight long-polls get to finish after a signal.
const shutdownGrace = 5 * time.Second

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := run(logger); err != nil {
		logger.Error("bridge stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	store, err := state.Open(cfg.StatePath)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	client := apclient.New(apclient.Options{
		URL:      cfg.ArchipelagoURL,
		SlotName: cfg.SlotName,
		Password: cfg.Password,
		Store:    store,
		Logger:   logger,
	})
	server := &http.Server{
		Addr:              cfg.Listen,
		Handler:           httpapi.New(store, client, cfg.PollTimeout, logger).Handler(),
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

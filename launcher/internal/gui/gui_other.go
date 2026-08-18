//go:build !windows

// Package gui is the launcher's window. It exists on Windows only: walk is a
// Win32 binding. Everywhere else the launcher is the console flow, which is
// also what the Docker stack uses.
package gui

import (
	"errors"
	"log/slog"

	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

// Available reports whether this build has a window.
func Available() bool { return false }

// Run always fails here. The caller falls back to the console flow.
func Run(settings.Settings, *slog.Logger) error {
	return errors.New("the window is Windows only")
}

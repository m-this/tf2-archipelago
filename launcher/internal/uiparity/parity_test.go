// Package uiparity holds one test: the window and the terminal offer the same
// settings.
//
// It reads the two sources rather than the two interfaces, because the window
// only builds on Windows and this has to fail on the machine that broke it.
package uiparity

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

/*
	Fields the window writes through a helper rather than by name.

botTeamFrom turns the seat and class widgets into one BotTeam, and
settings.WithBotTeam puts it on the settings, so no line in the window says
next.SrcdsBotTeamComp. The terminal writes them one at a time. Listed here with
that reason rather than dropped, so a field that goes missing for any other
reason still fails.
*/
var writtenThroughWithBotTeam = []string{
	"SrcdsBotLoadouts",
	"SrcdsBotSeatLoadouts",
	"SrcdsBotTeamComp",
}

var (
	guiWrites = regexp.MustCompile(`next\.([A-Za-z0-9]+) =`)
	tuiWrites = regexp.MustCompile(`f\.edited\.([A-Za-z0-9]+)\s*(?:=[^=]|\z)`)
	tuiTakes  = regexp.MustCompile(`&f\.edited\.([A-Za-z0-9]+)`)
)

func TestBothInterfacesWriteTheSameSettings(t *testing.T) {
	gui := fieldsIn(t, guiWrites, "../gui/settings_windows.go")
	gui = append(gui, writtenThroughWithBotTeam...)
	slices.Sort(gui)
	gui = slices.Compact(gui)

	tui := fieldsIn(t, tuiWrites, tuiSources(t)...)
	tui = append(tui, fieldsIn(t, tuiTakes, tuiSources(t)...)...)
	slices.Sort(tui)
	tui = slices.Compact(tui)

	for _, name := range gui {
		if !slices.Contains(tui, name) {
			t.Errorf("the window writes %s and the terminal does not", name)
		}
	}
	for _, name := range tui {
		if !slices.Contains(gui, name) {
			t.Errorf("the terminal writes %s and the window does not", name)
		}
	}
}

/*
The tabs are the same in both, name for name.

They differed once: the terminal kept a Loadouts tab that the window nests under
Bots, and this test carried an exception for it. An exception in a parity test
is a place the two are allowed to drift, so the tabs were made to match and the
exception went.
*/
func TestBothInterfacesShowTheSameTabs(t *testing.T) {
	/* The window builds most tabs in one literal and appends Networking after
	   it, so file order is not tab order. Both lists are sorted and compared as
	   sets: a tab in one and not the other is the failure worth catching, and
	   the order is a layout choice each interface makes. */
	gui := between(t, "../gui/settings_windows.go", regexp.MustCompile(`(?m)^\t+Title: +"([A-Za-z ]+)",\n(?:\t+//.*\n)*\t+Layout:`))
	tui := between(t, "../tui/settings.go", regexp.MustCompile(`\{title: "([A-Za-z ]+)"`))

	slices.Sort(gui)
	slices.Sort(tui)

	if !slices.Equal(gui, tui) {
		t.Errorf("tabs differ\n window:   %v\n terminal: %v", gui, tui)
	}
}

func tuiSources(t *testing.T) []string {
	t.Helper()
	found, err := filepath.Glob("../tui/*.go")
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, name := range found {
		if !strings.HasSuffix(name, "_test.go") {
			out = append(out, name)
		}
	}
	return out
}

func fieldsIn(t *testing.T, pattern *regexp.Regexp, paths ...string) []string {
	t.Helper()
	var out []string
	for _, path := range paths {
		for _, match := range pattern.FindAllStringSubmatch(read(t, path), -1) {
			out = append(out, match[1])
		}
	}
	return out
}

func between(t *testing.T, path string, pattern *regexp.Regexp) []string {
	t.Helper()
	var out []string
	for _, match := range pattern.FindAllStringSubmatch(read(t, path), -1) {
		out = append(out, match[1])
	}
	return out
}

func read(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

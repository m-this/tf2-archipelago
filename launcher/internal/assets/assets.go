// Package assets holds the files the launcher embeds in its own binary: the
// compiled plugin, the ripext Windows build, and the config templates. They
// ship inside tf2ap.exe so a download is the whole install for the parts this
// project owns.
//
// The .smx and the ripext zip are gitignored build artefacts: the Makefile
// fetches and copies them into embedded/ before `go build`. The cfg and the
// template are committed. Nothing in this package reads versions.env directly;
// the version strings below are injected via -ldflags from that file at build
// time, so it stays the single source of truth.
package assets

import (
	_ "embed"
	"fmt"
)

//go:embed embedded/tf2_archipelago.smx
var plugin []byte

//go:embed embedded/sm-ripext-windows.zip
var ripextZip []byte

//go:embed embedded/defender-bots-windows.zip
var defenderBotsZip []byte

//go:embed embedded/tf2_archipelago.cfg
var pluginConfig []byte

//go:embed embedded/server.cfg.tmpl
var serverCfgTemplate string

// Version constants. Injected via -ldflags "-X ...". Empty means the binary
// was built without the Makefile (a hand build), which is fine for development
// but the installer refuses to run without them set.
var (
	SourcemodBranch    = ""
	SourcemodVersion   = ""
	MetamodBranch      = ""
	MetamodVersion     = ""
	RipextVersion      = ""
	ArchipelagoVersion = ""
)

// Plugin returns the compiled SourceMod plugin bytecode.
func Plugin() []byte { return plugin }

// RipextZip returns the Windows ripext distribution, unpacked into the game's
// addons tree at install time.
func RipextZip() []byte { return ripextZip }

// DefenderBotsZip returns the MvM defender bot stack for Windows: the four
// plugins, the two extension .dlls, their gamedata and the per-map navigation
// hints, rooted at addons/.
func DefenderBotsZip() []byte { return defenderBotsZip }

// PluginConfig returns tf2_archipelago.cfg, copied verbatim into the server's
// cfg/sourcemod/ directory.
func PluginConfig() []byte { return pluginConfig }

// ServerCfgTemplate returns the server.cfg text/template source. See
// srcdsconfig for the fields it expects.
func ServerCfgTemplate() string { return serverCfgTemplate }

// Versions reports the pinned tool versions, for display and for refusing to
// install when the binary was built without them.
func Versions() map[string]string {
	return map[string]string{
		"sourcemod":   SourcemodVersion,
		"metamod":     MetamodVersion,
		"ripext":      RipextVersion,
		"archipelago": ArchipelagoVersion,
	}
}

// RequireVersions fails when any version string is empty. A binary built by
// hand (go build ./launcher/cmd/tf2ap) has empty strings and could install the
// wrong versions; the Makefile sets them, so the installer gates on this.
func RequireVersions() error {
	for name, value := range Versions() {
		if value == "" {
			return fmt.Errorf("asset version %s is empty: build with `make launcher` so -ldflags injects it from deploy/env/versions.env", name)
		}
	}
	return nil
}

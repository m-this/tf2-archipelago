// Package assets holds the files the launcher embeds in its own binary: the
// compiled plugin, the ripext build for this platform, and the config
// templates. They
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

//go:embed embedded/tf2_archipelago.txt
var pluginGameData []byte

//go:embed embedded/tf2_mvm.apworld
var apworld []byte

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

	// LauncherVersion is this build of the launcher itself: the release it
	// belongs to and the commit it was built from. The commit is the useful
	// half between releases, when several builds carry the same version and
	// the only question anybody has is which one they are running.
	LauncherVersion = ""

	// DefenderbotsVersion is the tag the embedded bot mod was built from. It
	// installs no separate download, so nothing gates on it; it is here because
	// a crash report that does not say which bots were playing cannot be read.
	DefenderbotsVersion = ""
)

// Plugin returns the compiled SourceMod plugin bytecode.
func Plugin() []byte { return plugin }

// PluginGameData returns the signatures used for native TF2 projectile calls.
func PluginGameData() []byte { return pluginGameData }

// RipextZip returns the ripext distribution for this platform, unpacked into
// the game's addons tree at install time.
func RipextZip() []byte { return ripextZip }

// DefenderBotsZip returns the MvM defender bot stack for this platform: the
// four plugins, the two extensions, their gamedata and the per-map navigation
// hints, rooted at addons/.
//
// One platform's binaries per build. SourceMod takes the .so or the .dll by
// platform and ignores the other, so shipping both would be half the bytes of
// the archive wasted in every download.
func DefenderBotsZip() []byte { return defenderBotsZip }

// Apworld returns the world file the Archipelago app generates seeds with. The
// launcher installs it into the app before it runs the generator, so the player
// never has to find the custom_worlds folder.
func Apworld() []byte { return apworld }

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
		"sourcemod":    SourcemodVersion,
		"metamod":      MetamodVersion,
		"ripext":       RipextVersion,
		"archipelago":  ArchipelagoVersion,
		"defenderbots": DefenderbotsVersion,
		"launcher":     LauncherVersion,
	}
}

// Title is what the window calls itself: the name, and the build behind it when
// there is one. A hand build has no version injected and simply says the name.
func Title(name string) string {
	if LauncherVersion == "" {
		return name
	}
	return name + "  " + LauncherVersion
}

// RequireVersions fails when any version string is empty. A binary built by
// hand (go build ./launcher/cmd/tf2ap) has empty strings and could install the
// wrong versions; the Makefile sets them, so the installer gates on this.
func RequireVersions() error {
	for name, value := range Versions() {
		// Not install gates: nothing downloads by these, they only name what
		// is already here.
		if name == "defenderbots" || name == "launcher" {
			continue
		}
		if value == "" {
			return fmt.Errorf("asset version %s is empty: build with `make launcher` so -ldflags injects it from deploy/env/versions.env", name)
		}
	}
	return nil
}

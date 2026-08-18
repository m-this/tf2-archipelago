//go:build !windows

package generate

import (
	"os"
	"path/filepath"
	"strings"
)

// candidateDirs lists where a source checkout or an AppImage extraction of the
// app tends to be. Anywhere else goes in Options.AppDir.
func candidateDirs() []string {
	var dirs []string
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, "Archipelago"))
	}
	return append(dirs, "/opt/Archipelago", "/ap")
}

func generatorPath(appDir string) string {
	if p := filepath.Join(appDir, "ArchipelagoGenerate"); exists(p) {
		return p
	}
	return filepath.Join(appDir, "Generate.py")
}

// generatorCommand runs a frozen build as it is, and a source checkout through
// python: a .py is not executable on its own.
func generatorCommand(exe string) (string, []string) {
	if strings.HasSuffix(exe, ".py") {
		return "python3", []string{exe}
	}
	return exe, nil
}

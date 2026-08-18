//go:build windows

package generate

import (
	"os"
	"path/filepath"
)

// candidateDirs lists where the Archipelago installer puts the app, most likely
// first. The default is per-user under Programs; an "install for everyone"
// lands in Program Files.
func candidateDirs() []string {
	var dirs []string
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		dirs = append(dirs, filepath.Join(local, "Programs", "Archipelago"))
	}
	for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)"} {
		if root := os.Getenv(env); root != "" {
			dirs = append(dirs, filepath.Join(root, "Archipelago"))
		}
	}
	dirs = append(dirs, `C:\ProgramData\Archipelago`)
	return dirs
}

func generatorPath(appDir string) string {
	return filepath.Join(appDir, "ArchipelagoGenerate.exe")
}

// generatorCommand runs the frozen exe as it is.
func generatorCommand(exe string) (string, []string) { return exe, nil }

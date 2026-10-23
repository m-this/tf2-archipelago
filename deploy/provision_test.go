package deploy_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// provisionBinary is launcher/cmd/provision, built once for the whole run: the
// entrypoint and the launch script call it the way the image does.
var provisionBinary string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "tf2ap-provision")
	if err != nil {
		panic(err)
	}
	provisionBinary = filepath.Join(dir, "provision")
	build := exec.Command("go", "build", "-o", provisionBinary, "./launcher/cmd/provision")
	build.Dir = ".."
	build.Stdout, build.Stderr = os.Stderr, os.Stderr
	if err := build.Run(); err != nil {
		panic("cannot build launcher/cmd/provision: " + err.Error())
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

/*
The image loads a selected mod only when the run needs it, as the launcher does.

The entrypoint used to sync every mod SRCDS_MODS named, whatever
SRCDS_MOD_LOADING said, so SigMod ran under a pool of stock missions that never
asked for it.
*/
func TestTheImageLoadsASelectedModOnlyWhenTheRunNeedsIt(t *testing.T) {
	t.Parallel()
	for loading, want := range map[string]string{"required": "", "always": "sigsegv-mvm"} {
		command := exec.Command(provisionBinary, "mods")
		command.Env = []string{
			"PATH=" + os.Getenv("PATH"),
			"SRCDS_MODS=sigsegv-mvm",
			"SRCDS_MOD_LOADING=" + loading,
		}
		output, err := command.Output()
		if err != nil {
			t.Fatalf("provision mods: %v", err)
		}
		if got := strings.TrimSpace(string(output)); got != want {
			t.Errorf("SRCDS_MOD_LOADING=%s loads %q, want %q", loading, got, want)
		}
	}
}

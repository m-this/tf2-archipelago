package deploy_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// SteamCMD that reports the app stuck at state 0x6 on its first `stuck`
// app_update runs, rewriting the manifest each time as the real one does, and
// installs the game after that.
const fakeSteamcmd = `#!/bin/sh
echo "$*" >> "$FAKE_CALLS"
case "$*" in
*app_update*)
	stuck=$(cat "$FAKE_STUCK")
	if [ "$stuck" -gt 0 ]; then
		echo $((stuck - 1)) > "$FAKE_STUCK"
		echo '"StateFlags" "6"' > "$STEAMAPPDIR/steamapps/appmanifest_232250.acf"
		echo "Error! App '232250' state is 0x6 after update job."
		exit 8
	fi
	echo "Success! App '232250' fully installed."
	;;
esac
`

type launchRun struct {
	output    string
	err       error
	calls     []string
	steamapps string
}

func runLaunchWithStuckUpdates(t *testing.T, stuck int) launchRun {
	t.Helper()
	root := t.TempDir()
	steamcmdDir := filepath.Join(root, "steamcmd")
	game := filepath.Join(root, "tf-dedicated")
	steamapps := filepath.Join(game, "steamapps")
	for _, dir := range []string{steamcmdDir, steamapps} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		filepath.Join(steamcmdDir, "steamcmd.sh"):          fakeSteamcmd,
		filepath.Join(game, "srcds_run"):                   "echo srcds started\n",
		filepath.Join(root, "stuck"):                       strconv.Itoa(stuck) + "\n",
		filepath.Join(steamapps, "appmanifest_232250.acf"): "\"StateFlags\" \"6\"\n",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	callsPath := filepath.Join(root, "calls")

	command := exec.Command("bash", "deploy/srcds-launch.sh")
	command.Dir = ".."
	command.Env = append(os.Environ(),
		"FAKE_CALLS="+callsPath,
		"FAKE_STUCK="+filepath.Join(root, "stuck"),
		"STEAMCMDDIR="+steamcmdDir,
		"STEAMAPPDIR="+game,
		"STEAMAPPID=232250",
		"STEAMAPP=tf",
		"METAMOD_VERSION=",
		"SOURCEMOD_VERSION=",
		"SRCDS_MAXPLAYERS=32",
		"SRCDS_STARTMAP=mvm_decoy",
		"SRCDS_PORT=27015",
		"SRCDS_RCONPW=test",
		"SRCDS_LAN=1",
		"SRCDS_SDR_FAKEIP=0",
	)
	output, err := command.CombinedOutput()
	calls, readErr := os.ReadFile(callsPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	return launchRun{
		output:    string(output),
		err:       err,
		calls:     strings.Split(strings.TrimSpace(string(calls)), "\n"),
		steamapps: steamapps,
	}
}

func asideManifests(t *testing.T, steamapps string) []string {
	t.Helper()
	aside, err := filepath.Glob(filepath.Join(steamapps, "appmanifest_232250.acf.0x6-*"))
	if err != nil {
		t.Fatal(err)
	}
	return aside
}

func TestAnUpdateStuckAtState0x6RecoversOnce(t *testing.T) {
	t.Parallel()

	run := runLaunchWithStuckUpdates(t, 1)
	if run.err != nil {
		t.Fatalf("launch: %v\n%s", run.err, run.output)
	}
	if !strings.Contains(run.output, "srcds started") {
		t.Fatalf("the server did not start after the recovery:\n%s", run.output)
	}

	update := "+force_install_dir " + filepath.Dir(run.steamapps) + " +login anonymous +app_update 232250 +quit"
	want := []string{update, "+quit", "+login anonymous +quit", update}
	if strings.Join(run.calls, "\n") != strings.Join(want, "\n") {
		t.Fatalf("SteamCMD runs = %q, want %q", run.calls, want)
	}
	aside := asideManifests(t, run.steamapps)
	if len(aside) != 1 {
		t.Fatalf("manifests moved aside = %q, want one", aside)
	}
	kept, err := os.ReadFile(aside[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(kept), `"StateFlags" "6"`) {
		t.Fatalf("the stuck manifest was not kept:\n%s", kept)
	}
}

func TestAnUpdateStillStuckAfterTheRecoveryFails(t *testing.T) {
	t.Parallel()

	run := runLaunchWithStuckUpdates(t, 2)
	var exit *exec.ExitError
	if !errors.As(run.err, &exit) || exit.ExitCode() != 8 {
		t.Fatalf("launch error = %v, want exit status 8:\n%s", run.err, run.output)
	}
	if strings.Contains(run.output, "srcds started") {
		t.Fatalf("the server started with the update still stuck:\n%s", run.output)
	}
	if !strings.Contains(run.output, "[AP] the TF2 update is still stuck at state 0x6 after moving") {
		t.Fatalf("no [AP] line names the stuck manifest:\n%s", run.output)
	}
	updates := 0
	for _, call := range run.calls {
		if strings.Contains(call, "+app_update") {
			updates++
		}
	}
	if updates != 2 {
		t.Fatalf("app_update ran %d times, want 2: %q", updates, run.calls)
	}
	if aside := asideManifests(t, run.steamapps); len(aside) != 1 {
		t.Fatalf("manifests moved aside = %q, want one", aside)
	}
}

package deploy_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestChangedAdminFileReloadsSourceModCache(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	game := filepath.Join(root, "tf-dedicated", "tf")
	configs := filepath.Join(game, "addons", "sourcemod", "configs")
	if err := os.MkdirAll(configs, 0o755); err != nil {
		t.Fatal(err)
	}
	calls := filepath.Join(root, "rcon-calls")
	fakeRCON := filepath.Join(root, "rcon")
	if err := os.WriteFile(fakeRCON, []byte("#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$TF2AP_RCON_CALLS\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	command := exec.Command("bash", "-c", `. deploy/srcds-entrypoint.sh
install_admin
wait
# An unchanged supervisor pass must not keep flushing SourceMod's cache.
install_admin
wait`)
	command.Dir = ".."
	command.Env = append(os.Environ(),
		"TF2AP_ENTRYPOINT_LIBRARY=1",
		"STEAMAPPDIR="+filepath.Join(root, "tf-dedicated"),
		"STEAMAPP=tf",
		"SRCDS_ADMIN_STEAMIDS=76561198019118556",
		"TF2AP_RCON="+fakeRCON,
		"TF2AP_RCON_CALLS="+calls,
		"TF2AP_ADMIN_RELOAD_ATTEMPTS=1",
		"TF2AP_ADMIN_RELOAD_INTERVAL=0",
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("entrypoint test: %v\n%s", err, output)
	}

	admins, err := os.ReadFile(filepath.Join(configs, "admins_simple.ini"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(admins), `"STEAM_0:0:29426414" "99:z"`) {
		t.Fatalf("admin file does not contain converted Steam id:\n%s", admins)
	}
	reloads, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(reloads); got != "sm_reloadadmins\n" {
		t.Fatalf("RCON calls = %q, want one admin-cache reload", got)
	}
}

/*
The loadout file is written into the staged tree and the convar follows it.

Both halves were missing in the stack: nothing wrote the file, so a loadout
picked on the admin page reached .env and stopped there, and nothing set
sm_redbots_manager_use_custom_loadouts, so the mod would have ignored the file
even if it had been there.

The renderer is stubbed here. What this covers is the entrypoint's half: that
it runs before the sync that carries the file over, and that what it printed is
what server.cfg ends up saying.
*/
func TestTheBotLoadoutIsStagedAndTheConvarFollowsIt(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := filepath.Join(root, "botfiles")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$TF2AP_BOTFILES_CALLS\"\necho 1\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	calls := filepath.Join(root, "botfiles-calls")

	command := exec.Command("bash", "-c", `. deploy/srcds-entrypoint.sh
install_bot_files
echo "convar=$bot_custom_loadouts"`)
	command.Dir = ".."
	command.Env = append(os.Environ(),
		"TF2AP_ENTRYPOINT_LIBRARY=1",
		"TF2AP_BOT_FILES="+fake,
		"TF2AP_BOTFILES_CALLS="+calls,
		"STEAMAPPDIR="+root,
		"STEAMAPP=tf",
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("entrypoint test: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "convar=1") {
		t.Fatalf("the convar did not follow the renderer:\n%s", output)
	}

	asked, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	// The staged tree the image builds, which sync_tree copies into the game.
	if got := strings.TrimSpace(string(asked)); got != "-root /opt/tf2-archipelago" {
		t.Fatalf("the renderer was asked %q, want the staged tree", got)
	}
}

// A renderer that fails leaves the bots on stock rather than taking the server
// down with it: the convar stays 0 and the mod ignores whatever is on disk.
func TestARefusedRenderLeavesTheBotsOnStock(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := filepath.Join(root, "botfiles")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\necho broken >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	command := exec.Command("bash", "-c", `. deploy/srcds-entrypoint.sh
install_bot_files
echo "convar=$bot_custom_loadouts"`)
	command.Dir = ".."
	command.Env = append(os.Environ(),
		"TF2AP_ENTRYPOINT_LIBRARY=1",
		"TF2AP_BOT_FILES="+fake,
		"STEAMAPPDIR="+root,
		"STEAMAPP=tf",
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("entrypoint test: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "convar=0") {
		t.Fatalf("a failed render did not leave the convar off:\n%s", output)
	}
}

func TestSigModDoesNotCountREDDefendersAsInvaders(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	cfgDir := filepath.Join(root, "tf", "cfg")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("bash", "-c", `. deploy/srcds-entrypoint.sh
install_server_cfg`)
	command.Dir = ".."
	command.Env = append(os.Environ(),
		"TF2AP_ENTRYPOINT_LIBRARY=1",
		"STEAMAPPDIR="+root,
		"STEAMAPP=tf",
		"SRCDS_RCONPW=test",
		"SRCDS_MODS=sigsegv-mvm",
	)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("entrypoint test: %v\n%s", err, output)
	}
	config, err := os.ReadFile(filepath.Join(cfgDir, "server.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(config), "sig_mvm_robot_limit_fix_red 0\n") {
		t.Fatalf("RED defenders still count toward the invader limit:\n%s", config)
	}
}

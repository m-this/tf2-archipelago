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
The loadout file is written into the staged tree.

Nothing wrote it once, so a loadout picked on the admin page reached .env and
stopped there. Into the staged tree rather than the game, because sync_tree
carries the staged file over. Whether the mod reads it is server.cfg's
sm_redbots_manager_use_custom_loadouts, which provision decides from the same
settings.
*/
func TestTheBotLoadoutIsStaged(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := filepath.Join(root, "botfiles")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$TF2AP_BOTFILES_CALLS\"\necho 1\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	calls := filepath.Join(root, "botfiles-calls")

	command := exec.Command("bash", "-c", `. deploy/srcds-entrypoint.sh
install_bot_files`)
	command.Dir = ".."
	command.Env = append(os.Environ(),
		"TF2AP_ENTRYPOINT_LIBRARY=1",
		"TF2AP_BOT_FILES="+fake,
		"TF2AP_BOTFILES_CALLS="+calls,
		"STEAMAPPDIR="+root,
		"STEAMAPP=tf",
	)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("entrypoint test: %v\n%s", err, output)
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
		"SRCDS_MOD_LOADING=always",
		"TF2AP_PROVISION="+provisionBinary,
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

func TestServerCfgWithoutSigModOmitsREDRobotLimit(t *testing.T) {
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
		"SRCDS_MODS=",
		"TF2AP_PROVISION="+provisionBinary,
	)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("entrypoint test: %v\n%s", err, output)
	}
	config, err := os.ReadFile(filepath.Join(cfgDir, "server.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(config), "sig_mvm_robot_limit_fix_red") || strings.Contains(string(config), "SigMod's RED robot-limit fix") {
		t.Fatalf("server.cfg contains SigMod-only settings with SigMod disabled:\n%s", config)
	}
}

// renderDockerServerCfg runs the entrypoint's server.cfg step the way the
// image does and returns what it wrote.
func renderDockerServerCfg(t *testing.T, env ...string) string {
	t.Helper()
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
		"TF2AP_PROVISION="+provisionBinary,
	)
	command.Env = append(command.Env, env...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("entrypoint test: %v\n%s", err, output)
	}
	config, err := os.ReadFile(filepath.Join(cfgDir, "server.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	return string(config)
}

// The start mission waits for ten seconds of game time, and an empty server
// that hibernates never lets them pass (376b0994). The launcher wrote this
// line; the image did not.
func TestTheDockerServerStaysAwakeWhenEmpty(t *testing.T) {
	t.Parallel()
	if config := renderDockerServerCfg(t); !strings.Contains(config, "sv_hibernate_when_empty 0\n") {
		t.Fatalf("server.cfg lets an empty server hibernate:\n%s", config)
	}
}

// The robot health chosen on the admin page reaches the plugin.
func TestTheDockerServerScalesRobotHealth(t *testing.T) {
	t.Parallel()
	config := renderDockerServerCfg(t, "SRCDS_BLU_HEALTH_PCT=150")
	if !strings.Contains(config, "tf2ap_robot_health_scale 1.50\n") {
		t.Fatalf("server.cfg does not carry SRCDS_BLU_HEALTH_PCT=150:\n%s", config)
	}
}

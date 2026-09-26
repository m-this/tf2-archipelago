package webapi

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/m-this/tf2-archipelago/gamedata"
	"github.com/m-this/tf2-archipelago/launcher/internal/assets"
	"github.com/m-this/tf2-archipelago/launcher/internal/botfiles"
	"github.com/m-this/tf2-archipelago/launcher/internal/botlive"
	"github.com/m-this/tf2-archipelago/launcher/internal/composeenv"
	"github.com/m-this/tf2-archipelago/launcher/internal/form"
	"github.com/m-this/tf2-archipelago/launcher/internal/generate"
	"github.com/m-this/tf2-archipelago/launcher/internal/installer"
	"github.com/m-this/tf2-archipelago/launcher/internal/roomcheck"
	"github.com/m-this/tf2-archipelago/launcher/internal/runshape"
	"github.com/m-this/tf2-archipelago/launcher/internal/saveplan"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
	"github.com/m-this/tf2-archipelago/launcher/internal/srcdsconfig"
	"github.com/m-this/tf2-archipelago/launcher/internal/winproc"
)

func (a *App) DraftSettings() (settings.Settings, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.draft == nil {
		return settings.Settings{}, errors.New("settings are not open")
	}
	return a.draft.Settings, nil
}

// SettingsNow is the draft while the screen is open and the saved settings
// otherwise. The debug bundle asks: it is offered from the console as well as
// from the settings, and a bundle nobody can save from the log they are
// reading is the one that never reaches a bug report.
func (a *App) SettingsNow() settings.Settings {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.draft != nil {
		return a.draft.Settings
	}
	return a.settings
}

func (a *App) OpenSettings(page string) {
	a.mu.Lock()
	state := form.NewState(a.settings)
	a.draft, a.formPage = &state, page
	a.community = availableCommunityPackNames(a.settings.CommunityContentDir)
	a.imported = importedCommunityPackNames(a.settings.CommunityContentDir, a.community)
	a.serverMods = installer.ReadyServerMods(a.settings.InstallRoot)
	a.publishLocked(Event{Name: "state", Data: struct{}{}})
	a.mu.Unlock()
}

func (a *App) Change(c form.Change) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.draft == nil {
		return errors.New("settings are not open")
	}
	next, err := form.Apply(*a.draft, a.formEnvLocked(), c)
	if err != nil {
		return err
	}
	*a.draft = next
	a.publishLocked(Event{Name: "state", Data: struct{}{}})
	return nil
}

func (a *App) SaveSettings(restart bool) error {
	a.mu.Lock()
	if a.draft == nil {
		a.mu.Unlock()
		return errors.New("settings are not open")
	}
	draft := *a.draft
	readyMods := slices.Clone(a.serverMods)
	attached, envFile := a.attached, a.attachedEnvFile
	a.mu.Unlock()
	roomErr := parseDraftRoom(&draft)
	before := a.supervisor.Settings()
	forgetOldRoomCards(before, &draft.Settings)
	written, err := persistDraft(draft.Settings, before, readyMods, attached, envFile)
	if err != nil {
		return err
	}
	if attached && botlive.TeamMoved(before, written) {
		if err := a.applyAttachedTeam(before, written); err != nil {
			return fmt.Errorf("settings saved to .env, but the live bot team was not applied: %w", err)
		}
	}
	a.mu.Lock()
	a.settings, a.draft = written, nil
	a.notice = "settings saved"
	a.noticeSeq++
	heldRestart := a.smHeld
	a.smHeld = false
	a.mu.Unlock()
	a.supervisor.SetSettings(written)
	if attached {
		a.finishAttachedSave(before, written)
		a.publishState()
		return nil
	}
	if _, err := settings.WritePlayerFile(written, assets.ArchipelagoVersion); err != nil {
		a.Say("%v", err)
	}
	if a.supervisor.Running() {
		plan := saveplan.For(before, written)
		if plan.Team {
			if err := srcdsconfig.Install(written); err != nil {
				a.Say("cannot write the bot files: %v", err)
			} else {
				for _, command := range botlive.Commands(before, written) {
					a.SendRCON(command)
				}
			}
		}
		switch {
		case plan.Restart && restart:
			a.Say("settings saved. Restarting the server to apply them.")
			a.Restart()
		case heldRestart:
			a.Say("SourceMod updated its gamedata. Restarting the server to load it.")
			a.Restart()
		case plan.Restart:
			a.Say("settings saved. The server is still playing on what it started with: press Restart to apply them.")
		case plan.Quiet():
			a.Say("settings saved. The server keeps playing: nothing here changes a run it is already in.")
		}
	}
	go a.reportRoom(written, draft.Draft.Room, roomErr)
	a.publishState()
	return nil
}

func parseDraftRoom(draft *form.State) error {
	room, err := settings.ParseRoom(draft.Draft.Room)
	if err == nil {
		draft.Settings.APHost, draft.Settings.APPort, draft.Settings.APTls = room.Host, room.Port, room.TLS
	} else if strings.TrimSpace(draft.Draft.Room) == "" {
		draft.Settings.APHost, draft.Settings.APPort = "", 0
	}
	return err
}

func persistDraft(draft, before settings.Settings, readyMods []string, attached bool, envFile string) (settings.Settings, error) {
	if attached {
		if envFile == "" {
			return settings.Settings{}, errors.New("the Compose .env file is not mounted into the admin container")
		}
		if err := composeenv.Write(envFile, before, draft); err != nil {
			return settings.Settings{}, err
		}
		return draft, nil
	}
	if err := settings.CheckServerModsReady(draft, readyMods); err != nil {
		return settings.Settings{}, err
	}
	return settings.Persist(draft)
}

func (a *App) finishAttachedSave(before, after settings.Settings) {
	plan := saveplan.For(before, after)
	if plan.Restart {
		a.Notify("Settings saved to .env. Apply them with: docker compose up -d --force-recreate")
		return
	}
	if plan.Team {
		a.Notify("Bot team applied to the running server without a map restart.")
		return
	}
	a.Notify("Settings saved to .env. Container settings apply with docker compose up -d --force-recreate; seed options apply on the next generation.")
}

// applyAttachedTeam uses the existing shared community overlay as a narrow
// hand-off. The admin cannot write the game volume and does not have Docker's
// socket; the AP plugin copies exactly the two bot files over RCON, then each
// lineup command is acknowledged in order on one connection.
func (a *App) applyAttachedTeam(before, after settings.Settings) error {
	root := filepath.Join(before.CommunityContentDir, "tf")
	if err := botfiles.StageForLive(root, after); err != nil {
		return fmt.Errorf("stage bot files: %w", err)
	}
	client, err := dialRCON(before)
	if err != nil {
		return fmt.Errorf("connect to the game: %w", err)
	}
	defer func() { _ = client.Close() }()
	reply, err := client.Exec("sm_ap_botcards_sync")
	if err != nil {
		return fmt.Errorf("copy bot files into the game: %w", err)
	}
	if !strings.Contains(reply, "Bot file sync OK") {
		return fmt.Errorf("copy bot files into the game: %s", strings.TrimSpace(reply))
	}
	for _, command := range botlive.Commands(before, after) {
		reply, err := client.Exec(command)
		if err != nil {
			return fmt.Errorf("%s: %w", command, err)
		}
		if strings.Contains(reply, "Unknown command") || strings.Contains(reply, "cards unchanged") ||
			strings.Contains(reply, "Cannot read the new lineup") ||
			strings.Contains(reply, "No card loadout file to reload") {
			return fmt.Errorf("%s: %s", command, reply)
		}
	}
	return nil
}

func (a *App) reportRoom(s settings.Settings, typed string, parseErr error) {
	if parseErr != nil && strings.TrimSpace(typed) != "" {
		a.Notify("the room address was not saved: " + parseErr.Error() + ". " + roomcheck.NotConfigured.Advice())
		return
	}
	result, err := roomcheck.Check(context.Background(), s)
	if err != nil {
		a.Notify("settings saved, but the room did not answer: " + err.Error() + ". " + result.Advice())
		return
	}
	a.Notify("settings saved. " + result.Advice())
}

func (a *App) CancelSettings() {
	a.mu.Lock()
	a.draft = nil
	heldRestart := a.smHeld
	a.smHeld = false
	a.publishLocked(Event{Name: "state", Data: struct{}{}})
	a.mu.Unlock()
	if heldRestart {
		a.Say("SourceMod updated its gamedata. Restarting the server to load it.")
		a.Restart()
	}
}

func (a *App) formEnvLocked() form.Env {
	dirs := generate.SearchPath("")
	appDir := ""
	if len(dirs) > 0 {
		appDir = dirs[0]
	}
	var received []string
	if !a.draft.Settings.TestMode {
		received = []string{}
		for _, unlock := range a.snapshot.Unlocks {
			if unlock.Kind == "Bot card" {
				received = append(received, unlock.Name)
			}
		}
	}
	return form.Env{
		BotCardItems:       received,
		CommunityAvailable: slices.Clone(a.community), ServerModsReady: slices.Clone(a.serverMods),
		CommunityHashMismatches: installer.PendingCommunityArchiveHashMismatches(settings.KnownCommunityArchives(a.draft.Settings.CommunityContentDir)),
		Platform:                runtime.GOOS, AppDirDefault: appDir, ManagedExternally: a.attached,
	}
}

func (a *App) Dispatch(id string) error {
	a.mu.Lock()
	if a.draft == nil {
		a.mu.Unlock()
		return errors.New("settings are not open")
	}
	if !form.Dispatchable(*a.draft, a.formEnvLocked(), id) {
		a.mu.Unlock()
		return fmt.Errorf("no settings action %q", id)
	}
	s := *a.draft
	a.mu.Unlock()

	// form.Act owns every button whose answer is a State and nothing else, and
	// it says which those are. Naming them here as well is what broke
	// bots.name_add: it was in wiredActions, so the test that compares the two
	// lists passed, and it was missing from the switch below, so the browser
	// got "settings action \"bots.name_add\" is not wired" and no name was
	// ever added. One list, and it is form.Act's.
	if next, said, ok := form.Act(s, id); ok {
		a.mutateDraft(func(state *form.State) { *state = next })
		a.Notify(said)
		return nil
	}

	switch id {
	case "missions.pool_all", "missions.pool_none":
		a.setPool(id == "missions.pool_all")
	case "missions.check_selection":
		a.checkMissionSelection(s.Settings)
	case "missions.download_packs":
		go a.downloadPacks(s.Settings)
	case "missions.ignore_hash_mismatch":
		go a.ignoreCommunityArchiveHashMismatch(s.Settings)
	case "missions.import_assets":
		return a.useLocalPacks(s.Settings.CommunityContentDir)
	case "missions.install_mods":
		go a.installSelectedMods(s.Settings)
	case "server.repair":
		go a.repair(s.Settings.InstallRoot)
	case "server.reset":
		return a.resetSettings()
	default:
		return fmt.Errorf("settings action %q is not wired", id)
	}
	return nil
}

func (a *App) checkMissionSelection(s settings.Settings) {
	if modErr := settings.CheckServerModsReady(s, a.readyServerMods()); modErr != nil {
		a.Notify(modErr.Error())
		return
	}
	result, err := settings.CheckRunSelection(s)
	if err != nil {
		a.Notify(err.Error())
		return
	}
	a.Notify(result.Summary())
}

var wiredActions = []string{
	"run.generate", "run.open_player_file", "run.open_folder", "run.open_settings_file",
	"missions.download_packs", "missions.import_assets", "missions.check_selection",
	"missions.ignore_hash_mismatch",
	"missions.install_mods",
	"missions.pool_all", "missions.pool_none",
	"server.debug_bundle", "server.repair", "server.reset",
	"net.check_funnel", "bots.save_team", "bots.remove_team", "bots.name_add", "loadout.save",
}

func (a *App) mutateDraft(change func(*form.State)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.draft != nil {
		change(a.draft)
		a.publishLocked(Event{Name: "state", Data: struct{}{}})
	}
}

func (a *App) setPool(all bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.draft == nil {
		return
	}
	activeMods := activeReadyServerMods(a.draft.Settings, a.serverMods)
	var excluded []string
	if !all {
		for _, mission := range gamedata.Missions {
			excluded = append(excluded, mission.PopFile)
		}
	} else {
		visible := runshape.VisibleMissions(a.community)
		for _, mission := range gamedata.Missions {
			if !gamedata.IsMissionPlayableWith(mission.ID, activeMods) ||
				gamedata.IsCommunityMission(mission.ID) && !a.draft.Settings.MvmCommunityMissions ||
				gamedata.MissionPack(mission.ID) != "" && !slices.ContainsFunc(visible, func(candidate gamedata.Mission) bool { return candidate.ID == mission.ID }) {
				excluded = append(excluded, mission.PopFile)
			}
		}
	}
	a.draft.Settings.MvmExcludedMissions = excluded
	if slices.Contains(excluded, a.draft.Settings.MvmStartMission) {
		a.draft.Settings.MvmStartMission = ""
	}
	a.notice = map[bool]string{true: "every mission is in the pool", false: "every mission is left out"}[all]
	a.noticeSeq++
	a.publishLocked(Event{Name: "state", Data: struct{}{}})
}

func activeReadyServerMods(s settings.Settings, ready []string) []string {
	var active []string
	for _, key := range settings.ServerModKeys(s) {
		if slices.Contains(ready, key) {
			active = append(active, key)
		}
	}
	return active
}

func (a *App) readyServerMods() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.serverMods)
}

func (a *App) installSelectedMods(s settings.Settings) {
	mods := settings.ServerModKeys(s)
	if len(mods) == 0 {
		a.Notify("select a server mod first")
		return
	}
	if a.attached {
		ready := installer.ReadyServerMods(s.InstallRoot)
		if slices.ContainsFunc(mods, func(mod string) bool { return !slices.Contains(ready, mod) }) {
			a.Notify("Docker includes the selected server mods. Save these settings, then apply them with: docker compose up -d --force-recreate")
			return
		}
		a.Notify("selected server mods are installed and verified")
		return
	}
	if a.supervisor.Running() {
		a.Notify("stop the server before installing or repairing a server mod")
		return
	}
	ctx, done, ok := a.beginSettingsActivity("Preparing selected server mods…")
	if !ok {
		return
	}
	defer done()
	if _, err := installer.Ensure(ctx, s.InstallRoot, nil, mods, a.reportSettingsActivity); err != nil {
		if ctx.Err() == nil {
			a.Notify("server mod setup: " + err.Error())
		}
		return
	}
	a.mu.Lock()
	a.serverMods = installer.ReadyServerMods(s.InstallRoot)
	ready := slices.Clone(a.serverMods)
	a.mu.Unlock()
	if err := settings.CheckServerModsReady(s, ready); err != nil && len(settings.RequiredServerMods(s)) > 0 {
		a.Notify(err.Error())
		return
	}
	a.Notify("selected server mods are installed and verified")
}

func (a *App) downloadPacks(s settings.Settings) {
	folder := strings.TrimSpace(s.CommunityContentDir)
	if folder == "" {
		a.Notify("choose an asset pack folder first")
		return
	}
	archives := settings.CommunityArchives(s)
	if len(archives) == 0 {
		a.Notify("select at least one community pack first")
		return
	}
	ctx, done, ok := a.beginSettingsActivity("Checking cached community assets…")
	if !ok {
		return
	}
	defer done()
	if err := installer.DownloadCommunityArchives(ctx, archives, a.reportSettingsActivity); err != nil {
		a.Notify("community assets: " + err.Error())
		return
	}
	if a.attached {
		contentTree := filepath.Join(folder, "tf")
		if err := installer.InstallCommunityArchives(archives, contentTree, s.SrcdsMods, a.reportSettingsActivity); err != nil {
			a.Notify("community assets: " + err.Error())
			return
		}
	}
	a.mu.Lock()
	a.community = availableCommunityPackNames(folder)
	a.mu.Unlock()
	if a.attached {
		a.Notify("selected community packs are installed; the game server will pick them up within 30 seconds")
		return
	}
	a.Notify("selected community packs are ready in " + folder)
}

func (a *App) ignoreCommunityArchiveHashMismatch(s settings.Settings) {
	folder := strings.TrimSpace(s.CommunityContentDir)
	if folder == "" {
		a.Notify("choose an asset pack folder first")
		return
	}
	approved, err := installer.IgnoreCommunityArchiveHashMismatch(settings.KnownCommunityArchives(folder))
	if err != nil {
		a.Notify("community assets: " + err.Error())
		return
	}
	a.mu.Lock()
	a.community = availableCommunityPackNames(folder)
	a.mu.Unlock()
	a.Notify("ignored hash mismatch for " + strings.Join(approved, ", ") + "; these missions may be unstable")
}

// beginSettingsActivity gives long-running settings actions one shared busy
// state and a line that remains visible on the settings page. Progress updates
// replace that line instead of making the player hunt through the server log.
func (a *App) beginSettingsActivity(message string) (context.Context, func(), bool) {
	a.mu.Lock()
	if a.busy {
		a.mu.Unlock()
		a.Notify("another install is already running")
		return nil, func() {}, false
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.busy, a.install, a.activity = true, cancel, message
	a.publishLocked(Event{Name: "state", Data: struct{}{}})
	a.mu.Unlock()
	a.Say("%s", message)
	return ctx, func() {
		cancel()
		a.mu.Lock()
		a.busy, a.install, a.activity = false, nil, ""
		a.publishLocked(Event{Name: "state", Data: struct{}{}})
		a.mu.Unlock()
	}, true
}

func (a *App) reportSettingsActivity(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	a.mu.Lock()
	a.activity = message
	a.publishLocked(Event{Name: "state", Data: struct{}{}})
	a.mu.Unlock()
	a.Say("%s", message)
}

func (a *App) repair(root string) {
	a.Stop()
	_, _ = winproc.KillUnder(root)
	removed, err := installer.Clean(root)
	if err != nil {
		a.Notify("repair: " + err.Error())
		return
	}
	if len(removed) == 0 {
		a.Notify("repair: nothing to remove")
	} else {
		a.Notify("repair removed " + strings.Join(removed, ", "))
	}
	a.mu.Lock()
	a.serverMods = installer.ReadyServerMods(root)
	a.mu.Unlock()
}

func (a *App) resetSettings() error {
	fresh := settings.Defaults()
	fresh.InstallRoot = a.supervisor.Settings().InstallRoot
	written, err := settings.Persist(fresh)
	if err != nil {
		return err
	}
	a.supervisor.SetSettings(written)
	a.mu.Lock()
	a.settings = written
	state := form.NewState(written)
	a.draft = &state
	a.mu.Unlock()
	a.Notify("every setting is back to its default")
	return nil
}

/*
useLocalPacks takes the asset packs already sitting in the content folder.

The packs run to gigabytes and they are on the same machine as the launcher, so
there is nothing to upload: the player points the content folder at where they
put the zips and this reads what is there. It is what the terminal interface has
always done. The browser's old multipart upload existed only because a page with
no folder picker had no other way to name a file.
*/
func (a *App) useLocalPacks(folder string) error {
	if strings.TrimSpace(folder) == "" {
		return errors.New("choose an asset pack folder first")
	}
	available := availableCommunityPackNames(folder)
	if len(available) == 0 {
		return fmt.Errorf("no archive-assets.zip or mlarchive-assets.zip was found in %s", folder)
	}
	if err := a.AcceptImportedPacks(available); err != nil {
		return err
	}
	a.Notify("using the community packs in " + folder)
	return nil
}

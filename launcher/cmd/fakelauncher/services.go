package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"connectrpc.com/connect"

	"github.com/m-this/tf2-archipelago/launcher/internal/form"
	launcherv1 "github.com/m-this/tf2-archipelago/launcher/internal/gen/tf2ap/launcher/v1"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

// The three services, over the fake. Each one is the shortest thing that is
// still true: the state moves, the stream says so, and a refusal is a refusal.

type launcherRPC struct{ fake *fake }

func (s launcherRPC) GetSnapshot(context.Context, *connect.Request[launcherv1.GetSnapshotRequest]) (*connect.Response[launcherv1.GetSnapshotResponse], error) {
	s.fake.mu.Lock()
	defer s.fake.mu.Unlock()
	return connect.NewResponse(&launcherv1.GetSnapshotResponse{Snapshot: s.fake.snapshotLocked().Proto()}), nil
}

func (s launcherRPC) Start(context.Context, *connect.Request[launcherv1.StartRequest]) (*connect.Response[launcherv1.StartResponse], error) {
	s.fake.mu.Lock()
	s.fake.running, s.fake.mission = true, "mvm_decoy_advanced"
	s.fake.mu.Unlock()
	s.fake.say("the server is up")
	s.fake.redraw()
	return connect.NewResponse(&launcherv1.StartResponse{}), nil
}

func (s launcherRPC) Stop(context.Context, *connect.Request[launcherv1.StopRequest]) (*connect.Response[launcherv1.StopResponse], error) {
	s.fake.mu.Lock()
	s.fake.running, s.fake.mission = false, ""
	s.fake.mu.Unlock()
	s.fake.say("the server stopped")
	s.fake.redraw()
	return connect.NewResponse(&launcherv1.StopResponse{}), nil
}

func (s launcherRPC) Restart(context.Context, *connect.Request[launcherv1.RestartRequest]) (*connect.Response[launcherv1.RestartResponse], error) {
	s.fake.say("restarting the server")
	s.fake.redraw()
	return connect.NewResponse(&launcherv1.RestartResponse{}), nil
}

func (s launcherRPC) Quit(context.Context, *connect.Request[launcherv1.QuitRequest]) (*connect.Response[launcherv1.QuitResponse], error) {
	s.fake.say("quit was pressed")
	return connect.NewResponse(&launcherv1.QuitResponse{}), nil
}

func (s launcherRPC) SendRcon(_ context.Context, request *connect.Request[launcherv1.SendRconRequest]) (*connect.Response[launcherv1.SendRconResponse], error) {
	s.fake.say("rcon: " + request.Msg.GetCommand())
	return connect.NewResponse(&launcherv1.SendRconResponse{}), nil
}

func (s launcherRPC) SetMission(_ context.Context, request *connect.Request[launcherv1.SetMissionRequest]) (*connect.Response[launcherv1.SetMissionResponse], error) {
	s.fake.mu.Lock()
	s.fake.mission = request.Msg.GetPopFile()
	s.fake.mu.Unlock()
	s.fake.say("next mission is " + request.Msg.GetPopFile())
	s.fake.redraw()
	return connect.NewResponse(&launcherv1.SetMissionResponse{}), nil
}

// ResumeMission loads the mission and says where it went back to, the way the
// plugin announces it. The fake has no game, so the wave is the record's.
func (s launcherRPC) ResumeMission(_ context.Context, request *connect.Request[launcherv1.ResumeMissionRequest]) (*connect.Response[launcherv1.ResumeMissionResponse], error) {
	popFile := request.Msg.GetPopFile()
	s.fake.mu.Lock()
	s.fake.mission = popFile
	s.fake.mu.Unlock()
	s.fake.say(fmt.Sprintf("resuming %s at wave %d", popFile, request.Msg.GetWave()))
	s.fake.redraw()
	return connect.NewResponse(&launcherv1.ResumeMissionResponse{}), nil
}

func (s launcherRPC) ApproveFunnel(context.Context, *connect.Request[launcherv1.ApproveFunnelRequest]) (*connect.Response[launcherv1.ApproveFunnelResponse], error) {
	return connect.NewResponse(&launcherv1.ApproveFunnelResponse{
		Message: "Tailscale Funnel is ready for this tailnet.",
	}), nil
}

type settingsRPC struct{ fake *fake }

func (s settingsRPC) OpenSettings(_ context.Context, request *connect.Request[launcherv1.OpenSettingsRequest]) (*connect.Response[launcherv1.OpenSettingsResponse], error) {
	s.fake.mu.Lock()
	state := form.NewState(s.fake.settings)
	s.fake.draft = &state
	screen := s.fake.screenLocked().Proto()
	s.fake.redrawLocked()
	s.fake.mu.Unlock()
	_ = request
	return connect.NewResponse(&launcherv1.OpenSettingsResponse{Screen: screen}), nil
}

func (s settingsRPC) ChangeSetting(_ context.Context, request *connect.Request[launcherv1.ChangeSettingRequest]) (*connect.Response[launcherv1.ChangeSettingResponse], error) {
	s.fake.mu.Lock()
	if s.fake.draft == nil {
		s.fake.mu.Unlock()
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("the settings are not open"))
	}
	change := form.Change{Field: request.Msg.GetChange().GetField(), Value: request.Msg.GetChange().GetValue()}
	next, err := form.Apply(*s.fake.draft, form.Env{}, change)
	if err != nil {
		s.fake.mu.Unlock()
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	s.fake.draft = &next
	screen := s.fake.screenLocked().Proto()
	s.fake.redrawLocked()
	s.fake.mu.Unlock()
	return connect.NewResponse(&launcherv1.ChangeSettingResponse{Screen: screen}), nil
}

func (s settingsRPC) DispatchAction(_ context.Context, request *connect.Request[launcherv1.DispatchActionRequest]) (*connect.Response[launcherv1.DispatchActionResponse], error) {
	s.fake.say("action: " + request.Msg.GetId())
	s.fake.mu.Lock()
	// The buttons whose whole effect is on the draft are answered the way
	// the launcher answers them, by the same code. The rest only log.
	if s.fake.draft != nil {
		if next, said, ok := form.Act(*s.fake.draft, request.Msg.GetId()); ok {
			s.fake.draft = &next
			s.fake.notice, s.fake.noticeSeq = said, s.fake.noticeSeq+1
		}
	}
	screen := s.fake.screenLocked().Proto()
	s.fake.redrawLocked()
	s.fake.mu.Unlock()
	return connect.NewResponse(&launcherv1.DispatchActionResponse{Screen: screen}), nil
}

// SaveSettings refuses a room nothing could connect to, which is the one thing
// the real launcher refuses a save for and the one the browser has to draw.
func (s settingsRPC) SaveSettings(_ context.Context, request *connect.Request[launcherv1.SaveSettingsRequest]) (*connect.Response[launcherv1.SaveSettingsResponse], error) {
	s.fake.mu.Lock()
	defer s.fake.mu.Unlock()
	if s.fake.draft == nil {
		return connect.NewResponse(&launcherv1.SaveSettingsResponse{
			Saved: false, Refusal: "the settings are not open",
		}), nil
	}
	if s.fake.draft.Settings.InstallRoot == "" {
		return connect.NewResponse(&launcherv1.SaveSettingsResponse{
			Saved: false, Refusal: "the server folder cannot be empty",
			Screen: s.fake.screenLocked().Proto(),
		}), nil
	}
	s.fake.settings = s.fake.draft.Settings
	s.fake.notice, s.fake.noticeSeq = "settings saved", s.fake.noticeSeq+1
	s.fake.redrawLocked()
	_ = request
	return connect.NewResponse(&launcherv1.SaveSettingsResponse{
		Saved: true, Screen: s.fake.screenLocked().Proto(),
	}), nil
}

func (s settingsRPC) CancelSettings(context.Context, *connect.Request[launcherv1.CancelSettingsRequest]) (*connect.Response[launcherv1.CancelSettingsResponse], error) {
	s.fake.mu.Lock()
	s.fake.draft = nil
	s.fake.redrawLocked()
	s.fake.mu.Unlock()
	return connect.NewResponse(&launcherv1.CancelSettingsResponse{}), nil
}

type filesRPC struct{ fake *fake }

func (s filesRPC) ShowFile(_ context.Context, request *connect.Request[launcherv1.ShowFileRequest]) (*connect.Response[launcherv1.ShowFileResponse], error) {
	path := map[launcherv1.FileTarget]string{
		launcherv1.FileTarget_FILE_TARGET_SETTINGS_FILE:  "/home/player/.config/tf2ap/settings.json",
		launcherv1.FileTarget_FILE_TARGET_PLAYER_FILE:    "/home/player/tf2-archipelago/Scout.yaml",
		launcherv1.FileTarget_FILE_TARGET_INSTALL_ROOT:   "/home/player/tf2-archipelago",
		launcherv1.FileTarget_FILE_TARGET_GENERATED_SEED: "/home/player/tf2-archipelago/seed.zip",
	}[request.Msg.GetTarget()]
	s.fake.say("opened " + path)
	return connect.NewResponse(&launcherv1.ShowFileResponse{Path: path}), nil
}

func (s filesRPC) DownloadDebugBundle(_ context.Context, _ *connect.Request[launcherv1.DownloadDebugBundleRequest], stream *connect.ServerStream[launcherv1.DownloadDebugBundleResponse]) error {
	if err := stream.Send(&launcherv1.DownloadDebugBundleResponse{Filename: "tf2ap-debug.zip"}); err != nil {
		return err
	}
	return stream.Send(&launcherv1.DownloadDebugBundleResponse{Chunk: []byte("not a real zip")})
}

var _ = settings.Defaults

// ListFolder reads the machine it runs on, like the real one. The browser tests
// only ask that Up and a child move, which any folder can show.
func (s filesRPC) ListFolder(_ context.Context, request *connect.Request[launcherv1.ListFolderRequest]) (*connect.Response[launcherv1.ListFolderResponse], error) {
	path := request.Msg.GetPath()
	if path == "" {
		path = "/"
	}
	path = filepath.Clean(path)
	answer := &launcherv1.ListFolderResponse{Path: path}
	if parent := filepath.Dir(path); parent != path {
		answer.Parent = parent
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		// A folder that cannot be read is drawn empty rather than as a failure:
		// a permission dialog on the way to somewhere else is noise.
		//nolint:nilerr // An unreadable folder is an empty folder here.
		return connect.NewResponse(answer), nil
	}
	for _, entry := range entries {
		if entry.IsDir() {
			answer.Folders = append(answer.Folders, entry.Name())
		}
	}
	slices.Sort(answer.Folders)
	return connect.NewResponse(answer), nil
}

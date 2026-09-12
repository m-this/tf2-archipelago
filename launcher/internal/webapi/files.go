package webapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"time"

	"connectrpc.com/connect"

	"github.com/m-this/tf2-archipelago/launcher/internal/assets"
	"github.com/m-this/tf2-archipelago/launcher/internal/debugbundle"
	launcherv1 "github.com/m-this/tf2-archipelago/launcher/internal/gen/tf2ap/launcher/v1"
	"github.com/m-this/tf2-archipelago/launcher/internal/generate"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
	"github.com/m-this/tf2-archipelago/launcher/internal/winproc"
)

// bundleChunk is how much of the zip travels in one message. The bundle is a
// few megabytes and the connection is loopback, so this is only about keeping
// one message under Connect's default ceiling.
const bundleChunk = 64 << 10

// foldersMax bounds one answer. A folder with more subfolders than this is not
// one anybody is picking from by scrolling, and the typed path stays beside the
// picker for exactly that case.
const foldersMax = 2000

// FilePath answers where the player's copy of something is, making it first
// where making it is what the button means: the player file and the seed are
// written from the settings on the screen, not read from disk.
func (a *App) FilePath(ctx context.Context, target launcherv1.FileTarget) (string, error) {
	if target == launcherv1.FileTarget_FILE_TARGET_SETTINGS_FILE {
		return settings.Path()
	}
	s, err := a.DraftSettings()
	if err != nil {
		return "", err
	}
	switch target {
	case launcherv1.FileTarget_FILE_TARGET_PLAYER_FILE:
		if err := settings.CheckServerModsReady(s, a.readyServerMods()); err != nil {
			return "", err
		}
		return settings.WritePlayerFile(s, assets.ArchipelagoVersion)
	case launcherv1.FileTarget_FILE_TARGET_INSTALL_ROOT:
		return s.InstallRoot, nil
	case launcherv1.FileTarget_FILE_TARGET_GENERATED_SEED:
		if err := settings.CheckServerModsReady(s, a.readyServerMods()); err != nil {
			return "", err
		}
		result, err := generate.Run(ctx, generate.Options{
			Settings: s, AppDir: s.ArchipelagoDir,
			Apworld: assets.Apworld(), ArchipelagoVersion: assets.ArchipelagoVersion,
		})
		if err != nil {
			return "", err
		}
		return result.Archive, nil
	case launcherv1.FileTarget_FILE_TARGET_UNSPECIFIED, launcherv1.FileTarget_FILE_TARGET_SETTINGS_FILE:
	}
	return "", fmt.Errorf("no file target %d", target)
}

// FilesRPC is FilesService over one App.
type FilesRPC struct {
	App *App

	// open shows a path to the player. A field rather than a call so a test can
	// ask what was opened without a desktop; nil means winproc.Open.
	open func(string) error
}

// NewFilesRPC wires the service to the desktop's own handler.
func NewFilesRPC(app *App) FilesRPC { return FilesRPC{App: app, open: winproc.Open} }

// ShowFile hands a path to the desktop, because a browser cannot open a folder.
// It answers with what it opened either way, so a machine with no handler for a
// .yaml still tells the player where to look.
func (s FilesRPC) ShowFile(ctx context.Context, request *connect.Request[launcherv1.ShowFileRequest]) (*connect.Response[launcherv1.ShowFileResponse], error) {
	path, err := s.App.FilePath(ctx, request.Msg.GetTarget())
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if err := s.open(path); err != nil {
		s.App.Say("could not open %s: %v", path, err)
	}
	return connect.NewResponse(&launcherv1.ShowFileResponse{Path: path}), nil
}

// DownloadDebugBundle streams the zip. The first message names it and carries
// no bytes; every message after it carries bytes and no name.
//
//nolint:contextcheck // Bundle collection owns its bridge timeout.
func (s FilesRPC) DownloadDebugBundle(_ context.Context, _ *connect.Request[launcherv1.DownloadDebugBundleRequest], stream *connect.ServerStream[launcherv1.DownloadDebugBundleResponse]) error {
	path, err := debugbundle.Write(s.App.SettingsNow(), assets.Versions(), time.Now())
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	file, err := os.Open(path) //nolint:gosec // The path is the launcher's own bundle, not the browser's.
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	defer func() { _ = file.Close() }()

	if err := stream.Send(&launcherv1.DownloadDebugBundleResponse{Filename: filepath.Base(path)}); err != nil {
		return err
	}
	buffer := make([]byte, bundleChunk)
	for {
		read, err := file.Read(buffer)
		if read > 0 {
			if err := stream.Send(&launcherv1.DownloadDebugBundleResponse{Chunk: buffer[:read]}); err != nil {
				return err
			}
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return connect.NewError(connect.CodeInternal, err)
		}
	}
}

/*
ListFolder reads one folder for the Browse picker.

Folders only, and no attempt to hide anything: this is the player's own machine
and their own launcher, so there is nothing here to keep them out of. What it
does do is refuse to answer with a file, because every Browse row in form names
a folder and a picker offering a file would be offering an answer no row holds.

An unreadable folder is not an error the player has to solve. It answers with
the folder and no children, and the picker draws it empty: a permission-denied
dialog on the way to somewhere else is noise.
*/
func (s FilesRPC) ListFolder(_ context.Context, request *connect.Request[launcherv1.ListFolderRequest]) (*connect.Response[launcherv1.ListFolderResponse], error) {
	path := request.Msg.GetPath()
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		path = home
	}
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("%q is not a full path", path))
	}

	answer := &launcherv1.ListFolderResponse{Path: path, Parent: parentOf(path)}
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
		if len(answer.Folders) == foldersMax {
			break
		}
	}
	slices.Sort(answer.Folders)
	return connect.NewResponse(answer), nil
}

// parentOf answers with the folder above, and empty at the top: filepath.Dir
// returns its argument at a root, and a picker that offered Up there would go
// nowhere for ever.
func parentOf(path string) string {
	parent := filepath.Dir(path)
	if parent == path {
		return ""
	}
	return parent
}

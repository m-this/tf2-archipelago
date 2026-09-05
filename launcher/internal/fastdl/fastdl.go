// Package fastdl serves the game's content tree over HTTP, so a client that
// joins without a community map fetches it from here rather than through the
// game server's in-band transfer. That transfer can reach the end of a large
// packed BSP without the client accepting it, and the map restarts forever.
//
// Everything is on the same machine as the game server and reachable exactly
// where the game port is: a player who can join can download, and nobody
// else can. There is no port forwarding here and no relay.
package fastdl

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// DefaultPort is one above the game port's neighbourhood so it never collides
// with srcds's own TCP listener on the game port.
const DefaultPort = 27080

// urlPrefix is what sv_downloadurl ends in. The client appends the file's
// game-relative path: <url>/maps/mvm_x.bsp.bz2, then <url>/maps/mvm_x.bsp.
const urlPrefix = "/tf/"

// contentDirs are the only directories a client is served from. The game tree
// also holds cfg/, with the rcon password in it, and addons/, with the
// plugins: neither is anything a client downloads, so neither is reachable.
var contentDirs = map[string]bool{
	"maps": true, "materials": true, "models": true, "sound": true,
	"particles": true, "resource": true,
}

// Timeouts bound one slow client without cutting off a slow one. A packed BSP
// is tens of megabytes, and a friend on a poor line can spend longer than five
// minutes on it, so the write gets a quarter of an hour; the idle timeout is
// what a blip mid-download has to fit in before the client asks again.
const (
	readHeaderTimeout = 10 * time.Second
	writeTimeout      = 15 * time.Minute
	idleTimeout       = 2 * time.Minute
)

// URL is the sv_downloadurl value for a server on host, serving on port.
func URL(host string, port int) string {
	return "http://" + net.JoinHostPort(host, strconv.Itoa(port)) + strings.TrimSuffix(urlPrefix, "/")
}

// Handler serves files under gameDir, the game's tf/ directory. Only regular
// files inside contentDirs answer; a directory, a path that climbs out, or
// anything in another directory is a 404, indistinguishable from a file that
// is not there.
func Handler(gameDir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		relative, ok := contentPath(r.URL.Path)
		if !ok {
			http.NotFound(w, r)
			return
		}
		target := filepath.Join(gameDir, filepath.FromSlash(relative))
		info, err := os.Stat(target)
		if err != nil || !info.Mode().IsRegular() {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, target)
	})
}

// contentPath turns a request path into a game-relative file path, or
// reports that the request names nothing this server hands out.
func contentPath(requestPath string) (string, bool) {
	if !strings.HasPrefix(requestPath, urlPrefix) {
		return "", false
	}
	cleaned := path.Clean("/" + strings.TrimPrefix(requestPath, urlPrefix))
	relative := strings.TrimPrefix(cleaned, "/")
	dir, _, found := strings.Cut(relative, "/")
	if !found || !contentDirs[dir] {
		return "", false
	}
	return relative, true
}

// Serve listens on listen until ctx ends. It returns nil on a clean stop and
// the listen error when the port is taken, which is worth a line in the log
// and not worth stopping the game server over.
func Serve(ctx context.Context, gameDir, listen string) error {
	server := &http.Server{
		Addr:              listen,
		Handler:           Handler(gameDir),
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
	done := make(chan error, 1)
	go func() { done <- server.ListenAndServe() }()
	select {
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.WithoutCancel(ctx), idleTimeout)
		defer cancel()
		_ = server.Shutdown(shutdown)
		return nil
	case err := <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("fastdl on %s: %w", listen, err)
	}
}

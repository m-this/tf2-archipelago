package runtime

import (
	"context"
	"time"

	"github.com/m-this/tf2-archipelago/launcher/internal/fastdl"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
	"github.com/m-this/tf2-archipelago/launcher/internal/tailscalefastdl"
)

const tailscaleSetupTimeout = 20 * time.Second

// prepareTailscaleFastDL returns the settings for this start. It changes only
// the FastDL endpoint: how players reach SRCDS remains exactly as the operator
// selected it. A setup failure is non-fatal because server.cfg keeps direct
// game-server downloads enabled.
func prepareTailscaleFastDL(parent context.Context, s settings.Settings, say func(string)) settings.Settings {
	if !s.TailscaleFastDL {
		return s
	}

	// Funnel publishes the launcher's listener through its public HTTPS
	// endpoint. The listener is bound to loopback below, not to the LAN.
	if s.FastDLPort <= 0 {
		s.FastDLPort = fastdl.DefaultPort
	}
	s.SrcdsDownloadURL = ""

	ctx, cancel := context.WithTimeout(parent, tailscaleSetupTimeout)
	defer cancel()
	result, err := tailscalefastdl.Configure(ctx, s.FastDLPort)
	if err != nil {
		s.FastDLPort = 0
		say("Tailscale Funnel FastDL is unavailable, so maps will come from the game server itself: " + err.Error() +
			". Open Settings > Networking and run Set up / check Tailscale Funnel")
		return s
	}

	s.SrcdsDownloadURL = result.URL
	say("public Tailscale Funnel FastDL ready at " + result.URL + "; players do not need Tailscale")
	return s
}

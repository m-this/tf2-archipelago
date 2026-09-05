package runtime

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/m-this/tf2-archipelago/launcher/internal/fastdl"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
	"github.com/m-this/tf2-archipelago/launcher/internal/tailscalefastdl"
)

const (
	tailscaleSetupTimeout   = 20 * time.Second
	tailscaleCleanupTimeout = 5 * time.Second
)

// TailscaleFastDLStartError stops Start when the operator enabled Funnel but
// the launcher cannot provide it. ApprovalURL is present for first-time or
// renewed tailnet authorization.
type TailscaleFastDLStartError struct {
	Err         error
	ApprovalURL string
}

func (e *TailscaleFastDLStartError) Error() string {
	action := "restore Tailscale or run -setup-funnel, then try Start again"
	if e.ApprovalURL != "" {
		action = "approve Funnel in the browser, then try Start again"
	}
	return fmt.Sprintf("tailscale Funnel FastDL is enabled but unavailable: %v; %s; the server was not started", e.Err, action)
}

func (e *TailscaleFastDLStartError) Unwrap() error { return e.Err }

type (
	tailscaleConfigure func(context.Context, int) (tailscalefastdl.Result, error)
	tailscaleDisable   func(context.Context) error
)

// prepareTailscaleFastDL verifies and applies the Funnel route for this run.
// It changes only the FastDL endpoint; how players reach SRCDS remains exactly
// as selected. Failure blocks Start because silently dropping an enabled
// download service would violate the operator's saved intent.
func prepareTailscaleFastDL(parent context.Context, s settings.Settings, say func(string)) (settings.Settings, error) {
	return prepareTailscaleFastDLWith(parent, s, say, tailscalefastdl.Configure)
}

// stopTailscaleFastDL removes only the route this launcher owns. Cleanup is
// deliberately best-effort: a Tailscale problem must not hide the reason the
// game server stopped, and its short independent context still works after a
// Ctrl-C has cancelled the run's context.
func stopTailscaleFastDL(parent context.Context, s settings.Settings, say func(context.Context, string)) {
	stopTailscaleFastDLWith(parent, s, say, tailscalefastdl.Disable)
}

func stopTailscaleFastDLWith(
	parent context.Context, s settings.Settings, say func(context.Context, string), disable tailscaleDisable,
) {
	if !s.TailscaleFastDL {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), tailscaleCleanupTimeout)
	defer cancel()
	if err := disable(ctx); err != nil {
		say(ctx, "could not remove the public Tailscale Funnel FastDL route: "+err.Error())
		return
	}
	say(ctx, "removed the public Tailscale Funnel FastDL route")
}

func prepareTailscaleFastDLWith(parent context.Context, s settings.Settings, say func(string), configure tailscaleConfigure) (settings.Settings, error) {
	if !s.TailscaleFastDL {
		return s, nil
	}

	// Funnel publishes the launcher's listener through its public HTTPS
	// endpoint. The listener is bound to loopback below, not to the LAN.
	if s.FastDLPort <= 0 {
		s.FastDLPort = fastdl.DefaultPort
	}
	s.SrcdsDownloadURL = ""

	ctx, cancel := context.WithTimeout(parent, tailscaleSetupTimeout)
	defer cancel()
	result, err := configure(ctx, s.FastDLPort)
	if err != nil {
		startErr := &TailscaleFastDLStartError{Err: err}
		if approval, ok := errors.AsType[*tailscalefastdl.ApprovalRequiredError](err); ok {
			startErr.ApprovalURL = approval.URL
		}
		return s, startErr
	}

	s.SrcdsDownloadURL = result.URL
	say("public Tailscale Funnel FastDL ready at " + result.URL + "; players do not need Tailscale")
	return s, nil
}

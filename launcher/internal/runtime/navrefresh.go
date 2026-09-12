package runtime

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/m-this/tf2-archipelago/launcher/internal/rcon"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

const navigationRefreshDelay = 3 * time.Second

var (
	cbaseNPCExtensionLine = regexp.MustCompile(`(?m)^\[([0-9]+)\].*\bCBaseNPC\b`)
	emptyNavigationLine   = regexp.MustCompile(`(?m)^0 nav areas on this map\r?$`)
)

type rconExecutor interface {
	Exec(string) (string, error)
}

func isMVMMapChange(line Line) bool {
	return line.Source == "srcds" && strings.Contains(strings.ToLower(line.Text), "mapchange to mvm_")
}

func refreshDefenderNavigation(client rconExecutor) (bool, error) {
	report, err := client.Exec("sm_dump_nest")
	if err != nil {
		return false, err
	}
	if !emptyNavigationLine.MatchString(report) {
		return false, nil
	}

	extensions, err := client.Exec("sm exts list")
	if err != nil {
		return false, err
	}
	match := cbaseNPCExtensionLine.FindStringSubmatch(extensions)
	if len(match) != 2 {
		return false, fmt.Errorf("CBaseNPC is not in SourceMod's extension list")
	}

	if _, err := client.Exec("sm plugins unload tf2_defenderbots"); err != nil {
		return false, fmt.Errorf("unload defender bots: %w", err)
	}
	loaded := false
	defer func() {
		if !loaded {
			_, _ = client.Exec("sm plugins load tf2_defenderbots")
		}
	}()
	if _, err := client.Exec("sm exts reload " + match[1]); err != nil {
		return false, fmt.Errorf("reload CBaseNPC: %w", err)
	}
	if _, err := client.Exec("sm plugins load tf2_defenderbots"); err != nil {
		return false, fmt.Errorf("reload defender bots: %w", err)
	}
	loaded = true
	return true, nil
}

func (s *Supervisor) watchNavigationRefresh(ctx context.Context, current settings.Settings, refresh <-chan struct{}) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-refresh:
		}

		timer := time.NewTimer(navigationRefreshDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}

		var client *rcon.Client
		var err error
		// Address discovery owns a short hostname-lookup timeout; the RCON dial
		// itself still stops immediately when the supervisor context is cancelled.
		for _, address := range RconAddresses(current) { //nolint:contextcheck // Address discovery owns its bounded timeout.
			client, err = rcon.DialContext(ctx, address, current.SrcdsRconPw)
			if err == nil {
				break
			}
		}
		if err != nil {
			s.emit("could not refresh defender navigation after the map loaded: " + err.Error())
			continue
		}
		refreshed, err := func() (bool, error) {
			defer func() { _ = client.Close() }()
			return refreshDefenderNavigation(client)
		}()
		if err != nil {
			s.emit("could not refresh defender navigation after the map loaded: " + err.Error())
			continue
		}
		if refreshed {
			s.emit("refreshed defender navigation after the map loaded")
		}
	}
}

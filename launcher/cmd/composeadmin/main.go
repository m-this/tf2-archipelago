// Command composeadmin serves the launcher UI beside a Compose-managed TF2
// server. It observes the bridge and reaches SRCDS over their shared loopback
// namespace; Docker remains the lifecycle and configuration authority.
package main

import (
	"log/slog"
	"net"
	"os"
	"strconv"

	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
	"github.com/m-this/tf2-archipelago/launcher/internal/webapi"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("admin UI stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	s := settings.ApplyEnv(settings.ComposeDefaults())
	authority := env("TF2AP_ADMIN_AUTHORITY", "127.0.0.1:8477")
	logger.Info("TF2 Archipelago admin UI", "url", "http://"+authority)
	logJoinAddress(logger, s)
	logger.Info("Docker Compose owns server lifecycle; the Settings tab writes .env")
	return webapi.Run(s, logger, webapi.Options{
		Address:           env("TF2AP_ADMIN_LISTEN", "0.0.0.0:8477"),
		Authority:         authority,
		Attached:          true,
		AttachedEnvFile:   env("TF2AP_ADMIN_ENV_FILE", "/config/compose.env"),
		TailscaleSocket:   env("TF2AP_TAILSCALE_SOCKET", "/run/tf2ap-fastdl/tailscaled.sock"),
		TailscaleHostname: env("TAILSCALE_HOSTNAME", "tf2-fastdl"),
		AttachedLogs: []webapi.AttachedLog{
			{Path: env("TF2AP_ADMIN_SRCDS_LOG", "/srv/tf-dedicated/tf/console.log"), Source: "srcds"},
			{Path: env("TF2AP_ADMIN_BRIDGE_LOG", "/srv/bridge/bridge.log"), Source: "bridge"},
		},
	})
}

// logJoinAddress says what a player types into the game. The address is the one
// TF2AP_JOIN_HOST names; left unset it is loopback, which only the machine
// running the stack can use, so that case says so instead of printing it as if
// it were an answer.
func logJoinAddress(logger *slog.Logger, s settings.Settings) {
	host := s.SrcdsJoinHost
	if host == "" {
		host = "127.0.0.1"
	}
	port := strconv.Itoa(s.SrcdsPort)
	logger.Info("players join with", "console", "connect "+net.JoinHostPort(host, port))
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		logger.Warn("TF2AP_JOIN_HOST is not set, so that address works on this machine only; set it in .env to this server's public IP or DNS name")
	}
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

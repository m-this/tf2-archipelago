// Package config reads the bridge's configuration from the environment.
//
// Environment only, no config file: the bridge runs in a container next to a
// compose file that already owns every other setting in this stack, and two
// places to look is one too many.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
)

// Config is everything the bridge needs to start. Nothing here changes while
// it runs.
type Config struct {
	// ArchipelagoURL is ws:// or wss://, host and port included.
	ArchipelagoURL string
	SlotName       string
	Password       string

	// Listen is the plugin-facing address. Loopback, always: srcds and the
	// bridge share a network namespace and nothing else may reach it.
	Listen string

	// StatePath is the file holding the check queue and the unlock set.
	StatePath string

	// PollTimeout is how long GET /grants is held open before answering 204.
	// Short enough that the plugin's own timeout never fires first.
	PollTimeout time.Duration
}

// Load reads the environment. Every value has a default that works inside the
// compose file; the ones that cannot have a sane default are checked here
// rather than at first use.
func Load() (Config, error) {
	host := env("AP_HOST", "archipelago")
	port := env("AP_PORT", "38281")
	scheme := "ws"
	tls, err := boolEnv("AP_TLS", false)
	if err != nil {
		return Config{}, err
	}
	if tls {
		scheme = "wss"
	}

	timeout, err := durationEnv("BRIDGE_POLL_TIMEOUT", 25*time.Second)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		ArchipelagoURL: (&url.URL{Scheme: scheme, Host: host + ":" + port}).String(),
		SlotName:       env("AP_SLOT_NAME", "tf2"),
		Password:       os.Getenv("AP_PASSWORD"),
		Listen:         env("BRIDGE_LISTEN", "127.0.0.1:24680"),
		StatePath:      env("BRIDGE_STATE", "/data/bridge.json"),
		PollTimeout:    timeout,
	}
	if cfg.SlotName == "" {
		return Config{}, fmt.Errorf("AP_SLOT_NAME is empty")
	}
	if _, err := strconv.Atoi(port); err != nil {
		return Config{}, fmt.Errorf("AP_PORT %q is not a number", port)
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if value, set := os.LookupEnv(key); set && value != "" {
		return value
	}
	return fallback
}

func boolEnv(key string, fallback bool) (bool, error) {
	value, set := os.LookupEnv(key)
	if !set || value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s %q is not a boolean", key, value)
	}
	return parsed, nil
}

func durationEnv(key string, fallback time.Duration) (time.Duration, error) {
	value, set := os.LookupEnv(key)
	if !set || value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s %q is not a duration", key, value)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be positive, got %s", key, parsed)
	}
	return parsed, nil
}

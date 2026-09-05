package main

import (
	"testing"

	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

func TestFastDLStatus(t *testing.T) {
	tests := []struct {
		name string
		s    settings.Settings
		want string
	}{
		{"off", settings.Settings{}, "off"},
		{"launcher", settings.Settings{FastDLPort: 27080}, "launcher HTTP server on port 27080"},
		{"funnel", settings.Settings{FastDLPort: 27080, TailscaleFastDL: true}, "Tailscale Funnel on local port 27080"},
		{"external", settings.Settings{FastDLPort: 27080, SrcdsDownloadURL: "https://cdn.example/tf"}, "https://cdn.example/tf"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := fastDLStatus(test.s); got != test.want {
				t.Fatalf("fastDLStatus() = %q, want %q", got, test.want)
			}
		})
	}
}

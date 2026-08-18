package settings

import "testing"

func TestParseRoomAcceptsWhatPlayersPaste(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want Room
	}{
		{"archipelago.gg:12345", Room{"archipelago.gg", 12345, true}},
		{"  archipelago.gg:12345  ", Room{"archipelago.gg", 12345, true}},
		{"wss://archipelago.gg:12345", Room{"archipelago.gg", 12345, true}},
		{"https://archipelago.gg:12345/", Room{"archipelago.gg", 12345, true}},
		{"ws://archipelago.gg:12345", Room{"archipelago.gg", 12345, false}},
		{"localhost:38281", Room{"localhost", 38281, false}},
		{"192.168.1.10:38281", Room{"192.168.1.10", 38281, false}},
		{"[::1]:38281", Room{"::1", 38281, false}},
	} {
		got, err := ParseRoom(tc.in)
		if err != nil {
			t.Errorf("ParseRoom(%q): %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseRoom(%q) = %+v, want %+v", tc.in, got, tc.want)
		}
	}
}

func TestParseRoomRejectsWhatCannotWork(t *testing.T) {
	for _, in := range []string{"", "   ", "archipelago.gg", "archipelago.gg:", ":12345", "archipelago.gg:port", "archipelago.gg:99999"} {
		if got, err := ParseRoom(in); err == nil {
			t.Errorf("ParseRoom(%q) accepted %+v", in, got)
		}
	}
}

func TestRoomString(t *testing.T) {
	if got := (Room{Host: "archipelago.gg", Port: 12345}).String(); got != "archipelago.gg:12345" {
		t.Errorf("got %q", got)
	}
	if got := (Room{}).String(); got != "" {
		t.Errorf("an unset room rendered as %q", got)
	}
}

func TestNewRconPasswordIsLongAndDifferentEveryTime(t *testing.T) {
	first, err := NewRconPassword()
	if err != nil {
		t.Fatalf("NewRconPassword: %v", err)
	}
	second, _ := NewRconPassword()
	if len(first) < 20 {
		t.Errorf("password is %d characters: %q", len(first), first)
	}
	if first == second {
		t.Error("two calls returned the same password")
	}
}

func TestApplyEnvReadsAWholeRoom(t *testing.T) {
	t.Setenv("AP_ROOM", "ws://localhost:38281")
	got := ApplyEnv(Defaults())
	if got.APHost != "localhost" || got.APPort != 38281 || got.APTls {
		t.Errorf("AP_ROOM gave %s:%d tls=%v", got.APHost, got.APPort, got.APTls)
	}
}

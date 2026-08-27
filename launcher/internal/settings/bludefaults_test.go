package settings

import "testing"

/*
	A settings file written before a field existed must not leave it at zero.

The window's number boxes for the robot scales take a minimum of ten. walk
refuses a value below the minimum and a refused widget takes the whole settings
dialog with it, so a zero here is not a wrong setting: it is a launcher nobody
can configure. Reported as the settings window no longer opening.

Zero is also wrong on its own terms. Nothing said is not nought per cent.
*/
func TestAnOlderFileGetsTheRobotScales(t *testing.T) {
	// A file from before the Balancing page existed: no srcds_blu_* at all.
	old := []byte(`{"srcds_hostname":"a server","srcds_bot_team_size":6}`)

	s, err := parse(old)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		got  int
	}{
		{"damage", s.SrcdsBluDamagePct},
		{"health", s.SrcdsBluHealthPct},
		{"speed", s.SrcdsBluSpeedPct},
	} {
		if test.got != 100 {
			t.Errorf("%s = %d, want 100: the window refuses anything under 10", test.name, test.got)
		}
	}
}

// A file that does name them keeps what it says, including the measured dose.
func TestAFileThatSetsTheScalesKeepsThem(t *testing.T) {
	s, err := parse([]byte(`{"srcds_blu_damage_pct":50,"srcds_blu_speed_pct":80}`))
	if err != nil {
		t.Fatal(err)
	}
	if s.SrcdsBluDamagePct != 50 {
		t.Errorf("damage = %d, want 50", s.SrcdsBluDamagePct)
	}
	if s.SrcdsBluSpeedPct != 80 {
		t.Errorf("speed = %d, want 80", s.SrcdsBluSpeedPct)
	}
	// The one it did not mention still gets a value the window will take.
	if s.SrcdsBluHealthPct != 100 {
		t.Errorf("health = %d, want 100", s.SrcdsBluHealthPct)
	}
}

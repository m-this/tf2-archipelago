package settings

import "testing"

/*
	A settings file written before a field existed must not leave it at zero.

The window's number box for the robot health scale takes a minimum of ten. walk
refuses a value below the minimum and a refused widget takes the whole settings
dialog with it, so a zero here is not a wrong setting: it is a launcher nobody
can configure. Reported as the settings window no longer opening.

Zero is also wrong on its own terms. Nothing said is not nought per cent.
*/
func TestAnOlderFileGetsTheRobotScale(t *testing.T) {
	// A file from before the Balancing page existed: no srcds_blu_* at all.
	old := []byte(`{"srcds_hostname":"a server","srcds_bot_team_size":6}`)

	s, err := parse(old)
	if err != nil {
		t.Fatal(err)
	}
	if s.SrcdsBluHealthPct != 100 {
		t.Errorf("health = %d, want 100: the window refuses anything under 10", s.SrcdsBluHealthPct)
	}
}

// A file that does name it keeps what it says.
func TestAFileThatSetsTheScaleKeepsIt(t *testing.T) {
	s, err := parse([]byte(`{"srcds_blu_health_pct":50}`))
	if err != nil {
		t.Fatal(err)
	}
	if s.SrcdsBluHealthPct != 50 {
		t.Errorf("health = %d, want 50", s.SrcdsBluHealthPct)
	}
}

/*
	The damage and speed scales were removed after they were measured.

Damage bent a mission and speed showed nothing, and one lever that works beats
three that need explaining. A file still carrying the old keys must load rather
than refuse, since every settings file written while they existed has them.
*/
func TestAFileWithTheRemovedScalesStillLoads(t *testing.T) {
	s, err := parse([]byte(`{"srcds_blu_damage_pct":50,"srcds_blu_speed_pct":80,"srcds_blu_health_pct":60}`))
	if err != nil {
		t.Fatal(err)
	}
	if s.SrcdsBluHealthPct != 60 {
		t.Errorf("health = %d, want 60", s.SrcdsBluHealthPct)
	}
}

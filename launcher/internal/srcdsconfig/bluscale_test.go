package srcdsconfig

import (
	"strings"
	"testing"

	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

// The Balancing page writes percentages and the mod reads floats, with 1.0
// meaning off. A page nobody has touched must leave the mission alone.
func TestTheRobotScalesReachServerCfg(t *testing.T) {
	for _, test := range []struct {
		name string
		s    settings.Settings
		want []string
	}{
		{
			"untouched settings leave the mission alone",
			settings.Defaults(),
			[]string{
				"sm_redbots_manager_blu_damage_scale 1.0",
				"sm_redbots_manager_blu_health_scale 1.0",
				"sm_redbots_manager_blu_speed_scale 1.0",
			},
		},
		{
			"a zero is somebody who never set it, not harmless robots",
			settings.Settings{},
			[]string{"sm_redbots_manager_blu_damage_scale 1.0"},
		},
		{
			"the measured damage dose",
			settings.Settings{SrcdsBluDamagePct: 50},
			[]string{"sm_redbots_manager_blu_damage_scale 0.50"},
		},
		{
			"below the mod's floor is clamped to it",
			settings.Settings{SrcdsBluSpeedPct: 5},
			[]string{"sm_redbots_manager_blu_speed_scale 0.10"},
		},
	} {
		got, err := RenderServerCfg(test.s)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range test.want {
			if !strings.Contains(got, want) {
				t.Errorf("%s: missing %q", test.name, want)
			}
		}
	}
}

package settings

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"go.yaml.in/yaml/v3"
)

const deployDir = "../../../deploy"

// composeEnvironments is each service's environment, by name, across the files
// a stack and `make seed` load. A release merges the same two.
func composeEnvironments(t *testing.T) map[string]map[string]bool {
	t.Helper()
	services := map[string]map[string]bool{}
	for _, file := range []string{"compose.yml", "compose.seed.yml"} {
		body, err := os.ReadFile(filepath.Join(deployDir, file))
		if err != nil {
			t.Fatal(err)
		}
		var parsed struct {
			Services map[string]struct {
				Environment map[string]any `yaml:"environment"`
			} `yaml:"services"`
		}
		if err := yaml.Unmarshal(body, &parsed); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		for name, service := range parsed.Services {
			services[name] = map[string]bool{}
			for key := range service.Environment {
				services[name][key] = true
			}
		}
	}
	return services
}

/*
Every variable reaches the services that read it, and no other.

The admin page writes every variable into .env, and a value Compose does not
hand to a container is a choice that quietly does nothing: playeryaml falls
back to Defaults() and the seed is generated without it. Twelve run options
were lost that way, because each service carries its own list and the lists
were copied by hand.
*/
func TestComposePassesEveryVariableToItsReaders(t *testing.T) {
	environments := composeEnvironments(t)
	owned := map[string]bool{}
	for _, v := range EnvVars {
		owned[v.Name] = true
		for _, service := range v.Services {
			if _, ok := environments[service]; !ok {
				t.Errorf("%s names service %q, which no compose file has", v.Name, service)
			}
		}
	}
	for service, environment := range environments {
		// It loads the whole of .env, so it has no list to keep in step.
		if service == "admin" {
			continue
		}
		for _, v := range EnvVars {
			reads := slices.Contains(v.Services, service)
			if reads && !environment[v.Name] {
				t.Errorf("%s reads %s, but compose does not pass it", service, v.Name)
			}
			if !reads && environment[v.Name] {
				t.Errorf("compose passes %s to %s, which EnvVars does not say reads it", v.Name, service)
			}
		}
	}
}

/*
A variable no service reads is a decision, made here with its reason.

Without this list, a new option added to ApplyEnv and composeenv with no
Services is exactly the bug the test above exists for, and it would pass.
*/
func TestEveryVariableOutsideTheContainersSaysWhy(t *testing.T) {
	launcherOnly := map[string]string{
		"TF2AP_INSTALL_ROOT":          "the admin service sets it to the mounted volume",
		"TF2AP_COMMUNITY_CONTENT_DIR": "the admin service sets it to the mounted overlay",
		"TF2AP_ARCHIPELAGO_DIR":       "the launcher's own Archipelago install",
		"AP_ROOM":                     "a launcher spelling of AP_HOST, AP_PORT and AP_TLS",
		"AP_ROOM_URL":                 "shown by the admin page only",
		"TF2AP_JOIN_HOST":             "shown by the admin page only",
		"SRCDS_BOT_TEAM_PRESETS":      "teams kept by the admin page, not played",
		"SRCDS_BOT_CARD_ROLLS":        "read by the admin page's live bot view",
		"BRIDGE_METRICS_PORT":         "compose interpolates it into the port and BRIDGE_METRICS_LISTEN",
	}
	for _, v := range EnvVars {
		_, listed := launcherOnly[v.Name]
		if len(v.Services) == 0 && !listed {
			t.Errorf("%s reaches no container; name its readers in EnvVars, or say why here", v.Name)
		}
		if len(v.Services) > 0 && listed {
			t.Errorf("%s has readers now; take it off this list", v.Name)
		}
	}
}

// A variable a container reads is one an operator can set, so .env.example
// names it.
func TestEnvExampleNamesEveryVariableAContainerReads(t *testing.T) {
	body, err := os.ReadFile(filepath.Join(deployDir, ".env.example"))
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range EnvVars {
		if len(v.Services) == 0 {
			continue
		}
		line := regexp.MustCompile(`(?m)^#?\s*` + v.Name + `=`)
		if !line.Match(body) {
			t.Errorf(".env.example does not name %s", v.Name)
		}
	}
}

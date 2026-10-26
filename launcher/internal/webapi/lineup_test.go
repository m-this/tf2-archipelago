package webapi

import (
	"errors"
	"strings"
	"testing"
)

// The plugin refuses a lineup by answering, not by failing the call: the RCON
// command succeeds and the reply says the cards are unchanged. A Save that only
// sent the commands told the player nothing (apw-fjv8).
func TestALineupStopsAtTheFirstRefusalAndSaysIt(t *testing.T) {
	for _, refusal := range []string{
		"[AP cards] Defender manager is unavailable; cards unchanged.",
		"[AP cards] Cannot read the new lineup; cards unchanged.",
		`Unknown command "sm_ap_botcards_reconcile"`,
		"[SM] No card loadout file to reload.",
	} {
		var sent []string
		err := applyLineup(func(command string) (string, error) {
			sent = append(sent, command)
			if command == "sm_ap_botcards_reconcile" {
				return refusal + "\n", nil
			}
			return "", nil
		}, []string{"sm_redbots_rebind_seats", "sm_ap_botcards_reconcile", "say the team changed"})
		if err == nil || !strings.Contains(err.Error(), refusal) {
			t.Errorf("%q was answered %v", refusal, err)
		}
		if len(sent) != 2 {
			t.Errorf("%q: sent %v, want nothing after the refusal", refusal, sent)
		}
	}
}

func TestALineupTheGameTakesIsApplied(t *testing.T) {
	var sent []string
	commands := []string{"sm_redbots_rebind_seats", "sm_ap_botcards_reconcile"}
	err := applyLineup(func(command string) (string, error) {
		sent = append(sent, command)
		return "[AP cards] Freed 1 seat.", nil
	}, commands)
	if err != nil || len(sent) != len(commands) {
		t.Fatalf("answered %v after sending %v", err, sent)
	}

	lost := errors.New("connection reset")
	err = applyLineup(func(string) (string, error) { return "", lost }, commands)
	if !errors.Is(err, lost) {
		t.Fatalf("a dropped connection answered %v", err)
	}
}

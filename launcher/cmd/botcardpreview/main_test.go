package main

import (
	"os"
	"testing"
)

// The browser reads the catalogue out of this file and nowhere else, so a
// card edited in Go and not regenerated is a card the two sides disagree about.
func TestCommittedCatalogueIsCurrent(t *testing.T) {
	want, err := render()
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile("../../web/src/bot-cards/cards.generated.ts")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatal("cards.generated.ts is behind the Go catalogue: run go generate ./launcher/internal/botcards")
	}
}

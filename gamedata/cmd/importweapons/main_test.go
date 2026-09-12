package main

import (
	"slices"
	"testing"
)

func TestGroupUsesInheritedWeaponNameForDecoratedDefinitions(t *testing.T) {
	weapons := group([]candidate{
		{DefIndex: 13, Name: "Scattergun", Slot: "primary", ItemClass: "tf_weapon_scattergun"},
		{DefIndex: 15001, Name: "Scattergun", Slot: "primary", ItemClass: "tf_weapon_scattergun"},
		{DefIndex: 15029, Name: "Scattergun", Slot: "primary", ItemClass: "tf_weapon_scattergun"},
		{DefIndex: 15157, Name: "Scattergun", Slot: "primary", ItemClass: "tf_weapon_scattergun"},
	})

	if len(weapons) != 1 {
		t.Fatalf("group returned %d weapons, want one Scattergun", len(weapons))
	}
	if got, want := weapons[0].DefIndexes, []int{13, 15001, 15029, 15157}; !slices.Equal(got, want) {
		t.Fatalf("Scattergun definitions = %v, want %v", got, want)
	}
}

func TestPaintKitPrototypeNumberIsNotTreatedAsAWeaponDefinition(t *testing.T) {
	weapons := group([]candidate{
		{DefIndex: 29, Name: "Medi Gun", Slot: "secondary", ItemClass: "tf_weapon_medigun"},
		// In the real schema Backcountry Blaster has paint-kit prototype 29.
		// Its inherited name, not that unrelated number, makes it a Scattergun.
		{DefIndex: 13, Name: "Scattergun", Slot: "primary", ItemClass: "tf_weapon_scattergun"},
		{DefIndex: 15029, Name: "Scattergun", Slot: "primary", ItemClass: "tf_weapon_scattergun"},
	})

	for _, weapon := range weapons {
		if weapon.Name == "Medi Gun" && slices.Contains(weapon.DefIndexes, 15029) {
			t.Fatal("Backcountry Blaster was attached to the weapon whose definition matches its paint-kit prototype")
		}
		if weapon.Name == "Scattergun" && !slices.Contains(weapon.DefIndexes, 15029) {
			t.Fatal("Backcountry Blaster was not attached to Scattergun")
		}
	}
}

func TestNamedFestiveAndBotkillerDefinitionsJoinTheBaseWeapon(t *testing.T) {
	weapons := group([]candidate{
		{DefIndex: 13, Name: "Scattergun", Slot: "primary", ItemClass: "tf_weapon_scattergun"},
		{DefIndex: 669, Name: "Festive Scattergun", Slot: "primary", ItemClass: "tf_weapon_scattergun"},
		{DefIndex: 799, Name: "Silver Botkiller Scattergun Mk.I", Slot: "primary", ItemClass: "tf_weapon_scattergun"},
	})

	if len(weapons) != 1 {
		t.Fatalf("group returned %d weapons, want one Scattergun", len(weapons))
	}
	if got, want := weapons[0].DefIndexes, []int{13, 669, 799}; !slices.Equal(got, want) {
		t.Fatalf("Scattergun definitions = %v, want %v", got, want)
	}
}

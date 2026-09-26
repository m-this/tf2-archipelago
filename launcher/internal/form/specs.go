package form

import "strings"

/*
	Specs is every row the launcher offers, resolved for one state.

A function rather than a package variable, because two pages have rows that are
not fixed. The Missions page has one per mission and which missions exist
depends on the community asset packs on disk; the Loadouts page has one per slot
and the Spy has three where everybody else has three different ones.

Order within a page is the order the page offers, and Tabs is the order of the
pages. Both are declared rather than derived, so moving a row does not silently
move a page.
*/
func Specs(s State, env Env) []Spec {
	specs := runSpecs(s, env)
	return append(specs, botSpecs(s, env)...)
}

// trim is what a text row does to what was typed. A path or a slot name with a
// space on the end is a value nobody meant and one the server would refuse
// later, somewhere less obvious.
func trim(v string) string { return strings.TrimSpace(v) }

package gamedata

// ClassID identifies one mercenary class. Explicit literals, append-only,
// never reused: these ids reach seeds through the class items.
type ClassID uint8

const (
	ClassScout    ClassID = 1
	ClassSoldier  ClassID = 2
	ClassPyro     ClassID = 3
	ClassDemoman  ClassID = 4
	ClassHeavy    ClassID = 5
	ClassEngineer ClassID = 6
	ClassMedic    ClassID = 7
	ClassSniper   ClassID = 8
	ClassSpy      ClassID = 9
)

// Class is one mercenary class. Key is what crosses the wire to the plugin,
// which maps it to TF2's own class enum.
type Class struct {
	ID   ClassID
	Key  string
	Name string
}

// Classes is the nine classes, in the order the class selection menu lists them.
var Classes = []Class{
	{ClassScout, "scout", "Scout"},
	{ClassSoldier, "soldier", "Soldier"},
	{ClassPyro, "pyro", "Pyro"},
	{ClassDemoman, "demoman", "Demoman"},
	{ClassHeavy, "heavy", "Heavy"},
	{ClassEngineer, "engineer", "Engineer"},
	{ClassMedic, "medic", "Medic"},
	{ClassSniper, "sniper", "Sniper"},
	{ClassSpy, "spy", "Spy"},
}

var classesByID = indexClasses()

func indexClasses() map[ClassID]Class {
	byID := make(map[ClassID]Class, len(Classes))
	for _, c := range Classes {
		byID[c.ID] = c
	}
	return byID
}

// ClassByID returns the class with that id.
func ClassByID(id ClassID) (Class, bool) {
	c, ok := classesByID[id]
	return c, ok
}

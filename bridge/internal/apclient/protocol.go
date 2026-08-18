package apclient

import (
	"encoding/json"
	"strings"
)

// The Archipelago wire format: a JSON array of objects, each with a "cmd". Only
// the messages this bridge acts on are modelled; anything else is read and
// dropped. See docs/network protocol.md in the Archipelago tree.

// itemsHandling: other worlds, our own, starting inventory. Without the third the run starts empty.
const itemsHandling = 0b111

// statusGoal is the ClientStatus that ends a run.
const statusGoal = 30

type header struct {
	Cmd string `json:"cmd"`
}

// Version is Archipelago's own version tuple. The "class" key is required:
// the server compares versions by it.
type Version struct {
	Class string `json:"class"`
	Major int    `json:"major"`
	Minor int    `json:"minor"`
	Build int    `json:"build"`
}

type connectMessage struct {
	Cmd           string   `json:"cmd"`
	Password      string   `json:"password"`
	Game          string   `json:"game"`
	Name          string   `json:"name"`
	UUID          string   `json:"uuid"`
	Version       Version  `json:"version"`
	ItemsHandling int      `json:"items_handling"`
	Tags          []string `json:"tags"`
	SlotData      bool     `json:"slot_data"`
}

type locationChecksMessage struct {
	Cmd       string  `json:"cmd"`
	Locations []int64 `json:"locations"`
}

type statusUpdateMessage struct {
	Cmd    string `json:"cmd"`
	Status int    `json:"status"`
}

type syncMessage struct {
	Cmd string `json:"cmd"`
}

// connectUpdateMessage changes what Connect said, once the slot data has been
// read. Tags are the only thing it changes here: DeathLink is a tag, and the
// seed is what says whether this slot wants it.
type connectUpdateMessage struct {
	Cmd           string   `json:"cmd"`
	ItemsHandling int      `json:"items_handling"`
	Tags          []string `json:"tags"`
}

// bounceMessage is a broadcast to every client holding one of the tags. It is
// how DeathLink travels: a Bounce out, a Bounced in.
type bounceMessage struct {
	Cmd  string        `json:"cmd"`
	Tags []string      `json:"tags"`
	Data deathLinkData `json:"data"`
}

// deathLinkData is the payload of a DeathLink bounce, as every client spells it.
type deathLinkData struct {
	Time   float64 `json:"time"`
	Source string  `json:"source"`
	Cause  string  `json:"cause,omitempty"`
}

type bounced struct {
	Tags []string        `json:"tags"`
	Data json.RawMessage `json:"data"`
}

type sayMessage struct {
	Cmd  string `json:"cmd"`
	Text string `json:"text"`
}

type roomInfo struct {
	SeedName string `json:"seed_name"`
	Password bool   `json:"password"`
}

type connected struct {
	Team             int             `json:"team"`
	Slot             int             `json:"slot"`
	CheckedLocations []int64         `json:"checked_locations"`
	SlotData         json.RawMessage `json:"slot_data"`
}

type connectionRefused struct {
	Errors []string `json:"errors"`
}

type networkItem struct {
	Item     int64 `json:"item"`
	Location int64 `json:"location"`
	Player   int   `json:"player"`
	Flags    int   `json:"flags"`
}

type receivedItems struct {
	Index int           `json:"index"`
	Items []networkItem `json:"items"`
}

type printJSON struct {
	Type string `json:"type"`
	Data []struct {
		Text string `json:"text"`
	} `json:"data"`
}

func (p printJSON) text() string {
	var message strings.Builder
	for _, part := range p.Data {
		message.WriteString(part.Text)
	}
	return message.String()
}

// SlotData is what the apworld put in the seed. gamedata knows every mission
// that exists; only the seed knows which ones are in play.
type SlotData struct {
	FormatVersion       int      `json:"format_version"`
	Missions            []string `json:"missions"`
	Goal                string   `json:"goal"`
	GoalMission         string   `json:"goal_mission"`
	MissionsanityTarget int      `json:"missionsanity_target"`
	DeathLink           bool     `json:"death_link"`
}

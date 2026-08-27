package apclient

import (
	"encoding/json"
	"strconv"
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
	Players          []networkPlayer `json:"players"`
	SlotInfo         map[string]struct {
		Name string `json:"name"`
		Game string `json:"game"`
	} `json:"slot_info"`
}

// networkPlayer is who else is in the multiworld. Alias is what somebody
// renamed themselves to and wins over the slot name when it is set.
type networkPlayer struct {
	Team  int    `json:"team"`
	Slot  int    `json:"slot"`
	Alias string `json:"alias"`
	Name  string `json:"name"`
}

// getDataPackage asks for the item and location names of the games in the room.
// Without it every id in a chat line prints as a number.
type getDataPackage struct {
	Cmd   string   `json:"cmd"`
	Games []string `json:"games"`
}

type dataPackage struct {
	Data struct {
		Games map[string]struct {
			ItemNameToID     map[string]int64 `json:"item_name_to_id"`
			LocationNameToID map[string]int64 `json:"location_name_to_id"`
		} `json:"games"`
	} `json:"data"`
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
		// Text carries the id itself when Type names one, which is why a
		// reader that takes Text alone prints numbers at people.
		Type   string `json:"type"`
		Text   string `json:"text"`
		Player int    `json:"player"`
	} `json:"data"`
}

// text assembles the line, turning every id part back into a name. A nil book
// is the pre-handshake case and leaves the ids alone.
func (p printJSON) text(names *nameBook) string {
	var message strings.Builder
	for _, part := range p.Data {
		if names == nil {
			message.WriteString(part.Text)
			continue
		}
		id, err := strconv.ParseInt(part.Text, 10, 64)
		switch {
		case part.Type == "player_id" && err == nil:
			message.WriteString(names.player(int(id)))
		case part.Type == "item_id" && err == nil:
			message.WriteString(names.item(id, part.Player))
		case part.Type == "location_id" && err == nil:
			message.WriteString(names.location(id, part.Player))
		default:
			message.WriteString(part.Text)
		}
	}
	return message.String()
}

// SlotData is what the apworld put in the seed. gamedata knows every mission
// that exists; only the seed knows which ones are in play.
type SlotData struct {
	FormatVersion           int      `json:"format_version"`
	Missions                []string `json:"missions"`
	StartMission            string   `json:"start_mission"`
	Goal                    string   `json:"goal"`
	GoalMission             string   `json:"goal_mission"`
	MissionsanityTarget     int      `json:"missionsanity_target"`
	DeathLink               bool     `json:"death_link"`
	MissionTicketImportance string   `json:"mission_ticket_importance"`
}

package settings

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
)

/*
codec is how one kind of value is spelled in an .env file.

parse reports false for a value it cannot read, and ApplyEnv then leaves the
field as it was: an empty AP_PORT must not make the launcher ask for a room
address it already had.
*/
type codec[T any] struct {
	parse  func(string) (T, bool)
	format func(T) string
}

// bind is the usual variable: one field, read and written with c.
func bind[T any](name string, services []string, c codec[T], field func(*Settings) *T) EnvVar {
	return EnvVar{
		Name:     name,
		Services: services,
		read: func(s *Settings, value string) {
			if parsed, ok := c.parse(value); ok {
				*field(s) = parsed
			}
		},
		write: func(s Settings) string { return c.format(*field(&s)) },
	}
}

// alias is another spelling of fields a later variable owns. It is read and
// never written.
func alias(name string, services []string, read func(*Settings, string)) EnvVar {
	return EnvVar{Name: name, Services: services, read: read}
}

func readOnly(v EnvVar) EnvVar {
	v.write = nil
	return v
}

var text = codec[string]{
	parse:  func(value string) (string, bool) { return value, true },
	format: func(value string) string { return value },
}

// named is text an empty value does not set. Compose passes a variable .env
// leaves out as empty, and that names nothing.
var named = codec[string]{
	parse: func(value string) (string, bool) {
		value = strings.TrimSpace(value)
		return value, value != ""
	},
	format: func(value string) string { return value },
}

var number = codec[int]{
	parse: func(value string) (int, bool) {
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		return parsed, err == nil
	},
	format: strconv.Itoa,
}

// boolean accepts the spellings an .env file and a shell both produce, and
// writes 1 or 0.
var boolean = codec[bool]{
	parse: func(value string) (bool, bool) {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "1", "true", "yes", "on":
			return true, true
		case "0", "false", "no", "off":
			return false, true
		}
		return false, false
	},
	format: func(value bool) string {
		if value {
			return "1"
		}
		return "0"
	},
}

// words is a comma-separated list. An empty value empties the list, which is
// how a shortcut clears a saved one.
var words = codec[[]string]{
	parse:  func(value string) ([]string, bool) { return SplitList(value), true },
	format: func(value []string) string { return strings.Join(value, ",") },
}

// seats is a list where entry n is seat n of RED. Drop the empties the way
// words does and every seat after a draw moves up one.
var seats = codec[[]string]{
	parse:  func(value string) ([]string, bool) { return SplitSeats(value), true },
	format: func(value []string) string { return strings.Join(value, ",") },
}

// pairs is "key=value,key=value", written sorted by key.
var pairs = codec[map[string]string]{
	parse: func(value string) (map[string]string, bool) {
		parsed := make(map[string]string)
		for _, entry := range SplitList(value) {
			key, item, found := strings.Cut(entry, "=")
			if found && key != "" {
				parsed[strings.TrimSpace(key)] = strings.TrimSpace(item)
			}
		}
		return parsed, true
	},
	format: func(value map[string]string) string {
		out := make([]string, 0, len(value))
		for _, key := range slices.Sorted(maps.Keys(value)) {
			out = append(out, key+"="+value[key])
		}
		return strings.Join(out, ",")
	},
}

/*
table is one of the tables somebody built in front of the page, which the
Compose stack carries as JSON because they are records rather than words.

A value that will not parse is left alone rather than emptied. The alternative
is a stack that loses every loadout or team somebody built to one bad edit of
.env, and the file is edited by hand. Empty is written empty rather than "{}":
a stack that has built none should not have a line of punctuation in its .env
explaining that.
*/
func table[T any]() codec[map[string]T] {
	return codec[map[string]T]{
		parse: func(value string) (map[string]T, bool) {
			if strings.TrimSpace(value) == "" {
				return nil, true
			}
			parsed := make(map[string]T)
			if err := json.Unmarshal([]byte(value), &parsed); err != nil {
				return nil, false
			}
			return parsed, true
		},
		format: func(value map[string]T) string {
			if len(value) == 0 {
				return ""
			}
			body, err := json.Marshal(value)
			if err != nil {
				// Strings, ints and slices of them: there is no value of these
				// that cannot be marshalled, and dropping one silently would
				// lose work somebody did by hand.
				panic(fmt.Sprintf("settings: cannot encode %T: %v", value, err))
			}
			return string(body)
		},
	}
}

var reach = codec[Reach]{
	parse:  func(value string) (Reach, bool) { return ParseReach(strings.ToLower(strings.TrimSpace(value))) },
	format: func(value Reach) string { return string(value) },
}

var modLoading = codec[ModLoading]{
	parse: func(value string) (ModLoading, bool) {
		loading := ModLoading(strings.ToLower(strings.TrimSpace(value)))
		return loading, loading.Valid()
	},
	format: func(value ModLoading) string { return string(value.OrDefault()) },
}

// classSlots was a boolean before 1.16.1 and still reads as one: a compose
// file that says 1 or true gets the progressive slots it was asking for.
var classSlots = codec[ClassSlots]{
	parse: func(value string) (ClassSlots, bool) {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			return ClassSlotsOff, true
		}
		if on, ok := boolean.parse(value); ok {
			if on {
				return ClassSlotsProgressive, true
			}
			return ClassSlotsOff, true
		}
		return ClassSlots(value).OrOff(), true
	},
	format: func(value ClassSlots) string { return string(value.OrOff()) },
}

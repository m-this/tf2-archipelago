// Package chat carries the multiworld's conversation to the game and back.
//
// Nothing here is durable on purpose. A check is a fact about a run and must
// survive anything; a chat line is something someone said, and a line missed
// while the game server was restarting is gone the way it is in any chat.
package chat

import (
	"slices"
	"strings"
	"sync"
	"unicode/utf8"
)

const (
	// LineMax is how much text fits in one line of game chat. The engine
	// refuses a user message over 255 bytes and drops it without a word, and
	// the plugin puts a tag and a colour code in front of every line. The
	// plugin holds the same number as ChatLineMax: both sides build a line, so
	// both sides have to know how long one is.
	LineMax = 200

	// LinesMax bounds one multiworld message. !help alone is two thousand
	// bytes, and every line of it goes to every player on the server.
	LinesMax = 12

	// What the players get instead of the rest. The whole text is in the
	// bridge's log either way.
	truncatedNote = "The rest of that answer is in the server log."
)

// Message is one line from the multiworld, flattened to text here rather than
// in SourcePawn.
type Message struct {
	Seq  int    `json:"seq"`
	Text string `json:"text"`
}

// Log is a bounded ring of recent messages.
type Log struct {
	capacity int

	mu       sync.Mutex
	messages []Message
	latest   int
	updated  chan struct{}
}

func New(capacity int) *Log {
	if capacity < 1 {
		capacity = 1
	}
	return &Log{
		capacity: capacity,
		messages: make([]Message, 0, capacity),
		updated:  make(chan struct{}),
	}
}

// Watch returns a channel closed the next time a message arrives.
func (l *Log) Watch() <-chan struct{} {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.updated
}

// Append records what the multiworld said, as one line per line of chat, and
// wakes anyone waiting. The oldest line falls off the end: a log nobody reads
// must not grow without bound.
func (l *Log) Append(text string) {
	lines := wrap(text)
	if len(lines) == 0 {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, line := range lines {
		l.latest++
		l.messages = append(l.messages, Message{Seq: l.latest, Text: line})
	}
	if len(l.messages) > l.capacity {
		l.messages = slices.Delete(l.messages, 0, len(l.messages)-l.capacity)
	}
	close(l.updated)
	l.updated = make(chan struct{})
}

// wrap cuts one multiworld message into lines the game chat can carry. The
// multiworld writes for a client with a scrollback; this is a chat line over a
// protocol that throws away anything too long.
func wrap(text string) []string {
	lines := make([]string, 0, LinesMax+1)
	for paragraph := range strings.SplitSeq(text, "\n") {
		paragraph = strings.TrimSpace(paragraph)
		for paragraph != "" {
			if len(lines) == LinesMax {
				return append(lines, truncatedNote)
			}
			if len(paragraph) <= LineMax {
				lines = append(lines, paragraph)
				break
			}
			cut := breakAt(paragraph)
			lines = append(lines, strings.TrimSpace(paragraph[:cut]))
			paragraph = strings.TrimSpace(paragraph[cut:])
		}
	}
	return lines
}

// breakAt is where to cut a paragraph longer than one line: the last space that
// still leaves a line worth reading, else the last rune boundary that fits. A
// rune is four bytes at most, so the walk back cannot run away.
func breakAt(paragraph string) int {
	if cut := strings.LastIndexByte(paragraph[:LineMax+1], ' '); cut >= LineMax/2 {
		return cut
	}
	cut := LineMax
	for cut > LineMax-utf8.UTFMax && !utf8.RuneStart(paragraph[cut]) {
		cut--
	}
	return cut
}

// Since returns the lines past a sequence, and the sequence to ask from next. A
// negative sequence means "nothing behind me", which is what the plugin sends
// on load so an evening's backlog does not land in chat.
func (l *Log) Since(seq int) ([]Message, int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if seq < 0 {
		return nil, l.latest
	}
	fresh := make([]Message, 0, len(l.messages))
	for _, message := range l.messages {
		if message.Seq > seq {
			fresh = append(fresh, message)
		}
	}
	return fresh, l.latest
}

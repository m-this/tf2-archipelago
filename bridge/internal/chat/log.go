// Package chat carries the multiworld's conversation to the game and back.
//
// Nothing here is durable on purpose. A check is a fact about a run and must
// survive anything; a chat line is something someone said, and a line missed
// while the game server was restarting is gone the way it is in any chat.
package chat

import (
	"slices"
	"sync"
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

// Append records a line and wakes anyone waiting. The oldest line falls off the
// end: a log nobody reads must not grow without bound.
func (l *Log) Append(text string) {
	if text == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	l.latest++
	l.messages = append(l.messages, Message{Seq: l.latest, Text: text})
	if len(l.messages) > l.capacity {
		l.messages = slices.Delete(l.messages, 0, len(l.messages)-l.capacity)
	}
	close(l.updated)
	l.updated = make(chan struct{})
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

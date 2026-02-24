// Package events provides a simple in-process fan-out pub/sub bus for
// streaming study status changes to Server-Sent Events (SSE) clients.
package events

import (
	"sync"
)

// StudyEvent is the payload published when a study changes.
type StudyEvent struct {
	Type      string `json:"type"`       // "study.status_changed" | "study.created" | "study.deleted"
	StudyID   string `json:"study_id"`
	ProjectID string `json:"project_id"`
	Status    string `json:"status,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// Bus is a fan-out pub/sub bus. Each subscriber gets its own buffered channel.
// Slow subscribers are dropped rather than blocking publishers.
type Bus struct {
	mu   sync.RWMutex
	subs map[uint64]chan StudyEvent
	next uint64
}

// NewBus creates a new event bus.
func NewBus() *Bus {
	return &Bus{subs: make(map[uint64]chan StudyEvent)}
}

// Subscribe registers a new subscriber and returns its channel and an
// unsubscribe function. The caller must call unsubscribe when done.
func (b *Bus) Subscribe() (<-chan StudyEvent, func()) {
	ch := make(chan StudyEvent, 32)

	b.mu.Lock()
	id := b.next
	b.next++
	b.subs[id] = ch
	b.mu.Unlock()

	return ch, func() {
		b.mu.Lock()
		delete(b.subs, id)
		close(ch) // stop Publish from sending to this channel
		b.mu.Unlock()
		// Drain any events already in the buffer.
		for range ch {
		}
	}
}

// Publish sends ev to all subscribers. Subscribers whose channels are full
// are skipped (best-effort delivery — SSE is not transactional).
func (b *Bus) Publish(ev StudyEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subs {
		select {
		case ch <- ev:
		default:
			// subscriber too slow — skip
		}
	}
}

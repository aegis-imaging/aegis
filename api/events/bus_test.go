package events

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBus_PublishReceive(t *testing.T) {
	b := NewBus()
	ch, unsub := b.Subscribe()
	defer unsub()

	ev := StudyEvent{Type: "study.status_changed", StudyID: "s1", Status: "approved"}
	b.Publish(ev)

	select {
	case got := <-ch:
		assert.Equal(t, ev, got)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestBus_MultiSubscriber(t *testing.T) {
	b := NewBus()
	ch1, unsub1 := b.Subscribe()
	ch2, unsub2 := b.Subscribe()
	defer unsub1()
	defer unsub2()

	b.Publish(StudyEvent{Type: "study.created", StudyID: "s2"})

	for _, ch := range []<-chan StudyEvent{ch1, ch2} {
		select {
		case got := <-ch:
			assert.Equal(t, "study.created", got.Type)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for event on subscriber")
		}
	}
}

func TestBus_Unsubscribe(t *testing.T) {
	b := NewBus()
	_, unsub := b.Subscribe()
	unsub()

	// After unsub the internal map should be empty.
	b.mu.RLock()
	count := len(b.subs)
	b.mu.RUnlock()
	require.Equal(t, 0, count)
}

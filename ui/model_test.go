package ui

import (
	"testing"

	"github.com/akib558/docker-tui/docker"
)

// TestNewEventFollowsTailWhenCursorAtEnd reproduces the tail-following
// regression: before this fix, m.eventsCursor was never advanced as events
// streamed in, so once more events arrived than fit on screen, the
// cursor-based windowing in renderListScreen stayed pinned to the oldest
// events. With the cursor already at the last index, each new event must
// land the cursor on the new last index too, the same way `tail -f` follows.
func TestNewEventFollowsTailWhenCursorAtEnd(t *testing.T) {
	m := Model{}
	for i := 0; i < 30; i++ {
		m.events = append(m.events, docker.DockerEvent{Type: "container", Action: "start"})
	}
	m.eventsCursor = len(m.events) - 1

	for i := 0; i < 50; i++ {
		model, _ := m.Update(newEventMsg{ev: docker.DockerEvent{Type: "container", Action: "die"}})
		m = model.(Model)
		if m.eventsCursor != len(m.events)-1 {
			t.Fatalf("iteration %d: eventsCursor = %d, want %d (len(m.events)-1) to keep following the tail",
				i, m.eventsCursor, len(m.events)-1)
		}
	}
}

// TestNewEventDoesNotMoveCursorWhenScrolledUp ensures new events don't yank
// the user's scroll position when they've moved off the tail to look at
// history — normal tail-f-with-manual-scroll behavior.
func TestNewEventDoesNotMoveCursorWhenScrolledUp(t *testing.T) {
	m := Model{}
	for i := 0; i < 30; i++ {
		m.events = append(m.events, docker.DockerEvent{Type: "container", Action: "start"})
	}
	m.eventsCursor = 5 // scrolled up, away from the tail

	model, _ := m.Update(newEventMsg{ev: docker.DockerEvent{Type: "container", Action: "die"}})
	m = model.(Model)

	if m.eventsCursor != 5 {
		t.Fatalf("eventsCursor = %d, want 5 (unchanged) when the user had scrolled away from the tail", m.eventsCursor)
	}
	if len(m.events) != 31 {
		t.Fatalf("len(m.events) = %d, want 31", len(m.events))
	}
}

// TestNewEventTrimsOldestAndKeepsCursorFollowing checks the 500-event cap:
// once the oldest event is dropped, an already-at-the-tail cursor must
// still land on the new last index (not run off the end).
func TestNewEventTrimsOldestAndKeepsCursorFollowing(t *testing.T) {
	m := Model{}
	for i := 0; i < 500; i++ {
		m.events = append(m.events, docker.DockerEvent{Type: "container", Action: "start"})
	}
	m.eventsCursor = len(m.events) - 1

	model, _ := m.Update(newEventMsg{ev: docker.DockerEvent{Type: "container", Action: "die"}})
	m = model.(Model)

	if len(m.events) != 500 {
		t.Fatalf("len(m.events) = %d, want 500 (capped)", len(m.events))
	}
	if m.eventsCursor != len(m.events)-1 {
		t.Fatalf("eventsCursor = %d, want %d after trim", m.eventsCursor, len(m.events)-1)
	}
}

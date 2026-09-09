package ui

import (
	"strings"
	"testing"
	"time"
)

// fixedNow is the instant every deterministic render test pins the clock to.
var fixedNow = time.Date(2026, 3, 14, 9, 26, 53, 0, time.UTC)

// withFixedNow pins nowFunc for the duration of a test.
func withFixedNow(t *testing.T, at time.Time) {
	t.Helper()
	prev := nowFunc
	nowFunc = func() time.Time { return at }
	t.Cleanup(func() { nowFunc = prev })
}

func TestHeaderClockUsesNowFunc(t *testing.T) {
	withFixedNow(t, fixedNow)
	m := Model{width: 120, height: 40}
	out := m.renderHeader(120)
	if !strings.Contains(out, "09:26:53") {
		t.Fatalf("header should render the injected clock 09:26:53, got %q", out)
	}
}

func TestNotificationExpiryUsesNowFunc(t *testing.T) {
	m := Model{
		width:        120,
		notification: "container started",
		notifyTime:   fixedNow.Add(-1 * time.Second),
	}
	withFixedNow(t, fixedNow)
	if !m.hasActiveNotification() {
		t.Fatal("a 1s-old notification should still be active under the injected clock")
	}
	withFixedNow(t, fixedNow.Add(10*time.Second))
	if m.hasActiveNotification() {
		t.Fatal("a 10s-old notification should have expired under the injected clock")
	}
}

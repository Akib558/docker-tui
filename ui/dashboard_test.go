package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/akib558/docker-tui/docker"
)

func TestContainerStateSummaryCountsAllContainers(t *testing.T) {
	m := Model{containers: []docker.ContainerInfo{
		{ID: "1", State: "running"},
		{ID: "2", State: "exited"},
		{ID: "3", State: "restarting"},
		{ID: "4", State: "created"},
		{ID: "5", State: "removing"},
	}}

	summary := m.containerStateSummary()
	total := 0
	labels := make([]string, 0, len(summary))
	for _, sc := range summary {
		total += sc.count
		labels = append(labels, sc.label)
	}
	if total != len(m.containers) {
		t.Fatalf("summary total = %d, want %d (%v)", total, len(m.containers), labels)
	}
	if !containsAll(labels, "running", "stopped", "restarting", "created", "removing") {
		t.Fatalf("missing expected labels: %v", labels)
	}
}

func TestRenderStatusStripIncludesRestarting(t *testing.T) {
	m := Model{
		containers: []docker.ContainerInfo{
			{ID: "1", State: "running"},
			{ID: "2", State: "restarting"},
		},
	}
	out := m.renderStatusStrip(120)
	if !strings.Contains(out, "restarting") {
		t.Fatalf("status strip should mention restarting, got %q", out)
	}
}

func TestRenderStatusStripStoppedLabelVisible(t *testing.T) {
	m := Model{
		containers: []docker.ContainerInfo{
			{State: "running"},
			{State: "exited"},
			{State: "exited"},
			{State: "exited"},
			{State: "paused"},
			{State: "restarting"},
		},
		lastRefresh:     time.Now(),
		refreshInterval: 2 * time.Second,
	}
	for _, w := range []int{80, 120} {
		out := m.renderStatusStrip(w)
		if !strings.Contains(out, "stopped") {
			t.Fatalf("width %d: missing stopped label in %q", w, out)
		}
	}
	for _, w := range []int{40, 60} {
		out := m.renderStatusStrip(w)
		if !strings.Contains(out, "○3") {
			t.Fatalf("width %d: expected compact stopped count, got %q", w, out)
		}
	}
}

func TestStateDisplayName(t *testing.T) {
	if got := stateDisplayName("exited"); got != "stopped" {
		t.Fatalf("exited label = %q, want stopped", got)
	}
	if got := stateDisplayName("running"); got != "running" {
		t.Fatalf("running label = %q, want running", got)
	}
}

func containsAll(hay []string, needles ...string) bool {
	for _, n := range needles {
		found := false
		for _, h := range hay {
			if h == n {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

package ui

import (
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/akib558/docker-tui/config"
	"github.com/akib558/docker-tui/docker"
	"github.com/charmbracelet/lipgloss"
)

var updateGolden = flag.Bool("update", false, "regenerate golden files")

type goldenCase struct {
	name    string
	view    viewState
	width   int
	height  int
	variant string // "ready" | "empty" | "nomatch" | "loading"
	cursor  string // "first" | "middle" | "last"
}

func goldenCases() []goldenCase {
	views := []struct {
		name string
		view viewState
	}{
		{"list", viewList},
		{"images", viewImages},
		{"volumes", viewVolumes},
		{"networks", viewNetworks},
		{"notifications", viewNotifications},
		{"events", viewEvents},
	}
	widths := []int{40, 80, 120, 200}
	var cases []goldenCase
	for _, v := range views {
		for _, w := range widths {
			cases = append(cases, goldenCase{
				name: v.name + "_w" + strconv.Itoa(w) + "_ready_first",
				view: v.view, width: w, height: 30, variant: "ready", cursor: "first",
			})
		}
		for _, variant := range []string{"empty", "nomatch", "loading"} {
			cases = append(cases, goldenCase{
				name: v.name + "_w120_" + variant,
				view: v.view, width: 120, height: 30, variant: variant, cursor: "first",
			})
		}
		for _, cur := range []string{"middle", "last"} {
			cases = append(cases, goldenCase{
				name: v.name + "_w120_ready_" + cur,
				view: v.view, width: 120, height: 30, variant: "ready", cursor: cur,
			})
		}
	}
	return cases
}

func cursorIndex(kind string, n int) int {
	switch {
	case n == 0:
		return 0
	case kind == "middle":
		return n / 2
	case kind == "last":
		return n - 1
	default:
		return 0
	}
}

func fixtureModel(c goldenCase) Model {
	m := Model{
		client:          fixtureClient(),
		cfg:             fixtureConfig(),
		view:            c.view,
		width:           c.width,
		height:          c.height,
		selected:        map[string]bool{},
		stats:           fixtureStats(),
		refreshInterval: 2 * time.Second,
		lastRefresh:     fixtureBase.Add(30 * time.Minute),
		startTime:       fixtureBase,
		systemMem:       docker.SystemMemory{Total: 16 << 30, Used: 8 << 30, Available: 8 << 30, Percent: 50},
		systemLoad:      docker.SystemLoad{Load1: 0.42, Load5: 0.51, Load15: 0.60},
	}

	switch c.variant {
	case "loading":
		m.loading = true
		return m
	case "empty":
		return m
	case "nomatch":
		m.containers = fixtureContainers()
		m.images = fixtureImages()
		m.volumes = fixtureVolumes()
		m.networks = fixtureNetworks()
		m.filterText = "zzzz-no-such-thing"
		m.volFilterText = "zzzz-no-such-thing"
		m.netFilterText = "zzzz-no-such-thing"
		return m
	}

	m.containers = fixtureContainers()
	m.images = fixtureImages()
	m.volumes = fixtureVolumes()
	m.networks = fixtureNetworks()
	m.events = fixtureEvents()
	m.notifyHistory = fixtureNotifications()

	m.cursor = cursorIndex(c.cursor, len(m.containers))
	m.imgCursor = cursorIndex(c.cursor, len(m.images))
	m.volCursor = cursorIndex(c.cursor, len(m.volumes))
	m.netCursor = cursorIndex(c.cursor, len(m.networks))
	m.eventsCursor = cursorIndex(c.cursor, len(m.events))
	m.notifyCursor = cursorIndex(c.cursor, len(m.notifyHistory))
	return m
}

func renderGolden(t *testing.T, c goldenCase) string {
	t.Helper()
	withFixedNow(t, fixedNow)
	applyTheme(config.FindTheme("dark-green"))
	return fixtureModel(c).View()
}

func goldenPath(name string) string {
	return filepath.Join("testdata", "golden", name+".txt")
}

func TestGolden(t *testing.T) {
	for _, c := range goldenCases() {
		t.Run(c.name, func(t *testing.T) {
			got := renderGolden(t, c)
			path := goldenPath(c.name)
			if *updateGolden {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("missing golden %s (run: go test ./ui/ -run TestGolden -update): %v", path, err)
			}
			if got != string(want) {
				t.Errorf("golden mismatch for %s\n--- want ---\n%s\n--- got ---\n%s",
					c.name, visualize(string(want)), visualize(got))
			}
		})
	}
}

// visualize strips ANSI so a failing diff is readable in terminal output.
func visualize(s string) string {
	var b strings.Builder
	for _, line := range strings.Split(s, "\n") {
		b.WriteString(lipgloss.NewStyle().Render(""))
		b.WriteString(stripANSI(line))
		b.WriteString("\n")
	}
	return b.String()
}

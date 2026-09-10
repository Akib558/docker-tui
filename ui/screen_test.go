package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func sampleScreen() ListScreen {
	cols := FitColumns([]ColumnSpec{
		{Label: "NAME", Percent: 50, Min: 10},
		{Label: "DRIVER", Percent: 50, Min: 6, Priority: 1},
	}, TableW(80))
	return ListScreen{
		Title: "Volumes",
		Count: 2,
		Cols:  cols,
		Rows: []ListTableRow{
			{Cells: []Cell{{Text: "pgdata", Width: cols[0].Width}, {Text: "local", Width: cols[1].Width}}},
			{Cells: []Cell{{Text: "app-cache", Width: cols[0].Width}, {Text: "local", Width: cols[1].Width}}},
		},
		State: ScreenReady,
		Help:  "j/k nav",
	}
}

func TestRenderListScreenFitsWidth(t *testing.T) {
	withFixedNow(t, fixedNow)
	m := Model{width: 80, height: 24, cfg: fixtureConfig()}
	out := m.renderListScreen(sampleScreen())
	for i, line := range strings.Split(out, "\n") {
		if w := lipgloss.Width(line); w > 80 {
			t.Errorf("line %d is %d cols, want <= 80: %q", i+1, w, stripANSI(line))
		}
	}
}

func TestRenderListScreenAlwaysEmitsToast(t *testing.T) {
	withFixedNow(t, fixedNow)
	m := Model{
		width: 80, height: 24, cfg: fixtureConfig(),
		notification: "Removed volume pgdata", notifyTime: fixedNow,
	}
	out := m.renderListScreen(sampleScreen())
	if !strings.Contains(stripANSI(out), "Removed volume pgdata") {
		t.Fatal("the scaffold must render the active toast in every view")
	}
}

func TestRenderListScreenStates(t *testing.T) {
	withFixedNow(t, fixedNow)
	m := Model{width: 80, height: 24, cfg: fixtureConfig()}
	cases := map[ScreenState]string{
		ScreenLoading: "Loading volumes...",
		ScreenEmpty:   "No volumes found.",
		ScreenNoMatch: "No volumes match the current filter.",
		ScreenError:   "Cannot connect to Docker",
	}
	for state, msg := range cases {
		s := sampleScreen()
		s.State = state
		s.Message = msg
		s.Rows = nil
		out := stripANSI(m.renderListScreen(s))
		if !strings.Contains(out, msg) {
			t.Errorf("state %d should render %q, got:\n%s", state, msg, out)
		}
	}
}

func TestRenderListScreenHeaderMatchesRows(t *testing.T) {
	withFixedNow(t, fixedNow)
	m := Model{width: 80, height: 24, cfg: fixtureConfig()}
	s := sampleScreen()
	out := stripANSI(m.renderListScreen(s))
	lines := strings.Split(out, "\n")
	var hdrIdx int
	for i, l := range lines {
		if strings.Contains(l, "NAME") && strings.Contains(l, "DRIVER") {
			hdrIdx = i
			break
		}
	}
	if hdrIdx == 0 {
		t.Fatal("could not find the table header line")
	}
	// listHeaderStyle has BorderBottom, so the rendered header is two lines
	// (the content line, then a "---" rule) before the first data row —
	// see ui/layout.go's detailLogsChromeLines comment, which documents the
	// same fact. The first data row is also the cursor row, whose gutter
	// carries the rail glyph (Task 7) — a multi-byte UTF-8 character that
	// is still one display column, so compare display-column offsets, not
	// raw byte offsets, or the rail's extra bytes throw off the comparison.
	hdr := lines[hdrIdx]
	row := lines[hdrIdx+2]
	hdrCol := displayWidth(hdr[:strings.Index(hdr, "DRIVER")])
	rowCol := displayWidth(row[:strings.Index(row, "local")])
	if hdrCol != rowCol {
		t.Fatalf("DRIVER header at column %d but its cell at column %d", hdrCol, rowCol)
	}
}

func TestRenderListScreenSelectionCountInTitle(t *testing.T) {
	withFixedNow(t, fixedNow)
	m := Model{width: 80, height: 24, cfg: fixtureConfig()}
	s := sampleScreen()
	s.Selected = 2
	out := stripANSI(m.renderListScreen(s))
	if !strings.Contains(out, "2 selected") {
		t.Fatalf("title should report the selection count, got:\n%s", out)
	}
	if !strings.Contains(out, glyphMarkerSelect) {
		t.Fatalf("title should use the registry glyph %q", glyphMarkerSelect)
	}
}

func TestRenderListScreenScrollIndicatorReservesItsOwnLine(t *testing.T) {
	withFixedNow(t, fixedNow)
	m := Model{width: 80, height: 24, cfg: fixtureConfig()}
	s := sampleScreen()

	// Enough rows that the body can't show them all at this height, forcing
	// the two-pass chrome calculation's scroll-indicator branch.
	var rows []ListTableRow
	for i := 0; i < 50; i++ {
		rows = append(rows, ListTableRow{Cells: []Cell{
			{Text: fmt.Sprintf("vol-%d", i), Width: s.Cols[0].Width},
			{Text: "local", Width: s.Cols[1].Width},
		}})
	}
	s.Rows = rows
	s.Count = len(rows)

	out := m.renderListScreen(s)
	lines := strings.Split(out, "\n")
	if len(lines) > m.height {
		t.Fatalf("rendered %d lines, want <= %d (height) — the scroll indicator's own line must be reserved in advance, not pushed past the terminal height", len(lines), m.height)
	}
	if !strings.Contains(stripANSI(out), glyphScroll) {
		t.Fatal("scroll indicator should render when rows exceed the visible area")
	}
}

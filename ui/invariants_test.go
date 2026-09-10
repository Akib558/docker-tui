package ui

import (
	"os"
	"strings"
	"testing"

	"github.com/akib558/docker-tui/config"
	"github.com/charmbracelet/lipgloss"
)

// Invariant 1: no rendered line may exceed the terminal width. A line that
// does will wrap and break the fixed-height layout.
func TestEveryLineFitsTerminalWidth(t *testing.T) {
	for _, c := range goldenCases() {
		t.Run(c.name, func(t *testing.T) {
			out := renderGolden(t, c)
			for i, line := range strings.Split(out, "\n") {
				if w := lipgloss.Width(line); w > c.width {
					t.Errorf("line %d is %d cols wide, terminal is %d: %q",
						i+1, w, c.width, stripANSI(line))
				}
			}
		})
	}
}

// Invariant 2: a highlighted row's background must run unbroken from column 0
// to the terminal width. Any gap reads as a hole and stops the row looking
// like one line.
func TestCursorRowBackgroundIsContiguous(t *testing.T) {
	cellSets := map[string][]Cell{
		"plain": {
			{Text: glyphPad(glyphStateRunning, 3), Width: 3, FG: colorSuccess},
			{Text: "web-api", Width: 24, FG: colorText},
			{Text: "Up 3 hours", Width: 16, FG: colorSubtext},
		},
		"with_dim_id": {
			{Text: glyphPad(glyphStateRunning, 3), Width: 3, FG: colorSuccess},
			{Text: "web-api", Width: 24, FG: colorText},
			{Text: "a1b2c3d4e5f6", Width: 14, FG: colorDim},
		},
		"with_bar": {
			{Text: glyphPad(glyphStateRunning, 3), Width: 3, FG: colorSuccess},
			{Text: "web-api", Width: 24, FG: colorText},
			{Width: 12, Bar: true, BarPercent: 42},
		},
	}
	kinds := []ListRowKind{ListRowCursor, ListRowSelected, ListRowCursorSelected}

	for _, theme := range config.Themes {
		applyTheme(&theme)
		for name, cells := range cellSets {
			for _, kind := range kinds {
				for _, w := range []int{40, 80, 120, 200} {
					row := renderRowFromKind(w, kind, 0, cells)
					cols := bgColumns(row)
					if len(cols) != w {
						t.Fatalf("%s/%s/kind=%d/w=%d: probe saw %d columns, want %d",
							theme.Name, name, kind, w, len(cols), w)
					}
					for i, bg := range cols {
						if bg != cols[0] {
							t.Errorf("%s/%s/kind=%d/w=%d: background breaks at column %d (%q != %q)",
								theme.Name, name, kind, w, i, bg, cols[0])
							break
						}
					}
				}
			}
		}
	}
	applyTheme(config.FindTheme("dark-green"))
}

// columnOffsets renders a sentinel row — each column holds a unique letter —
// and reports the display column each cell starts at.
func columnOffsets(termW int, cols []ListColumn, rs RowStyle) []int {
	letters := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	cells := make([]Cell, len(cols))
	for i, c := range cols {
		cells[i] = Cell{Text: string(letters[i%len(letters)]), Width: c.Width}
	}
	plain := stripANSI(RenderRow(termW, cells, rs))
	offsets := make([]int, 0, len(cols))
	for i := range cols {
		idx := strings.Index(plain, string(letters[i%len(letters)]))
		offsets = append(offsets, displayWidth(plain[:idx]))
	}
	return offsets
}

// Invariant 3: a view's table header must start each column at exactly the
// display column its data rows do.
func TestHeaderAndRowColumnsAlign(t *testing.T) {
	for _, w := range []int{40, 80, 120, 200} {
		cols := FitColumns([]ColumnSpec{
			{Label: "", Width: 3},
			{Label: "NAME", Percent: 40, Min: 10},
			{Label: "STATUS", Percent: 35, Min: 8, Priority: 1},
			{Label: "ID", Percent: 25, Min: 8, Priority: 2},
		}, TableW(w))

		header := columnOffsets(w, cols, RowStyle{Marker: " "})
		row := columnOffsets(w, cols, rowStyleFromKind(ListRowCursor, 0))
		if len(header) != len(row) {
			t.Fatalf("w=%d: header has %d columns, row has %d", w, len(header), len(row))
		}
		for i := range header {
			if header[i] != row[i] {
				t.Errorf("w=%d: column %d starts at %d in the header but %d in the row",
					w, i, header[i], row[i])
			}
		}
	}
}

// TestViewHeadersUseSharedGeometry checks that each view's table header is no
// longer hand-built with a hardcoded gap between columns — once a view
// migrates onto the shared ListScreen scaffold (Tasks 12-15), its header is
// built by RenderRow using the same single-space gap as its data rows, and
// the literal `+ "  " +` join this check looks for disappears from the file
// entirely. A rendered-text comparison was tried first and abandoned: each
// column's own padding scales with the view's computed width, so the real
// inter-column gap is never a small, text-position-derivable constant
// without assuming the very hardcoded value this test exists to catch —
// which would make the test unsatisfiable even after a correct migration.
func TestViewHeadersUseSharedGeometry(t *testing.T) {
	handBuilt := []string{
		"view_images.go",
		"view_volumes.go",
		"view_networks.go",
		"view_events.go",
	}
	const hardcodedGap = `+ "  " +`
	for _, f := range handBuilt {
		t.Run(f, func(t *testing.T) {
			src, err := os.ReadFile(f)
			if err != nil {
				t.Fatalf("reading %s: %v", f, err)
			}
			if strings.Contains(string(src), hardcodedGap) {
				t.Errorf("%s still hand-builds its table header with a literal two-space gap (%q) instead of RenderRow's shared, single-space geometry", f, hardcodedGap)
			}
		})
	}
}

// Invariant 4: the detail view's identity line (name/state/image/ID) and tab
// bar must never overflow the terminal width, even for a deliberately long
// container name.
func TestDetailIdentityLineFitsWidth(t *testing.T) {
	withFixedNow(t, fixedNow)
	long := fixtureContainers()[4] // the deliberately long name
	for _, w := range []int{40, 80, 120, 200} {
		m := fixtureModel(goldenCase{view: viewDetail, width: w, height: 30, variant: "ready", cursor: "first"})
		m.inspected = &long
		m.view = viewDetail
		for i, line := range strings.Split(m.View(), "\n") {
			if got := lipgloss.Width(line); got > w {
				t.Errorf("w=%d line %d is %d cols: %q", w, i+1, got, stripANSI(line))
			}
		}
	}
}

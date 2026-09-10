package ui

import (
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

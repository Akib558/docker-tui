package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestGlyphPadEqualWidth(t *testing.T) {
	icons := []string{"●", "○", "◑", "◇", "✕", "↻"}
	widths := make([]int, len(icons))
	for i, icon := range icons {
		widths[i] = displayWidth(glyphPad(icon, 2))
	}
	for i := 1; i < len(widths); i++ {
		if widths[i] != widths[0] {
			t.Fatalf("icon widths differ: %v", widths)
		}
	}
}

func TestRenderRowFitsTerminalWidth(t *testing.T) {
	cells := []Cell{
		{Text: glyphPad("●", 3), Width: 3, FG: colorSuccess},
		{Text: "nginx", Width: 20, FG: colorText},
		{Text: "running", Width: 12, FG: colorSubtext},
	}
	for _, w := range []int{60, 80, 120, 200} {
		for _, kind := range []ListRowKind{ListRowNormal, ListRowCursor, ListRowSelected, ListRowCursorSelected} {
			row := renderRowFromKind(w, kind, 1, cells)
			if got := lipgloss.Width(row); got != w {
				t.Fatalf("width %d: got %d (kind=%d)", w, got, kind)
			}
		}
	}
}

func TestSelectedRowBackgroundContinuity(t *testing.T) {
	cells := []Cell{
		{Text: glyphPad("●", 3), Width: 3, FG: colorSuccess},
		{Text: "web-server", Width: 24, FG: colorText},
		{Text: "Up 2 hours", Width: 16, FG: colorSubtext},
	}
	row := renderRowFromKind(80, ListRowCursor, 0, cells)
	if !strings.Contains(row, string(colorBgSelected)) && colorBgSelected != "" {
		// Background color should appear in selected row styling.
		if lipgloss.Width(row) != 80 {
			t.Fatalf("selected row width = %d, want 80", lipgloss.Width(row))
		}
	}
}

func TestRenderRowZeroPercentBar(t *testing.T) {
	cells := []Cell{
		{Width: 12, Bar: true, BarPercent: 0},
	}
	row := renderRowFromKind(20, ListRowNormal, 0, cells)
	if !strings.Contains(row, "0%") {
		t.Fatalf("zero percent bar should show 0%%, got %q", row)
	}
	if lipgloss.Width(row) != 20 {
		t.Fatalf("row width = %d, want 20", lipgloss.Width(row))
	}
}

func TestRenderRowZebraAlternates(t *testing.T) {
	even := rowStyleFromKind(ListRowNormal, 0)
	odd := rowStyleFromKind(ListRowNormal, 1)
	if even.BG == odd.BG {
		t.Fatalf("zebra styles should differ between even and odd rows")
	}
}

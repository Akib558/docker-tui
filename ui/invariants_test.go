package ui

import (
	"strings"
	"testing"

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

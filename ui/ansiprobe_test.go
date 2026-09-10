package ui

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// bgColumns returns the background SGR signature in effect at each display
// column of a rendered line. "" means no background (terminal default).
func bgColumns(line string) []string {
	var cols []string
	cur := ""
	i := 0
	for i < len(line) {
		if line[i] == 0x1b && i+1 < len(line) && line[i+1] == '[' {
			end := strings.IndexByte(line[i:], 'm')
			if end < 0 {
				break
			}
			cur = applySGRBackground(cur, line[i+2:i+end])
			i += end + 1
			continue
		}
		r, size := utf8.DecodeRuneInString(line[i:])
		for k := 0; k < runewidth.RuneWidth(r); k++ {
			cols = append(cols, cur)
		}
		i += size
	}
	return cols
}

// applySGRBackground folds one SGR parameter list into the current background.
func applySGRBackground(cur, seq string) string {
	params := strings.Split(seq, ";")
	for i := 0; i < len(params); i++ {
		p := params[i]
		switch {
		case p == "" || p == "0" || p == "49":
			cur = ""
		case p == "48":
			if i+2 < len(params) && params[i+1] == "5" {
				cur = strings.Join(params[i:i+3], ";")
				i += 2
			} else if i+4 < len(params) && params[i+1] == "2" {
				cur = strings.Join(params[i:i+5], ";")
				i += 4
			}
		case len(p) == 2 && p[0] == '4' && p[1] >= '0' && p[1] <= '7':
			cur = p
		case len(p) == 3 && p[0] == '1' && p[1] == '0' && p[2] >= '0' && p[2] <= '7':
			cur = p
		}
	}
	return cur
}

func TestBGColumnsTracksBackground(t *testing.T) {
	styled := lipgloss.NewStyle().Background(lipgloss.Color("#1B5E20")).Render("ab")
	got := bgColumns(styled + "cd")
	if len(got) != 4 {
		t.Fatalf("expected 4 columns, got %d (%v)", len(got), got)
	}
	if got[0] == "" || got[0] != got[1] {
		t.Fatalf("styled columns should share one background, got %v", got[:2])
	}
	if got[2] != "" || got[3] != "" {
		t.Fatalf("unstyled columns should have no background, got %v", got[2:])
	}
}

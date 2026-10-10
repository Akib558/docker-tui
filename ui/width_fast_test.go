package ui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestPlainDisplayWidthMatchesLipgloss(t *testing.T) {
	cases := []string{
		"",
		"hello",
		"nginx-proxy",
		"日本語",
		"foo🐳bar",
		"café",
		"e\u0301",
		"col\tab",
	}
	for _, s := range cases {
		if got := displayWidth(s); got != lipgloss.Width(s) {
			t.Fatalf("displayWidth(%q) = %d, lipgloss = %d", s, got, lipgloss.Width(s))
		}
	}
}

func TestRenderedWidthMatchesLipgloss(t *testing.T) {
	cases := []string{
		"\x1b[38;2;255;0;0mred\x1b[0m",
		"\x1b[1;32mrunning\x1b[0m",
		"\x1b[38;5;196m\x1b[1mERR\x1b[0m fail",
		lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).Render("hello"),
		lipgloss.NewStyle().Background(colorBgAlt).Foreground(colorText).Render("cell"),
	}
	for _, s := range cases {
		if got := renderedWidth(s); got != lipgloss.Width(s) {
			t.Fatalf("renderedWidth(%q) = %d, lipgloss = %d", s, got, lipgloss.Width(s))
		}
	}
}

func TestTruncateDisplayASCII(t *testing.T) {
	if got := truncateDisplay("abcdef", 3); got != "abc" {
		t.Fatalf("got %q", got)
	}
}

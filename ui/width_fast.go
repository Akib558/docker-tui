package ui

import (
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// ponytail: width fast paths assume EastAsianWidth=false (see init in cell.go); upgrade via runewidth fallback.

func hasESC(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			return true
		}
	}
	return false
}

func isASCIINoEsc(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\x1b' {
			return false
		}
		if c == '\t' {
			return false
		}
		if c < 0x20 || c > 0x7e {
			return false
		}
	}
	return true
}

func plainDisplayWidth(s string) int {
	if isASCIINoEsc(s) {
		return len(s)
	}
	return runewidth.StringWidth(s)
}

// displayWidth is the visible width of plain (unstyled) text.
func displayWidth(s string) int {
	return plainDisplayWidth(s)
}

// renderedWidth is the visible width of a string that may contain ANSI SGR/CSI sequences.
func renderedWidth(s string) int {
	if s == "" {
		return 0
	}
	if !hasESC(s) {
		return plainDisplayWidth(s)
	}
	w := 0
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' {
			if end, ok := skipANSIAt(s, i); ok {
				i = end
				continue
			}
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == '\t' {
			w += runewidth.RuneWidth(r)
		} else if r < 0x80 {
			w++
		} else {
			w += runewidth.RuneWidth(r)
		}
		i += size
	}
	return w
}

func skipANSIAt(s string, i int) (end int, ok bool) {
	if i >= len(s) || s[i] != '\x1b' {
		return i, false
	}
	if i+1 >= len(s) {
		return len(s), true
	}
	switch s[i+1] {
	case '[':
		j := i + 2
		for j < len(s) {
			c := s[j]
			if c >= 0x40 && c <= 0x7e {
				return j + 1, true
			}
			j++
		}
		return len(s), true
	case ']':
		j := i + 2
		for j < len(s) {
			if s[j] == '\x07' {
				return j + 1, true
			}
			if s[j] == '\x1b' && j+1 < len(s) && s[j+1] == '\\' {
				return j + 2, true
			}
			j++
		}
		return len(s), true
	case '(', ')', '*', '+', '-', '.', '/':
		if i+2 < len(s) {
			return i + 3, true
		}
		return len(s), true
	default:
		return i + 2, true
	}
}

func truncateDisplay(s string, maxW int) string {
	if maxW < 1 {
		return ""
	}
	if !hasESC(s) && isASCIINoEsc(s) {
		if len(s) <= maxW {
			return s
		}
		return s[:maxW]
	}
	return runewidth.Truncate(s, maxW, "")
}

func stripANSIFast(s string) string {
	if !hasESC(s) {
		return s
	}
	var b []byte
	b = make([]byte, 0, len(s))
	for i := 0; i < len(s); {
		if s[i] == '\x1b' {
			if end, ok := skipANSIAt(s, i); ok {
				i = end
				continue
			}
		}
		b = append(b, s[i])
		i++
	}
	return string(b)
}

// lipglossDisplayWidth matches lipgloss.Width; used in tests.
func lipglossDisplayWidth(s string) int { return lipgloss.Width(s) }

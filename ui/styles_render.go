package ui

import (
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
)

const styleSentinel = "\x7f"

type stylePair struct {
	open  string
	close string
}

var (
	stylePairMu sync.RWMutex
	stylePairs  = make(map[styleKey]stylePair)
)

type styleKey struct {
	fg   lipgloss.Color
	bg   lipgloss.Color
	bold bool
}

func clearStylePairs() {
	stylePairMu.Lock()
	stylePairs = make(map[styleKey]stylePair)
	stylePairMu.Unlock()
}

func styleOpenClose(fg, bg lipgloss.Color, bold bool) (open, close string) {
	if fg == "" && bg == "" && !bold {
		return "", ""
	}
	k := styleKey{fg: fg, bg: bg, bold: bold}
	stylePairMu.RLock()
	p, ok := stylePairs[k]
	stylePairMu.RUnlock()
	if ok {
		return p.open, p.close
	}
	st := lipgloss.NewStyle()
	if fg != "" {
		st = st.Foreground(fg)
	}
	if bg != "" {
		st = st.Background(bg)
	}
	if bold {
		st = st.Bold(true)
	}
	raw := st.Render(styleSentinel)
	idx := strings.Index(raw, styleSentinel)
	if idx < 0 {
		return raw, ""
	}
	p = stylePair{open: raw[:idx], close: raw[idx+len(styleSentinel):]}
	stylePairMu.Lock()
	stylePairs[k] = p
	stylePairMu.Unlock()
	return p.open, p.close
}

func wrapWithStyle(text string, fg, bg lipgloss.Color, bold bool) string {
	open, close := styleOpenClose(fg, bg, bold)
	if open == "" && close == "" {
		return text
	}
	return open + text + close
}

func bgPad(n int, bg lipgloss.Color) string {
	if n < 1 {
		return ""
	}
	open, close := styleOpenClose("", bg, false)
	if open == "" && close == "" {
		return strings.Repeat(" ", n)
	}
	sp := strings.Repeat(" ", n)
	return open + sp + close
}

func prewarmStylePairs() {
	colors := []lipgloss.Color{
		colorPrimary, colorSecondary, colorSuccess, colorDanger, colorWarning,
		colorMuted, colorText, colorSubtext, colorDim, colorSecondary, colorTitleFg,
		colorBarTrack,
	}
	bgs := []lipgloss.Color{"", colorBgAlt, colorBgSelected}
	for _, bg := range bgs {
		for _, fg := range colors {
			styleOpenClose(fg, bg, false)
			styleOpenClose(fg, bg, true)
		}
		styleOpenClose("", bg, false)
	}
	for _, fg := range colors {
		styleOpenClose(fg, "", false)
		styleOpenClose(fg, "", true)
	}
}

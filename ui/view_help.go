package ui

import (
	"strings"
	"time"

	"github.com/akib558/docker-tui/config"
	"github.com/charmbracelet/lipgloss"
)

// ── List help bar ────────────────────────────────────────────────────────

func (m *Model) dismissHint() {
	if m.cfg != nil && !m.cfg.HintDismissed {
		m.cfg.HintDismissed = true
		_ = config.Save(m.cfg)
	}
}

func (m Model) renderGettingStartedHint(w int) string {
	line := lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
		Render("  press ? for help  ·  : for command palette")
	return truncateDisplay(line, w)
}

func (m Model) helpCentered(w int) string {
	var keys []struct{ key, desc string }
	if m.filtering {
		keys = []struct{ key, desc string }{
			{"type", "search"},
			{"backspace", "delete"},
			{"enter/esc", "done"},
			{"ctrl+u", "clear"},
		}
	} else if len(m.selected) > 0 {
		keys = []struct{ key, desc string }{
			{"space", "toggle"},
			{"a", "all"},
			{"s", "start/stop"},
			{"d", "remove"},
			{"esc", "deselect"},
			{"?", "more"},
		}
	} else {
		keys = []struct{ key, desc string }{
			{"j/k", "nav"},
			{"enter", "details"},
			{"/", "filter"},
			{":", "commands"},
			{"s", "start/stop"},
			{"D", "events"},
			{"space", "select"},
			{"?", "help"},
			{"q", "quit"},
		}
	}
	return renderHelpBar(w, fmtKeys(keys))
}

func (m Model) centralLogsHelp(w int) string {
	if m.centralLogFiltering {
		return renderHelpBar(w, fmtKeys([]struct{ key, desc string }{
			{"type", "filter"},
			{"r", "regex"},
			{"backspace", "delete"},
			{"enter/esc", "done"},
			{"ctrl+u", "clear"},
		}))
	}
	return renderHelpBar(w, fmtKeys([]struct{ key, desc string }{
		{"j/k", "focus line"},
		{"pgup/pgdn", "page"},
		{"space/V", "select"},
		{"1-9", "toggle container"},
		{"/", "filter"},
		{"y", "copy"},
		{"L", "legend"},
		{"E", "export"},
		{"?", "help"},
		{"esc", "back"},
	}))
}

// ── Detail help bar ──────────────────────────────────────────────────────

func (m Model) detailHelp(w int) string {
	var keys []struct{ key, desc string }
	if m.detailTab == tabLogs {
		live := "start live"
		if m.liveLogging {
			live = "stop live"
		}
		keys = []struct{ key, desc string }{
			{"tab", "switch tab"},
			{"j/k", "focus line"},
			{"space", "select"},
			{"V", "range select"},
			{"y", "copy selection"},
			{"l", live},
			{"L", "legend"},
			{"E", "export"},
			{"?", "help"},
			{"esc", "back"},
		}
	} else if m.detailTab == tabTerminal {
		keys = []struct{ key, desc string }{
			{"tab", "switch tab"},
			{"i", "focus input"},
			{"enter", "send"},
			{"ctrl+\\", "detach"},
			{"x", "reconnect"},
			{"?", "help"},
			{"esc", "back"},
		}
	} else if m.detailTab == tabDiff {
		keys = []struct{ key, desc string }{
			{"tab", "switch tab"},
			{"j/k", "scroll"},
			{"f", "fetch diff"},
			{"?", "help"},
			{"esc", "back"},
		}
	} else if m.detailTab == tabProcesses {
		keys = []struct{ key, desc string }{
			{"tab", "switch tab"},
			{"j/k", "scroll"},
			{"p", "refresh"},
			{"?", "help"},
			{"esc", "back"},
		}
	} else {
		keys = []struct{ key, desc string }{
			{"tab", "switch tab"},
			{"j/k", "scroll"},
			{"s", "start/stop"},
			{"e", "exec"},
			{"?", "help"},
			{"esc", "back"},
		}
	}
	return renderHelpBar(w, fmtKeys(keys))
}

// ── Notification ─────────────────────────────────────────────────────────

func (m Model) renderNotification() string {
	if m.notification == "" || time.Since(m.notifyTime) > 4*time.Second {
		return ""
	}
	if m.notifyIsErr {
		return "  " + notifyErrorStyle.Render(m.notification) + "\n"
	}
	return "  " + notifySuccessStyle.Render(m.notification) + "\n"
}

// ── Key formatting ───────────────────────────────────────────────────────

func fmtKeys(keys []struct{ key, desc string }) string {
	sep := " " + lipgloss.NewStyle().Foreground(colorDim).Render("·") + " "
	var parts []string
	for _, k := range keys {
		parts = append(parts, helpKeyStyle.Render(k.key)+" "+helpDescStyle.Render(k.desc))
	}
	return strings.Join(parts, sep)
}

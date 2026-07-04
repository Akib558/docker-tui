package ui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/akib558/docker-tui/docker"
	"github.com/charmbracelet/lipgloss"
)

// ── KV rendering ─────────────────────────────────────────────────────────

func renderKV(key, value string) string {
	return "  " + detailLabelStyle.Render(key) + " " + detailValueStyle.Render(value) + "\n"
}

func renderHealthKV(key, health string) string {
	var style lipgloss.Style
	switch health {
	case "healthy":
		style = lipgloss.NewStyle().Foreground(colorSuccess).Bold(true)
	case "unhealthy":
		style = lipgloss.NewStyle().Foreground(colorDanger).Bold(true)
	case "starting":
		style = lipgloss.NewStyle().Foreground(colorWarning).Bold(true)
	default:
		style = lipgloss.NewStyle().Foreground(colorMuted)
	}
	return "  " + detailLabelStyle.Render(key) + " " + style.Render(health) + "\n"
}

func renderRestartKV(key string, count int) string {
	var style lipgloss.Style
	switch {
	case count >= 10:
		style = lipgloss.NewStyle().Foreground(colorDanger).Bold(true)
	case count >= 3:
		style = lipgloss.NewStyle().Foreground(colorWarning).Bold(true)
	default:
		style = lipgloss.NewStyle().Foreground(colorText)
	}
	icons := strings.Repeat("↻", min(count, 5))
	return "  " + detailLabelStyle.Render(key) + " " + style.Render(icons+" "+fmt.Sprintf("%d", count)) + "\n"
}

func formatUptime(d time.Duration) string {
	if d < time.Minute {
		return "< 1m"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		if m == 0 {
			return fmt.Sprintf("%dh", h)
		}
		return fmt.Sprintf("%dh %dm", h, m)
	}
	days := int(d.Hours()) / 24
	h := int(d.Hours()) % 24
	if h == 0 {
		return fmt.Sprintf("%dd", days)
	}
	return fmt.Sprintf("%dd %dh", days, h)
}

// ── Truncate ─────────────────────────────────────────────────────────────

func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// ── Ports ────────────────────────────────────────────────────────────────

func formatPortsSummary(ports []docker.PortBinding) string {
	if len(ports) == 0 {
		return "-"
	}
	var parts []string
	seen := make(map[string]bool)
	for _, p := range ports {
		var s string
		if p.HostPort != "" && p.HostPort != "0" {
			s = p.HostPort + "->" + p.ContPort
		} else {
			s = p.ContPort
		}
		if !seen[s] {
			parts = append(parts, s)
			seen[s] = true
		}
	}
	return strings.Join(parts, ",")
}

// ── Log cleaning ──────────────────────────────────────────────────────────

func cleanDockerLogs(s string) string {
	var cleaned strings.Builder
	for _, line := range strings.Split(s, "\n") {
		line = docker.StripDockerLogHeaderForUI(line)
		cleaned.WriteString(line + "\n")
	}
	return strings.TrimRight(sanitizeOutputText(cleaned.String()), "\n")
}

var ansiEscapeRE = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]|\x1b\][^\a]*(\a|\x1b\\)|\x1b[@-_]`)

func sanitizeOutputText(s string) string {
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = ansiEscapeRE.ReplaceAllString(s, "")
	s = strings.Map(func(r rune) rune {
		switch r {
		case '\n', '\t':
			return r
		}
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, s)

	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	emptyRun := 0
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if strings.TrimSpace(line) == "" {
			emptyRun++
			if emptyRun > 1 {
				continue
			}
			out = append(out, "")
			continue
		}
		emptyRun = 0
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func normalizeTerminalScroll(scroll, maxScroll int, follow bool) (int, bool) {
	if maxScroll < 0 {
		maxScroll = 0
	}
	if follow {
		return maxScroll, true
	}
	if scroll < 0 {
		return 0, false
	}
	if scroll >= maxScroll {
		return maxScroll, true
	}
	return scroll, false
}

func detailPageStep(height int) int {
	step := height / 3
	if step < 3 {
		return 3
	}
	return step
}

// ── Layout helpers ────────────────────────────────────────────────────────

func interleave(items []string, spacer string) []string {
	if len(items) == 0 {
		return items
	}
	result := make([]string, 0, len(items)*2-1)
	for i, item := range items {
		if i > 0 {
			result = append(result, spacer)
		}
		result = append(result, item)
	}
	return result
}

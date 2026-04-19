package ui

import (
	"regexp"
	"strings"

	"github.com/akib/docker-tui/docker"
)

// ── KV rendering ─────────────────────────────────────────────────────────

func renderKV(key, value string) string {
	return "  " + detailLabelStyle.Render(key) + " " + detailValueStyle.Render(value) + "\n"
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
		if len(line) > 8 {
			if line[0] == 1 || line[0] == 2 {
				line = line[8:]
			}
		}
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


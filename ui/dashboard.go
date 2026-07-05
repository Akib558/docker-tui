package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/akib558/docker-tui/config"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderDashboard(w int) string {
	if len(m.containers) == 0 {
		return ""
	}
	return m.renderStatusStrip(w)
}

func (m Model) dashboardHeight(w int) int {
	if len(m.containers) == 0 {
		return 0
	}
	if w < 80 {
		return 1
	}
	return 2
}

func (m Model) containerStateSummary() []struct {
	state string
	label string
	count int
} {
	counts := make(map[string]int, len(m.containers))
	for _, c := range m.containers {
		counts[c.State]++
	}

	type segment struct {
		state string
		label string
		count int
	}
	var out []segment
	add := func(state, label string, n int) {
		if n > 0 {
			out = append(out, segment{state: state, label: label, count: n})
		}
	}

	add("running", "running", counts["running"])
	stopped := counts["exited"] + counts["dead"]
	add("exited", "stopped", stopped)
	add("restarting", "restarting", counts["restarting"])
	add("paused", "paused", counts["paused"])
	add("created", "created", counts["created"])
	add("removing", "removing", counts["removing"])

	known := map[string]bool{
		"running": true, "restarting": true, "paused": true,
		"created": true, "removing": true, "exited": true, "dead": true,
	}
	otherStates := make([]string, 0)
	for state, n := range counts {
		if known[state] || n == 0 {
			continue
		}
		otherStates = append(otherStates, state)
	}
	sort.Strings(otherStates)
	for _, state := range otherStates {
		add(state, state, counts[state])
	}

	result := make([]struct {
		state string
		label string
		count int
	}, len(out))
	for i, s := range out {
		result[i] = struct {
			state string
			label string
			count int
		}{s.state, s.label, s.count}
	}
	return result
}

func (m Model) hostCPUHistory() []float64 {
	if len(m.cpuHistory) == 0 {
		return nil
	}
	maxLen := 0
	for _, h := range m.cpuHistory {
		if len(h) > maxLen {
			maxLen = len(h)
		}
	}
	if maxLen == 0 {
		return nil
	}
	out := make([]float64, maxLen)
	counts := make([]int, maxLen)
	for _, h := range m.cpuHistory {
		offset := maxLen - len(h)
		for i, v := range h {
			out[offset+i] += v
			counts[offset+i]++
		}
	}
	for i := range out {
		if counts[i] > 0 {
			out[i] /= float64(counts[i])
		}
	}
	return out
}

func (m Model) avgRunningCPUPercent() float64 {
	var sum float64
	var n int
	for _, c := range m.containers {
		if c.State != "running" {
			continue
		}
		if s, ok := m.stats[c.ID]; ok {
			sum += s.CPUPercent
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

func formatStateSummarySegment(state, label string, count int) string {
	glyph := stateGlyph(state)
	if label == "stopped" {
		glyph = glyphStateStopped
	}
	text := fmt.Sprintf("%s %d %s", glyph, count, label)
	return lipgloss.NewStyle().Foreground(stateFGColor(state)).Bold(true).Render(text)
}

func formatStateSummaryCompact(state string, count int) string {
	glyph := stateGlyph(state)
	if state == "exited" {
		glyph = glyphStateStopped
	}
	return lipgloss.NewStyle().Foreground(stateFGColor(state)).Bold(true).
		Render(fmt.Sprintf("%s%d", glyph, count))
}

func (m Model) renderRefreshMeta() string {
	var parts []string
	if !m.lastRefresh.IsZero() {
		age := time.Since(m.lastRefresh).Round(time.Second)
		parts = append(parts, lipgloss.NewStyle().Foreground(colorDim).Render(
			fmt.Sprintf("⟳ %s ago", formatUptime(age))))
	}
	if m.refreshInterval > 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(colorDim).Render(
			fmt.Sprintf("every %s", config.FormatRefreshInterval(m.refreshInterval))))
	}
	return strings.Join(parts, "   ")
}

func (m Model) renderStatusStrip(w int) string {
	summary := m.containerStateSummary()

	labelStyle := lipgloss.NewStyle().Foreground(colorMuted).Bold(true)

	var countParts []string
	countParts = append(countParts, labelStyle.Render("CONTAINERS"))
	for _, sc := range summary {
		if w < 80 {
			countParts = append(countParts, formatStateSummaryCompact(sc.state, sc.count))
		} else {
			countParts = append(countParts, formatStateSummarySegment(sc.state, sc.label, sc.count))
		}
	}
	line1 := strings.Join(countParts, "   ")
	meta := m.renderRefreshMeta()

	if w < 80 {
		if meta != "" {
			return truncateDisplay(line1+"   "+meta, w)
		}
		return truncateDisplay(line1, w)
	}

	var line2 strings.Builder
	if meta != "" {
		line2.WriteString(meta)
	}
	mem := m.systemMem
	load := m.systemLoad

	if hist := m.hostCPUHistory(); len(hist) > 0 {
		if line2.Len() > 0 {
			line2.WriteString("   ")
		}
		sparkW := 10
		if w >= 140 {
			sparkW = 14
		}
		avg := m.avgRunningCPUPercent()
		line2.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Bold(true).Render("CPU "))
		line2.WriteString(sparklineColored(hist, sparkW, 100, colorPrimary))
		line2.WriteString(lipgloss.NewStyle().Foreground(colorSubtext).Render(fmt.Sprintf("  %.0f%%", avg)))
		line2.WriteString("   ")
	}

	if mem.Total > 0 {
		barW := 16
		if w < 120 {
			barW = 10
		}
		line2.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Bold(true).Render("MEM "))
		line2.WriteString(hostMemBar(mem.Percent, barW))
		line2.WriteString(lipgloss.NewStyle().Foreground(colorDim).Render(
			fmt.Sprintf("  %s/%s", formatBytes(mem.Used), formatBytes(mem.Total))))
		line2.WriteString("   ")
	}

	if load.Load1 > 0 {
		line2.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Bold(true).Render("LOAD "))
		line2.WriteString(lipgloss.NewStyle().Foreground(colorSubtext).Render(
			fmt.Sprintf("%.1f %.1f %.1f", load.Load1, load.Load5, load.Load15)))
	}

	if line2.Len() == 0 {
		return truncateDisplay(line1, w)
	}
	return truncateDisplay(line1, w) + "\n" + truncateDisplay(line2.String(), w)
}

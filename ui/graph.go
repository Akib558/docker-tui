package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var sparkBlocks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

func sparkline(data []float64, width int, maxVal float64) string {
	if width <= 0 {
		return ""
	}
	if len(data) == 0 {
		return strings.Repeat(string(sparkBlocks[0]), width)
	}
	if len(data) > width {
		data = data[len(data)-width:]
	}
	if maxVal <= 0 {
		for _, v := range data {
			if v > maxVal {
				maxVal = v
			}
		}
	}
	if maxVal == 0 {
		maxVal = 1
	}
	var b strings.Builder
	for i := 0; i < width-len(data); i++ {
		b.WriteRune(sparkBlocks[0])
	}
	for _, v := range data {
		idx := int(math.Round(v / maxVal * float64(len(sparkBlocks)-1)))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(sparkBlocks) {
			idx = len(sparkBlocks) - 1
		}
		b.WriteRune(sparkBlocks[idx])
	}
	return b.String()
}

func progressBar(percent float64, width int, fillColor, emptyColor lipgloss.Color) string {
	if width <= 0 {
		return ""
	}
	if percent > 100 {
		percent = 100
	}
	if percent < 0 {
		percent = 0
	}
	filled := int(math.Round(percent / 100 * float64(width)))
	empty := width - filled
	return lipgloss.NewStyle().Foreground(fillColor).Render(strings.Repeat("━", filled)) +
		lipgloss.NewStyle().Foreground(emptyColor).Render(strings.Repeat("─", empty))
}

func miniBar(percent float64, width int) string {
	if width <= 5 {
		return fmt.Sprintf("%3.0f%%", percent)
	}
	barWidth := width - 5
	if barWidth < 2 {
		barWidth = 2
	}
	barPct := percent
	if barPct > 100 {
		barPct = 100
	}
	filled := int(math.Round(barPct / 100 * float64(barWidth)))
	empty := barWidth - filled

	var fillColor lipgloss.Color
	switch {
	case percent >= 80:
		fillColor = colorDanger
	case percent >= 50:
		fillColor = colorWarning
	default:
		fillColor = colorPrimary
	}
	bar := lipgloss.NewStyle().Foreground(fillColor).Render(strings.Repeat("█", filled)) +
		lipgloss.NewStyle().Foreground(colorDim).Render(strings.Repeat("░", empty))
	return bar + " " + lipgloss.NewStyle().Foreground(colorSubtext).Render(fmt.Sprintf("%3.0f%%", percent))
}

func hostMemBar(percent float64, width int) string {
	if width <= 4 {
		return fmt.Sprintf("%.0f%%", percent)
	}
	barW := width - 5
	if barW < 4 {
		barW = 4
	}
	var fillColor lipgloss.Color
	switch {
	case percent >= 85:
		fillColor = colorDanger
	case percent >= 60:
		fillColor = colorWarning
	default:
		fillColor = colorPrimary
	}
	return progressBar(percent, barW, fillColor, colorDim) + " " +
		lipgloss.NewStyle().Foreground(colorSubtext).Render(fmt.Sprintf("%.0f%%", percent))
}

func sparklineColored(data []float64, width int, maxVal float64, color lipgloss.Color) string {
	return lipgloss.NewStyle().Foreground(color).Render(sparkline(data, width, maxVal))
}

func formatBytes(b uint64) string {
	const (
		kib = 1024
		mib = kib * 1024
		gib = mib * 1024
	)
	switch {
	case b >= gib:
		return fmt.Sprintf("%.1f GiB", float64(b)/float64(gib))
	case b >= mib:
		return fmt.Sprintf("%.0f MiB", float64(b)/float64(mib))
	case b >= kib:
		return fmt.Sprintf("%.0f KiB", float64(b)/float64(kib))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

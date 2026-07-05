package ui

import (
	"fmt"
	"strings"

	"github.com/akib558/docker-tui/config"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderDialogOverlay() string {
	var d string
	switch m.dialog {
	case dialogConfirm:
		d = m.renderConfirmDialog()
	case dialogTheme:
		d = m.renderThemeDialog()
	case dialogInput:
		d = m.renderInputDialog()
	case dialogHelp:
		d = m.renderHelpDialog()
		if d == "" {
			return ""
		}
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Top,
			lipgloss.NewStyle().MarginTop(1).Render(d))
	case dialogCommandPalette:
		d = m.renderCommandPaletteDialog()
	}
	if d == "" {
		return ""
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, d)
}

func (m Model) renderConfirmDialog() string {
	title := dialogTitleStyle.Render("⚠  Confirm Action")
	msg := lipgloss.NewStyle().Foreground(colorText).Render(m.confirmMsg)
	btns := "\n" + helpKeyStyle.Render("y") + " " + helpDescStyle.Render("confirm") +
		"  " + lipgloss.NewStyle().Foreground(colorDim).Render("|") + "  " +
		helpKeyStyle.Render("n/esc") + " " + helpDescStyle.Render("cancel")
	content := title + "\n\n" + msg + "\n" + btns
	w := min(64, m.width-8)
	return dialogStyle.Width(w).Render(content)
}

func (m Model) renderThemeDialog() string {
	title := dialogTitleStyle.Render("  Select Theme")
	dialogW := min(44, m.width-8)
	var lines []string
	for i, t := range config.Themes {
		if i == m.themeCursor {
			lines = append(lines, renderSelectableRow(dialogW, ListRowCursor, t.Name))
		} else {
			lines = append(lines, renderSelectableRow(dialogW, ListRowNormal, t.Name))
		}
	}
	help := "\n" + helpKeyStyle.Render("j/k") + " " + helpDescStyle.Render("navigate") +
		"  " + lipgloss.NewStyle().Foreground(colorDim).Render("|") + "  " +
		helpKeyStyle.Render("enter") + " " + helpDescStyle.Render("select") +
		"  " + lipgloss.NewStyle().Foreground(colorDim).Render("|") + "  " +
		helpKeyStyle.Render("esc") + " " + helpDescStyle.Render("cancel")
	content := title + "\n\n" + strings.Join(lines, "\n") + "\n" + help
	return dialogStyle.Width(dialogW).Render(content)
}

func (m Model) renderInputDialog() string {
	title := dialogTitleStyle.Render(m.inputPrompt)
	inputW := min(44, m.width-16)
	cursor := lipgloss.NewStyle().Foreground(colorPrimary).Render("█")
	input := inputStyle.Width(inputW).Render(m.inputText + cursor)
	help := "\n" + helpKeyStyle.Render("enter") + " " + helpDescStyle.Render("submit") +
		"  " + lipgloss.NewStyle().Foreground(colorDim).Render("|") + "  " +
		helpKeyStyle.Render("esc") + " " + helpDescStyle.Render("cancel")
	content := title + "\n\n" + input + "\n" + help
	w := min(54, m.width-8)
	return dialogStyle.Width(w).Render(content)
}

func (m Model) renderHelpDialog() string {
	title := dialogTitleStyle.Render("Keyboard Reference")
	body := m.helpDialogBodyLines()
	maxVisible := m.helpDialogMaxVisible()
	maxScroll := max(0, len(body)-maxVisible)
	scroll := m.helpScroll
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}
	end := min(scroll+maxVisible, len(body))
	visible := body[scroll:end]

	var content strings.Builder
	content.WriteString(title)
	content.WriteString("\n")
	for _, line := range visible {
		content.WriteString(line)
		content.WriteString("\n")
	}

	footer := "\n" + helpKeyStyle.Render("j/k") + " " + helpDescStyle.Render("scroll") +
		"  " + lipgloss.NewStyle().Foreground(colorDim).Render("|") + "  " +
		helpKeyStyle.Render("esc/q/?") + " " + helpDescStyle.Render("close")
	if maxScroll > 0 {
		pct := float64(scroll) / float64(maxScroll) * 100
		footer += "  " + lipgloss.NewStyle().Foreground(colorMuted).
			Render(fmt.Sprintf("(%.0f%%)", pct))
	}

	w := min(64, m.width-4)
	return dialogStyle.Width(w).Render(content.String() + footer)
}

func (m Model) helpDialogMaxVisible() int {
	// margin(1) + dialog chrome(4) + title(1) + footer(2)
	n := m.height - 10
	if n < 4 {
		return 4
	}
	return n
}

func (m Model) helpDialogMaxScroll() int {
	return max(0, len(m.helpDialogBodyLines())-m.helpDialogMaxVisible())
}

func (m Model) helpDialogBodyLines() []string {
	renderSection := func(name string, keys []struct{ key, desc string }) []string {
		lines := []string{
			lipgloss.NewStyle().Bold(true).Foreground(colorSecondary).Render(name),
		}
		for _, k := range keys {
			lines = append(lines, "  "+helpKeyStyle.Width(12).Render(k.key)+" "+helpDescStyle.Render(k.desc))
		}
		return lines
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, renderSection("Common", []struct{ key, desc string }{
		{"ctrl+c", "quit immediately"},
		{"q", "quit / go back"},
		{"esc", "cancel / go back"},
		{"?", "show this help"},
		{":", "command palette"},
		{"t", "theme picker"},
	})...)
	lines = append(lines, "")
	lines = append(lines, renderSection("Container List", []struct{ key, desc string }{
		{"j/k ↑↓", "navigate"},
		{"g/G", "first / last"},
		{"enter/l", "open details"},
		{"space", "toggle selection"},
		{"a", "select all / none"},
		{"/", "filter"},
		{"C", "clear filter"},
		{"c", "compose grouping"},
		{"S", "cycle sort mode"},
		{"s", "start / stop"},
		{"P", "pause / unpause"},
		{"R", "restart"},
		{"K", "kill (SIGKILL)"},
		{"X", "system prune"},
		{"d", "remove"},
		{"e", "exec shell"},
		{"D", "docker events"},
		{"L", "centralized logs"},
		{"N", "notification center"},
		{"i", "images view"},
		{"v", "volumes view"},
		{"n", "networks view"},
		{"+/-", "refresh interval"},
		{"r", "force refresh"},
	})...)
	lines = append(lines, "")
	lines = append(lines, renderSection("Detail View", []struct{ key, desc string }{
		{"1-7", "jump to tab"},
		{"tab/←/→", "switch tab"},
		{"j/k ↑↓", "scroll content"},
		{"pgup/pgdn", "page scroll"},
		{"home/end", "top / bottom"},
		{"l", "toggle live logs"},
		{"L", "toggle log legend"},
		{"space/V", "select log lines"},
		{"f", "fetch filesystem diff"},
		{"p", "refresh process list"},
		{"i", "focus terminal input"},
		{"x", "reconnect terminal"},
		{"ctrl+\\", "detach terminal"},
		{"s/R/P/K/d/e", "container actions (Info/Env tabs only)"},
	})...)
	lines = append(lines, "")
	lines = append(lines, renderSection("Logs", []struct{ key, desc string }{
		{"/", "filter logs"},
		{"j/k", "focus line"},
		{"space/V", "select lines"},
		{"1-9", "toggle container (central)"},
		{"a", "show all containers (central)"},
		{"y", "copy selection/focused line"},
		{"L", "toggle legend"},
		{"r", "toggle regex (central)"},
		{"ctrl+u", "clear filter"},
		{"E", "export to file"},
		{"end", "resume follow"},
	})...)
	lines = append(lines, "")
	lines = append(lines, renderSection("Images / Volumes / Networks", []struct{ key, desc string }{
		{"p", "pull image / prune volumes"},
		{"P", "prune dangling images"},
		{"d", "remove selected"},
		{"space", "toggle selection"},
		{"a", "select all / none"},
		{"/", "filter"},
		{"r", "refresh"},
	})...)
	return lines
}

func (m Model) renderCommandPaletteDialog() string {
	title := dialogTitleStyle.Render("  Command Palette")

	// Input field
	inputW := min(50, m.width-16)
	cursor := lipgloss.NewStyle().Foreground(colorPrimary).Render("█")
	input := inputStyle.Width(inputW).Render(">" + m.commandPaletteText + cursor)

	// Results
	var results strings.Builder
	maxResults := 10
	startIdx := 0
	if m.commandPaletteCursor >= maxResults {
		startIdx = m.commandPaletteCursor - maxResults + 1
	}
	endIdx := min(startIdx+maxResults, len(m.commandPaletteResults))

	w := min(60, m.width-8)
	for i := startIdx; i < endIdx; i++ {
		cmd := m.commandPaletteResults[i]
		line := cmd.Name + " - " + cmd.Description
		if i == m.commandPaletteCursor {
			results.WriteString(renderSelectableRow(w, ListRowCursor, line) + "\n")
		} else {
			results.WriteString(renderSelectableRow(w, ListRowNormal, line) + "\n")
		}
	}

	if len(m.commandPaletteResults) == 0 {
		results.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render("  No matching commands") + "\n")
	}

	help := "\n" + helpKeyStyle.Render("↑/↓") + " " + helpDescStyle.Render("navigate") +
		"  " + lipgloss.NewStyle().Foreground(colorDim).Render("|") + "  " +
		helpKeyStyle.Render("enter") + " " + helpDescStyle.Render("execute") +
		"  " + lipgloss.NewStyle().Foreground(colorDim).Render("|") + "  " +
		helpKeyStyle.Render("esc") + " " + helpDescStyle.Render("cancel")

	content := title + "\n\n" + input + "\n\n" + results.String() + help
	return dialogStyle.Width(w).Render(content)
}

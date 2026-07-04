package ui

import (
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
	var lines []string
	for i, t := range config.Themes {
		if i == m.themeCursor {
			line := cursorStyle.Render("▸ ") + listItemSelStyle.Render(" "+t.Name+" ")
			lines = append(lines, line)
		} else {
			lines = append(lines, "  "+lipgloss.NewStyle().Foreground(colorText).Render(t.Name))
		}
	}
	help := "\n" + helpKeyStyle.Render("j/k") + " " + helpDescStyle.Render("navigate") +
		"  " + lipgloss.NewStyle().Foreground(colorDim).Render("|") + "  " +
		helpKeyStyle.Render("enter") + " " + helpDescStyle.Render("select") +
		"  " + lipgloss.NewStyle().Foreground(colorDim).Render("|") + "  " +
		helpKeyStyle.Render("esc") + " " + helpDescStyle.Render("cancel")
	content := title + "\n\n" + strings.Join(lines, "\n") + "\n" + help
	w := min(44, m.width-8)
	return dialogStyle.Width(w).Render(content)
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
	title := dialogTitleStyle.Render("  Keyboard Reference")

	renderSection := func(name string, keys []struct{ key, desc string }) string {
		var b strings.Builder
		b.WriteString("\n" + lipgloss.NewStyle().Bold(true).Foreground(colorSecondary).Render(name) + "\n")
		for _, k := range keys {
			b.WriteString("  " + helpKeyStyle.Width(12).Render(k.key) + " " + helpDescStyle.Render(k.desc) + "\n")
		}
		return b.String()
	}

	var content strings.Builder
	content.WriteString(title)

	content.WriteString(renderSection("Common", []struct{ key, desc string }{
		{"ctrl+c", "quit immediately"},
		{"q", "quit / go back"},
		{"esc", "cancel / go back"},
		{"?", "show this help"},
		{"t", "theme picker"},
	}))

	content.WriteString(renderSection("Container List", []struct{ key, desc string }{
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
		{"L", "centralized logs"},
		{"N", "notification center"},
		{"i", "images view"},
		{"v", "volumes view"},
		{"n", "networks view"},
		{"+/-", "refresh interval"},
		{"r", "force refresh"},
	}))

	content.WriteString(renderSection("Detail View", []struct{ key, desc string }{
		{"1-7", "jump to tab"},
		{"tab/←/→", "switch tab"},
		{"j/k ↑↓", "scroll content"},
		{"pgup/pgdn", "page scroll"},
		{"home/end", "top / bottom"},
		{"l", "toggle live logs"},
		{"f", "fetch filesystem diff"},
		{"p", "refresh process list"},
		{"x", "reconnect terminal"},
		{"ctrl+\\", "detach terminal"},
		{"s", "start / stop"},
		{"P", "pause / unpause"},
		{"R", "restart"},
		{"K", "kill (SIGKILL)"},
		{"d", "remove"},
		{"e", "exec shell"},
	}))

	content.WriteString(renderSection("Logs", []struct{ key, desc string }{
		{"/", "filter logs"},
		{"r", "toggle regex (central)"},
		{"ctrl+u", "clear filter"},
		{"y", "copy line (central)"},
		{"E", "export to file"},
		{"end", "resume follow"},
	}))

	content.WriteString(renderSection("Images / Volumes / Networks", []struct{ key, desc string }{
		{"p", "pull image / prune volumes"},
		{"P", "prune dangling images"},
		{"d", "remove selected"},
		{"space", "toggle selection"},
		{"a", "select all / none"},
		{"/", "filter"},
		{"r", "refresh"},
	}))

	help := "\n" + helpKeyStyle.Render("esc/q/?") + " " + helpDescStyle.Render("close")
	w := min(64, m.width-8)
	return dialogStyle.Width(w).Render(content.String() + "\n" + help)
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

	for i := startIdx; i < endIdx; i++ {
		cmd := m.commandPaletteResults[i]
		if i == m.commandPaletteCursor {
			line := cursorStyle.Render("▸ ") + listItemSelStyle.Render(cmd.Name+" ") + helpDescStyle.Render(" - "+cmd.Description)
			results.WriteString(line + "\n")
		} else {
			line := "  " + lipgloss.NewStyle().Foreground(colorText).Render(cmd.Name) + helpDescStyle.Render(" - "+cmd.Description)
			results.WriteString(line + "\n")
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

	w := min(60, m.width-8)
	content := title + "\n\n" + input + "\n\n" + results.String() + help
	return dialogStyle.Width(w).Render(content)
}

package ui

import (
	"strings"

	"github.com/akib/docker-tui/config"
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

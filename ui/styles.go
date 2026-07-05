package ui

import (
	"github.com/akib558/docker-tui/config"
	"github.com/charmbracelet/lipgloss"
)

// ── Color vars (updated by applyTheme) ──────────────────────────────────

var (
	colorPrimary    lipgloss.Color
	colorSecondary  lipgloss.Color
	colorSuccess    lipgloss.Color
	colorDanger     lipgloss.Color
	colorWarning    lipgloss.Color
	colorMuted      lipgloss.Color
	colorText       lipgloss.Color
	colorSubtext    lipgloss.Color
	colorBgAlt      lipgloss.Color
	colorBgSelected lipgloss.Color
	colorBorder     lipgloss.Color
	colorDim        lipgloss.Color
	colorBarTrack   lipgloss.Color
	colorCyan       lipgloss.Color
	colorTitleFg    lipgloss.Color
)

// ── Style vars (rebuilt by rebuildStyles) ────────────────────────────────

var (
	titleStyle         lipgloss.Style
	listHeaderStyle    lipgloss.Style
	statusRunning      lipgloss.Style
	statusStopped      lipgloss.Style
	statusOther        lipgloss.Style
	detailBoxStyle     lipgloss.Style
	detailLabelStyle   lipgloss.Style
	detailValueStyle   lipgloss.Style
	sectionHeaderStyle lipgloss.Style
	activeTabStyle     lipgloss.Style
	inactiveTabStyle   lipgloss.Style
	helpBarStyle       lipgloss.Style
	helpKeyStyle       lipgloss.Style
	helpDescStyle      lipgloss.Style
	notifySuccessStyle lipgloss.Style
	notifyErrorStyle   lipgloss.Style
	tableHeaderStyle   lipgloss.Style
	dialogStyle        lipgloss.Style
	dialogTitleStyle   lipgloss.Style
	inputStyle         lipgloss.Style
	filterBarStyle     lipgloss.Style
	selectedMarkStyle  lipgloss.Style
)

func init() {
	applyTheme(config.FindTheme("dark-green"))
}

// applyTheme updates all color and style variables to the given theme.
func applyTheme(t *config.Theme) {
	colorPrimary = lipgloss.Color(t.Primary)
	colorSecondary = lipgloss.Color(t.Secondary)
	colorSuccess = lipgloss.Color(t.Success)
	colorDanger = lipgloss.Color(t.Danger)
	colorWarning = lipgloss.Color(t.Warning)
	colorMuted = lipgloss.Color(t.Muted)
	colorText = lipgloss.Color(t.Text)
	colorSubtext = lipgloss.Color(t.Subtext)
	colorBgAlt = lipgloss.Color(t.BgAlt)
	colorBgSelected = lipgloss.Color(t.BgSelected)
	colorBorder = lipgloss.Color(t.Border)
	colorDim = lipgloss.Color(t.Dim)
	colorBarTrack = lipgloss.Color(t.Border)
	colorCyan = lipgloss.Color(t.Cyan)
	colorTitleFg = lipgloss.Color(t.TitleFg)
	rebuildStyles()
}

func rebuildStyles() {
	titleStyle = lipgloss.NewStyle().
		Bold(true).Foreground(colorTitleFg).Background(colorPrimary).Padding(0, 2)

	listHeaderStyle = lipgloss.NewStyle().
		Bold(true).Foreground(colorMuted).
		BorderBottom(true).BorderStyle(lipgloss.NormalBorder()).BorderForeground(colorBorder)

	statusRunning = lipgloss.NewStyle().Bold(true).Foreground(colorSuccess)
	statusStopped = lipgloss.NewStyle().Bold(true).Foreground(colorDanger)
	statusOther = lipgloss.NewStyle().Bold(true).Foreground(colorWarning)

	detailBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(colorBorder).Padding(1, 2)

	detailLabelStyle = lipgloss.NewStyle().Foreground(colorSecondary).Bold(true).Width(16)
	detailValueStyle = lipgloss.NewStyle().Foreground(colorText)

	sectionHeaderStyle = lipgloss.NewStyle().
		Bold(true).Foreground(colorPrimary).
		BorderBottom(true).BorderStyle(lipgloss.NormalBorder()).BorderForeground(colorBorder)

	activeTabStyle = lipgloss.NewStyle().
		Bold(true).Foreground(colorTitleFg).Background(colorPrimary).
		Padding(0, 1)

	inactiveTabStyle = lipgloss.NewStyle().
		Foreground(colorMuted).Padding(0, 1)

	helpBarStyle = lipgloss.NewStyle().
		Foreground(colorMuted).Background(colorBgAlt).Padding(0, 1).
		BorderTop(true).BorderStyle(lipgloss.NormalBorder()).BorderForeground(colorBorder)
	helpKeyStyle = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	helpDescStyle = lipgloss.NewStyle().Foreground(colorMuted)

	notifySuccessStyle = lipgloss.NewStyle().
		Foreground(colorSuccess).Bold(true).
		BorderLeft(true).BorderStyle(lipgloss.NormalBorder()).BorderForeground(colorSuccess).
		PaddingLeft(1)
	notifyErrorStyle = lipgloss.NewStyle().
		Foreground(colorDanger).Bold(true).
		BorderLeft(true).BorderStyle(lipgloss.NormalBorder()).BorderForeground(colorDanger).
		PaddingLeft(1)

	tableHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(colorMuted)

	dialogStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(colorPrimary).
		Padding(1, 3).Background(colorBgAlt)

	dialogTitleStyle = lipgloss.NewStyle().
		Bold(true).Foreground(colorPrimary).MarginBottom(1)

	inputStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).BorderForeground(colorPrimary).
		Foreground(colorText).Padding(0, 1)

	filterBarStyle = lipgloss.NewStyle().
		Foreground(colorWarning).Bold(true).
		Background(colorBgAlt).Padding(0, 1).
		BorderLeft(true).BorderStyle(lipgloss.NormalBorder()).BorderForeground(colorWarning)

	selectedMarkStyle = lipgloss.NewStyle().Foreground(colorWarning).Bold(true)
}

func stateStyle(state string) lipgloss.Style {
	switch state {
	case "running":
		return statusRunning
	case "exited", "dead":
		return statusStopped
	default:
		return statusOther
	}
}

func stateIcon(state string) string { return stateGlyph(state) }

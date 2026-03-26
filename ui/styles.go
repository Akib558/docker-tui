package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Dark green color palette
	colorPrimary    = lipgloss.Color("#00E676") // bright green
	colorSecondary  = lipgloss.Color("#69F0AE") // light green
	colorAccent     = lipgloss.Color("#B9F6CA") // pale green
	colorSuccess    = lipgloss.Color("#00E676") // green
	colorDanger     = lipgloss.Color("#FF5252") // red
	colorWarning    = lipgloss.Color("#FFD740") // amber
	colorMuted      = lipgloss.Color("#4E6E5D") // muted green-gray
	colorText       = lipgloss.Color("#E8F5E9") // near white green tint
	colorSubtext    = lipgloss.Color("#A5D6A7") // light green gray
	colorBgAlt      = lipgloss.Color("#0D2818") // dark green surface
	colorBgSelected = lipgloss.Color("#1B5E20") // selected row green
	colorBorder     = lipgloss.Color("#2E7D32") // green border
	colorDim        = lipgloss.Color("#1B5E20") // dim green for decorative

	// Title / header bar
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#0A0F0D")).
			Background(colorPrimary).
			Padding(0, 2)

	// Container list styles
	listHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorSecondary).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colorDim)

	listItemStyle = lipgloss.NewStyle().
			Foreground(colorText).
			PaddingLeft(2)

	listItemSelectedStyle = lipgloss.NewStyle().
				Foreground(colorText).
				Background(colorBgSelected).
				Bold(true).
				PaddingLeft(1)

	// Status badges
	statusRunning = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorSuccess)

	statusStopped = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorDanger)

	statusOther = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWarning)

	// Detail panel styles
	detailBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 2)

	detailLabelStyle = lipgloss.NewStyle().
				Foreground(colorSecondary).
				Bold(true).
				Width(16)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(colorText)

	// Section headers in detail view
	sectionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorPrimary).
				MarginTop(1).
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(colorDim)

	// Tab styles
	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#0A0F0D")).
			Background(colorPrimary).
			Padding(0, 2)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				Background(colorBgAlt).
				Padding(0, 2)

	// Help bar
	helpBarStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	// Notifications
	notifySuccessStyle = lipgloss.NewStyle().
				Foreground(colorSuccess).
				Bold(true)

	notifyErrorStyle = lipgloss.NewStyle().
				Foreground(colorDanger).
				Bold(true)

	// Table column header
	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorSecondary).
				PaddingRight(2)

	// Cursor for selected row
	cursorStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	// Stat card styles
	statCardBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 2).
			Align(lipgloss.Center)

	statCardLabel = lipgloss.NewStyle().
			Foreground(colorMuted).
			Bold(true).
			Align(lipgloss.Center)

	statCardValue = lipgloss.NewStyle().
			Bold(true).
			Align(lipgloss.Center)
)

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

func stateIcon(state string) string {
	switch state {
	case "running":
		return "●"
	case "exited":
		return "○"
	case "paused":
		return "◑"
	case "restarting":
		return "↻"
	case "dead":
		return "✕"
	case "created":
		return "◇"
	default:
		return "?"
	}
}

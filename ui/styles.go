package ui

import "github.com/charmbracelet/lipgloss"

var (
	// ── Dark green palette ──────────────────────────────────────────
	colorPrimary    = lipgloss.Color("#00E676")
	colorSecondary  = lipgloss.Color("#69F0AE")
	colorAccent     = lipgloss.Color("#B9F6CA")
	colorSuccess    = lipgloss.Color("#00E676")
	colorDanger     = lipgloss.Color("#FF5252")
	colorWarning    = lipgloss.Color("#FFD740")
	colorMuted      = lipgloss.Color("#4E6E5D")
	colorText       = lipgloss.Color("#E8F5E9")
	colorSubtext    = lipgloss.Color("#A5D6A7")
	colorBgAlt      = lipgloss.Color("#0D2818")
	colorBgSelected = lipgloss.Color("#1B5E20")
	colorBorder     = lipgloss.Color("#2E7D32")
	colorDim        = lipgloss.Color("#1B5E20")
	colorCyan       = lipgloss.Color("#00BCD4")

	// ── Title / header ──────────────────────────────────────────────
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#0A0F0D")).
			Background(colorPrimary).
			Padding(0, 2)

	// ── List ────────────────────────────────────────────────────────
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

	// ── Status indicators ───────────────────────────────────────────
	statusRunning = lipgloss.NewStyle().Bold(true).Foreground(colorSuccess)
	statusStopped = lipgloss.NewStyle().Bold(true).Foreground(colorDanger)
	statusOther   = lipgloss.NewStyle().Bold(true).Foreground(colorWarning)

	// ── Detail panel ────────────────────────────────────────────────
	detailBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 2)

	detailLabelStyle = lipgloss.NewStyle().
				Foreground(colorSecondary).
				Bold(true).
				Width(16)

	detailValueStyle = lipgloss.NewStyle().Foreground(colorText)

	sectionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorPrimary).
				MarginTop(1).
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(colorDim)

	// ── Tabs ────────────────────────────────────────────────────────
	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#0A0F0D")).
			Background(colorPrimary).
			Padding(0, 2)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				Background(colorBgAlt).
				Padding(0, 2)

	// ── Help bar ────────────────────────────────────────────────────
	helpBarStyle = lipgloss.NewStyle().Foreground(colorMuted)
	helpKeyStyle = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	helpDescStyle = lipgloss.NewStyle().Foreground(colorMuted)

	// ── Notifications ───────────────────────────────────────────────
	notifySuccessStyle = lipgloss.NewStyle().Foreground(colorSuccess).Bold(true)
	notifyErrorStyle   = lipgloss.NewStyle().Foreground(colorDanger).Bold(true)

	// ── Table ───────────────────────────────────────────────────────
	tableHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(colorSecondary).PaddingRight(2)
	cursorStyle      = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)

	// ── Stat cards ──────────────────────────────────────────────────
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

package ui

// Semantic glyph registry — no glyph literals outside this file.

const (
	glyphLogo         = "⬡"
	glyphRefresh      = "⟳"
	glyphSearch       = "⌕"
	glyphScroll       = "↕"
	glyphWarn         = "⚠"
	glyphStateRunning = "●"
	glyphStateStopped = "○"
	glyphStatePaused  = "◑"
	glyphStateCreated = "◇"
	glyphStateRestart = "↻"
	glyphStateDead    = "✕"
	glyphHealthOK     = "♥"
	glyphHealthBad    = "✕"
	glyphHealthStart  = "…"
	glyphMarkerCursor = "▸"
	glyphMarkerSelect = "◈"
	glyphMarkerBoth   = "◉"
	glyphToastOK      = "✓"
	glyphToastErr     = "✗"
	glyphBarFill      = "█"
	glyphBarEmpty     = "░"
)

func stateGlyph(state string) string {
	switch state {
	case "running":
		return glyphStateRunning
	case "exited":
		return glyphStateStopped
	case "paused":
		return glyphStatePaused
	case "restarting":
		return glyphStateRestart
	case "dead":
		return glyphStateDead
	case "created":
		return glyphStateCreated
	default:
		return "?"
	}
}

func stateDisplayName(state string) string {
	switch state {
	case "exited", "dead":
		return "stopped"
	default:
		return state
	}
}

func healthGlyphPlain(health string) string {
	switch health {
	case "healthy":
		return glyphHealthOK
	case "unhealthy":
		return glyphHealthBad
	case "starting":
		return glyphHealthStart
	default:
		return "?"
	}
}

package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ScreenState selects which body a list screen renders.
type ScreenState int

const (
	ScreenReady   ScreenState = iota // rows are present
	ScreenLoading                    // data is being fetched
	ScreenEmpty                      // no objects exist
	ScreenNoMatch                    // objects exist but the filter matches none
	ScreenError                      // Message holds the error text
)

// ListScreen describes a list view as data. Views build one of these; the
// renderer below is the only thing that turns it into text, so every view
// gets identical chrome in identical order.
type ListScreen struct {
	Title    string
	Count    int    // total objects, before filtering
	Selected int    // multi-selection count; 0 hides the badge
	Filter   string // active filter text; "" hides the filter bar
	Cols     []ListColumn
	Rows     []ListTableRow
	Cursor   int
	State    ScreenState
	Message  string   // body text for Loading / Empty / NoMatch / Error
	Extra    []string // pre-rendered lines between the title and the table
	Help     string   // pre-rendered help bar content
}

// screenChrome counts the non-row lines a screen draws, so BodyRows can be
// derived rather than guessed.
func screenChrome(s ListScreen, hasToast bool) int {
	chrome := 3 // app header — renderHeader (ui/view_list.go) renders 3 lines, not the 2
	// its own code intends: a pre-existing width/padding wrap bug, out of scope
	// for this plan (no task touches renderHeader). Accounted for here as a known
	// fact so a future fix to that bug has a clear place to also update.
	chrome++ // title
	chrome++ // blank line under the title
	for _, e := range s.Extra {
		chrome += strings.Count(e, "\n") + 1
	}
	if s.Filter != "" {
		chrome++
	}
	if s.State == ScreenReady {
		chrome += 2 // table header + its bottom border line
	}
	if hasToast {
		chrome++
	}
	chrome += 2 // help bar: its BorderTop rule plus its content line. There is no blank
	// line before it — renderNotification returns "" (zero bytes) when there's
	// no active toast, and nothing else writes one here.
	return chrome
}

// renderListScreen paints a complete list view.
func (m Model) renderListScreen(s ListScreen) string {
	w := m.width
	var b strings.Builder

	b.WriteString(m.renderHeader(w))
	b.WriteString(renderViewTitle(s.Title, s.Count, s.Selected))
	b.WriteString("\n")

	for _, line := range s.Extra {
		b.WriteString(line + "\n")
	}
	if s.Filter != "" {
		b.WriteString(filterBarStyle.Render(glyphSearch+" "+s.Filter) + "\n")
	}

	if s.State != ScreenReady {
		b.WriteString("\n" + m.renderScreenMessage(w, s) + "\n\n")
		b.WriteString(m.renderNotification())
		b.WriteString(renderHelpBar(w, s.Help))
		return b.String()
	}

	baseChrome := screenChrome(s, m.hasActiveNotification())
	visible := newFrame(w, m.height, baseChrome).BodyRows
	if len(s.Rows) > visible {
		// The scroll indicator itself takes a line that baseChrome didn't
		// reserve. Recompute with it counted, so the indicator doesn't push
		// the output one line past the terminal height — the same
		// two-pass check the pre-scaffold listChrome/listBodyRowsEstimate
		// used for this exact problem.
		visible = newFrame(w, m.height, baseChrome+1).BodyRows
	}

	b.WriteString(m.renderScreenTableHeader(s) + "\n")

	start := 0
	if s.Cursor >= visible {
		start = s.Cursor - visible + 1
	}
	end := min(start+visible, len(s.Rows))
	for i := start; i < end; i++ {
		kind := listRowKind(i, s.Cursor, s.Rows[i].Selected)
		b.WriteString(renderRowFromKind(w, kind, i, s.Rows[i].Cells) + "\n")
	}

	if len(s.Rows) > visible {
		b.WriteString(m.renderScrollIndicator(s.Cursor, len(s.Rows)) + "\n")
	}

	b.WriteString(m.renderNotification())
	b.WriteString(renderHelpBar(w, s.Help))
	return b.String()
}

// renderScreenTableHeader builds the header from the same columns the rows
// use, so the two can never disagree.
func (m Model) renderScreenTableHeader(s ListScreen) string {
	cells := make([]Cell, len(s.Cols))
	for i, c := range s.Cols {
		cells[i] = Cell{Text: c.Label, Width: c.Width, FG: colorMuted, Bold: true}
	}
	row := RenderRow(m.width, cells, RowStyle{Marker: " "})
	return listHeaderStyle.Render(row)
}

func (m Model) renderScreenMessage(w int, s ListScreen) string {
	style := lipgloss.NewStyle().Foreground(colorMuted).Italic(true)
	if s.State == ScreenError {
		style = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).BorderForeground(colorDanger).
			Foreground(colorDanger).Padding(1, 2).Width(min(w-4, 70))
	}
	return lipgloss.PlaceHorizontal(w, lipgloss.Center, style.Render(s.Message))
}

func (m Model) renderScrollIndicator(cursor, total int) string {
	pct := float64(cursor) / float64(max(total-1, 1)) * 100
	return lipgloss.NewStyle().Foreground(colorMuted).
		Render(fmt.Sprintf("  %s %d/%d (%.0f%%)", glyphScroll, cursor+1, total, pct))
}

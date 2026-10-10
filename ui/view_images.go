package ui

import (
	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewImages() string {
	cols := FitColumns([]ColumnSpec{
		{Label: "TAG", Percent: 45, Min: 14},
		{Label: "IMAGE ID", Width: 14, Priority: 3},
		{Label: "SIZE", Width: 10, Priority: 1},
		{Label: "CREATED", Width: 16, Priority: 2},
	}, TableW(m.width))

	s := ListScreen{
		Title:  "Images",
		Count:  len(m.images),
		Cols:   cols,
		Cursor: m.imgCursor,
		State:  ScreenReady,
		Help:   m.imagesHelpKeys(),
	}
	if m.imagePullProgress != "" {
		s.Extra = []string{
			lipgloss.NewStyle().Foreground(colorSecondary).
				Render("  Pull: " + truncate(m.imagePullProgress, max(m.width-10, 20))),
		}
	}

	switch {
	case m.loading:
		s.State, s.Message = ScreenLoading, "Loading images..."
		return m.renderListScreen(s)
	case len(m.images) == 0:
		s.State, s.Message = ScreenEmpty, "No images found."
		return m.renderListScreen(s)
	}

	m.fillVisibleListRows(&s, len(m.images), func(i int) ListTableRow {
		img := m.images[i]
		cells := []Cell{{Text: truncate(img.DisplayTag(), cols[0].Width), Width: cols[0].Width, FG: colorText}}
		for j, c := range cols[1:] {
			var text string
			var fg lipgloss.Color
			switch c.Label {
			case "IMAGE ID":
				text, fg = img.ID, colorDim
			case "SIZE":
				text, fg = formatBytes(uint64(img.Size)), colorSubtext
			case "CREATED":
				text, fg = img.Created.Format("2006-01-02 15:04"), colorMuted
			}
			cells = append(cells, Cell{Text: truncate(text, cols[j+1].Width), Width: cols[j+1].Width, FG: fg})
		}
		return ListTableRow{Cells: cells}
	})
	return m.renderListScreen(s)
}

func (m Model) imagesHelpKeys() string {
	return fmtKeys([]struct{ key, desc string }{
		{"j/k", "nav"},
		{"p", "pull"},
		{"P", "prune"},
		{"d", "remove"},
		{"?", "help"},
		{"esc", "back"},
	})
}

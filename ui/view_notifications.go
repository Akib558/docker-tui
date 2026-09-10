package ui

func (m Model) viewNotifications() string {
	cols := FitColumns([]ColumnSpec{
		{Label: "TIME", Width: 8},
		{Label: "", Width: 1},
		{Label: "MESSAGE", Percent: 100, Min: 20},
	}, TableW(m.width))

	s := ListScreen{
		Title:  "Notifications",
		Count:  len(m.notifyHistory),
		Cols:   cols,
		Cursor: m.notifyCursor,
		State:  ScreenReady,
		Help:   m.notificationsHelpKeys(),
	}

	if len(m.notifyHistory) == 0 {
		s.State = ScreenEmpty
		s.Message = "No notifications yet."
		return m.renderListScreen(s)
	}

	s.Rows = make([]ListTableRow, 0, len(m.notifyHistory))
	for _, n := range m.notifyHistory {
		icon, fg := glyphToastOK, colorSuccess
		if n.IsError {
			icon, fg = glyphToastErr, colorDanger
		}
		s.Rows = append(s.Rows, ListTableRow{Cells: []Cell{
			{Text: n.Timestamp.Format("15:04:05"), Width: cols[0].Width, FG: colorMuted},
			{Text: icon, Width: cols[1].Width, FG: fg},
			{Text: truncate(n.Message, cols[2].Width), Width: cols[2].Width, FG: colorText},
		}})
	}
	return m.renderListScreen(s)
}

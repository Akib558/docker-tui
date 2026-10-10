package ui

func listScrollWindow(cursor, total, visible int) (start, end int) {
	if visible < 1 {
		visible = 1
	}
	start = 0
	if cursor >= visible {
		start = cursor - visible + 1
	}
	end = min(start+visible, total)
	return start, end
}

func (m Model) listVisibleBodyRows(s ListScreen, totalRows int) int {
	baseChrome := screenChrome(s, m.hasActiveNotification())
	visible := newFrame(m.width, m.height, baseChrome).BodyRows
	if totalRows > visible {
		visible = newFrame(m.width, m.height, baseChrome+1).BodyRows
	}
	if visible < 1 {
		return 1
	}
	return visible
}

// fillVisibleListRows builds table rows only for the scroll window.
func (m Model) fillVisibleListRows(s *ListScreen, total int, at func(i int) ListTableRow) {
	if total == 0 {
		s.Rows = nil
		s.RowCount = 0
		s.RowOffset = 0
		return
	}
	visible := m.listVisibleBodyRows(*s, total)
	start, end := listScrollWindow(s.Cursor, total, visible)
	s.RowCount = total
	s.RowOffset = start
	s.Rows = make([]ListTableRow, 0, end-start)
	for i := start; i < end; i++ {
		s.Rows = append(s.Rows, at(i))
	}
}

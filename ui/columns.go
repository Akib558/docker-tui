package ui

// ColumnSpec declares one column's width policy. Views declare specs; the
// fitter turns them into concrete ListColumns for the available width.
type ColumnSpec struct {
	Label    string
	Width    int // fixed display cells; when > 0, Percent and Min are ignored
	Percent  int // share of the flexible width, 0-100
	Min      int // floor below which the column is dropped instead of shrunk
	Priority int // higher drops first; 0 never drops
}

// FitColumns lays specs out into availW display columns, accounting for the
// single-space gap the row renderer places between columns. Columns are
// dropped highest-Priority-first when the remainder cannot honour every Min.
func FitColumns(specs []ColumnSpec, availW int) []ListColumn {
	if availW < 1 || len(specs) == 0 {
		return nil
	}
	keep := append([]ColumnSpec(nil), specs...)
	for {
		cols, ok := layoutColumns(keep, availW)
		if ok || !dropLowestPriority(&keep) {
			return cols
		}
	}
}

// layoutColumns distributes availW across specs, reporting whether every
// column cleared its Min.
func layoutColumns(specs []ColumnSpec, availW int) ([]ListColumn, bool) {
	if len(specs) == 0 {
		return nil, false
	}
	flexible := availW - (len(specs) - 1) // gaps between columns
	pctTotal := 0
	for _, s := range specs {
		if s.Width > 0 {
			flexible -= s.Width
			continue
		}
		pctTotal += s.Percent
	}
	if flexible < 0 {
		flexible = 0
	}

	cols := make([]ListColumn, 0, len(specs))
	used := 0
	lastFlex := -1
	for i, s := range specs {
		if s.Width > 0 {
			cols = append(cols, ListColumn{Label: s.Label, Width: s.Width})
			continue
		}
		w := 0
		if pctTotal > 0 {
			w = flexible * s.Percent / pctTotal
		}
		cols = append(cols, ListColumn{Label: s.Label, Width: w})
		used += w
		lastFlex = i
	}
	// Hand any rounding remainder to the last flexible column so the row
	// fills the width exactly.
	if lastFlex >= 0 && used < flexible {
		cols[lastFlex].Width += flexible - used
	}

	ok := true
	for i, s := range specs {
		if s.Width == 0 && cols[i].Width < s.Min {
			ok = false
			break
		}
	}
	return cols, ok
}

// dropLowestPriority removes the highest-Priority (least important) spec.
// It reports false when only never-drop columns remain.
func dropLowestPriority(specs *[]ColumnSpec) bool {
	worst, worstIdx := 0, -1
	for i, s := range *specs {
		if s.Priority > worst {
			worst, worstIdx = s.Priority, i
		}
	}
	if worstIdx < 0 {
		return false
	}
	*specs = append((*specs)[:worstIdx], (*specs)[worstIdx+1:]...)
	return true
}

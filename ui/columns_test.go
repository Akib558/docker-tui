package ui

import "testing"

func specs() []ColumnSpec {
	return []ColumnSpec{
		{Label: "", Width: 3, Priority: 0},
		{Label: "NAME", Percent: 30, Min: 10, Priority: 0},
		{Label: "IMAGE", Percent: 30, Min: 10, Priority: 2},
		{Label: "STATUS", Percent: 25, Min: 8, Priority: 1},
		{Label: "PORTS", Percent: 15, Min: 8, Priority: 3},
	}
}

func totalWidth(cols []ListColumn) int {
	total := 0
	for _, c := range cols {
		total += c.Width
	}
	return total
}

func TestFitColumnsNeverExceedsAvailable(t *testing.T) {
	for _, avail := range []int{20, 33, 50, 73, 120, 193} {
		cols := FitColumns(specs(), avail)
		gaps := 0
		if len(cols) > 1 {
			gaps = len(cols) - 1
		}
		if got := totalWidth(cols) + gaps; got > avail {
			t.Errorf("avail=%d: columns+gaps = %d, want <= %d", avail, got, avail)
		}
	}
}

func TestFitColumnsDropsByPriority(t *testing.T) {
	cols := FitColumns(specs(), 30)
	for _, c := range cols {
		if c.Label == "PORTS" {
			t.Fatalf("PORTS (priority 3) should drop before IMAGE/STATUS at width 30, got %v", cols)
		}
	}
	labels := map[string]bool{}
	for _, c := range cols {
		labels[c.Label] = true
	}
	if !labels["NAME"] {
		t.Fatalf("NAME (priority 0) must never drop, got %v", cols)
	}
}

func TestFitColumnsRespectsFixedWidths(t *testing.T) {
	cols := FitColumns(specs(), 120)
	if cols[0].Width != 3 {
		t.Fatalf("fixed-width column should stay 3, got %d", cols[0].Width)
	}
}

func TestFitColumnsHonoursMinimums(t *testing.T) {
	cols := FitColumns(specs(), 45)
	for _, c := range cols {
		if c.Label == "NAME" && c.Width < 10 {
			t.Fatalf("NAME below its Min of 10: %d", c.Width)
		}
	}
}

// neverDropSpecs mirrors the shape real call sites use where a panic was
// observed: every column is Priority 0 (never-drop), mixing a fixed-width
// column with a flexible one, as in the notifications view, the events
// view, and the detail identity line.
func neverDropSpecs() []ColumnSpec {
	return []ColumnSpec{
		{Label: "TIME", Width: 8},
		{Label: "", Width: 1},
		{Label: "MESSAGE", Percent: 100, Min: 20},
	}
}

// TestFitColumnsNeverPanicsAtNonPositiveWidth is a regression test for a
// panic at terminal widths 1-7: FitColumns used to return nil whenever
// availW < 1 (and TableW(termW) is < 1 for termW <= 7), but every call site
// indexes the result unconditionally (e.g. cols[0].Width). For a never-drop
// spec slice, FitColumns must always return exactly one ListColumn per spec,
// even at availW <= 0, and must not panic.
func TestFitColumnsNeverPanicsAtNonPositiveWidth(t *testing.T) {
	for _, avail := range []int{0, -5, -1000} {
		cols := FitColumns(neverDropSpecs(), avail)
		if len(cols) != len(neverDropSpecs()) {
			t.Fatalf("avail=%d: len(cols) = %d, want %d (one per surviving spec)", avail, len(cols), len(neverDropSpecs()))
		}
		_ = cols[0].Width
		_ = cols[1].Width
		_ = cols[2].Width
	}
}

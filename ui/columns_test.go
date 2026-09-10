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

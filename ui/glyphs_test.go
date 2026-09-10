package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ui/glyphs.go declares itself the only home for glyph literals. This test
// enforces that claim.
func TestNoGlyphLiteralsOutsideRegistry(t *testing.T) {
	banned := []string{"⬡", "⟳", "⌕", "↕", "⚠", "●", "○", "◑", "◇", "↻", "♥", "✕", "▸", "◈", "◉", "✓", "✗", "█", "░", "┃"}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if f == "glyphs.go" || strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, g := range banned {
			if strings.Contains(string(src), g) {
				t.Errorf("%s contains the glyph literal %q — use the glyph* constant from glyphs.go", f, g)
			}
		}
	}
}

package config

import "testing"

// A foreground role must be distinguishable from the backgrounds it is drawn
// over. 1.6:1 is well below the WCAG text minimum, but it is the floor at
// which a terminal colour stops being invisible.
const minContrast = 1.6

func TestDimIsNotABackground(t *testing.T) {
	for _, th := range Themes {
		if th.Dim == th.BgSelected {
			t.Errorf("theme %s: Dim (%s) equals BgSelected — dim text is invisible on the cursor row",
				th.Name, th.Dim)
		}
	}
}

func TestForegroundRolesClearContrastFloor(t *testing.T) {
	for _, th := range Themes {
		fgs := map[string]string{
			"Text": th.Text, "Subtext": th.Subtext, "Muted": th.Muted, "Dim": th.Dim,
			"Success": th.Success, "Danger": th.Danger, "Warning": th.Warning,
			"Primary": th.Primary, "Secondary": th.Secondary,
		}
		bgs := map[string]string{"BgAlt": th.BgAlt, "BgSelected": th.BgSelected}
		for fgName, fg := range fgs {
			for bgName, bg := range bgs {
				ratio, ok := ContrastRatio(fg, bg)
				if !ok {
					t.Fatalf("theme %s: cannot parse %s=%s over %s=%s", th.Name, fgName, fg, bgName, bg)
				}
				if ratio < minContrast {
					t.Errorf("theme %s: %s (%s) over %s (%s) = %.2f, want >= %.2f",
						th.Name, fgName, fg, bgName, bg, ratio, minContrast)
				}
			}
		}
	}
}

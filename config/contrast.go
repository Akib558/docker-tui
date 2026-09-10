package config

import (
	"math"
	"strconv"
	"strings"
)

// Luminance returns the WCAG relative luminance of a "#rrggbb" colour.
func Luminance(hex string) (float64, bool) {
	h := strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(h) != 6 {
		return 0, false
	}
	channel := func(s string) (float64, bool) {
		v, err := strconv.ParseUint(s, 16, 8)
		if err != nil {
			return 0, false
		}
		c := float64(v) / 255
		if c <= 0.03928 {
			return c / 12.92, true
		}
		return math.Pow((c+0.055)/1.055, 2.4), true
	}
	r, okR := channel(h[0:2])
	g, okG := channel(h[2:4])
	b, okB := channel(h[4:6])
	if !okR || !okG || !okB {
		return 0, false
	}
	return 0.2126*r + 0.7152*g + 0.0722*b, true
}

// ContrastRatio returns the WCAG contrast ratio between two "#rrggbb" colours.
func ContrastRatio(fg, bg string) (float64, bool) {
	lf, okF := Luminance(fg)
	lb, okB := Luminance(bg)
	if !okF || !okB {
		return 0, false
	}
	hi, lo := math.Max(lf, lb), math.Min(lf, lb)
	return (hi + 0.05) / (lo + 0.05), true
}

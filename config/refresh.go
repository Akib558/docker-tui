package config

import (
	"fmt"
	"time"
)

// RefreshPresetsMS are the btop-style refresh intervals (fastest → slowest).
var RefreshPresetsMS = []int{500, 1000, 1500, 2000, 3000, 5000, 10000, 30000}

// RefreshDuration returns the configured refresh interval.
func (c *Config) RefreshDuration() time.Duration {
	ms := c.RefreshMS
	if ms <= 0 && c.RefreshSeconds > 0 {
		ms = c.RefreshSeconds * 1000
	}
	if ms <= 0 {
		ms = 2000
	}
	if ms < RefreshPresetsMS[0] {
		ms = RefreshPresetsMS[0]
	}
	return time.Duration(ms) * time.Millisecond
}

// SetRefreshDuration updates both RefreshMS and the legacy RefreshSeconds field.
func (c *Config) SetRefreshDuration(d time.Duration) {
	ms := int(d / time.Millisecond)
	if ms < RefreshPresetsMS[0] {
		ms = RefreshPresetsMS[0]
	}
	c.RefreshMS = ms
	secs := ms / 1000
	if secs < 1 {
		secs = 1
	}
	c.RefreshSeconds = secs
}

// StepRefresh moves to the next faster (faster=true) or slower preset.
func (c *Config) StepRefresh(faster bool) time.Duration {
	cur := c.RefreshMS
	if cur <= 0 {
		cur = c.RefreshSeconds * 1000
	}
	idx := 0
	for i, ms := range RefreshPresetsMS {
		if ms >= cur {
			idx = i
			break
		}
		idx = i
	}
	if faster {
		if idx > 0 {
			idx--
		}
	} else if idx < len(RefreshPresetsMS)-1 {
		idx++
	}
	c.RefreshMS = RefreshPresetsMS[idx]
	secs := c.RefreshMS / 1000
	if secs < 1 {
		secs = 1
	}
	c.RefreshSeconds = secs
	return c.RefreshDuration()
}

// FormatRefreshInterval renders a human-readable refresh rate.
func FormatRefreshInterval(d time.Duration) string {
	ms := d.Milliseconds()
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	s := d.Seconds()
	if s == float64(int(s)) {
		return fmt.Sprintf("%.0fs", s)
	}
	return fmt.Sprintf("%.1fs", s)
}

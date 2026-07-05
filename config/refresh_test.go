package config

import (
	"testing"
	"time"
)

func TestRefreshDurationMigratesLegacySeconds(t *testing.T) {
	cfg := Config{RefreshSeconds: 5}
	if got := cfg.RefreshDuration(); got != 5*time.Second {
		t.Fatalf("RefreshDuration() = %v, want 5s", got)
	}
}

func TestLoadMigratesRefreshMSToLegacySeconds(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeConfig(t, home, `{"refresh_seconds":3}`)

	cfg := Load()
	if cfg.RefreshMS != 3000 {
		t.Fatalf("RefreshMS = %d, want 3000", cfg.RefreshMS)
	}
	if cfg.RefreshSeconds != 3 {
		t.Fatalf("RefreshSeconds = %d, want 3", cfg.RefreshSeconds)
	}
}

func TestStepRefreshPresets(t *testing.T) {
	cfg := Config{RefreshMS: 2000}

	cfg.StepRefresh(true)
	if cfg.RefreshMS != 1500 {
		t.Fatalf("faster from 2s = %dms, want 1500", cfg.RefreshMS)
	}

	cfg.StepRefresh(false)
	if cfg.RefreshMS != 2000 {
		t.Fatalf("slower back = %dms, want 2000", cfg.RefreshMS)
	}

	cfg.RefreshMS = RefreshPresetsMS[0]
	cfg.StepRefresh(true)
	if cfg.RefreshMS != RefreshPresetsMS[0] {
		t.Fatalf("faster at min should stay at %d, got %d", RefreshPresetsMS[0], cfg.RefreshMS)
	}

	cfg.RefreshMS = RefreshPresetsMS[len(RefreshPresetsMS)-1]
	cfg.StepRefresh(false)
	if cfg.RefreshMS != RefreshPresetsMS[len(RefreshPresetsMS)-1] {
		t.Fatalf("slower at max should stay at %d, got %d", RefreshPresetsMS[len(RefreshPresetsMS)-1], cfg.RefreshMS)
	}
}

func TestFormatRefreshInterval(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{500 * time.Millisecond, "500ms"},
		{time.Second, "1s"},
		{1500 * time.Millisecond, "1.5s"},
		{2 * time.Second, "2s"},
	}
	for _, tc := range tests {
		if got := FormatRefreshInterval(tc.d); got != tc.want {
			t.Fatalf("FormatRefreshInterval(%v) = %q, want %q", tc.d, got, tc.want)
		}
	}
}

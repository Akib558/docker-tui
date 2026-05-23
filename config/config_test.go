package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadContainerColorsDoesNotAliasDefaultOrOtherLoads(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Cleanup(func() {
		delete(Default.ContainerColors, "api")
	})

	first := Load()
	second := Load()

	if first.ContainerColors == nil {
		t.Fatal("first load returned nil ContainerColors")
	}
	if second.ContainerColors == nil {
		t.Fatal("second load returned nil ContainerColors")
	}

	first.ContainerColors["api"] = "#111111"
	if got, ok := Default.ContainerColors["api"]; ok {
		t.Errorf("first load aliases Default.ContainerColors: got %q", got)
	}
	if got, ok := second.ContainerColors["api"]; ok {
		t.Errorf("first load aliases second load ContainerColors: got %q", got)
	}
}

func TestLoadConfiguredContainerColorsDoesNotMutateDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Cleanup(func() {
		delete(Default.ContainerColors, "api")
		delete(Default.ContainerColors, "worker")
	})
	writeConfig(t, home, `{"container_colors":{"api":"#111111"}}`)

	cfg := Load()
	if got := cfg.ContainerColors["api"]; got != "#111111" {
		t.Fatalf("loaded api color = %q, want #111111", got)
	}
	if got, ok := Default.ContainerColors["api"]; ok {
		t.Fatalf("Load mutated Default.ContainerColors: got %q", got)
	}

	cfg.ContainerColors["worker"] = "#222222"
	if got, ok := Default.ContainerColors["worker"]; ok {
		t.Fatalf("loaded config aliases Default.ContainerColors after mutation: got %q", got)
	}
}

func TestLoadNormalizesNilContainerColors(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeConfig(t, home, `{"container_colors":null}`)

	cfg := Load()
	if cfg.ContainerColors == nil {
		t.Fatal("Load returned nil ContainerColors")
	}
	cfg.ContainerColors["api"] = "#111111"
}

func writeConfig(t *testing.T, home, contents string) {
	t.Helper()
	path := filepath.Join(home, ".config", "docker-tui", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

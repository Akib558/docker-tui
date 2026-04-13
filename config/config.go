package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ── Theme ────────────────────────────────────────────────────────────────

type Theme struct {
	Name       string
	Primary    string
	Secondary  string
	Success    string
	Danger     string
	Warning    string
	Muted      string
	Text       string
	Subtext    string
	BgAlt      string
	BgSelected string
	Border     string
	Dim        string
	Cyan       string
	TitleFg    string
}

var Themes = []Theme{
	{
		Name: "dark-green", Primary: "#00E676", Secondary: "#69F0AE",
		Success: "#00E676", Danger: "#FF5252", Warning: "#FFD740",
		Muted: "#4E6E5D", Text: "#E8F5E9", Subtext: "#A5D6A7",
		BgAlt: "#0D2818", BgSelected: "#1B5E20", Border: "#2E7D32",
		Dim: "#1B5E20", Cyan: "#00BCD4", TitleFg: "#0A0F0D",
	},
	{
		Name: "dracula", Primary: "#BD93F9", Secondary: "#FF79C6",
		Success: "#50FA7B", Danger: "#FF5555", Warning: "#FFB86C",
		Muted: "#6272A4", Text: "#F8F8F2", Subtext: "#BFBFBF",
		BgAlt: "#21222C", BgSelected: "#44475A", Border: "#44475A",
		Dim: "#44475A", Cyan: "#8BE9FD", TitleFg: "#282A36",
	},
	{
		Name: "nord", Primary: "#88C0D0", Secondary: "#81A1C1",
		Success: "#A3BE8C", Danger: "#BF616A", Warning: "#EBCB8B",
		Muted: "#4C566A", Text: "#ECEFF4", Subtext: "#D8DEE9",
		BgAlt: "#2E3440", BgSelected: "#3B4252", Border: "#4C566A",
		Dim: "#3B4252", Cyan: "#8FBCBB", TitleFg: "#2E3440",
	},
	{
		Name: "gruvbox", Primary: "#FE8019", Secondary: "#FABD2F",
		Success: "#B8BB26", Danger: "#FB4934", Warning: "#FABD2F",
		Muted: "#665C54", Text: "#EBDBB2", Subtext: "#BDAE93",
		BgAlt: "#282828", BgSelected: "#3C3836", Border: "#504945",
		Dim: "#3C3836", Cyan: "#83A598", TitleFg: "#282828",
	},
	{
		Name: "tokyo-night", Primary: "#7AA2F7", Secondary: "#BB9AF7",
		Success: "#9ECE6A", Danger: "#F7768E", Warning: "#E0AF68",
		Muted: "#565F89", Text: "#C0CAF5", Subtext: "#9AA5CE",
		BgAlt: "#1A1B26", BgSelected: "#283457", Border: "#3B4261",
		Dim: "#24283B", Cyan: "#2AC3DE", TitleFg: "#1A1B26",
	},
}

func FindTheme(name string) *Theme {
	for i := range Themes {
		if Themes[i].Name == name {
			return &Themes[i]
		}
	}
	return &Themes[0]
}

func ThemeIndex(name string) int {
	for i, t := range Themes {
		if t.Name == name {
			return i
		}
	}
	return 0
}

// ── Config ───────────────────────────────────────────────────────────────

type Config struct {
	Theme          string  `json:"theme"`
	RefreshSeconds int     `json:"refresh_seconds"`
	AlertCPU       float64 `json:"alert_cpu"`
	AlertMem       float64 `json:"alert_mem"`
}

var Default = Config{
	Theme:          "dark-green",
	RefreshSeconds: 3,
	AlertCPU:       80.0,
	AlertMem:       80.0,
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "docker-tui", "config.json")
}

func Load() *Config {
	cfg := Default
	data, err := os.ReadFile(configPath())
	if err != nil {
		return &cfg
	}
	_ = json.Unmarshal(data, &cfg)
	if cfg.RefreshSeconds < 1 {
		cfg.RefreshSeconds = 1
	}
	return &cfg
}

func Save(cfg *Config) error {
	path := configPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

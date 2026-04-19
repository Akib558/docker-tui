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
	{
		Name: "catppuccin-mocha", Primary: "#CBA6F7", Secondary: "#89B4FA",
		Success: "#A6E3A1", Danger: "#F38BA8", Warning: "#FAB387",
		Muted: "#585B70", Text: "#CDD6F4", Subtext: "#BAC2DE",
		BgAlt: "#181825", BgSelected: "#313244", Border: "#45475A",
		Dim: "#313244", Cyan: "#89DCEB", TitleFg: "#1E1E2E",
	},
	{
		Name: "catppuccin-latte", Primary: "#8839EF", Secondary: "#1E66F5",
		Success: "#40A02B", Danger: "#D20F39", Warning: "#FE640B",
		Muted: "#9CA0B0", Text: "#4C4F69", Subtext: "#5C5F77",
		BgAlt: "#E6E9EF", BgSelected: "#CCD0DA", Border: "#ACB0BE",
		Dim: "#CCD0DA", Cyan: "#04A5E5", TitleFg: "#EFF1F5",
	},
	{
		Name: "rose-pine", Primary: "#C4A7E7", Secondary: "#9CCFD8",
		Success: "#31748F", Danger: "#EB6F92", Warning: "#F6C177",
		Muted: "#403D52", Text: "#E0DEF4", Subtext: "#908CAA",
		BgAlt: "#1F1D2E", BgSelected: "#26233A", Border: "#403D52",
		Dim: "#26233A", Cyan: "#9CCFD8", TitleFg: "#191724",
	},
	{
		Name: "ayu-dark", Primary: "#FFB454", Secondary: "#39BAE6",
		Success: "#AAD94C", Danger: "#FF3333", Warning: "#E6B450",
		Muted: "#3D424D", Text: "#BFBDB6", Subtext: "#626A73",
		BgAlt: "#0D1017", BgSelected: "#1A1F29", Border: "#2D3340",
		Dim: "#1A1F29", Cyan: "#39BAE6", TitleFg: "#0B0E14",
	},
	{
		Name: "monokai", Primary: "#A6E22E", Secondary: "#66D9E8",
		Success: "#A6E22E", Danger: "#F92672", Warning: "#E6DB74",
		Muted: "#49483E", Text: "#F8F8F2", Subtext: "#75715E",
		BgAlt: "#1E1F1C", BgSelected: "#3E3D32", Border: "#49483E",
		Dim: "#3E3D32", Cyan: "#66D9E8", TitleFg: "#272822",
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

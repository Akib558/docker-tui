// Package config loads user preferences and theme definitions for docker-tui.
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
		Muted: "#5A7A69", Text: "#E8F5E9", Subtext: "#A5D6A7",
		BgAlt: "#0D2818", BgSelected: "#1B5E20", Border: "#2E7D32",
		Dim: "#5C8A62", Cyan: "#00BCD4", TitleFg: "#0A0F0D",
	},
	{
		Name: "dracula", Primary: "#BD93F9", Secondary: "#FF79C6",
		Success: "#50FA7B", Danger: "#FF5555", Warning: "#FFB86C",
		Muted: "#6272A4", Text: "#F8F8F2", Subtext: "#BFBFBF",
		BgAlt: "#21222C", BgSelected: "#44475A", Border: "#44475A",
		Dim: "#7D8199", Cyan: "#8BE9FD", TitleFg: "#282A36",
	},
	{
		Name: "nord", Primary: "#88C0D0", Secondary: "#81A1C1",
		Success: "#A3BE8C", Danger: "#BF616A", Warning: "#EBCB8B",
		Muted: "#586276", Text: "#ECEFF4", Subtext: "#D8DEE9",
		BgAlt: "#2E3440", BgSelected: "#3B4252", Border: "#4C566A",
		Dim: "#7B879C", Cyan: "#8FBCBB", TitleFg: "#2E3440",
	},
	{
		Name: "gruvbox", Primary: "#FE8019", Secondary: "#FABD2F",
		Success: "#B8BB26", Danger: "#FB4934", Warning: "#FABD2F",
		Muted: "#665C54", Text: "#EBDBB2", Subtext: "#BDAE93",
		BgAlt: "#282828", BgSelected: "#3C3836", Border: "#504945",
		Dim: "#8A8175", Cyan: "#83A598", TitleFg: "#282828",
	},
	{
		Name: "tokyo-night", Primary: "#7AA2F7", Secondary: "#BB9AF7",
		Success: "#9ECE6A", Danger: "#F7768E", Warning: "#E0AF68",
		Muted: "#565F89", Text: "#C0CAF5", Subtext: "#9AA5CE",
		BgAlt: "#1A1B26", BgSelected: "#283457", Border: "#3B4261",
		Dim: "#6B7394", Cyan: "#2AC3DE", TitleFg: "#1A1B26",
	},
	{
		Name: "catppuccin-mocha", Primary: "#CBA6F7", Secondary: "#89B4FA",
		Success: "#A6E3A1", Danger: "#F38BA8", Warning: "#FAB387",
		Muted: "#585B70", Text: "#CDD6F4", Subtext: "#BAC2DE",
		BgAlt: "#181825", BgSelected: "#313244", Border: "#45475A",
		Dim: "#7F849C", Cyan: "#89DCEB", TitleFg: "#1E1E2E",
	},
	{
		Name: "catppuccin-latte", Primary: "#8839EF", Secondary: "#1E66F5",
		Success: "#40A02B", Danger: "#D20F39", Warning: "#FE640B",
		Muted: "#9CA0B0", Text: "#4C4F69", Subtext: "#5C5F77",
		BgAlt: "#E6E9EF", BgSelected: "#CCD0DA", Border: "#ACB0BE",
		Dim: "#7C7F93", Cyan: "#04A5E5", TitleFg: "#EFF1F5",
	},
	{
		Name: "rose-pine", Primary: "#C4A7E7", Secondary: "#9CCFD8",
		Success: "#31748F", Danger: "#EB6F92", Warning: "#F6C177",
		Muted: "#48455A", Text: "#E0DEF4", Subtext: "#908CAA",
		BgAlt: "#1F1D2E", BgSelected: "#26233A", Border: "#403D52",
		Dim: "#7B7391", Cyan: "#9CCFD8", TitleFg: "#191724",
	},
	{
		Name: "ayu-dark", Primary: "#FFB454", Secondary: "#39BAE6",
		Success: "#AAD94C", Danger: "#FF3333", Warning: "#E6B450",
		Muted: "#3D424D", Text: "#BFBDB6", Subtext: "#626A73",
		BgAlt: "#0D1017", BgSelected: "#1A1F29", Border: "#2D3340",
		Dim: "#6E7681", Cyan: "#39BAE6", TitleFg: "#0B0E14",
	},
	{
		Name: "monokai", Primary: "#A6E22E", Secondary: "#66D9E8",
		Success: "#A6E22E", Danger: "#F92672", Warning: "#E6DB74",
		Muted: "#5D5C52", Text: "#F8F8F2", Subtext: "#75715E",
		BgAlt: "#1E1F1C", BgSelected: "#3E3D32", Border: "#49483E",
		Dim: "#8A8778", Cyan: "#66D9E8", TitleFg: "#272822",
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

type LogHighlightPattern struct {
	Pattern string `json:"pattern"`
	Color   string `json:"color"`
}

type Config struct {
	Theme                string                `json:"theme"`
	RefreshMS            int                   `json:"refresh_ms"`
	RefreshSeconds       int                   `json:"refresh_seconds"` // legacy; kept in sync with RefreshMS
	AlertCPU             float64               `json:"alert_cpu"`
	AlertMem             float64               `json:"alert_mem"`
	ContainerColors      map[string]string     `json:"container_colors"`
	LogHighlightPatterns []LogHighlightPattern `json:"log_highlight_patterns"`
	HintDismissed        bool                  `json:"hint_dismissed"`
}

var Default = Config{
	Theme:           "dark-green",
	RefreshMS:       2000,
	RefreshSeconds:  2,
	AlertCPU:        80.0,
	AlertMem:        80.0,
	ContainerColors: map[string]string{},
	LogHighlightPatterns: []LogHighlightPattern{
		{Pattern: `(?i)\berror\b`, Color: "#FF5252"},
		{Pattern: `(?i)\bfatal\b`, Color: "#FF5252"},
		{Pattern: `(?i)\bcritical\b`, Color: "#FF5252"},
		{Pattern: `(?i)\bwarn\b`, Color: "#FFD740"},
		{Pattern: `(?i)\bwarning\b`, Color: "#FFD740"},
		{Pattern: `(?i)\bpanic\b`, Color: "#FF5252"},
		{Pattern: `(?i)\bexception\b`, Color: "#FF5252"},
		{Pattern: `(?i)\bfailed\b`, Color: "#FF5252"},
		{Pattern: `(?i)\btimeout\b`, Color: "#FFD740"},
		{Pattern: `(?i)\bsuccess\b`, Color: "#00E676"},
		{Pattern: `(?i)\bcompleted\b`, Color: "#00E676"},
	},
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "docker-tui", "config.json")
}

func cloneStringMap(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func Load() *Config {
	cfg := Default
	cfg.ContainerColors = cloneStringMap(Default.ContainerColors)
	cfg.LogHighlightPatterns = make([]LogHighlightPattern, len(Default.LogHighlightPatterns))
	copy(cfg.LogHighlightPatterns, Default.LogHighlightPatterns)
	data, err := os.ReadFile(configPath())
	if err != nil {
		return &cfg
	}
	_ = json.Unmarshal(data, &cfg)
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(data, &raw)
	if _, hasMS := raw["refresh_ms"]; !hasMS && cfg.RefreshSeconds > 0 {
		cfg.RefreshMS = cfg.RefreshSeconds * 1000
	} else if cfg.RefreshMS <= 0 {
		cfg.RefreshMS = Default.RefreshMS
	}
	if cfg.RefreshSeconds < 1 {
		cfg.RefreshSeconds = max(1, cfg.RefreshMS/1000)
	}
	if cfg.ContainerColors == nil {
		cfg.ContainerColors = map[string]string{}
	}
	cfg.ContainerColors = cloneStringMap(cfg.ContainerColors)
	if cfg.LogHighlightPatterns == nil {
		cfg.LogHighlightPatterns = make([]LogHighlightPattern, len(Default.LogHighlightPatterns))
		copy(cfg.LogHighlightPatterns, Default.LogHighlightPatterns)
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

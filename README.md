# docker-tui

A fast, keyboard-driven terminal UI for Docker — manage containers, images, and events without leaving your terminal.

![CI](https://github.com/akib/docker-tui/actions/workflows/ci.yml/badge.svg)
![Go version](https://img.shields.io/badge/go-1.25%2B-blue)
![License](https://img.shields.io/badge/license-MIT-green)

---

## Features

- **Container list** — live CPU/mem bars, state icons, filter, multi-select
- **Detail view** — info, resources (sparklines + progress bars), environment, live logs, embedded shell
- **Images view** — list, remove, pull
- **Events stream** — real-time Docker events with color-coded actions
- **Host stats** — system memory + load average dashboard
- **10 themes** — dark-green, dracula, nord, gruvbox, tokyo-night, catppuccin-mocha, catppuccin-latte, rose-pine, ayu-dark, monokai
- **History** — CPU/mem sparklines persist across restarts
- **Responsive** — adapts columns to terminal width (80 → 220+ cols)
- **Mouse support** — scroll wheel + click to select rows

## Install

### go install
```bash
go install github.com/akib/docker-tui@latest
```

### Build from source
```bash
git clone https://github.com/akib/docker-tui
cd docker-tui
make build          # produces ./docker-tui
make install        # installs to $GOPATH/bin
```

### Binary releases
Download a pre-built binary from the [Releases page](https://github.com/akib/docker-tui/releases).

## Requirements

- Go 1.25+ (build from source)
- Docker daemon running and accessible (socket or `DOCKER_HOST`)
- Terminal with 256-color support (most modern terminals)

## Usage

```bash
docker-tui
```

## Key Bindings

### List view

| Key | Action |
|-----|--------|
| `j` / `k` / `↑` / `↓` | Navigate |
| `enter` / `l` | Open container detail |
| `space` | Toggle multi-select |
| `a` | Select / deselect all |
| `s` | Start / stop container(s) |
| `R` | Restart container(s) |
| `d` | Remove container(s) (confirm) |
| `e` | `docker exec -it` in new terminal |
| `/` | Enter filter mode |
| `C` | Clear filter |
| `c` | Toggle compose grouping |
| `i` | Images view |
| `v` | Events view |
| `t` | Theme picker |
| `+` / `-` | Faster / slower refresh interval |
| `r` | Force refresh |
| `q` | Quit |

### Detail view

| Key | Action |
|-----|--------|
| `tab` / `→` | Next tab |
| `shift+tab` / `←` | Previous tab |
| `j` / `k` | Scroll |
| `l` | Toggle live log stream (Logs tab) |
| `x` | Reconnect embedded shell (Terminal tab) |
| `ctrl+\` | Detach embedded shell |
| `s` | Start / stop |
| `R` | Restart |
| `d` | Remove (confirm) |
| `e` | `docker exec -it` in new terminal |
| `t` | Theme picker |
| `esc` | Back to list |

### Filter mode

| Key | Action |
|-----|--------|
| type | Search by name / image / state |
| `backspace` | Delete character |
| `ctrl+u` | Clear filter |
| `enter` / `esc` | Exit filter mode |

### Images view

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate |
| `p` | Pull image (enter ref) |
| `d` | Remove image (confirm) |
| `r` | Refresh |
| `esc` | Back |

## Config

Config file: `~/.config/docker-tui/config.json`

```json
{
  "theme": "dark-green",
  "refresh_seconds": 3,
  "alert_cpu": 80.0,
  "alert_mem": 80.0
}
```

| Field | Default | Description |
|-------|---------|-------------|
| `theme` | `"dark-green"` | Active theme name |
| `refresh_seconds` | `3` | Container list refresh interval (1–30) |
| `alert_cpu` | `80.0` | CPU % threshold for alert notification |
| `alert_mem` | `80.0` | Memory % threshold for alert notification |

History cache: `~/.cache/docker-tui/history.json`

## Themes

| Name | Style |
|------|-------|
| `dark-green` | Green on dark (default) |
| `dracula` | Purple/pink dark |
| `nord` | Arctic blue dark |
| `gruvbox` | Warm retro dark |
| `tokyo-night` | Blue/purple dark |
| `catppuccin-mocha` | Soft lavender dark |
| `catppuccin-latte` | Soft lavender light |
| `rose-pine` | Muted rose dark |
| `ayu-dark` | Orange/blue dark |
| `monokai` | Classic green/pink dark |

Switch theme at runtime with `t` → `j/k` → `enter`.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE)

# docker-tui

Terminal UI for Docker containers, images, events, live metrics, and an embedded shell tab.

## Run

```bash
go run .
```

## Key controls

### List view
- `j/k`, `up/down`: move selection
- `enter`: open selected container details
- `/`: start filter mode
- `C`: clear filter
- `s`: start/stop selected (or selected set)
- `R`: restart
- `d`: remove
- `e`: open external `docker exec -it`
- `i`: images view
- `v`: events view
- `t`: theme dialog
- `+` / `-`: faster/slower refresh interval
- `q`: quit

### Detail view
- `tab` / `shift+tab`: switch tabs
- `j/k`: scroll tab content
- `l`: toggle live log stream (Logs tab)
- `x`: reconnect embedded terminal (Terminal tab, running container only)
- `ctrl+\`: detach embedded terminal
- `s`, `R`, `d`: container actions
- `esc`: back

### Filter mode
- Type to filter by name/image/state
- `backspace`: delete
- `ctrl+u`: clear filter text
- `enter` / `esc`: leave filter mode

## Config

Config file:

`~/.config/docker-tui/config.json`

Fields:
- `theme`
- `refresh_seconds`
- `alert_cpu`
- `alert_mem`

History cache:

`~/.cache/docker-tui/history.json`

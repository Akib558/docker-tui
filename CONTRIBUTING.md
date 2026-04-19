# Contributing to docker-tui

Thanks for your interest! Contributions are welcome.

## Development setup

```bash
git clone https://github.com/akib/docker-tui
cd docker-tui
make build    # build binary
make test     # run tests
make lint     # run linter (requires golangci-lint)
make run      # run directly
```

## Code structure

```
main.go                 Entry point
config/
  config.go             Theme definitions + JSON config persistence
docker/
  interface.go          ClientAPI interface (mock-friendly)
  client.go             Docker SDK wrapper (containers, stats, exec)
  events.go             Real-time event streaming
  images.go             Image operations
  system.go             Linux /proc host stats
  system_others.go      Stub for non-Linux
  system_types.go       Shared types (SystemMemory, SystemLoad)
ui/
  state.go              Model struct, messages, constructor, Init
  model.go              Update() + per-view key handlers
  commands.go           tea.Cmd functions (Docker operations)
  history.go            CPU/mem history persistence
  alerts.go             CPU/mem alert logic
  helpers.go            Notification, stream lifecycle, cursor helpers
  styles.go             Lipgloss style vars + theme application
  graph.go              Sparklines, progress bars, byte formatter
  views.go              View() router
  view_list.go          Container list view
  view_detail.go        Container detail tabs
  view_images.go        Images view
  view_events.go        Events view
  view_dialog.go        Confirm / theme / input dialogs
  view_help.go          Help bars, notification, key formatter
  view_utils.go         Shared render utilities (truncate, log clean, etc.)
```

## Making changes

- Run `make test` before submitting.
- Keep each PR focused on one concern.
- Add tests for new logic (especially in `docker/` and `ui/helpers_test.go`).
- Follow existing code style — no external formatter config needed beyond `gofmt`.

## Reporting bugs

Open a GitHub issue with:
1. docker-tui version (`docker-tui --version` once implemented)
2. Docker daemon version (`docker version`)
3. OS + terminal emulator
4. Steps to reproduce

# Volumes View — Design Spec

## Overview

Add a dedicated volumes view to docker-tui for listing, inspecting, and managing Docker volumes. Includes filter, multi-select, bulk delete, and orphaned volume cleanup.

## Architecture

New view `view_volumes.go` in `ui/`, following the pattern of `view_images.go` and `view_events.go`. Reached via `v` key from container list. Bubble Tea model drives state; Docker SDK in new `docker/volumes.go` package provides volume data.

```
ui/view_volumes.go   — list UI, filter, selection, key handling
docker/volumes.go   — Docker SDK wrapper (new)
ui/model.go         — volumes state (volumes []Volume, selectedVolumes []string)
ui/commands.go — volume commands (list, inspect, prune, remove)
ui/state.go        — VolumesView added to view enum
```

## Data Model

```go
type Volume struct {
    Name       string
    Driver string
    Mountpoint string
    Labels     map[string]string
    Scope      string
    CreatedAt  time.Time
}
```

## User Flows

**List volumes**
1. User presses `v` from container list
2. `updateVolumes()` fetches all volumes via `docker.ListVolumes()`
3. List renders with name, driver, mountpoint

**Filter**
- `/` enters filter mode, filters by volume name in real-time
- `ctrl+u` clears filter

**Multi-select + delete**
1. `space` toggles selection on focused volume
2. `a` selects all visible volumes
3. `d` shows confirmation dialog if volumes selected
4. Confirmed → `docker.RemoveVolumes()` called in parallel
5. List refreshes, status bar shows success/failure count

**Prune orphaned**
1. User presses `p`
2. Confirmation dialog: "Remove N orphaned volumes?"
3. Confirmed → `docker.PruneVolumes()` → refresh list

## Key Bindings

| Key | Action |
|-----|--------|
| `v` | Open volumes view |
| `j/k`, arrows | Navigate list |
| `space` | Toggle selection |
| `a` | Select all visible |
| `/` | Filter by name |
| `ctrl+u` | Clear filter |
| `d` | Delete selected (confirm first) |
| `p` | Prune orphaned volumes (confirm first) |
| `r` | Refresh |
| `esc` | Return to container list |

## Error Handling

- Docker API errors → status bar message, no crash
- Failed deletions → individual error shown, partial success possible
- Empty state → "No volumes found" message with helpful context

## Testing

- Unit tests for `docker/volumes.go` (mock Docker API)
- Integration test for key flow (list → select → delete)
- Filter behavior tests

## Out of Scope

- Volume creation (out of scope for a TUI focused on monitoring/management)
- Volume content inspection (mount and read files)
- Volume attach/detach to containers

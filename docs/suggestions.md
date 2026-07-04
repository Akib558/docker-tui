# docker-tui: Codebase Analysis & Feature Suggestions

## Current State Summary

docker-tui is a well-structured Bubble Tea TUI with the following feature set:

- **Container Dashboard** — live list with state, CPU/memory mini-bars, sparklines, compose grouping
- **Detail View** — 5 tabs: Info, Resources, Environment, Logs, Terminal
- **Operations** — start/stop/restart/remove containers (single + batch), exec into shell
- **Images** — list, pull, remove
- **Volumes** — list, remove, prune, filter, multi-select
- **Networks** — list, remove, filter, multi-select
- **Events** — real-time Docker event stream
- **Centralized Logs** — multi-container log aggregation with filtering (substring + regex), clipboard copy
- **Embedded Terminal** — in-app shell session per container
- **Alerts** — CPU/memory threshold notifications
- **Themes** — 10 built-in themes with runtime switching
- **Persistence** — config file + sparkline history cache

**Architecture:** Go 1.25, Bubble Tea + Lipgloss, Docker SDK. Clean separation: `config/`, `docker/`, `ui/`. Interface-based Docker client (`ClientAPI`). All tests pass.

---

## A. Missing Docker Operations

### A1. Container Pause / Unpause
- **What:** Add `p` key to pause/unpause a running container.
- **Why:** Docker supports this natively; useful for debugging or temporarily freezing a container without stopping it.
- **Effort:** Small — one Docker SDK call + UI keybinding.

### A2. Container Kill
- **What:** Add `K` key to send SIGKILL (or configurable signal) to a container.
- **Why:** Stop is graceful (SIGTERM + timeout). Sometimes you need immediate kill for hung containers.
- **Effort:** Small.

### A3. Container Rename
- **What:** Add rename via input dialog on the detail view.
- **Why:** Common need when managing containers manually outside compose.
- **Effort:** Small.

### A4. Container Top (Process List)
- **What:** New detail tab or popup showing running processes inside the container (`docker top`).
- **Why:** Essential for debugging — see what's actually running, PID count, CPU per process.
- **Effort:** Medium — new tab rendering + Docker `ContainerTop` API.

### A5. Image Prune (Dangling Images)
- **What:** Add `P` key in images view to remove dangling/unused images.
- **Why:** Images accumulate quickly; pruning reclaims disk space. Currently only individual remove exists.
- **Effort:** Small.

### A6. Image Inspect
- **What:** Detail view for selected image showing layers, size breakdown, labels, env, created history.
- **Why:** Understanding what's inside an image without running `docker inspect` manually.
- **Effort:** Medium.

### A7. Network Inspect
- **What:** Detail view for selected network showing subnet, gateway, IPAM config, connected containers.
- **Why:** Network debugging is painful from the CLI; a visual summary helps.
- **Effort:** Medium.

### A8. Volume Inspect
- **What:** Detail view for selected volume showing options, labels, which containers mount it.
- **Why:** Know what's using a volume before removing it.
- **Effort:** Medium.

### A9. Network Create
- **What:** Input dialog to create a new network with driver selection.
- **Why:** Completes the network CRUD cycle.
- **Effort:** Medium.

### A10. Docker System Prune
- **What:** Global cleanup command (accessible from main list) — removes stopped containers, dangling images, unused networks, and orphaned volumes.
- **Why:** One-key cleanup instead of visiting each view separately.
- **Effort:** Small — single API call + confirmation dialog.

### A11. Container Filesystem Diff (Expose Existing)
- **What:** The `GetContainerDiff` method exists in the Docker client but is never called from the UI. Add a way to view filesystem changes (added/modified/deleted files) in the detail view.
- **Why:** Useful for debugging what a container has changed on its writable layer.
- **Effort:** Small — the backend is already implemented.

---

## B. UX & Navigation Improvements

### B1. Full Help Screen
- **What:** Dedicated help overlay (press `?`) showing all keybindings organized by view, not just the bottom bar.
- **Why:** The bottom help bar is truncated and context-specific. New users need a complete reference.
- **Effort:** Small.

### B2. Container Sorting
- **What:** Sort containers by column: name, state, CPU, memory, image. Toggle with `S` or column header click.
- **Why:** Default sort (running first, then alphabetical) isn't always what you want. Finding the highest CPU consumer should be one keypress.
- **Effort:** Medium.

### B3. Number Key Tab Switching
- **What:** Press `1`-`5` to jump directly to a detail tab (Info, Resources, Environment, Logs, Terminal).
- **Why:** Tab cycling through 5 tabs is slow. Direct access is faster.
- **Effort:** Trivial.

### B4. Log Export to File
- **What:** Press `E` in any log view to save current logs (filtered or full) to a file.
- **Why:** Sharing logs with teammates or saving for post-mortem analysis.
- **Effort:** Small.

### B5. Auto-Reconnect on Docker Disconnect
- **What:** When Docker daemon becomes unreachable, show a reconnecting state and retry instead of staying stuck on the error.
- **Why:** Docker daemon restarts (updates, crashes) shouldn't require restarting docker-tui.
- **Effort:** Medium.

### B6. Container Quick-Jump (Fuzzy Search)
- **What:** Press `g` then type to fuzzy-jump to a container by name, similar to vim's `f` or CtrlP.
- **Why:** With 50+ containers, scrolling is slow even with filtering.
- **Effort:** Medium.

### B7. Status Bar with Connection Health
- **What:** Persistent bottom status bar showing Docker connection status, daemon version, uptime, and last refresh time.
- **Why:** Currently connection status is only visible in the header or on error. A persistent indicator helps during long sessions.
- **Effort:** Small.

### B8. Notification Center / Alert History
- **What:** Replace transient 4-second notifications with a scrollable notification panel (press `N`).
- **Why:** CPU/memory alerts disappear too fast. Users miss important events.
- **Effort:** Medium.

### B9. Compose-Aware Batch Operations
- **What:** When compose grouping is active, add a key to select/operate on all containers in the same compose project.
- **Why:** "Restart all containers in my project" is a very common workflow.
- **Effort:** Medium.

### B10. Image Pull Progress Indicator
- **What:** Show pull progress (layer download status) in the images view instead of blocking silently.
- **Why:** Large image pulls can take minutes with no feedback.
- **Effort:** Medium — requires parsing Docker pull JSON stream.

---

## C. Monitoring & Observability

### C1. Container Health Check Status
- **What:** Show health check status (healthy/unhealthy/none) in the container list and detail view.
- **Why:** Health checks are a core Docker feature; knowing a container is "running" but "unhealthy" is critical.
- **Effort:** Small — data is available in `ContainerInspect`.

### C2. Resource Usage History Graph (Extended)
- **What:** Expand sparkline history from 60 data points to configurable length. Add a dedicated full-screen graph view.
- **Why:** 60 points at 3-second refresh is only 3 minutes of history. Longer trends help diagnose recurring issues.
- **Effort:** Medium.

### C3. Network I/O and Block I/O in List View
- **What:** Add optional columns for network RX/TX and block read/write in the container list.
- **Why:** CPU and memory are shown, but I/O-heavy containers are invisible until you open the detail view.
- **Effort:** Small — data is already collected in stats.

### C4. Disk Usage Overview
- **What:** New view (press `D` from main list) showing disk usage breakdown: images, containers (writable layer), volumes, build cache.
- **Why:** `docker system df` equivalent — know where your disk space is going.
- **Effort:** Medium.

### C5. Container Restart Tracking
- **What:** Show restart count prominently in the list view (badge/icon) and highlight containers with high restart counts.
- **Why:** A container restarting in a loop (crash loop) is one of the most common problems. `RestartCount` is already collected in `InspectContainer` but not shown in the list.
- **Effort:** Small.

### C6. Container Uptime Display
- **What:** Show container uptime (since last start) in the list or detail view.
- **Why:** Knowing a container has been up for 2 seconds vs 30 days is immediately useful.
- **Effort:** Small — `StartedAt` is available in inspect data.

---

## D. Unique / Power-User Features

### D1. Command Palette
- **What:** Press `:` to open a command palette for quick access to all actions (like VS Code's Ctrl+Shift+P).
- **Why:** Power users prefer typing commands over remembering keybindings. Also serves as discoverability for hidden features.
- **Effort:** Medium.

### D2. Container Comparison Mode
- **What:** Select exactly 2 containers and press `C` to see a side-by-side comparison of resources, env vars, labels, network config.
- **Why:** Debugging "why does this one work but that one doesn't" is a daily task.
- **Effort:** Medium.

### D3. Log Pattern Highlighting
- **What:** Configurable regex patterns that highlight log lines (e.g., `ERROR`, `WARN`, `panic`, stack traces) with distinct colors.
- **Why:** Scanning logs visually is much faster when errors are color-coded.
- **Effort:** Medium — extend the existing log viewer.

### D4. Saved Container Groups
- **What:** Save named groups of containers in config (e.g., "backend", "monitoring"). Quick-select a group with a keypress.
- **Why:** Users often work with the same set of containers repeatedly.
- **Effort:** Medium.

### D5. Container Dependency View
- **What:** Visualize container relationships based on shared networks and volume mounts. Show which containers talk to each other.
- **Why:** Understanding service topology at a glance is valuable for debugging connectivity issues.
- **Effort:** Large.

### D6. Multi-Host Docker Support
- **What:** Support connecting to multiple Docker daemons (via DOCKER_HOST or Docker contexts). Switch between hosts from within the TUI.
- **Why:** Users managing staging/production or remote hosts need this.
- **Effort:** Large.

### D7. Container Resource Limits Display
- **What:** Show configured resource limits (CPU quota, memory limit, restart policy) in the detail view.
- **Why:** Knowing the limits helps understand why a container is being OOM-killed or throttled.
- **Effort:** Small — data available in inspect.

---

## E. Code Quality & Architecture Improvements

### E1. Expose Container Filesystem Diff (A11)
- The `GetContainerDiff` / `DiffEntry` types and `getDiff` command exist but are dead code. Wire them into the UI.

### E2. Remove Dead Fields
- `eventsCtx` in `state.go:109` is marked "unused field kept for future use" — remove or use it.

### E3. Add UI Tests
- Current test coverage is limited to `config/`, `docker/` log parsing, and a few `ui/` helpers. Add tests for:
  - Log viewer scroll behavior
  - Filter logic
  - Column calculation at different widths
  - State transitions

### E4. Break Up Model Struct
- The `Model` struct has ~50 fields. Consider grouping into sub-structs:
  - `DetailState` (scroll, tab, log viewer, terminal fields)
  - `CentralLogsState` (central log fields)
  - `DialogState` (dialog, confirm, input fields)
- This improves readability without changing behavior.

### E5. Add `go vet` and Race Detection to Makefile
- The CI runs these but the local Makefile `test` target doesn't include `-race`. Add a `test-race` target.

### E6. Clipboard Dependency
- Clipboard uses `exec.Command("wayland-copy")` and `exec.Command("xclip")` as external processes. Consider using `github.com/atotto/clipboard` for cross-platform support including macOS.

---

## Priority Matrix

| Priority | Items | Rationale |
|----------|-------|-----------|
| **High** | A11 (diff view), B1 (help screen), B3 (tab shortcuts), C1 (health checks), C5 (restart tracking), C6 (uptime), E1 (dead code), E2 (dead fields) | Low effort, high impact, some are already partially implemented |
| **Medium** | A1 (pause), A2 (kill), A4 (process list), A5 (image prune), A10 (system prune), B2 (sorting), B4 (log export), B5 (auto-reconnect), B8 (notification center), B10 (pull progress), C3 (I/O columns), D1 (command palette), D3 (log highlighting), D7 (resource limits) | Moderate effort, strong user value |
| **Lower** | A3 (rename), A6 (image inspect), A7 (network inspect), A8 (volume inspect), A9 (network create), B6 (fuzzy jump), B7 (status bar), B9 (compose batch), C2 (extended graphs), C4 (disk usage), D2 (comparison), D4 (saved groups), D5 (dependency view), D6 (multi-host), E3-E6 (code quality) | Higher effort or niche value |

---

## Recommendation

Start with the **High** priority items — they're quick wins that fix dead code, improve discoverability, and surface critical container health information that's already available but not displayed. Then tackle the **Medium** items based on which workflows you use most.

# Volumes View Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add dedicated volumes view for listing, inspecting, and managing Docker volumes.

**Architecture:** New `docker/volumes.go` wraps Docker SDK volume operations. New `ui/view_volumes.go` provides list UI following images/events pattern. State added to `model.go` and `state.go`. Key handlers in `model.go` update switch and `commands.go`.

**Tech Stack:** Go, Bubble Tea, Docker SDK

---

## File Overview

| File | Role |
|------|------|
| `docker/volumes.go` | New — ListVolumes, RemoveVolume, PruneVolumes |
| `ui/view_volumes.go` | New — volumes list rendering |
| `ui/state.go` | Modify — add `viewVolumes` to viewState enum |
| `ui/model.go` | Modify — add volumes fields, messages, view routing |
| `ui/commands.go` | Modify — add volume commands |
| `ui/view_list.go` | Modify — add `v` key to open volumes |
| `ui/view_images.go` | Read — reference for patterns |

---

## Task 1: Docker Volume Wrapper

**Files:**
- Create: `docker/volumes.go`

- [ ] **Step 1: Write test file**

```go
package docker

import (
	"testing"
	"time"
)

func TestVolumeInfoDisplayName(t *testing.T) {
	v := VolumeInfo{Name: "my-volume", Driver: "local", Scope: "local", CreatedAt: time.Now()}
	if v.DisplayName() != "my-volume" {
		t.Errorf("expected my-volume, got %s", v.DisplayName())
	}
}

func TestVolumeInfoDisplayNameEmpty(t *testing.T) {
	v := VolumeInfo{}
	if v.DisplayName() != "" {
		t.Errorf("expected empty string, got %s", v.DisplayName())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./docker/... -run TestVolumeInfo -v`
Expected: FAIL — function not defined

- [ ] **Step 3: Write minimal types in volumes.go**

```go
package docker

import (
	"context"
	"fmt"
	"time"
)

type VolumeInfo struct {
	Name       string
	Driver     string
	Mountpoint string
	Labels     map[string]string
	Scope      string
	CreatedAt  time.Time
}

func (v VolumeInfo) DisplayName() string {
	return v.Name
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./docker/... -run TestVolumeInfo -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add docker/volumes.go
git commit -m "feat(volumes): add VolumeInfo type and DisplayName method"
```

---

## Task 2: ListVolumes Docker client method

**Files:**
- Modify: `docker/volumes.go:1-20`

- [ ] **Step 1: Add failing test for ListVolumes**

Add to `docker/volumes_test.go`:

```go
func TestClient_ListVolumes(t *testing.T) {
	// This test will skip if Docker not available
	c, err := NewClient()
	if err != nil {
		t.Skip("docker not available")
	}
	vols, err := c.ListVolumes()
	if err != nil {
		t.Fatalf("ListVolumes failed: %v", err)
	}
	// Just verify we get a slice back
	_ = vols
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./docker/... -run TestClient_ListVolumes -v`
Expected: FAIL — method not defined

- [ ] **Step 3: Write ListVolumes method**

Add to `docker/volumes.go`:

```go
func (c *Client) ListVolumes() ([]VolumeInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	list, err := c.cli.VolumeList(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}

	result := make([]VolumeInfo, 0, len(list.Volumes))
	for _, vol := range list.Volumes {
		result = append(result, VolumeInfo{
			Name:       vol.Name,
			Driver:     vol.Driver,
			Mountpoint: vol.Mountpoint,
			Labels:     vol.Labels,
			Scope:      vol.Scope,
			CreatedAt:  vol.CreatedAt,
		})
	}
	return result, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./docker/... -run TestClient_ListVolumes -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add docker/volumes.go docker/volumes_test.go
git commit -m "feat(volumes): add ListVolumes method"
```

---

## Task 3: RemoveVolume and PruneVolumes

**Files:**
- Modify: `docker/volumes.go`

- [ ] **Step 1: Add tests for RemoveVolume and PruneVolumes**

Add to `docker/volumes_test.go`:

```go
func TestClient_RemoveVolume(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Skip("docker not available")
	}
	// Removing non-existent should error
	err = c.RemoveVolume("nonexistent-volume-12345")
	if err == nil {
		t.Error("expected error when removing nonexistent volume")
	}
}

func TestClient_PruneVolumes(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Skip("docker not available")
	}
	// Prune should not error even with nothing to prune
	deleted, err := c.PruneVolumes()
	if err != nil {
		t.Fatalf("PruneVolumes failed: %v", err)
	}
	t.Logf("Pruned %d volumes", len(deleted))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./docker/... -run "TestClient_Remove|TestClient_Prune" -v`
Expected: FAIL — methods not defined

- [ ] **Step 3: Write RemoveVolume and PruneVolumes**

Add to `docker/volumes.go`:

```go
func (c *Client) RemoveVolume(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return c.cli.VolumeRemove(ctx, name, false)
}

func (c *Client) PruneVolumes() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := c.cli.VolumesPrune(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to prune volumes: %w", err)
	}
	return resp.VolumesDeleted, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./docker/... -run "TestClient_Remove|TestClient_Prune" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add docker/volumes.go
git commit -m "feat(volumes): add RemoveVolume and PruneVolumes"
```

---

## Task 4: Add volumes state to model

**Files:**
- Modify: `ui/state.go:16-22`
- Modify: `ui/model.go`

- [ ] **Step 1: Add viewVolumes to viewState enum in state.go**

In `state.go`, change:

```go
const (
	viewList viewState = iota
	viewDetail
	viewImages
	viewEvents
	viewLogs
)
```

To:

```go
const (
	viewList viewState = iota
	viewDetail
	viewImages
	viewEvents
	viewLogs
	viewVolumes
)
```

- [ ] **Step 2: Add volumes fields and message to model.go**

In `ui/model.go`, add after `events []docker.DockerEvent` field:

```go
volumes    []docker.VolumeInfo
volCursor  int
```

Add `volumesMsg` type after `imagesMsg`:

```go
type volumesMsg []docker.VolumeInfo
```

Add `volumeActionDoneMsg` after `imageActionDoneMsg`:

```go
type volumeActionDoneMsg struct{ action, name string }
```

- [ ] **Step 3: Handle volumesMsg in Update switch**

In `model.go` Update method, after `case imagesMsg`:

```go
case volumesMsg:
	m.volumes = []docker.VolumeInfo(msg)
	m.loading = false
	return m, nil
```

After `case imageActionDoneMsg`:

```go
case volumeActionDoneMsg:
	m.notify(fmt.Sprintf("%s: %s", msg.action, msg.name), false)
	return m, m.fetchVolumes()
```

- [ ] **Step 4: Add view routing for viewVolumes in Update**

In `model.go` Update method, after `case viewLogs:` in the view switch:

```go
case viewVolumes:
	return m.updateVolumes(msg)
```

- [ ] **Step 5: Commit**

```bash
git add ui/state.go ui/model.go
git commit -m "feat(volumes): add volumes state to model"
```

---

## Task 5: Add fetchVolumes command

**Files:**
- Modify: `ui/commands.go`

- [ ] **Step 1: Add fetchVolumes command**

Add to `ui/commands.go`, after `fetchImages`:

```go
func (m Model) fetchVolumes() tea.Cmd {
	return func() tea.Msg {
		vols, err := m.client.ListVolumes()
		if err != nil {
			return errMsg{err}
		}
		return volumesMsg(vols)
	}
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add ui/commands.go
git commit -m "feat(volumes): add fetchVolumes command"
```

---

## Task 6: Add remove volume command

**Files:**
- Modify: `ui/commands.go`

- [ ] **Step 1: Add removeVolume command**

Add to `ui/commands.go`, after `pullImage`:

```go
func (m Model) removeVolume(name string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.RemoveVolume(name); err != nil {
			return errMsg{err}
		}
		return volumeActionDoneMsg{"Removed", name}
	}
}

func (m Model) pruneVolumesCmd() tea.Cmd {
	return func() tea.Msg {
		deleted, err := m.client.PruneVolumes()
		if err != nil {
			return errMsg{err}
		}
		return volumeActionDoneMsg{"Pruned", fmt.Sprintf("%d volumes", len(deleted))}
	}
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add ui/commands.go
git commit -m "feat(volumes): add remove and prune volume commands"
```

---

## Task 7: Create view_volumes.go

**Files:**
- Create: `ui/view_volumes.go`

- [ ] **Step 1: Write view_volumes.go**

```go
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewVolumes() string {
	var b strings.Builder
	w := m.width

	b.WriteString(m.renderHeader(w))
	b.WriteString("  " + lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).
		Render(fmt.Sprintf("Volumes  (%d)", len(m.volumes))) + "\n\n")

	if m.loading {
		b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
			Render("  Loading volumes...") + "\n")
		b.WriteString(m.volumesHelp(w))
		return b.String()
	}
	if len(m.volumes) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
			Render("  No volumes found.") + "\n")
		b.WriteString(m.volumesHelp(w))
		return b.String()
	}

	nameW := max(w*40/100, 20)
	driverW := 12
	mountW := max(w*35/100, 20)
	scopeW := 10
	usedW := nameW + driverW + mountW + scopeW + 8
	if usedW > w-4 {
		mountW = max(w-nameW-driverW-scopeW-12, 12)
	}

	hdr := "  " +
		tableHeaderStyle.Width(nameW).Render("NAME") + "  " +
		tableHeaderStyle.Width(driverW).Render("DRIVER") + "  " +
		tableHeaderStyle.Width(mountW).Render("MOUNTPOINT") + "  " +
		tableHeaderStyle.Width(scopeW).Render("SCOPE")
	b.WriteString(listHeaderStyle.Width(w).Render(hdr) + "\n")

	visibleRows := max(3, m.height-9)
	startIdx := 0
	if m.volCursor >= visibleRows {
		startIdx = m.volCursor - visibleRows + 1
	}
	endIdx := min(startIdx+visibleRows, len(m.volumes))

	for i := startIdx; i < endIdx; i++ {
		vol := m.volumes[i]
		row := lipgloss.NewStyle().Width(nameW).Foreground(colorText).Render(truncate(vol.DisplayName(), nameW-1)) + "  " +
			lipgloss.NewStyle().Width(driverW).Foreground(colorDim).Render(truncate(vol.Driver, driverW-1)) + "  " +
			lipgloss.NewStyle().Width(mountW).Foreground(colorSubtext).Render(truncate(vol.Mountpoint, mountW-1)) + "  " +
			lipgloss.NewStyle().Width(scopeW).Foreground(colorMuted).Render(vol.Scope)
		if i == m.volCursor {
			b.WriteString(cursorStyle.Render("▸ ") + listItemSelStyle.Width(w-4).Render(row) + "\n")
		} else {
			b.WriteString("  " + listItemStyle.Width(w-4).Render(row) + "\n")
		}
	}

	if len(m.volumes) > visibleRows {
		pct := float64(m.volCursor) / float64(max(len(m.volumes)-1, 1)) * 100
		b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).
			Render(fmt.Sprintf("  ↕ %d/%d (%.0f%%)", m.volCursor+1, len(m.volumes), pct)) + "\n")
	}

	b.WriteString("\n" + m.volumesHelp(w))
	return b.String()
}

func (m Model) volumesHelp(w int) string {
	keys := []struct{ key, desc string }{
		{"j/k", "navigate"},
		{"space", "select"},
		{"a", "select all"},
		{"d", "remove"},
		{"p", "prune orphaned"},
		{"/", "filter"},
		{"ctrl+u", "clear filter"},
		{"r", "refresh"},
		{"t", "theme"},
		{"esc", "back"},
	}
	return helpBarStyle.Width(w).Render(lipgloss.PlaceHorizontal(w-2, lipgloss.Center, fmtKeys(keys)))
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add ui/view_volumes.go
git commit -m "feat(volumes): add volumes view rendering"
```

---

## Task 8: Add updateVolumes key handler

**Files:**
- Modify: `ui/model.go`

- [ ] **Step 1: Write updateVolumes method**

Add to `ui/model.go`, after `updateEvents`:

```go
func (m Model) updateVolumes(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "h":
		m.view = viewList
	case "up", "k":
		if m.volCursor > 0 {
			m.volCursor--
		}
	case "down", "j":
		if m.volCursor < len(m.volumes)-1 {
			m.volCursor++
		}
	case "home", "g":
		m.volCursor = 0
	case "end", "G":
		if n := len(m.volumes); n > 0 {
			m.volCursor = n - 1
		}
	case " ":
		if m.volCursor < len(m.volumes) {
			name := m.volumes[m.volCursor].Name
			if m.selected[name] {
				delete(m.selected, name)
			} else {
				m.selected[name] = true
			}
		}
	case "a":
		if len(m.selected) > 0 {
			m.selected = make(map[string]bool)
		} else {
			for _, vol := range m.volumes {
				m.selected[vol.Name] = true
			}
		}
	case "d":
		return m.confirmRemoveVolumes()
	case "p":
		return m.confirmPruneVolumes()
	case "/":
		m.filtering = true
		m.filterText = ""
	case "ctrl+u":
		m.filterText = ""
		m.volCursor = 0
	case "r":
		m.loading = true
		return m, m.fetchVolumes()
	case "t":
		m.dialog = dialogTheme
	}
	return m, nil
}

func (m Model) confirmRemoveVolumes() (tea.Model, tea.Cmd) {
	if len(m.selected) == 0 && m.volCursor < len(m.volumes) {
		m.selected[m.volumes[m.volCursor].Name] = true
	}
	if len(m.selected) == 0 {
		return m, nil
	}
	names := make([]string, 0, len(m.selected))
	for name := range m.selected {
		names = append(names, name)
	}
	msg := fmt.Sprintf("Remove %d volume(s)?\n\n  %s\n\nThis cannot be undone.", len(names), strings.Join(names, ", "))
	m.dialog = dialogConfirm
	m.confirmMsg = msg
	m.confirmOK = func() tea.Msg {
		var cmds []tea.Cmd
		for name := range m.selected {
			cmds = append(cmds, m.removeVolume(name))
		}
		m.selected = make(map[string]bool)
		return tea.Batch(cmds...)
	}
	return m, nil
}

func (m Model) confirmPruneVolumes() (tea.Model, tea.Cmd) {
	m.dialog = dialogConfirm
	m.confirmMsg = "Remove all orphaned volumes?\n\nThis cannot be undone."
	m.confirmOK = m.pruneVolumesCmd
	return m, nil
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add ui/model.go
git commit -m "feat(volumes): add volumes key handlers and dialogs"
```

---

## Task 9: Wire `v` key to open volumes from list

**Files:**
- Modify: `ui/model.go` (updateList function)

- [ ] **Step 1: Change `v` key in updateList**

In `model.go` `updateList` function, find:

```go
case "v":
	m.view = viewEvents
	if m.eventsCancel == nil {
		return m, m.startEventStream()
	}
```

Change to:

```go
case "v":
	m.view = viewVolumes
	m.volCursor = 0
	m.loading = true
	return m, m.fetchVolumes()
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add ui/model.go
git commit -m "feat(volumes): wire v key to open volumes view"
```

---

## Task 10: Add volumes filter support

**Files:**
- Modify: `ui/model.go`

- [ ] **Step 1: Add filteredVolumes method**

Add to `ui/model.go`, after `filteredContainers`:

```go
func (m Model) filteredVolumes() []docker.VolumeInfo {
	if m.filterText == "" {
		return m.volumes
	}
	q := strings.ToLower(m.filterText)
	var out []docker.VolumeInfo
	for _, vol := range m.volumes {
		if strings.Contains(strings.ToLower(vol.Name), q) ||
			strings.Contains(strings.ToLower(vol.Driver), q) {
			out = append(out, vol)
		}
	}
	return out
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add ui/model.go
git commit -m "feat(volumes): add volumes filtering"
```

---

## Task 11: Run full test suite

- [ ] **Step 1: Run tests**

Run: `go test ./... -v`
Expected: all tests pass

- [ ] **Step 2: Run linter**

Run: `golangci-lint run`
Expected: no errors

- [ ] **Step 3: Build binary**

Run: `make build && ./docker-tui --help`
Expected: builds and runs

---

## Task 12: Integration test — run app and verify

- [ ] **Step 1: Start app and navigate**

```bash
./docker-tui
# Press v to open volumes
# Verify list renders
# Press esc to return
# Press q to quit
```

Expected: volumes view opens, shows list, returns to list on esc

- [ ] **Step 2: Commit all work**

```bash
git add -A && git status
# Review changes, then commit
git commit -m "feat: add volumes management view

- List all Docker volumes with driver, mountpoint, scope
- Filter volumes by name or driver
- Multi-select and bulk delete with confirmation
- Prune orphaned volumes with confirmation
- Follows images/events pattern for consistent UX"
```

---

## Self-Review Checklist

- [ ] Spec coverage: all requirements have corresponding tasks
- [ ] No placeholders (TBD, TODO, "implement later")
- [ ] Type consistency: VolumeInfo, DisplayName, ListVolumes, RemoveVolume, PruneVolumes all match across tasks
- [ ] All tests pass
- [ ] Build succeeds
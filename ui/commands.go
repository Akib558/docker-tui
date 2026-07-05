package ui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/akib558/docker-tui/config"
	"github.com/akib558/docker-tui/docker"
	tea "github.com/charmbracelet/bubbletea"
)

// ── Docker commands ─────────────────────────────────────────────────────

func (m Model) refreshContainers() tea.Cmd {
	return func() tea.Msg {
		containers, err := m.client.ListContainers()
		if err != nil {
			return errMsg{err}
		}
		return containersMsg(containers)
	}
}

func (m Model) fetchImages() tea.Cmd {
	return func() tea.Msg {
		imgs, err := m.client.ListImages()
		if err != nil {
			return errMsg{err}
		}
		return imagesMsg(imgs)
	}
}

func (m Model) collectStats() tea.Cmd {
	return func() tea.Msg {
		var ids []string
		for _, c := range m.containers {
			if c.State == "running" {
				ids = append(ids, c.ID)
			}
		}
		return statsMsg{
			stats:   m.client.GetAllContainerStats(ids),
			sysMem:  docker.GetSystemMemory(),
			sysLoad: docker.GetSystemLoad(),
		}
	}
}

func (m Model) inspectContainer(id string) tea.Cmd {
	return func() tea.Msg {
		info, err := m.client.InspectContainer(id)
		if err != nil {
			return errMsg{err}
		}
		return inspectMsg{info}
	}
}

func (m Model) fetchLogs(id string) tea.Cmd {
	return func() tea.Msg {
		records, err := m.client.GetContainerLogRecords(id, 100)
		if err != nil {
			return logsMsg{{Message: "(unable to fetch logs)", System: true}}
		}
		entries := make([]LogEntry, 0, len(records))
		name := ""
		if m.inspected != nil {
			name = m.inspected.Name
		}
		for _, record := range records {
			entries = append(entries, LogEntry{
				ContainerID:   id,
				ContainerName: name,
				Timestamp:     record.Timestamp,
				Message:       record.Message,
			})
		}
		SortLogEntries(entries)
		return logsMsg(entries)
	}
}

func (m Model) streamLogs(id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		ch := make(chan LogEntry, 200)
		name := ""
		if m.inspected != nil {
			name = m.inspected.Name
		}

		go func() {
			defer close(ch)
			reader, err := m.client.GetContainerLogRecordsStream(ctx, id, 0)
			if err != nil {
				cancel()
				return
			}
			defer reader.Close()
			scanner := bufio.NewScanner(reader)
			scanner.Buffer(make([]byte, 64*1024), 1024*1024)
			for scanner.Scan() {
				record := docker.ParseDockerLogRecord(scanner.Text())
				entry := LogEntry{
					ContainerID:   id,
					ContainerName: name,
					Timestamp:     record.Timestamp,
					Message:       record.Message,
				}
				select {
				case ch <- entry:
				case <-ctx.Done():
					return
				}
			}
			if err := scanner.Err(); err != nil {
				select {
				case ch <- LogEntry{
					ContainerID:   id,
					ContainerName: name,
					Message:       fmt.Sprintf("(log stream error: %v)", err),
					System:        true,
				}:
				case <-ctx.Done():
				}
			}
		}()

		var readNext tea.Cmd
		readNext = func() tea.Msg {
			line, ok := <-ch
			if !ok {
				return logStreamDoneMsg{}
			}
			return logLineMsg{entry: line, next: readNext}
		}
		return logStreamStartMsg{cancel: cancel, next: readNext}
	}
}

func (m Model) startTerminal(id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		started := false
		var attach io.ReadWriteCloser
		var shell string

		for _, sh := range []string{"/bin/bash", "/bin/sh"} {
			resp, err := m.client.StartContainerExecShell(ctx, id, sh)
			if err == nil {
				attach = resp
				shell = sh
				started = true
				break
			}
		}
		if !started {
			cancel()
			return terminalDoneMsg{err: fmt.Errorf("unable to start shell (/bin/bash or /bin/sh)")}
		}

		ch := make(chan string, 256)
		go func() {
			defer close(ch)
			defer attach.Close()
			buf := make([]byte, 4096)
			for {
				n, err := attach.Read(buf)
				if n > 0 {
					ch <- string(buf[:n])
				}
				if err != nil {
					return
				}
			}
		}()

		var readNext tea.Cmd
		readNext = func() tea.Msg {
			select {
			case <-ctx.Done():
				return terminalDoneMsg{}
			case chunk, ok := <-ch:
				if !ok {
					return terminalDoneMsg{}
				}
				return terminalChunkMsg{chunk: chunk, next: readNext}
			}
		}

		stop := func() {
			cancel()
			_ = attach.Close()
		}
		return terminalStartMsg{cancel: stop, writer: attach, shell: shell, next: readNext}
	}
}

func (m Model) sendTerminalInput(text string) tea.Cmd {
	return func() tea.Msg {
		if m.terminalWriter == nil {
			return terminalDoneMsg{err: fmt.Errorf("terminal not connected")}
		}
		if _, err := io.WriteString(m.terminalWriter, text); err != nil {
			return terminalDoneMsg{err: err}
		}
		return nil
	}
}

func (m Model) startEventStream() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		ch := m.client.StreamEvents(ctx)

		var readNext tea.Cmd
		readNext = func() tea.Msg {
			ev, ok := <-ch
			if !ok {
				return nil
			}
			return newEventMsg{ev: ev, next: readNext}
		}
		return eventStreamStartMsg{cancel: cancel, next: readNext}
	}
}

func (m Model) getDiff(id string) tea.Cmd {
	return func() tea.Msg {
		diff, err := m.client.GetContainerDiff(id)
		if err != nil {
			return diffMsg{}
		}
		return diffMsg(diff)
	}
}

func (m Model) getTop(id string) tea.Cmd {
	return func() tea.Msg {
		top, err := m.client.GetContainerTop(id)
		if err != nil {
			return errMsg{err}
		}
		return topMsg{top: top}
	}
}

func (m Model) startContainer(id, name string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.StartContainer(id); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{"Started", name}
	}
}

func (m Model) stopContainer(id, name string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.StopContainer(id); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{"Stopped", name}
	}
}

func (m Model) restartContainer(id, name string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.RestartContainer(id); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{"Restarted", name}
	}
}

func (m Model) pauseContainer(id, name string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.PauseContainer(id); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{"Paused", name}
	}
}

func (m Model) unpauseContainer(id, name string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.UnpauseContainer(id); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{"Unpaused", name}
	}
}

func (m Model) killContainer(id, name string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.KillContainer(id, "SIGKILL"); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{"Killed", name}
	}
}

func (m Model) pullImage(ref string) tea.Cmd {
	return func() tea.Msg {
		ch := make(chan string, 128)
		errCh := make(chan error, 1)

		go func() {
			defer close(ch)
			errCh <- m.client.PullImageWithProgress(ref, func(progress string) {
				select {
				case ch <- progress:
				default:
				}
			})
		}()

		var readNext tea.Cmd
		readNext = func() tea.Msg {
			progress, ok := <-ch
			if ok {
				return pullProgressMsg{text: progress, next: readNext}
			}
			if err := <-errCh; err != nil {
				return errMsg{err}
			}
			return imageActionDoneMsg{"Pulled", ref}
		}

		return pullProgressMsg{text: "Starting pull...", next: readNext}
	}
}

func (m Model) pruneDanglingImages() tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.PruneDanglingImages()
		if err != nil {
			return errMsg{err}
		}
		reclaimed := formatBytes(result.SpaceReclaimed)
		return imageActionDoneMsg{"Pruned images", fmt.Sprintf("%d deleted (%s)", len(result.DeletedRefs), reclaimed)}
	}
}

func (m Model) pruneSystem() tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.SystemPrune()
		if err != nil {
			return errMsg{err}
		}
		summary := fmt.Sprintf("containers:%d images:%d networks:%d volumes:%d reclaimed:%s",
			result.ContainersDeleted,
			result.ImagesDeleted,
			result.NetworksDeleted,
			result.VolumesDeleted,
			formatBytes(result.SpaceReclaimed),
		)
		return actionDoneMsg{"System prune complete", summary}
	}
}

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

func (m Model) execIntoContainerCmd(id string) tea.Cmd {
	return tea.ExecProcess(exec.Command("docker", "exec", "-it", id, "/bin/sh"), func(err error) tea.Msg {
		return execDoneMsg{err}
	})
}

func (m Model) quit() (tea.Model, tea.Cmd) {
	m.stopLogStreaming()
	m.stopCentralLogStreaming()
	m.stopTerminalSession()
	if m.eventsCancel != nil {
		m.eventsCancel()
	}
	if m.client != nil {
		m.client.Close()
	}
	go config.Save(m.cfg)
	return m, tea.Quit
}

func (m Model) reconnect() tea.Cmd {
	return tea.Tick(time.Duration(m.reconnectAttempts)*2*time.Second, func(t time.Time) tea.Msg {
		client, err := docker.NewClient()
		if err != nil {
			return reconnectMsg{success: false, err: err}
		}
		_, err = client.ListContainers()
		if err != nil {
			client.Close()
			return reconnectMsg{success: false, err: err}
		}
		return reconnectMsg{success: true, client: client}
	})
}

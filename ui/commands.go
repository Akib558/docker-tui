package ui

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/akib/docker-tui/config"
	"github.com/akib/docker-tui/docker"
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
		logs, err := m.client.GetContainerLogs(id, 100)
		if err != nil {
			return logsMsg("(unable to fetch logs)")
		}
		return logsMsg(logs)
	}
}

func (m Model) streamLogs(id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		ch := make(chan string, 200)

		go func() {
			defer close(ch)
			reader, err := m.client.GetContainerLogsStream(ctx, id)
			if err != nil {
				cancel()
				return
			}
			defer reader.Close()
			buf := make([]byte, 4096)
			var partial string
			for {
				n, err := reader.Read(buf)
				if n > 0 {
					data := partial + string(buf[:n])
					lines := strings.Split(data, "\n")
					for i, line := range lines {
						if i == len(lines)-1 {
							partial = line
						} else {
							// Strip Docker log header (8 bytes)
							if len(line) > 8 && (line[0] == 1 || line[0] == 2) {
								line = line[8:]
							}
							select {
							case ch <- line:
							case <-ctx.Done():
								return
							}
						}
					}
				}
				if err != nil {
					break
				}
			}
		}()

		var readNext tea.Cmd
		readNext = func() tea.Msg {
			line, ok := <-ch
			if !ok {
				return logStreamDoneMsg{}
			}
			return logLineMsg{line: line, next: readNext}
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

func (m Model) pullImage(ref string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.PullImage(ref); err != nil {
			return errMsg{err}
		}
		return imageActionDoneMsg{"Pulled", ref}
	}
}

func (m Model) execIntoContainerCmd(id string) tea.Cmd {
	return tea.ExecProcess(exec.Command("docker", "exec", "-it", id, "/bin/sh"), func(err error) tea.Msg {
		return execDoneMsg{err}
	})
}

func (m Model) quit() (tea.Model, tea.Cmd) {
	m.stopLogStreaming()
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

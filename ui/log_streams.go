package ui

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/akib558/docker-tui/docker"
	tea "github.com/charmbracelet/bubbletea"
)

const centralLogTailConcurrency = 8

func (m Model) fetchCentralLogTails(targets []LogTarget) tea.Cmd {
	return func() tea.Msg {
		if len(targets) == 0 {
			return centralLogTailsBatchMsg{}
		}
		sem := make(chan struct{}, centralLogTailConcurrency)
		var mu sync.Mutex
		var all []LogEntry
		var wg sync.WaitGroup
		for _, target := range targets {
			wg.Add(1)
			go func(target LogTarget) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				records, err := m.client.GetContainerLogRecords(target.ID, 100)
				var entries []LogEntry
				if err != nil {
					entries = []LogEntry{systemLogEntry(target, fmt.Sprintf("unable to fetch logs: %v", err))}
				} else {
					entries = make([]LogEntry, 0, len(records))
					for _, record := range records {
						entries = append(entries, LogEntry{
							ContainerID:   target.ID,
							ContainerName: target.Name,
							Timestamp:     record.Timestamp,
							Message:       record.Message,
						})
					}
				}
				mu.Lock()
				all = append(all, entries...)
				mu.Unlock()
			}(target)
		}
		wg.Wait()
		SortLogEntries(all)
		return centralLogTailsBatchMsg{entries: all}
	}
}

func (m Model) startCentralLogStreams(targets []LogTarget) tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(targets))
	for _, target := range targets {
		target := target
		cmds = append(cmds, m.startCentralLogStream(target))
	}
	return tea.Batch(cmds...)
}

func (m Model) startCentralLogStream(target LogTarget) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		ch := make(chan LogEntry, 200)
		errCh := make(chan error, 1)

		go func() {
			var streamErr error
			defer func() {
				errCh <- streamErr
				close(ch)
			}()

			reader, err := m.client.GetContainerLogRecordsStream(ctx, target.ID, 0)
			if err != nil {
				streamErr = err
				return
			}
			defer reader.Close()

			streamErr = forwardDockerStreamEntries(ctx, reader, func(record docker.LogRecord) bool {
				entry := LogEntry{
					ContainerID:   target.ID,
					ContainerName: target.Name,
					Timestamp:     record.Timestamp,
					Message:       record.Message,
				}
				select {
				case ch <- entry:
					return true
				case <-ctx.Done():
					return false
				}
			})
		}()

		var readNext tea.Cmd
		readNext = func() tea.Msg {
			batch, more := drainLogChannel(ch, ctx)
			if len(batch) == 0 && !more {
				var err error
				select {
				case err = <-errCh:
				default:
				}
				return centralLogStreamDoneMsg{target: target, err: err}
			}
			var next tea.Cmd
			if more {
				next = readNext
			}
			if len(batch) == 0 {
				if next != nil {
					return next()
				}
				return centralLogStreamDoneMsg{target: target}
			}
			return centralLogBatchMsg{entries: batch, next: next}
		}

		return centralLogStreamStartMsg{cancel: cancel, next: readNext}
	}
}

func forwardDockerStreamEntries(ctx context.Context, reader io.Reader, onRecord func(docker.LogRecord) bool) error {
	buffered := bufio.NewReaderSize(reader, 64*1024)
	for {
		line, err := buffered.ReadString('\n')
		if len(line) > 0 {
			line = strings.TrimSuffix(line, "\n")
			line = strings.TrimSuffix(line, "\r")
			if !onRecord(docker.ParseDockerLogRecord(line)) {
				return nil
			}
		}
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

func systemLogEntry(target LogTarget, message string) LogEntry {
	return LogEntry{
		ContainerID:   target.ID,
		ContainerName: target.Name,
		Message:       message,
		System:        true,
	}
}

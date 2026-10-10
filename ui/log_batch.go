package ui

import (
	"context"
	"time"

	"github.com/akib558/docker-tui/docker"
)

const (
	logBatchMaxEntries = 256
	logBatchMaxWait    = 16 * time.Millisecond
)

// drainLogChannel reads up to logBatchMaxEntries or until logBatchMaxWait elapses.
func drainLogChannel(ch <-chan LogEntry, ctx context.Context) ([]LogEntry, bool) {
	var batch []LogEntry
	deadline := time.Now().Add(logBatchMaxWait)
	for len(batch) < logBatchMaxEntries {
		select {
		case entry, ok := <-ch:
			if !ok {
				return batch, false
			}
			batch = append(batch, entry)
		default:
			if len(batch) > 0 || time.Now().After(deadline) {
				return batch, true
			}
			timer := time.NewTimer(time.Until(deadline))
			select {
			case entry, ok := <-ch:
				timer.Stop()
				if !ok {
					return batch, false
				}
				batch = append(batch, entry)
			case <-timer.C:
				return batch, true
			case <-ctx.Done():
				timer.Stop()
				return batch, true
			}
		}
	}
	return batch, true
}

func drainEventChannel(ch <-chan docker.DockerEvent, ctx context.Context) ([]docker.DockerEvent, bool) {
	var batch []docker.DockerEvent
	deadline := time.Now().Add(logBatchMaxWait)
	for len(batch) < logBatchMaxEntries {
		select {
		case ev, ok := <-ch:
			if !ok {
				return batch, false
			}
			batch = append(batch, ev)
		default:
			if len(batch) > 0 || time.Now().After(deadline) {
				return batch, true
			}
			timer := time.NewTimer(time.Until(deadline))
			select {
			case ev, ok := <-ch:
				timer.Stop()
				if !ok {
					return batch, false
				}
				batch = append(batch, ev)
			case <-timer.C:
				return batch, true
			case <-ctx.Done():
				timer.Stop()
				return batch, true
			}
		}
	}
	return batch, true
}

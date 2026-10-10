package docker

import (
	"context"
	"sync"
	"time"
)

const (
	statsMaxConcurrency = 16
	statsBatchInterval  = 250 * time.Millisecond
	statsCallTimeout    = 5 * time.Second
)

// StatsBatchInterval is how often partial stats maps are delivered.
func StatsBatchInterval() time.Duration { return statsBatchInterval }

// StatsBatch is one partial result from CollectContainerStats.
type StatsBatch struct {
	Stats map[string]*ContainerResourceStats
	Done  bool
}

type statsFetcher interface {
	GetContainerStats(id string) (*ContainerResourceStats, error)
}

// CollectContainerStats fetches stats for ids with bounded concurrency and
// delivers partial maps on batchEvery until all requests finish.
func CollectContainerStats(client statsFetcher, ids []string, batchEvery time.Duration, deliver func(StatsBatch)) {
	if len(ids) == 0 {
		deliver(StatsBatch{Stats: map[string]*ContainerResourceStats{}, Done: true})
		return
	}
	if batchEvery <= 0 {
		batchEvery = statsBatchInterval
	}
	concurrency := statsMaxConcurrency
	if concurrency > len(ids) {
		concurrency = len(ids)
	}

	pending := make(map[string]*ContainerResourceStats)
	var pendingMu sync.Mutex
	flush := func(done bool) {
		pendingMu.Lock()
		if len(pending) == 0 && !done {
			pendingMu.Unlock()
			return
		}
		out := pending
		pending = make(map[string]*ContainerResourceStats)
		pendingMu.Unlock()
		deliver(StatsBatch{Stats: out, Done: done})
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			ctx, cancel := context.WithTimeout(context.Background(), statsCallTimeout)
			defer cancel()
			done := make(chan struct {
				stats *ContainerResourceStats
				err   error
			}, 1)
			go func() {
				s, err := client.GetContainerStats(id)
				done <- struct {
					stats *ContainerResourceStats
					err   error
				}{s, err}
			}()
			select {
			case <-ctx.Done():
				return
			case res := <-done:
				if res.err != nil || res.stats == nil {
					return
				}
				pendingMu.Lock()
				pending[id] = res.stats
				pendingMu.Unlock()
			}
		}(id)
	}

	ticker := time.NewTicker(batchEvery)
	defer ticker.Stop()
	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	for {
		select {
		case <-ticker.C:
			flush(false)
		case <-doneCh:
			flush(true)
			return
		}
	}
}

package docker

import (
	"fmt"
	"testing"
	"time"
)

type slowStatsClient struct {
	FakeClient
	delay time.Duration
}

func (s *slowStatsClient) GetContainerStats(id string) (*ContainerResourceStats, error) {
	time.Sleep(s.delay)
	return &ContainerResourceStats{CPUPercent: 1.5, MemPercent: 2.0}, nil
}

func BenchmarkGetAllContainerStats50(b *testing.B) {
	ids := make([]string, 50)
	for i := range ids {
		ids[i] = fmt.Sprintf("id%02d", i)
	}
	client := &slowStatsClient{delay: 200 * time.Millisecond}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		CollectContainerStats(client, ids, statsBatchInterval, func(batch StatsBatch) {
			_ = batch
		})
		if elapsed := time.Since(start); elapsed > 2*time.Second {
			b.Fatalf("collect took %v, want under ~1s wall with concurrency 16", elapsed)
		}
	}
}

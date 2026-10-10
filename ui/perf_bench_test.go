package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/akib558/docker-tui/docker"
)

func benchLogViewer(n int, max int) *LogViewerState {
	v := NewLogViewerState(max, []LogTarget{{ID: "c1", Name: "api"}})
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		v.Append(LogEntry{
			ContainerID: "c1", ContainerName: "api",
			Timestamp: base.Add(time.Duration(i) * time.Millisecond),
			Message:   fmt.Sprintf("line %d payload", i),
			Sequence:  int64(i + 1),
		})
	}
	v.FilteredEntries() // warm cache
	return &v
}

func benchModelCentralLogs(n int) Model {
	m := benchModelBase()
	m.view = viewLogs
	m.centralLogs = *benchLogViewer(n, centralLogBufferMax)
	m.centralLogTargets = []LogTarget{{ID: "c1", Name: "api"}}
	m.width = 120
	m.height = 40
	return m
}

func benchModelBase() Model {
	m := NewModel(fixtureConfig())
	m.client = &docker.FakeClient{}
	m.containers = makeContainersN(20)
	m.loading = false
	m.stats = fixtureStats()
	m.rebuildFilteredCache()
	m.width = 120
	m.height = 40
	m.dashboardCache = m.renderDashboard(m.width)
	m.dashboardCacheW = m.width
	return m
}

func makeContainersN(n int) []docker.ContainerInfo {
	out := make([]docker.ContainerInfo, n)
	for i := 0; i < n; i++ {
		out[i] = docker.ContainerInfo{
			ID:    fmt.Sprintf("id%04d", i),
			Name:  fmt.Sprintf("svc-%04d", i),
			Image: "img:latest",
			State: "running",
		}
	}
	return out
}

func BenchmarkAppendLogLine5k(b *testing.B) {
	v := benchLogViewer(5000, detailLogBufferMax)
	entry := LogEntry{ContainerID: "c1", ContainerName: "api", Message: "new", Sequence: 99999}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry.Sequence = int64(100000 + i)
		v.Append(entry)
	}
}

func BenchmarkAppendLogLine50k(b *testing.B) {
	v := NewLogViewerState(50000, nil)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 50000; i++ {
		v.Append(LogEntry{
			ContainerID: "c1", Message: fmt.Sprintf("line %d", i),
			Timestamp: base.Add(time.Duration(i) * time.Millisecond),
			Sequence:  int64(i + 1),
		})
	}
	entry := LogEntry{ContainerID: "c1", Message: "new", Sequence: 99999}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry.Sequence = int64(100000 + i)
		v.Append(entry)
	}
}

func BenchmarkUpdateLogLineMsg(b *testing.B) {
	m := benchModelBase()
	m.view = viewDetail
	m.detailTab = tabLogs
	m.liveLogging = true
	m.logViewer = *benchLogViewer(5000, detailLogBufferMax)
	msg := logBatchMsg{
		entries: []LogEntry{{ContainerID: "c1", Message: "batch line", Sequence: 1}},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg.entries[0].Sequence = int64(i + 1)
		m.Update(msg)
	}
}

func BenchmarkViewCentralLogs50k(b *testing.B) {
	m := benchModelCentralLogs(50000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkViewCentralLogs50kFilter(b *testing.B) {
	m := benchModelCentralLogs(50000)
	m.centralLogs.SetFilter("payload")
	m.centralLogs.FilteredEntries()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkViewCentralLogs50kRegex(b *testing.B) {
	m := benchModelCentralLogs(50000)
	m.centralLogs.SetFilter("line [0-9]+")
	m.centralLogs.SetUseRegex(true)
	m.centralLogs.FilteredEntries()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkViewCentralLogs50kHidden(b *testing.B) {
	m := benchModelCentralLogs(50000)
	m.centralLogs.HiddenContainers = map[string]bool{"c1": true, "c2": true, "c3": true}
	m.centralLogs.invalidateVisible()
	m.centralLogs.FilteredEntries()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkViewDetailLogs50k(b *testing.B) {
	m := benchModelBase()
	m.view = viewDetail
	m.detailTab = tabLogs
	m.inspected = &m.containers[0]
	m.logViewer = *benchLogViewer(50000, detailLogBufferMax)
	m.width = 120
	m.height = 40
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkViewList20(b *testing.B) {
	m := benchModelBase()
	m.width = 120
	m.height = 40
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkViewList200(b *testing.B) {
	m := benchModelBase()
	m.containers = makeContainersN(200)
	m.rebuildFilteredCache()
	m.dashboardCache = m.renderDashboard(m.width)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func makeContainersUnicodeN(n int) []docker.ContainerInfo {
	out := makeContainersN(n)
	for i := range out {
		switch i % 3 {
		case 0:
			out[i].Name = "svc-🐳-" + out[i].Name
		case 1:
			out[i].Name = "日本-" + out[i].Name
		}
	}
	return out
}

func BenchmarkViewList20Unicode(b *testing.B) {
	m := benchModelBase()
	m.containers = makeContainersUnicodeN(20)
	m.rebuildFilteredCache()
	m.dashboardCache = m.renderDashboard(m.width)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkViewList1000Unicode(b *testing.B) {
	m := benchModelBase()
	m.containers = makeContainersUnicodeN(1000)
	m.rebuildFilteredCache()
	m.dashboardCache = m.renderDashboard(m.width)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkViewList1000(b *testing.B) {
	m := benchModelBase()
	m.containers = makeContainersN(1000)
	m.rebuildFilteredCache()
	m.dashboardCache = m.renderDashboard(m.width)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkFilteredContainers(b *testing.B) {
	m := benchModelBase()
	m.containers = makeContainersN(1000)
	m.filterText = "svc-01"
	m.rebuildFilteredCache()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.filteredContainers()
	}
}

func BenchmarkMoveFocus50k(b *testing.B) {
	v := benchLogViewer(50000, 50000)
	height := 20
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.MoveFocus(1, height)
	}
}

func BenchmarkLogModalOpen5k(b *testing.B) {
	m := benchModelCentralLogs(5000)
	for i := 0; i < 5000; i++ {
		m.centralLogs.Selection[int64(i+1)] = true
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copyM := m
		copyM.openLogDetailModal(&copyM.centralLogs, true)
	}
}

func BenchmarkCentralLogBatchUpdate256(b *testing.B) {
	m := benchModelCentralLogs(5000)
	entries := make([]LogEntry, 256)
	for i := range entries {
		entries[i] = LogEntry{ContainerID: "c1", Message: strings.Repeat("x", 40), Sequence: int64(i + 1)}
	}
	msg := centralLogBatchMsg{entries: entries}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Update(msg)
	}
}

func BenchmarkCentralLogFilterKeystroke50k(b *testing.B) {
	v := benchLogViewer(50000, 50000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.SetFilter(fmt.Sprintf("payload%d", i%10))
		v.FilteredEntries()
	}
}

package ui

import (
	"hash/fnv"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/akib558/docker-tui/docker"
)

const (
	centralLogBufferMax = 2000
	detailLogBufferMax  = 1000
)

var (
	hexColorRE = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
	logPalette = []string{
		"#00E676", "#7AA2F7", "#FFB454", "#BD93F9", "#50FA7B",
		"#FF79C6", "#8BE9FD", "#FABD2F", "#F7768E", "#A3BE8C",
		"#39BAE6", "#C4A7E7", "#9CCFD8", "#E0AF68", "#66D9E8",
	}
)

type LogTarget struct {
	ID    string
	Name  string
	State string
	Color string
}

type LogEntry struct {
	ContainerID   string
	ContainerName string
	Timestamp     time.Time
	Message       string
	System        bool
	Sequence      int64
}

type LogViewerState struct {
	Entries    []LogEntry
	Targets    map[string]LogTarget
	Scroll     int
	Follow     bool
	Filter     string
	MaxEntries int
	nextSeq    int64
}

func NewLogViewerState(maxEntries int, targets []LogTarget) LogViewerState {
	if maxEntries <= 0 {
		maxEntries = detailLogBufferMax
	}
	targetMap := make(map[string]LogTarget, len(targets))
	for _, target := range targets {
		targetMap[target.ID] = target
	}
	return LogViewerState{
		Targets:    targetMap,
		Follow:     true,
		MaxEntries: maxEntries,
	}
}

func SelectCentralLogTargets(containers []docker.ContainerInfo, selected map[string]bool, colors map[string]string) []LogTarget {
	var targets []LogTarget
	if len(selected) > 0 {
		for _, c := range containers {
			if selected[c.ID] {
				target := LogTarget{ID: c.ID, Name: c.Name, State: c.State}
				target.Color = ResolveLogTargetColor(colors, target)
				targets = append(targets, target)
			}
		}
		return targets
	}
	for _, c := range containers {
		if c.State != "running" {
			continue
		}
		target := LogTarget{ID: c.ID, Name: c.Name, State: c.State}
		target.Color = ResolveLogTargetColor(colors, target)
		targets = append(targets, target)
	}
	return targets
}

func ResolveLogTargetColor(colors map[string]string, target LogTarget) string {
	if colors != nil {
		if color := validConfiguredColor(colors[target.ID]); color != "" {
			return color
		}
		if color := validConfiguredColor(colors[target.Name]); color != "" {
			return color
		}
		prefixes := make([]string, 0, len(colors))
		for key := range colors {
			prefixes = append(prefixes, key)
		}
		sort.Slice(prefixes, func(i, j int) bool { return len(prefixes[i]) > len(prefixes[j]) })
		for _, prefix := range prefixes {
			if prefix == target.ID || prefix == target.Name {
				continue
			}
			if strings.HasPrefix(target.ID, prefix) {
				if color := validConfiguredColor(colors[prefix]); color != "" {
					return color
				}
			}
		}
	}
	return stableLogColor(target)
}

func validConfiguredColor(color string) string {
	if hexColorRE.MatchString(color) {
		return color
	}
	return ""
}

func stableLogColor(target LogTarget) string {
	key := target.ID
	if key == "" {
		key = target.Name
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return logPalette[int(h.Sum32())%len(logPalette)]
}

func (s *LogViewerState) Append(entries ...LogEntry) {
	for _, entry := range entries {
		if entry.Sequence == 0 {
			s.nextSeq++
			entry.Sequence = s.nextSeq
		} else if entry.Sequence > s.nextSeq {
			s.nextSeq = entry.Sequence
		}
		entry.Message = sanitizeOutputText(entry.Message)
		if strings.TrimSpace(entry.Message) == "" {
			continue
		}
		s.Entries = append(s.Entries, entry)
	}
	if len(s.Entries) > s.MaxEntries {
		trim := len(s.Entries) - s.MaxEntries
		s.Entries = s.Entries[trim:]
		if !s.Follow {
			s.Scroll -= trim
			if s.Scroll < 0 {
				s.Scroll = 0
			}
		}
	}
}

func (s *LogViewerState) SetFilter(filter string) {
	s.Filter = strings.TrimSpace(filter)
	s.normalize(1)
}

func (s LogViewerState) FilteredEntries() []LogEntry {
	if s.Filter == "" {
		return append([]LogEntry(nil), s.Entries...)
	}
	q := strings.ToLower(s.Filter)
	out := make([]LogEntry, 0, len(s.Entries))
	for _, entry := range s.Entries {
		if strings.Contains(strings.ToLower(entry.ContainerName), q) ||
			strings.Contains(strings.ToLower(entry.Message), q) {
			out = append(out, entry)
		}
	}
	return out
}

func (s LogViewerState) VisibleWindow(height int) (int, int, int) {
	if height < 1 {
		height = 1
	}
	total := len(s.FilteredEntries())
	if total == 0 {
		return 0, 0, 0
	}
	maxScroll := max(0, total-height)
	scroll := s.Scroll
	if s.Follow {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}
	return scroll, min(scroll+height, total), total
}

func (s LogViewerState) VisibleEntries(height int) []LogEntry {
	filtered := s.FilteredEntries()
	start, end, _ := s.VisibleWindow(height)
	return filtered[start:end]
}

func (s *LogViewerState) ScrollBy(delta, height int) {
	if delta < 0 && s.Follow {
		start, _, _ := s.VisibleWindow(height)
		s.Scroll = start
	}
	if delta < 0 {
		s.Follow = false
	}
	s.Scroll += delta
	s.normalize(height)
}

func (s *LogViewerState) ScrollPage(delta, height int) {
	step := detailPageStep(height)
	s.ScrollBy(delta*step, height)
}

func (s *LogViewerState) ScrollHome(height int) {
	s.Follow = false
	s.Scroll = 0
	s.normalize(height)
}

func (s *LogViewerState) ScrollEnd() {
	s.Follow = true
	s.Scroll = 1 << 20
}

func (s *LogViewerState) normalize(height int) {
	_, _, total := s.VisibleWindow(height)
	maxScroll := max(0, total-height)
	if s.Scroll < 0 {
		s.Scroll = 0
	}
	if s.Scroll >= maxScroll {
		s.Scroll = maxScroll
		if total > 0 {
			s.Follow = true
		}
	}
}

func SortLogEntries(entries []LogEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if !a.Timestamp.IsZero() && !b.Timestamp.IsZero() && !a.Timestamp.Equal(b.Timestamp) {
			return a.Timestamp.Before(b.Timestamp)
		}
		if !a.Timestamp.IsZero() && b.Timestamp.IsZero() {
			return true
		}
		if a.Timestamp.IsZero() && !b.Timestamp.IsZero() {
			return false
		}
		return a.Sequence < b.Sequence
	})
}

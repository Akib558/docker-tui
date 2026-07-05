package ui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/akib558/docker-tui/docker"
	"github.com/aymanbagabas/go-osc52/v2"
	"hash/fnv"
	"regexp"
	"sort"
	"time"
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
	Entries          []LogEntry
	Targets          map[string]LogTarget
	Scroll           int
	Follow           bool
	Filter           string
	UseRegex         bool
	SelectedIndex    int
	LogCursor        int
	Focused          int
	Selection        map[int64]bool
	HiddenContainers map[string]bool
	SelectAnchor     int
	ShowLegend       bool
	MaxEntries       int
	nextSeq          int64
	filterCache      []LogEntry
	filterKey        string
	filterRE         *regexp.Regexp
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
		Selection:  make(map[int64]bool),
		ShowLegend: true,
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
		if len(targets) > 0 {
			return targets
		}
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

func (s *LogViewerState) invalidateFilterCache() {
	s.filterCache = nil
	s.filterKey = ""
}

func (s *LogViewerState) ToggleContainer(id string) {
	if id == "" {
		return
	}
	if s.HiddenContainers == nil {
		s.HiddenContainers = make(map[string]bool)
	}
	s.HiddenContainers[id] = !s.HiddenContainers[id]
	s.invalidateFilterCache()
}

func (s *LogViewerState) ShowAllContainers() {
	if len(s.HiddenContainers) == 0 {
		return
	}
	s.HiddenContainers = make(map[string]bool)
	s.invalidateFilterCache()
}

func (s LogViewerState) IsContainerHidden(id string) bool {
	return s.HiddenContainers != nil && s.HiddenContainers[id]
}

func (s LogViewerState) hiddenContainerKey() string {
	if len(s.HiddenContainers) == 0 {
		return ""
	}
	ids := make([]string, 0, len(s.HiddenContainers))
	for id, hidden := range s.HiddenContainers {
		if hidden {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return strings.Join(ids, ",")
}

func (s LogViewerState) entriesAfterContainerFilter() []LogEntry {
	if len(s.HiddenContainers) == 0 {
		return s.Entries
	}
	out := make([]LogEntry, 0, len(s.Entries))
	for _, entry := range s.Entries {
		if !s.HiddenContainers[entry.ContainerID] {
			out = append(out, entry)
		}
	}
	return out
}

func (s *LogViewerState) Append(entries ...LogEntry) {
	for _, entry := range entries {
		s.insertEntry(entry)
	}
	s.invalidateFilterCache()
}

func (s *LogViewerState) AppendSorted(entries ...LogEntry) {
	for _, entry := range entries {
		s.insertEntrySorted(entry)
	}
	s.invalidateFilterCache()
}

func (s *LogViewerState) insertEntry(entry LogEntry) {
	if entry.Sequence == 0 {
		s.nextSeq++
		entry.Sequence = s.nextSeq
	} else if entry.Sequence > s.nextSeq {
		s.nextSeq = entry.Sequence
	}
	entry.Message = sanitizeOutputText(entry.Message)
	if strings.TrimSpace(entry.Message) == "" {
		return
	}
	s.Entries = append(s.Entries, entry)
	s.trimEntries()
}

func (s *LogViewerState) insertEntrySorted(entry LogEntry) {
	if entry.Sequence == 0 {
		s.nextSeq++
		entry.Sequence = s.nextSeq
	} else if entry.Sequence > s.nextSeq {
		s.nextSeq = entry.Sequence
	}
	entry.Message = sanitizeOutputText(entry.Message)
	if strings.TrimSpace(entry.Message) == "" {
		return
	}
	idx := len(s.Entries)
	for idx > 0 && logEntryLess(s.Entries[idx-1], entry) {
		idx--
	}
	s.Entries = append(s.Entries, LogEntry{})
	copy(s.Entries[idx+1:], s.Entries[idx:])
	s.Entries[idx] = entry
	s.trimEntries()
}

func (s *LogViewerState) trimEntries() {
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

func logEntryLess(a, b LogEntry) bool {
	if !a.Timestamp.IsZero() && !b.Timestamp.IsZero() {
		if a.Timestamp.Equal(b.Timestamp) {
			return a.Sequence < b.Sequence
		}
		return a.Timestamp.Before(b.Timestamp)
	}
	return a.Sequence < b.Sequence
}

func (s *LogViewerState) SetFilter(filter string) {
	s.Filter = strings.TrimSpace(filter)
	s.invalidateFilterCache()
	s.compileFilterRE()
	s.normalize(1)
}

func (s *LogViewerState) SetUseRegex(useRegex bool) {
	s.UseRegex = useRegex
	s.invalidateFilterCache()
	s.compileFilterRE()
}

func (s *LogViewerState) compileFilterRE() {
	s.filterRE = nil
	if s.Filter == "" || !s.UseRegex {
		return
	}
	re, err := regexp.Compile("(?i)" + s.Filter)
	if err == nil {
		s.filterRE = re
	}
}

func (s LogViewerState) FilteredEntries() []LogEntry {
	base := s.entriesAfterContainerFilter()
	if s.Filter == "" {
		return base
	}
	key := s.Filter + "|" + fmt.Sprint(s.UseRegex) + "|" + fmt.Sprint(len(base)) + "|" + s.hiddenContainerKey()
	if s.filterCache != nil && s.filterKey == key {
		return s.filterCache
	}
	var out []LogEntry
	if s.UseRegex && s.filterRE != nil {
		out = s.filteredEntriesRegexFrom(base)
	} else {
		out = s.filteredEntriesSubstringFrom(base)
	}
	return out
}

func (s LogViewerState) filteredEntriesSubstringFrom(entries []LogEntry) []LogEntry {
	q := strings.ToLower(s.Filter)
	out := make([]LogEntry, 0, len(entries))
	for _, entry := range entries {
		if strings.Contains(strings.ToLower(entry.ContainerName), q) ||
			strings.Contains(strings.ToLower(entry.ContainerID), q) ||
			strings.Contains(strings.ToLower(entry.Message), q) {
			out = append(out, entry)
		}
	}
	return out
}

func (s LogViewerState) filteredEntriesRegexFrom(entries []LogEntry) []LogEntry {
	re := s.filterRE
	if re == nil {
		return s.filteredEntriesSubstringFrom(entries)
	}
	out := make([]LogEntry, 0, len(entries))
	for _, entry := range entries {
		if re.MatchString(entry.Message) ||
			re.MatchString(entry.ContainerName) ||
			re.MatchString(entry.ContainerID) {
			out = append(out, entry)
		}
	}
	return out
}

func (s LogViewerState) VisibleEntries(height int) []LogEntry {
	filtered := s.FilteredEntries()
	start, end, _ := s.visibleWindowFor(filtered, height)
	return filtered[start:end]
}

func (s LogViewerState) visibleWindowFor(filtered []LogEntry, height int) (int, int, int) {
	if height < 1 {
		height = 1
	}
	total := len(filtered)
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

func (s LogViewerState) VisibleWindow(height int) (int, int, int) {
	return s.visibleWindowFor(s.FilteredEntries(), height)
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
	if delta > 0 {
		total := len(s.FilteredEntries())
		if height < 1 {
			height = 1
		}
		if total > 0 && s.Scroll >= max(0, total-height) {
			s.Follow = true
		}
	}
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
	if height < 1 {
		height = 1
	}
	filtered := s.FilteredEntries()
	total := len(filtered)
	maxScroll := max(0, total-height)
	if s.Scroll < 0 {
		s.Scroll = 0
	}
	if s.Scroll > maxScroll {
		s.Scroll = maxScroll
	}
	if total == 0 {
		s.Focused = 0
		return
	}
	if s.Focused < 0 {
		s.Focused = 0
	}
	if s.Focused >= total {
		s.Focused = total - 1
	}
	if s.Follow && total > 0 {
		s.Focused = total - 1
	}
}

func (s *LogViewerState) MoveFocus(delta int, height int) {
	filtered := s.FilteredEntries()
	if len(filtered) == 0 {
		return
	}
	if delta < 0 && s.Follow {
		start, _, _ := s.VisibleWindow(height)
		s.Scroll = start
		s.Follow = false
	}
	s.Focused += delta
	if s.Focused < 0 {
		s.Focused = 0
	}
	if s.Focused >= len(filtered) {
		s.Focused = len(filtered) - 1
	}
	s.ensureFocusVisible(height)
}

func (s *LogViewerState) ensureFocusVisible(height int) {
	if height < 1 {
		height = 1
	}
	start, end, _ := s.VisibleWindow(height)
	if s.Focused < start {
		s.Scroll = s.Focused
		s.Follow = false
	}
	if s.Focused >= end {
		s.Scroll = s.Focused - height + 1
		s.Follow = false
	}
	s.normalize(height)
}

func (s *LogViewerState) ToggleSelectFocused() {
	filtered := s.FilteredEntries()
	if s.Focused < 0 || s.Focused >= len(filtered) {
		return
	}
	seq := filtered[s.Focused].Sequence
	if s.Selection == nil {
		s.Selection = make(map[int64]bool)
	}
	if s.Selection[seq] {
		delete(s.Selection, seq)
	} else {
		s.Selection[seq] = true
	}
	s.SelectAnchor = s.Focused
}

func (s *LogViewerState) SelectRangeToFocused() {
	filtered := s.FilteredEntries()
	if len(filtered) == 0 {
		return
	}
	if s.Selection == nil {
		s.Selection = make(map[int64]bool)
	}
	lo, hi := s.SelectAnchor, s.Focused
	if lo > hi {
		lo, hi = hi, lo
	}
	for i := lo; i <= hi; i++ {
		s.Selection[filtered[i].Sequence] = true
	}
}

func (s *LogViewerState) ClearSelection() {
	s.Selection = make(map[int64]bool)
}

func (s LogViewerState) SelectedEntries() []LogEntry {
	if len(s.Selection) == 0 {
		return nil
	}
	filtered := s.FilteredEntries()
	out := make([]LogEntry, 0, len(s.Selection))
	for _, entry := range filtered {
		if s.Selection[entry.Sequence] {
			out = append(out, entry)
		}
	}
	return out
}

func (s LogViewerState) FocusedEntry() (LogEntry, bool) {
	filtered := s.FilteredEntries()
	if s.Focused < 0 || s.Focused >= len(filtered) {
		return LogEntry{}, false
	}
	return filtered[s.Focused], true
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

func CopyToClipboard(text string) error {
	return copyToSystemClipboard(text)
}

func CopyLogEntries(entries []LogEntry) error {
	if len(entries) == 0 {
		return fmt.Errorf("no log lines to copy")
	}
	return CopyToClipboard(formatLogLinesForCopy(entries))
}

func copyToSystemClipboard(text string) error {
	if text == "" {
		return fmt.Errorf("empty clipboard text")
	}
	try := func(fn func() error) bool {
		return fn() == nil
	}

	switch runtime.GOOS {
	case "darwin":
		if try(func() error {
			cmd := exec.Command("pbcopy")
			cmd.Stdin = strings.NewReader(text)
			return cmd.Run()
		}) {
			return nil
		}
	case "windows":
		if try(func() error {
			cmd := exec.Command("cmd", "/c", "clip")
			cmd.Stdin = strings.NewReader(text)
			return cmd.Run()
		}) {
			return nil
		}
		if try(func() error {
			cmd := exec.Command("powershell", "-NoProfile", "-Command", "Set-Clipboard -Value ([Console]::In.ReadToEnd())")
			cmd.Stdin = strings.NewReader(text)
			return cmd.Run()
		}) {
			return nil
		}
	default:
		if try(func() error {
			cmd := exec.Command("wl-copy")
			cmd.Stdin = strings.NewReader(text)
			return cmd.Run()
		}) {
			return nil
		}
		if try(func() error { return copyViaWayland(text) }) {
			return nil
		}
		if try(func() error { return copyViaXclip(text) }) {
			return nil
		}
		if try(func() error {
			cmd := exec.Command("xsel", "--clipboard", "--input")
			cmd.Stdin = strings.NewReader(text)
			return cmd.Run()
		}) {
			return nil
		}
	}

	if err := copyViaOSC52(text); err == nil {
		return nil
	}
	return fmt.Errorf("clipboard unavailable; install wl-copy, xclip, pbcopy, or use an OSC52-capable terminal")
}

func copyViaOSC52(text string) error {
	seq := osc52.New(text)
	if _, err := fmt.Fprint(os.Stdout, seq); err == nil {
		return nil
	}
	_, err := fmt.Fprint(os.Stderr, seq)
	return err
}

func copyViaWayland(text string) error {
	cmd := exec.Command("wayland-copy")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func copyViaXclip(text string) error {
	cmd := exec.Command("xclip", "-selection", "clipboard")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

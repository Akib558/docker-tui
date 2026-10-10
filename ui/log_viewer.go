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
	msgLower      string
	nameLower     string
	idLower       string
}

type LogViewerState struct {
	Entries            []LogEntry
	Targets            map[string]LogTarget
	Scroll             int
	Follow             bool
	Filter             string
	UseRegex           bool
	SelectedIndex      int
	LogCursor          int
	Focused            int
	Selection          map[int64]bool
	HiddenContainers   map[string]bool
	SelectAnchor       int
	ShowLegend         bool
	MaxEntries         int
	nextSeq            int64
	filterRE           *regexp.Regexp
	filterLower        string
	visibleEntries     []LogEntry
	visibleSig         string
	visibleIncremental bool
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

func (s *LogViewerState) invalidateVisible() {
	s.visibleEntries = nil
	s.visibleSig = ""
	s.visibleIncremental = false
}

func (s *LogViewerState) visibilitySig() string {
	return s.Filter + "|" + fmt.Sprint(s.UseRegex) + "|" + fmt.Sprint(len(s.Entries)) + "|" + s.hiddenContainerKey()
}

func (s *LogViewerState) prepareEntry(entry LogEntry) LogEntry {
	entry.Message = sanitizeOutputText(entry.Message)
	if strings.TrimSpace(entry.Message) == "" {
		return LogEntry{}
	}
	entry.msgLower = strings.ToLower(entry.Message)
	entry.nameLower = strings.ToLower(entry.ContainerName)
	entry.idLower = strings.ToLower(entry.ContainerID)
	return entry
}

func (s *LogViewerState) entryPassesFilter(entry LogEntry) bool {
	if s.HiddenContainers != nil && s.HiddenContainers[entry.ContainerID] {
		return false
	}
	if s.Filter == "" {
		return true
	}
	if s.UseRegex && s.filterRE != nil {
		return s.filterRE.MatchString(entry.Message) ||
			s.filterRE.MatchString(entry.ContainerName) ||
			s.filterRE.MatchString(entry.ContainerID)
	}
	q := s.filterLower
	return strings.Contains(entry.nameLower, q) ||
		strings.Contains(entry.idLower, q) ||
		strings.Contains(entry.msgLower, q)
}

func (s *LogViewerState) rebuildVisible() {
	s.visibleEntries = make([]LogEntry, 0, len(s.Entries))
	for _, entry := range s.Entries {
		if s.entryPassesFilter(entry) {
			s.visibleEntries = append(s.visibleEntries, entry)
		}
	}
	s.visibleSig = s.visibilitySig()
	s.visibleIncremental = true
}

func (s *LogViewerState) ensureVisible() {
	sig := s.visibilitySig()
	if s.visibleSig == sig && s.visibleEntries != nil {
		return
	}
	s.rebuildVisible()
}

func (s *LogViewerState) noteAppended(entry LogEntry) {
	if !s.visibleIncremental || s.visibleSig == "" {
		s.invalidateVisible()
		return
	}
	prevLen := len(s.visibleEntries)
	sig := s.visibilitySig()
	if s.entryPassesFilter(entry) {
		s.visibleEntries = append(s.visibleEntries, entry)
	}
	s.visibleSig = sig
	if len(s.visibleEntries) < prevLen {
		s.invalidateVisible()
	}
}

func (s *LogViewerState) ToggleContainer(id string) {
	if id == "" {
		return
	}
	if s.HiddenContainers == nil {
		s.HiddenContainers = make(map[string]bool)
	}
	s.HiddenContainers[id] = !s.HiddenContainers[id]
	s.invalidateVisible()
}

func (s *LogViewerState) ShowAllContainers() {
	if len(s.HiddenContainers) == 0 {
		return
	}
	s.HiddenContainers = make(map[string]bool)
	s.invalidateVisible()
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

func (s *LogViewerState) Append(entries ...LogEntry) {
	for _, entry := range entries {
		s.insertEntry(entry)
	}
}

func (s *LogViewerState) AppendSorted(entries ...LogEntry) {
	if len(entries) == 0 {
		return
	}
	for _, entry := range entries {
		s.insertEntrySorted(entry, false)
	}
	s.maybeCompactEntries()
	s.invalidateVisible()
}

func (s *LogViewerState) AppendSortedBatch(entries []LogEntry) {
	if len(entries) == 0 {
		return
	}
	for _, entry := range entries {
		s.insertEntrySorted(entry, false)
	}
	s.maybeCompactEntries()
	s.invalidateVisible()
}

func (s *LogViewerState) insertEntry(entry LogEntry) {
	if entry.Sequence == 0 {
		s.nextSeq++
		entry.Sequence = s.nextSeq
	} else if entry.Sequence > s.nextSeq {
		s.nextSeq = entry.Sequence
	}
	entry = s.prepareEntry(entry)
	if entry.Message == "" {
		return
	}
	s.Entries = append(s.Entries, entry)
	s.trimEntries()
	s.noteAppended(entry)
}

func (s *LogViewerState) insertEntrySorted(entry LogEntry, compact bool) {
	if entry.Sequence == 0 {
		s.nextSeq++
		entry.Sequence = s.nextSeq
	} else if entry.Sequence > s.nextSeq {
		s.nextSeq = entry.Sequence
	}
	entry = s.prepareEntry(entry)
	if entry.Message == "" {
		return
	}
	idx := sort.Search(len(s.Entries), func(i int) bool {
		return !logEntryLess(s.Entries[i], entry)
	})
	s.Entries = append(s.Entries, LogEntry{})
	copy(s.Entries[idx+1:], s.Entries[idx:])
	s.Entries[idx] = entry
	s.trimEntries()
	if compact {
		s.maybeCompactEntries()
	}
}

func (s *LogViewerState) maybeCompactEntries() {
	if len(s.Entries) == 0 {
		return
	}
	if cap(s.Entries) > s.MaxEntries*4 && cap(s.Entries) > len(s.Entries)*2 {
		compact := make([]LogEntry, len(s.Entries))
		copy(compact, s.Entries)
		s.Entries = compact
	}
}

func (s *LogViewerState) trimEntries() {
	if len(s.Entries) <= s.MaxEntries {
		return
	}
	trim := len(s.Entries) - s.MaxEntries
	if !s.Follow {
		s.Scroll -= trim
		if s.Scroll < 0 {
			s.Scroll = 0
		}
	}
	if trim > len(s.Entries)/2 {
		compact := make([]LogEntry, s.MaxEntries)
		copy(compact, s.Entries[trim:])
		s.Entries = compact
	} else {
		s.Entries = s.Entries[trim:]
	}
	s.invalidateVisible()
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
	s.filterLower = strings.ToLower(s.Filter)
	s.invalidateVisible()
	s.compileFilterRE()
	s.normalize(1)
}

func (s *LogViewerState) SetUseRegex(useRegex bool) {
	s.UseRegex = useRegex
	s.invalidateVisible()
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

func (s *LogViewerState) FilteredEntries() []LogEntry {
	s.ensureVisible()
	return s.visibleEntries
}

func (s *LogViewerState) VisibleEntries(height int) []LogEntry {
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

func (s *LogViewerState) VisibleWindow(height int) (int, int, int) {
	return s.visibleWindowFor(s.FilteredEntries(), height)
}

func (s *LogViewerState) ScrollBy(delta, height int) {
	filtered := s.FilteredEntries()
	if delta < 0 && s.Follow {
		start, _, _ := s.visibleWindowFor(filtered, height)
		s.Scroll = start
	}
	if delta < 0 {
		s.Follow = false
	}
	s.Scroll += delta
	s.normalizeWith(filtered, height)
	if delta > 0 {
		total := len(filtered)
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
	s.normalizeWith(s.FilteredEntries(), height)
}

func (s *LogViewerState) normalizeWith(filtered []LogEntry, height int) {
	if height < 1 {
		height = 1
	}
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
		start, _, _ := s.visibleWindowFor(filtered, height)
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
	s.ensureFocusVisible(height, filtered)
}

func (s *LogViewerState) ensureFocusVisible(height int, filtered []LogEntry) {
	if height < 1 {
		height = 1
	}
	start, end, _ := s.visibleWindowFor(filtered, height)
	if s.Focused < start {
		s.Scroll = s.Focused
		s.Follow = false
	}
	if s.Focused >= end {
		s.Scroll = s.Focused - height + 1
		s.Follow = false
	}
	s.normalizeWith(filtered, height)
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

func (s *LogViewerState) SelectedEntries() []LogEntry {
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

func (s *LogViewerState) FocusedEntry() (LogEntry, bool) {
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

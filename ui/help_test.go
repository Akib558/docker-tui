package ui

import (
	"strings"
	"testing"
)

// The ? help dialog must document every reachable view. Events is reachable
// via D from the container list and via :events from the command palette,
// but the help dialog historically didn't mention it by name.
func TestHelpDialogDocumentsEvents(t *testing.T) {
	withFixedNow(t, fixedNow)
	m := Model{width: 120, height: 40, dialog: dialogHelp}
	out := stripANSI(m.View())
	if !strings.Contains(out, "events") {
		t.Fatal("the ? help dialog must document the events view — it is reachable via D and :events")
	}
}

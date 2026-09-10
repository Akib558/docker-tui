package ui

import (
	"strings"
	"testing"
)

// The ? help dialog must document every reachable view. Events is reachable
// via D from the container list and via :events from the command palette;
// the dialog already named it before this plan ({"D", "docker events"}), and
// this plan's Task 19 only reworded that entry to "docker events stream" —
// so this test pins the specific new wording rather than mere presence of
// the word "events" (which the old wording already satisfied too).
func TestHelpDialogDocumentsEvents(t *testing.T) {
	withFixedNow(t, fixedNow)
	m := Model{width: 120, height: 40, dialog: dialogHelp}
	out := stripANSI(m.View())
	if !strings.Contains(out, "docker events stream") {
		t.Fatal("the ? help dialog must document the events view as \"docker events stream\" — it is reachable via D and :events")
	}
}

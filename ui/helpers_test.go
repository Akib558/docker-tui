package ui

import (
	"testing"

	"github.com/akib/docker-tui/docker"
)

func TestAppendHistBounds(t *testing.T) {
	var h []float64
	for i := 0; i < historyLen+10; i++ {
		h = appendHist(h, float64(i))
	}
	if len(h) != historyLen {
		t.Fatalf("expected history length %d, got %d", historyLen, len(h))
	}
	if h[0] != 10 {
		t.Fatalf("expected oldest value 10 after trim, got %.0f", h[0])
	}
}

func TestTruncate(t *testing.T) {
	got := truncate("docker-container-name", 10)
	if got != "docker-..." {
		t.Fatalf("unexpected truncate result: %q", got)
	}
	if truncate("abc", 10) != "abc" {
		t.Fatalf("short strings should be unchanged")
	}
}

func TestCleanDockerLogs(t *testing.T) {
	raw := string([]byte{1, 0, 0, 0, 0, 0, 0, 5}) + "hello\nplain"
	got := cleanDockerLogs(raw)
	if got != "hello\nplain" {
		t.Fatalf("unexpected cleaned logs: %q", got)
	}
}

func TestFormatPortsSummary(t *testing.T) {
	ports := []docker.PortBinding{
		{HostPort: "8080", ContPort: "80"},
		{HostPort: "8080", ContPort: "80"},
		{HostPort: "", ContPort: "443"},
	}
	got := formatPortsSummary(ports)
	if got != "8080->80,443" {
		t.Fatalf("unexpected ports summary: %q", got)
	}
}

func TestSanitizeOutputText(t *testing.T) {
	in := "line1\rline2\x1b[31m red\x1b[0m\n\n\nline3\x00"
	got := sanitizeOutputText(in)
	want := "line1\nline2 red\n\nline3"
	if got != want {
		t.Fatalf("unexpected sanitized output:\nwant: %q\ngot:  %q", want, got)
	}
}

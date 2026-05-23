package docker

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestParseDockerLogRecord_StripsHeaderAndParsesTimestampMessage(t *testing.T) {
	tsText := "2026-05-23T10:20:30.123456789Z"
	wantTime, err := time.Parse(time.RFC3339Nano, tsText)
	if err != nil {
		t.Fatalf("failed to parse expected timestamp: %v", err)
	}

	line := string([]byte{1, 0, 0, 0, 0, 0, 0, 0}) + tsText + " service started"
	got := ParseDockerLogRecord(line)

	if !got.Timestamp.Equal(wantTime) {
		t.Fatalf("unexpected timestamp: want %s, got %s", wantTime.Format(time.RFC3339Nano), got.Timestamp.Format(time.RFC3339Nano))
	}
	if got.Message != "service started" {
		t.Fatalf("unexpected message: want %q, got %q", "service started", got.Message)
	}
}

func TestParseDockerLogRecord_WithoutTimestampReturnsZeroTimestampAndOriginalMessage(t *testing.T) {
	line := "plain log line"
	got := ParseDockerLogRecord(line)

	if !got.Timestamp.IsZero() {
		t.Fatalf("expected zero timestamp, got %s", got.Timestamp.Format(time.RFC3339Nano))
	}
	if got.Message != line {
		t.Fatalf("unexpected message: want %q, got %q", line, got.Message)
	}
}

func TestDecodeDockerLogRecords_SkipsBlankLines(t *testing.T) {
	ts1 := "2026-05-23T10:20:30Z"
	ts2 := "2026-05-23T10:20:31.000000001Z"
	wantTime1, err := time.Parse(time.RFC3339Nano, ts1)
	if err != nil {
		t.Fatalf("failed to parse expected ts1: %v", err)
	}
	wantTime2, err := time.Parse(time.RFC3339Nano, ts2)
	if err != nil {
		t.Fatalf("failed to parse expected ts2: %v", err)
	}

	raw := strings.Join([]string{
		string([]byte{1, 0, 0, 0, 0, 0, 0, 0}) + ts1 + " first message",
		"",
		"   ",
		string([]byte{2, 0, 0, 0, 0, 0, 0, 0}) + ts2 + " second message",
		"",
	}, "\n")

	got, err := DecodeDockerLogRecords(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("DecodeDockerLogRecords returned error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 log records, got %d", len(got))
	}
	if !got[0].Timestamp.Equal(wantTime1) || got[0].Message != "first message" {
		t.Fatalf("unexpected first record: %+v", got[0])
	}
	if !got[1].Timestamp.Equal(wantTime2) || got[1].Message != "second message" {
		t.Fatalf("unexpected second record: %+v", got[1])
	}
}

func TestDecodeDockerLogRecords_AllowsLinesLargerThanOneMiB(t *testing.T) {
	tsText := "2026-05-23T10:20:30Z"
	largeMessage := strings.Repeat("x", 2*1024*1024)
	raw := string([]byte{1, 0, 0, 0, 0, 0, 0, 0}) + tsText + " " + largeMessage + "\n"

	got, err := DecodeDockerLogRecords(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("DecodeDockerLogRecords returned error for large line: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 log record, got %d", len(got))
	}
	if got[0].Message != largeMessage {
		t.Fatalf("unexpected large message length: want %d, got %d", len(largeMessage), len(got[0].Message))
	}
}

func TestWriteDockerLogMessages_PreservesLegacyMessageOnlyOutput(t *testing.T) {
	tsText := "2026-05-23T10:20:30Z"
	input := strings.Join([]string{
		string([]byte{1, 0, 0, 0, 0, 0, 0, 0}) + tsText + " first",
		"",
		string([]byte{2, 0, 0, 0, 0, 0, 0, 0}) + tsText + " second",
		"plain",
	}, "\n")

	var out bytes.Buffer
	if err := writeDockerLogMessages(strings.NewReader(input), &out); err != nil {
		t.Fatalf("writeDockerLogMessages returned error: %v", err)
	}

	if out.String() != "first\n\nsecond\nplain\n" {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

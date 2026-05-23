package docker

import (
	"bufio"
	"io"
	"strings"
	"time"
)

type LogRecord struct {
	Timestamp time.Time
	Message   string
}

func DecodeDockerLogRecords(reader io.Reader) ([]LogRecord, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	records := make([]LogRecord, 0)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		records = append(records, ParseDockerLogRecord(line))
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func ParseDockerLogRecord(line string) LogRecord {
	line = stripDockerLogHeader(line)

	tsText, message, ok := strings.Cut(line, " ")
	if ok {
		if ts, err := time.Parse(time.RFC3339Nano, tsText); err == nil {
			return LogRecord{
				Timestamp: ts,
				Message:   message,
			}
		}
	}

	return LogRecord{
		Message: line,
	}
}

func stripDockerLogHeader(line string) string {
	if len(line) < 8 {
		return line
	}

	if (line[0] == 1 || line[0] == 2) && line[1] == 0 && line[2] == 0 && line[3] == 0 {
		return line[8:]
	}

	return line
}

func StripDockerLogHeaderForUI(line string) string {
	return stripDockerLogHeader(line)
}

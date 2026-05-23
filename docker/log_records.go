package docker

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

type LogRecord struct {
	Timestamp time.Time
	Message   string
}

func DecodeDockerLogRecords(reader io.Reader) ([]LogRecord, error) {
	records := make([]LogRecord, 0)
	err := forEachDockerLogLine(reader, func(line string) error {
		if strings.TrimSpace(line) == "" {
			return nil
		}
		records = append(records, ParseDockerLogRecord(line))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return records, nil
}

func writeDockerLogMessages(reader io.Reader, writer io.Writer) error {
	return forEachDockerLogLine(reader, func(line string) error {
		record := ParseDockerLogRecord(line)
		_, err := io.WriteString(writer, record.Message+"\n")
		return err
	})
}

func forEachDockerLogLine(reader io.Reader, onLine func(string) error) error {
	buffer := make([]byte, 0, 64*1024)
	chunk := make([]byte, 64*1024)

	for {
		n, err := reader.Read(chunk)
		if n > 0 {
			buffer = append(buffer, chunk[:n]...)
			for {
				newlineIdx := bytes.IndexByte(buffer, '\n')
				if newlineIdx < 0 {
					break
				}
				line := strings.TrimSuffix(string(buffer[:newlineIdx]), "\r")
				if onLineErr := onLine(line); onLineErr != nil {
					return onLineErr
				}
				buffer = buffer[newlineIdx+1:]
			}
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				if len(buffer) > 0 {
					line := strings.TrimSuffix(string(buffer), "\r")
					if onLineErr := onLine(line); onLineErr != nil {
						return onLineErr
					}
				}
				return nil
			}
			return fmt.Errorf("failed to read docker logs: %w", err)
		}
	}
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

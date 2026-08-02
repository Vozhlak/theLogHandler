package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

type LogEntry struct {
	Timestamp time.Time
	Level     string
	Service   string
	Message   string
	RequestID string
	UserID    string
}

var (
	requestIDRe = regexp.MustCompile(`request_id=([a-zA-Z0-9_]+)`)
	userIDRe    = regexp.MustCompile(`user_id=([a-zA-Z0-9_]+)`)
)

func ParseLogLine(line string) (LogEntry, error) {
	var entry LogEntry

	line = strings.TrimSpace(line)
	if line == "" {
		return entry, errors.New("log line is empty")
	}

	parts := strings.SplitN(line, " ", 4)
	if len(parts) < 4 {
		return entry, errors.New("invalid log format: expected timestamp, level, service and message")
	}

	timestampStr := parts[0]
	levelRaw := parts[1]
	serviceRaw := parts[2]
	message := strings.TrimSpace(parts[3])

	timestamp, err := time.Parse(time.RFC3339Nano, timestampStr)
	if err != nil {
		return entry, fmt.Errorf("parse timestamp: %w", err)
	}

	if !strings.HasPrefix(levelRaw, "[") || !strings.HasSuffix(levelRaw, "]") {
		return entry, errors.New("invalid log format: level must be in [LEVEL] form")
	}

	level := strings.Trim(levelRaw, "[]")
	if level == "" {
		return entry, errors.New("invalid log format: empty level")
	}

	service := strings.TrimSuffix(serviceRaw, ":")
	if service == "" || service == serviceRaw && !strings.HasSuffix(serviceRaw, ":") {
		return entry, errors.New("invalid log format: service must end with ':'")
	}

	if message == "" {
		return entry, errors.New("invalid log format: empty message")
	}

	requestMatch := requestIDRe.FindStringSubmatch(line)
	if len(requestMatch) < 2 {
		return entry, errors.New("invalid log format: request_id not found")
	}
	requestID := requestMatch[1]

	userID := ""
	userMatch := userIDRe.FindStringSubmatch(line)
	if len(userMatch) > 1 {
		userID = userMatch[1]
	}

	entry = LogEntry{
		Timestamp: timestamp,
		Level:     level,
		Service:   service,
		Message:   message,
		RequestID: requestID,
		UserID:    userID,
	}

	return entry, nil
}

func ReadLogFile(filepath string) ([]LogEntry, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("error file open: %w", err)
	}
	defer file.Close()

	var entries []LogEntry

	totalLines := 0
	parseErrors := 0

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		totalLines++

		line := scanner.Text()
		entry, err := ParseLogLine(line)
		if err != nil {
			parseErrors++

			fmt.Printf("warning: line: %d: %s\n", totalLines, err)
			continue
		}

		entries = append(entries, entry)
	}

	if err = scanner.Err(); err != nil {
		return nil, err
	}

	fmt.Printf("\nProcessing file: %s", filepath)
	fmt.Println("Total lines:", totalLines)
	fmt.Println("Valid entries:", len(entries))
	fmt.Println("Parse errors:", parseErrors)

	return entries, nil
}

func main() {
	fmt.Println("Happy coding!!!")
}

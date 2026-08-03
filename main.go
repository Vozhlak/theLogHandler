package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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

type FailedRequestReport struct {
	RequestID      string   `json:"request_id"`
	FailingService string   `json:"failing_service"`
	ErrorMessage   string   `json:"error_message"`
	Timeline       []string `json:"timeline"`
}

type AnalysisResult struct {
	TotalEntriesProcessed int                   `json:"total_entries_processed"`
	FailedRequestsFound   int                   `json:"failed_requests_found"`
	ProcessingTimeSeconds float64               `json:"processing_time_seconds"`
	FailedRequests        []FailedRequestReport `json:"failed_requests"`
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

	return entries, nil
}

func ScanLogDirectory(dirPath string) ([]string, error) {
	var logFiles []string

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && strings.HasSuffix(path, ".log") {
			logFiles = append(logFiles, path)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk directory %s: %w", dirPath, err)
	}

	sort.Strings(logFiles)

	return logFiles, err
}

func ProcessMultipleFiles(filePaths []string) ([]LogEntry, error) {
	if len(filePaths) == 0 {
		return nil, errors.New("file paths is empty")
	}

	var allEntries []LogEntry

	for _, filePath := range filePaths {
		entries, err := ReadLogFile(filePath)
		if err != nil {
			fmt.Printf("warning: failed to process file %s: %v\n", filePath, err)
			continue
		}

		fmt.Printf("%s: %d entries\n", filePath, len(entries))

		allEntries = append(allEntries, entries...)
	}

	return allEntries, nil
}

func CorrelateRequests(entries []LogEntry) map[string][]LogEntry {
	grouped := make(map[string][]LogEntry)
	const KeyNoRequestId = "no request_id"

	for _, item := range entries {
		if item.RequestID == "" {
			grouped[KeyNoRequestId] = append(grouped[KeyNoRequestId], item)

			continue
		}

		grouped[item.RequestID] = append(grouped[item.RequestID], item)
	}

	return grouped
}

func DetectFailedRequests(correlatedRequests map[string][]LogEntry) []string {
	if len(correlatedRequests) == 0 {
		return nil
	}

	keys := make([]string, 0, len(correlatedRequests))

	for key := range correlatedRequests {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	var failedIDs []string

	for _, key := range keys {
		if key == "no request_id" {
			continue
		}

		for _, entry := range correlatedRequests[key] {
			if entry.Level == "WARN" || entry.Level == "ERROR" {
				failedIDs = append(failedIDs, key)
				break
			}
		}
	}

	return failedIDs
}

func FindFirstFailure(requestEntries []LogEntry) (LogEntry, bool) {
	requestEntriesCopy := make([]LogEntry, len(requestEntries))
	copy(requestEntriesCopy, requestEntries)

	sort.Slice(requestEntriesCopy, func(i, j int) bool {
		return requestEntriesCopy[i].Timestamp.Before(requestEntriesCopy[j].Timestamp)
	})

	for i := range requestEntriesCopy {
		if requestEntriesCopy[i].Level == "WARN" || requestEntriesCopy[i].Level == "ERROR" {
			return requestEntriesCopy[i], true
		}
	}

	return LogEntry{}, false
}

func SortTimelineByTimestamp(entries []LogEntry) []LogEntry {
	entriesCopy := make([]LogEntry, len(entries))
	copy(entriesCopy, entries)

	sort.Slice(entriesCopy, func(i, j int) bool {
		return entriesCopy[i].Timestamp.Before(entriesCopy[j].Timestamp)
	})

	return entriesCopy
}

func WriteJSONReport(result AnalysisResult, filename string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal result json: %w", err)
	}

	if err = os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("write result file: %w", err)
	}

	return nil
}

func main() {
	dirPath := "./logs"

	start := time.Now()

	fmt.Println("Scanning directory:", dirPath)

	logFiles, err := ScanLogDirectory(dirPath)
	if err != nil {
		fmt.Println("error scanning directory:", err)
		return
	}

	fmt.Printf("Found %d log files:\n", len(logFiles))
	for _, file := range logFiles {
		fmt.Println(" -", file)
	}

	if len(logFiles) == 0 {
		fmt.Println("No log files found")
		return
	}

	fmt.Println("\nProcessing files...")

	logEntries, err := ProcessMultipleFiles(logFiles)
	if err != nil {
		fmt.Println("error processing files:", err)
		return
	}

	groupedEntries := CorrelateRequests(logEntries)
	failedRequestIDs := DetectFailedRequests(groupedEntries)

	totalRequests := len(groupedEntries)
	if _, ok := groupedEntries["no request_id"]; ok {
		totalRequests--
	}

	failedReports := make([]FailedRequestReport, 0, len(failedRequestIDs))

	for _, requestID := range failedRequestIDs {
		entries := groupedEntries[requestID]
		if len(entries) == 0 {
			continue
		}

		firstFailure, found := FindFirstFailure(entries)
		if !found {
			continue
		}

		timelineEntries := SortTimelineByTimestamp(entries)
		timeline := make([]string, 0, len(timelineEntries))

		for _, entry := range timelineEntries {
			timeline = append(timeline, fmt.Sprintf(
				"%s [%s] %s: %s",
				entry.Timestamp.Format("15:04:05.000Z07:00"),
				entry.Level,
				entry.Service,
				entry.Message,
			))
		}

		failedReports = append(failedReports, FailedRequestReport{
			RequestID:      requestID,
			FailingService: firstFailure.Service,
			ErrorMessage:   firstFailure.Message,
			Timeline:       timeline,
		})
	}

	result := AnalysisResult{
		TotalEntriesProcessed: len(logEntries),
		FailedRequestsFound:   len(failedRequestIDs),
		ProcessingTimeSeconds: time.Since(start).Seconds(),
		FailedRequests:        failedReports,
	}

	if err = WriteJSONReport(result, "analysis-result.json"); err != nil {
		fmt.Println("error writing report:", err)
		return
	}
}

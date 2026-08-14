package processor

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"sort"
	"the-log-handler/internal/parser"
)

func ReadLogFile(filepath string) ([]parser.LogEntry, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("error file open: %w", err)
	}
	defer file.Close()

	var entries []parser.LogEntry

	totalLines := 0
	parseErrors := 0

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		totalLines++

		line := scanner.Text()
		entry, err := parser.ParseLogLine(line)
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

func ProcessMultipleFiles(filePaths []string) ([]parser.LogEntry, error) {
	if len(filePaths) == 0 {
		return nil, errors.New("file paths is empty")
	}

	var allEntries []parser.LogEntry

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

func CorrelateRequests(entries []parser.LogEntry) map[string][]parser.LogEntry {
	grouped := make(map[string][]parser.LogEntry)
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

func DetectFailedRequests(correlatedRequests map[string][]parser.LogEntry) []string {
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

func FindFirstFailure(requestEntries []parser.LogEntry) (parser.LogEntry, bool) {
	requestEntriesCopy := make([]parser.LogEntry, len(requestEntries))
	copy(requestEntriesCopy, requestEntries)

	sort.Slice(requestEntriesCopy, func(i, j int) bool {
		return requestEntriesCopy[i].Timestamp.Before(requestEntriesCopy[j].Timestamp)
	})

	for i := range requestEntriesCopy {
		if requestEntriesCopy[i].Level == "WARN" || requestEntriesCopy[i].Level == "ERROR" {
			return requestEntriesCopy[i], true
		}
	}

	return parser.LogEntry{}, false
}

func SortTimelineByTimestamp(entries []parser.LogEntry) []parser.LogEntry {
	entriesCopy := make([]parser.LogEntry, len(entries))
	copy(entriesCopy, entries)

	sort.Slice(entriesCopy, func(i, j int) bool {
		return entriesCopy[i].Timestamp.Before(entriesCopy[j].Timestamp)
	})

	return entriesCopy
}

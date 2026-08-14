package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"the-log-handler/internal/cli"
	"the-log-handler/internal/processor"
	"the-log-handler/internal/reporter"
	"the-log-handler/internal/scanner"
)

const numWorkers = 4

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	go func() {
		<-sigChan

		fmt.Println("\nReceived interrupt signal, shutting down gracefully...")
		cancel()
	}()

	args, err := cli.ParseCommandLineArgs()
	if err != nil {
		fmt.Println("error get args:", err)
		return
	}

	start := time.Now()

	dirPath := args.InputDir

	fmt.Println("Scanning directory:", dirPath)

	logFiles, err := scanner.ScanLogDirectory(dirPath)
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

	logEntries, err := processor.ProcessFilesConcurrently(ctx, logFiles, numWorkers)
	if err != nil {
		fmt.Println("error processing files concurrently:", err)
		return
	}

	if ctx.Err() != nil {
		fmt.Println("Received interrupt signal, shutting down gracefully...")
		fmt.Printf(
			"Processed entries from interrupted processing: %d\n",
			len(logEntries),
		)
		fmt.Println("Saving partial results...")
	} else {
		fmt.Println("Processing completed successfully.")
	}

	groupedEntries := processor.CorrelateRequests(logEntries)
	failedRequestIDs := processor.DetectFailedRequests(groupedEntries)

	totalRequests := len(groupedEntries)
	if _, ok := groupedEntries["no request_id"]; ok {
		totalRequests--
	}

	failedReports := make([]reporter.FailedRequestReport, 0, len(failedRequestIDs))

	for _, requestID := range failedRequestIDs {
		entries := groupedEntries[requestID]
		if len(entries) == 0 {
			continue
		}

		firstFailure, found := processor.FindFirstFailure(entries)
		if !found {
			continue
		}

		timelineEntries := processor.SortTimelineByTimestamp(entries)
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

		failedReports = append(failedReports, reporter.FailedRequestReport{
			RequestID:      requestID,
			FailingService: firstFailure.Service,
			ErrorMessage:   firstFailure.Message,
			Timeline:       timeline,
		})
	}

	result := reporter.AnalysisResult{
		TotalEntriesProcessed: len(logEntries),
		FailedRequestsFound:   len(failedRequestIDs),
		ProcessingTimeSeconds: time.Since(start).Seconds(),
		FailedRequests:        failedReports,
	}

	if err = reporter.WriteJSONReport(result, args.OutputFile); err != nil {
		fmt.Println("error writing report:", err)
		return
	}
}

package reporter

import (
	"encoding/json"
	"fmt"
	"os"
)

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

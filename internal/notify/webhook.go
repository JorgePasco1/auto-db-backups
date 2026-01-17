package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type WebhookPayload struct {
	Status        string    `json:"status"`
	DatabaseType  string    `json:"database_type"`
	DatabaseName  string    `json:"database_name"`
	BackupKey     string    `json:"backup_key,omitempty"`
	BackupSize    int64     `json:"backup_size,omitempty"`
	Compressed    bool      `json:"compressed"`
	Encrypted     bool      `json:"encrypted"`
	Duration      string    `json:"duration"`
	Error         string    `json:"error,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
	Repository    string    `json:"repository,omitempty"`
	RunID         string    `json:"run_id,omitempty"`
	RunURL        string    `json:"run_url,omitempty"`
}

type WebhookNotifier struct {
	url     string
	client  *http.Client
}

func NewWebhookNotifier(url string) *WebhookNotifier {
	return &WebhookNotifier{
		url: url,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (n *WebhookNotifier) Notify(ctx context.Context, summary *BackupSummary) error {
	if n.url == "" {
		return nil
	}

	payload := buildWebhookPayload(summary)

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "auto-db-backups/1.0")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-success status: %d", resp.StatusCode)
	}

	return nil
}

func buildWebhookPayload(summary *BackupSummary) *WebhookPayload {
	payload := &WebhookPayload{
		DatabaseType: summary.DatabaseType,
		DatabaseName: summary.DatabaseName,
		Compressed:   summary.Compressed,
		Encrypted:    summary.Encrypted,
		Duration:     summary.Duration.String(),
		Timestamp:    time.Now().UTC(),
	}

	if summary.Success {
		payload.Status = "success"
		payload.BackupKey = summary.BackupKey
		payload.BackupSize = summary.BackupSize
	} else {
		payload.Status = "failure"
		if summary.Error != nil {
			payload.Error = summary.Error.Error()
		}
	}

	// Add GitHub context if available
	if repo := os.Getenv("GITHUB_REPOSITORY"); repo != "" {
		payload.Repository = repo
	}
	if runID := os.Getenv("GITHUB_RUN_ID"); runID != "" {
		payload.RunID = runID
		if serverURL := os.Getenv("GITHUB_SERVER_URL"); serverURL != "" {
			if repo := os.Getenv("GITHUB_REPOSITORY"); repo != "" {
				payload.RunURL = fmt.Sprintf("%s/%s/actions/runs/%s", serverURL, repo, runID)
			}
		}
	}

	return payload
}

package notify

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for BackupSummary struct
func TestBackupSummary_Fields(t *testing.T) {
	t.Parallel()

	summary := BackupSummary{
		DatabaseType:   "postgres",
		DatabaseName:   "mydb",
		BackupKey:      "backups/mydb-2024.dump.gz",
		BackupSize:     1024 * 1024,
		Compressed:     true,
		Encrypted:      true,
		Duration:       5 * time.Minute,
		Success:        true,
		Error:          nil,
		DeletedBackups: 3,
	}

	assert.Equal(t, "postgres", summary.DatabaseType)
	assert.Equal(t, "mydb", summary.DatabaseName)
	assert.Equal(t, "backups/mydb-2024.dump.gz", summary.BackupKey)
	assert.Equal(t, int64(1024*1024), summary.BackupSize)
	assert.True(t, summary.Compressed)
	assert.True(t, summary.Encrypted)
	assert.Equal(t, 5*time.Minute, summary.Duration)
	assert.True(t, summary.Success)
	assert.Nil(t, summary.Error)
	assert.Equal(t, 3, summary.DeletedBackups)
}

// Tests for formatBytes
func TestFormatBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"zero bytes", 0, "0 B"},
		{"small bytes", 512, "512 B"},
		{"exactly 1KB", 1024, "1.0 KB"},
		{"KB range", 1536, "1.5 KB"},
		{"exactly 1MB", 1024 * 1024, "1.0 MB"},
		{"MB range", 5 * 1024 * 1024, "5.0 MB"},
		{"exactly 1GB", 1024 * 1024 * 1024, "1.0 GB"},
		{"GB range", int64(2.5 * 1024 * 1024 * 1024), "2.5 GB"},
		{"large GB", 100 * int64(1024*1024*1024), "100.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := formatBytes(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Tests for boolToEmoji
func TestBoolToEmoji(t *testing.T) {
	t.Parallel()

	assert.Equal(t, ":white_check_mark:", boolToEmoji(true))
	assert.Equal(t, ":x:", boolToEmoji(false))
}

// Tests for buildSummaryMarkdown
func TestBuildSummaryMarkdown_Success(t *testing.T) {
	t.Parallel()

	summary := &BackupSummary{
		DatabaseType:   "postgres",
		DatabaseName:   "production",
		BackupKey:      "backups/prod-2024.dump.gz",
		BackupSize:     10 * 1024 * 1024,
		Compressed:     true,
		Encrypted:      true,
		Duration:       2 * time.Minute,
		Success:        true,
		DeletedBackups: 5,
	}

	markdown := buildSummaryMarkdown(summary)

	assert.Contains(t, markdown, "## Database Backup Summary")
	assert.Contains(t, markdown, ":white_check_mark: Success")
	assert.Contains(t, markdown, "| Database Type | postgres |")
	assert.Contains(t, markdown, "| Database Name | production |")
	assert.Contains(t, markdown, "| Backup Key | `backups/prod-2024.dump.gz` |")
	assert.Contains(t, markdown, "| Backup Size | 10.0 MB |")
	assert.Contains(t, markdown, "| Compressed | :white_check_mark: |")
	assert.Contains(t, markdown, "| Encrypted | :white_check_mark: |")
	assert.Contains(t, markdown, "| Duration |")
	assert.Contains(t, markdown, "| Old Backups Deleted | 5 |")
}

func TestBuildSummaryMarkdown_Failure(t *testing.T) {
	t.Parallel()

	summary := &BackupSummary{
		DatabaseType: "mysql",
		DatabaseName: "users",
		Success:      false,
		Error:        errors.New("connection refused"),
	}

	markdown := buildSummaryMarkdown(summary)

	assert.Contains(t, markdown, "## Database Backup Summary")
	assert.Contains(t, markdown, ":x: Failed")
	assert.Contains(t, markdown, "| Database Type | mysql |")
	assert.Contains(t, markdown, "| Database Name | users |")
	assert.Contains(t, markdown, "| Error | connection refused |")
	assert.NotContains(t, markdown, "Backup Key")
	assert.NotContains(t, markdown, "Backup Size")
}

func TestBuildSummaryMarkdown_NoDeletedBackups(t *testing.T) {
	t.Parallel()

	summary := &BackupSummary{
		DatabaseType:   "postgres",
		DatabaseName:   "test",
		BackupKey:      "test.dump.gz",
		BackupSize:     1024,
		Success:        true,
		DeletedBackups: 0,
	}

	markdown := buildSummaryMarkdown(summary)

	assert.NotContains(t, markdown, "Old Backups Deleted")
}

func TestBuildSummaryMarkdown_NoCompressionOrEncryption(t *testing.T) {
	t.Parallel()

	summary := &BackupSummary{
		DatabaseType: "mongodb",
		DatabaseName: "analytics",
		BackupKey:    "analytics.tar",
		BackupSize:   2048,
		Compressed:   false,
		Encrypted:    false,
		Success:      true,
	}

	markdown := buildSummaryMarkdown(summary)

	assert.Contains(t, markdown, "| Compressed | :x: |")
	assert.Contains(t, markdown, "| Encrypted | :x: |")
}

// Tests for WriteGitHubSummary
func TestWriteGitHubSummary_NotInGitHubActions(t *testing.T) {
	// When GITHUB_STEP_SUMMARY is not set, should return nil without writing
	summary := &BackupSummary{
		DatabaseType: "postgres",
		DatabaseName: "test",
		Success:      true,
	}

	err := WriteGitHubSummary(summary)
	assert.NoError(t, err)
}

func TestWriteGitHubSummary_InGitHubActions(t *testing.T) {
	// Create a temp file to simulate GITHUB_STEP_SUMMARY
	tempDir := t.TempDir()
	summaryFile := filepath.Join(tempDir, "summary.md")

	t.Setenv("GITHUB_STEP_SUMMARY", summaryFile)

	summary := &BackupSummary{
		DatabaseType: "postgres",
		DatabaseName: "testdb",
		BackupKey:    "test.dump.gz",
		BackupSize:   1024,
		Success:      true,
	}

	err := WriteGitHubSummary(summary)
	require.NoError(t, err)

	// Verify file was created and contains expected content
	content, err := os.ReadFile(summaryFile)
	require.NoError(t, err)

	assert.Contains(t, string(content), "## Database Backup Summary")
	assert.Contains(t, string(content), "postgres")
	assert.Contains(t, string(content), "testdb")
}

func TestWriteGitHubSummary_AppendsToExisting(t *testing.T) {
	tempDir := t.TempDir()
	summaryFile := filepath.Join(tempDir, "summary.md")

	// Write initial content
	err := os.WriteFile(summaryFile, []byte("# Existing Content\n"), 0644)
	require.NoError(t, err)

	t.Setenv("GITHUB_STEP_SUMMARY", summaryFile)

	summary := &BackupSummary{
		DatabaseType: "postgres",
		DatabaseName: "testdb",
		Success:      true,
	}

	err = WriteGitHubSummary(summary)
	require.NoError(t, err)

	content, err := os.ReadFile(summaryFile)
	require.NoError(t, err)

	// Should contain both original and new content
	assert.Contains(t, string(content), "# Existing Content")
	assert.Contains(t, string(content), "## Database Backup Summary")
}

// Tests for SetGitHubOutput
func TestSetGitHubOutput_NotInGitHubActions(t *testing.T) {
	// When GITHUB_OUTPUT is not set, should return nil
	err := SetGitHubOutput("test_key", "test_value")
	assert.NoError(t, err)
}

func TestSetGitHubOutput_InGitHubActions(t *testing.T) {
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "output.txt")

	t.Setenv("GITHUB_OUTPUT", outputFile)

	err := SetGitHubOutput("backup_key", "backups/test.dump.gz")
	require.NoError(t, err)

	err = SetGitHubOutput("backup_size", "1024")
	require.NoError(t, err)

	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	assert.Contains(t, string(content), "backup_key=backups/test.dump.gz")
	assert.Contains(t, string(content), "backup_size=1024")
}

// Tests for WebhookPayload struct
func TestWebhookPayload_Fields(t *testing.T) {
	t.Parallel()

	now := time.Now()
	payload := WebhookPayload{
		Status:       "success",
		DatabaseType: "postgres",
		DatabaseName: "mydb",
		BackupKey:    "test.dump.gz",
		BackupSize:   1024,
		Compressed:   true,
		Encrypted:    true,
		Duration:     "5m0s",
		Error:        "",
		Timestamp:    now,
		Repository:   "owner/repo",
		RunID:        "12345",
		RunURL:       "https://github.com/owner/repo/actions/runs/12345",
	}

	assert.Equal(t, "success", payload.Status)
	assert.Equal(t, "postgres", payload.DatabaseType)
	assert.Equal(t, "mydb", payload.DatabaseName)
	assert.Equal(t, "test.dump.gz", payload.BackupKey)
	assert.Equal(t, int64(1024), payload.BackupSize)
	assert.True(t, payload.Compressed)
	assert.True(t, payload.Encrypted)
	assert.Equal(t, "5m0s", payload.Duration)
	assert.Empty(t, payload.Error)
	assert.Equal(t, now, payload.Timestamp)
	assert.Equal(t, "owner/repo", payload.Repository)
	assert.Equal(t, "12345", payload.RunID)
	assert.Equal(t, "https://github.com/owner/repo/actions/runs/12345", payload.RunURL)
}

// Tests for NewWebhookNotifier
func TestNewWebhookNotifier(t *testing.T) {
	t.Parallel()

	notifier := NewWebhookNotifier("https://hooks.example.com/webhook")

	require.NotNil(t, notifier)
	assert.Equal(t, "https://hooks.example.com/webhook", notifier.url)
	require.NotNil(t, notifier.client)
	assert.Equal(t, 30*time.Second, notifier.client.Timeout)
}

// Tests for buildWebhookPayload
func TestBuildWebhookPayload_Success(t *testing.T) {
	t.Parallel()

	summary := &BackupSummary{
		DatabaseType: "postgres",
		DatabaseName: "proddb",
		BackupKey:    "backups/prod.dump.gz",
		BackupSize:   5 * 1024 * 1024,
		Compressed:   true,
		Encrypted:    false,
		Duration:     3 * time.Minute,
		Success:      true,
	}

	payload := buildWebhookPayload(summary)

	assert.Equal(t, "success", payload.Status)
	assert.Equal(t, "postgres", payload.DatabaseType)
	assert.Equal(t, "proddb", payload.DatabaseName)
	assert.Equal(t, "backups/prod.dump.gz", payload.BackupKey)
	assert.Equal(t, int64(5*1024*1024), payload.BackupSize)
	assert.True(t, payload.Compressed)
	assert.False(t, payload.Encrypted)
	assert.Equal(t, "3m0s", payload.Duration)
	assert.Empty(t, payload.Error)
	assert.False(t, payload.Timestamp.IsZero())
}

func TestBuildWebhookPayload_Failure(t *testing.T) {
	t.Parallel()

	summary := &BackupSummary{
		DatabaseType: "mysql",
		DatabaseName: "users",
		Compressed:   true,
		Encrypted:    true,
		Duration:     30 * time.Second,
		Success:      false,
		Error:        errors.New("connection timeout"),
	}

	payload := buildWebhookPayload(summary)

	assert.Equal(t, "failure", payload.Status)
	assert.Equal(t, "mysql", payload.DatabaseType)
	assert.Equal(t, "users", payload.DatabaseName)
	assert.Empty(t, payload.BackupKey)
	assert.Equal(t, int64(0), payload.BackupSize)
	assert.True(t, payload.Compressed)
	assert.True(t, payload.Encrypted)
	assert.Equal(t, "connection timeout", payload.Error)
}

func TestBuildWebhookPayload_WithGitHubContext(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")
	t.Setenv("GITHUB_RUN_ID", "12345")
	t.Setenv("GITHUB_SERVER_URL", "https://github.com")

	summary := &BackupSummary{
		DatabaseType: "postgres",
		DatabaseName: "test",
		Success:      true,
	}

	payload := buildWebhookPayload(summary)

	assert.Equal(t, "owner/repo", payload.Repository)
	assert.Equal(t, "12345", payload.RunID)
	assert.Equal(t, "https://github.com/owner/repo/actions/runs/12345", payload.RunURL)
}

func TestBuildWebhookPayload_WithoutGitHubContext(t *testing.T) {
	t.Parallel()

	summary := &BackupSummary{
		DatabaseType: "postgres",
		DatabaseName: "test",
		Success:      true,
	}

	payload := buildWebhookPayload(summary)

	assert.Empty(t, payload.Repository)
	assert.Empty(t, payload.RunID)
	assert.Empty(t, payload.RunURL)
}

// Tests for WebhookNotifier.Notify
func TestWebhookNotifier_Notify_EmptyURL(t *testing.T) {
	t.Parallel()

	notifier := NewWebhookNotifier("")
	summary := &BackupSummary{
		DatabaseType: "postgres",
		DatabaseName: "test",
		Success:      true,
	}

	err := notifier.Notify(context.Background(), summary)
	assert.NoError(t, err)
}

func TestWebhookNotifier_Notify_Success(t *testing.T) {
	t.Parallel()

	var receivedPayload WebhookPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "auto-db-backups/1.0", r.Header.Get("User-Agent"))

		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		assert.NoError(t, err)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewWebhookNotifier(server.URL)
	summary := &BackupSummary{
		DatabaseType: "postgres",
		DatabaseName: "testdb",
		BackupKey:    "test.dump.gz",
		BackupSize:   2048,
		Compressed:   true,
		Encrypted:    false,
		Duration:     time.Minute,
		Success:      true,
	}

	err := notifier.Notify(context.Background(), summary)
	require.NoError(t, err)

	assert.Equal(t, "success", receivedPayload.Status)
	assert.Equal(t, "postgres", receivedPayload.DatabaseType)
	assert.Equal(t, "testdb", receivedPayload.DatabaseName)
	assert.Equal(t, "test.dump.gz", receivedPayload.BackupKey)
	assert.Equal(t, int64(2048), receivedPayload.BackupSize)
	assert.True(t, receivedPayload.Compressed)
	assert.False(t, receivedPayload.Encrypted)
}

func TestWebhookNotifier_Notify_NonSuccessStatus(t *testing.T) {
	t.Parallel()

	tests := []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusInternalServerError,
		http.StatusServiceUnavailable,
	}

	for _, statusCode := range tests {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(statusCode)
			}))
			defer server.Close()

			notifier := NewWebhookNotifier(server.URL)
			summary := &BackupSummary{
				DatabaseType: "postgres",
				DatabaseName: "test",
				Success:      true,
			}

			err := notifier.Notify(context.Background(), summary)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "non-success status")
		})
	}
}

func TestWebhookNotifier_Notify_AcceptableSuccessStatuses(t *testing.T) {
	t.Parallel()

	tests := []int{
		http.StatusOK,
		http.StatusCreated,
		http.StatusAccepted,
		http.StatusNoContent,
	}

	for _, statusCode := range tests {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(statusCode)
			}))
			defer server.Close()

			notifier := NewWebhookNotifier(server.URL)
			summary := &BackupSummary{
				DatabaseType: "postgres",
				DatabaseName: "test",
				Success:      true,
			}

			err := notifier.Notify(context.Background(), summary)
			assert.NoError(t, err)
		})
	}
}

func TestWebhookNotifier_Notify_NetworkError(t *testing.T) {
	t.Parallel()

	// Use an invalid URL that will fail to connect
	notifier := NewWebhookNotifier("http://localhost:1")
	summary := &BackupSummary{
		DatabaseType: "postgres",
		DatabaseName: "test",
		Success:      true,
	}

	err := notifier.Notify(context.Background(), summary)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send webhook")
}

func TestWebhookNotifier_Notify_ContextCancellation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Slow response to allow context cancellation
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewWebhookNotifier(server.URL)
	summary := &BackupSummary{
		DatabaseType: "postgres",
		DatabaseName: "test",
		Success:      true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := notifier.Notify(ctx, summary)
	assert.Error(t, err)
}

func TestWebhookNotifier_Notify_FailureSummary(t *testing.T) {
	t.Parallel()

	var receivedPayload WebhookPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewWebhookNotifier(server.URL)
	summary := &BackupSummary{
		DatabaseType: "mysql",
		DatabaseName: "users",
		Success:      false,
		Error:        errors.New("database connection failed"),
	}

	err := notifier.Notify(context.Background(), summary)
	require.NoError(t, err)

	assert.Equal(t, "failure", receivedPayload.Status)
	assert.Equal(t, "database connection failed", receivedPayload.Error)
	assert.Empty(t, receivedPayload.BackupKey)
}

// Test JSON serialization of WebhookPayload
func TestWebhookPayload_JSONSerialization(t *testing.T) {
	t.Parallel()

	payload := WebhookPayload{
		Status:       "success",
		DatabaseType: "postgres",
		DatabaseName: "test",
		BackupKey:    "test.dump.gz",
		BackupSize:   1024,
		Compressed:   true,
		Encrypted:    false,
		Duration:     "1m0s",
		Timestamp:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	jsonBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	jsonStr := string(jsonBytes)
	assert.Contains(t, jsonStr, `"status":"success"`)
	assert.Contains(t, jsonStr, `"database_type":"postgres"`)
	assert.Contains(t, jsonStr, `"database_name":"test"`)
	assert.Contains(t, jsonStr, `"backup_key":"test.dump.gz"`)
	assert.Contains(t, jsonStr, `"backup_size":1024`)
	assert.Contains(t, jsonStr, `"compressed":true`)
	assert.Contains(t, jsonStr, `"encrypted":false`)
	assert.Contains(t, jsonStr, `"duration":"1m0s"`)
}

func TestWebhookPayload_JSONOmitempty(t *testing.T) {
	t.Parallel()

	// Test that omitempty fields are actually omitted
	payload := WebhookPayload{
		Status:       "failure",
		DatabaseType: "postgres",
		DatabaseName: "test",
		Compressed:   false,
		Encrypted:    false,
		Duration:     "30s",
		Error:        "some error",
		Timestamp:    time.Now(),
	}

	jsonBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	jsonStr := string(jsonBytes)
	// BackupKey and BackupSize should be omitted when empty/zero
	assert.NotContains(t, jsonStr, `"backup_key"`)
	// Note: BackupSize with value 0 is not omitted due to omitempty behavior with 0
	assert.Contains(t, jsonStr, `"error":"some error"`)
}

// Edge case tests
func TestFormatBytes_NegativeValue(t *testing.T) {
	t.Parallel()

	// This tests boundary condition - negative values are unusual but shouldn't crash
	result := formatBytes(-1)
	// The function doesn't handle negative values specially, but it shouldn't panic
	assert.NotEmpty(t, result)
}

func TestBuildSummaryMarkdown_LongDuration(t *testing.T) {
	t.Parallel()

	summary := &BackupSummary{
		DatabaseType: "postgres",
		DatabaseName: "huge_database",
		BackupKey:    "huge.dump.gz",
		BackupSize:   100 * 1024 * 1024 * 1024, // 100GB
		Duration:     2 * time.Hour,
		Success:      true,
	}

	markdown := buildSummaryMarkdown(summary)

	assert.Contains(t, markdown, "100.0 GB")
	assert.Contains(t, markdown, "| Duration |")
}

func TestBuildSummaryMarkdown_SpecialCharactersInDatabaseName(t *testing.T) {
	t.Parallel()

	summary := &BackupSummary{
		DatabaseType: "postgres",
		DatabaseName: "my-db_test.prod",
		BackupKey:    "backup.dump",
		BackupSize:   1024,
		Success:      true,
	}

	markdown := buildSummaryMarkdown(summary)

	assert.Contains(t, markdown, "my-db_test.prod")
}

func TestWriteGitHubSummary_InvalidPath(t *testing.T) {
	// Set an invalid path that can't be written to
	t.Setenv("GITHUB_STEP_SUMMARY", "/nonexistent/path/that/doesnt/exist/summary.md")

	summary := &BackupSummary{
		DatabaseType: "postgres",
		DatabaseName: "test",
		Success:      true,
	}

	err := WriteGitHubSummary(summary)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "failed to open summary file") ||
		strings.Contains(err.Error(), "no such file"))
}

func TestSetGitHubOutput_InvalidPath(t *testing.T) {
	t.Setenv("GITHUB_OUTPUT", "/nonexistent/path/output.txt")

	err := SetGitHubOutput("key", "value")
	assert.Error(t, err)
}

package notify

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type BackupSummary struct {
	DatabaseType   string
	DatabaseName   string
	BackupKey      string
	BackupSize     int64
	Compressed     bool
	Encrypted      bool
	Duration       time.Duration
	Success        bool
	Error          error
	DeletedBackups int
}

func WriteGitHubSummary(summary *BackupSummary) error {
	summaryFile := os.Getenv("GITHUB_STEP_SUMMARY")
	if summaryFile == "" {
		return nil // Not running in GitHub Actions
	}

	content := buildSummaryMarkdown(summary)

	f, err := os.OpenFile(summaryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open summary file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("failed to write summary: %w", err)
	}

	return nil
}

func buildSummaryMarkdown(summary *BackupSummary) string {
	var sb strings.Builder

	sb.WriteString("## Database Backup Summary\n\n")

	if summary.Success {
		sb.WriteString("**Status:** :white_check_mark: Success\n\n")
	} else {
		sb.WriteString("**Status:** :x: Failed\n\n")
	}

	sb.WriteString("| Property | Value |\n")
	sb.WriteString("|----------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Database Type | %s |\n", summary.DatabaseType))
	sb.WriteString(fmt.Sprintf("| Database Name | %s |\n", summary.DatabaseName))

	if summary.Success {
		sb.WriteString(fmt.Sprintf("| Backup Key | `%s` |\n", summary.BackupKey))
		sb.WriteString(fmt.Sprintf("| Backup Size | %s |\n", formatBytes(summary.BackupSize)))
		sb.WriteString(fmt.Sprintf("| Compressed | %s |\n", boolToEmoji(summary.Compressed)))
		sb.WriteString(fmt.Sprintf("| Encrypted | %s |\n", boolToEmoji(summary.Encrypted)))
		sb.WriteString(fmt.Sprintf("| Duration | %s |\n", summary.Duration.Round(time.Millisecond)))

		if summary.DeletedBackups > 0 {
			sb.WriteString(fmt.Sprintf("| Old Backups Deleted | %d |\n", summary.DeletedBackups))
		}
	} else {
		sb.WriteString(fmt.Sprintf("| Error | %s |\n", summary.Error.Error()))
	}

	sb.WriteString("\n")

	return sb.String()
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func boolToEmoji(b bool) string {
	if b {
		return ":white_check_mark:"
	}
	return ":x:"
}

func SetGitHubOutput(name, value string) error {
	outputFile := os.Getenv("GITHUB_OUTPUT")
	if outputFile == "" {
		return nil
	}

	f, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open output file: %w", err)
	}
	defer f.Close()

	if _, err := fmt.Fprintf(f, "%s=%s\n", name, value); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	return nil
}

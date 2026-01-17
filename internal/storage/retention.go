package storage

import (
	"context"
	"fmt"
	"log"
	"time"
)

type RetentionPolicy struct {
	Days  int
	Count int
}

type RetentionResult struct {
	DeletedCount int
	DeletedKeys  []string
	Errors       []error
}

func (p *RetentionPolicy) IsEnabled() bool {
	return p.Days > 0 || p.Count > 0
}

func ApplyRetention(ctx context.Context, client *R2Client, policy RetentionPolicy) (*RetentionResult, error) {
	if !policy.IsEnabled() {
		return &RetentionResult{}, nil
	}

	backups, err := client.ListBackups(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list backups: %w", err)
	}

	toDelete := determineBackupsToDelete(backups, policy)

	result := &RetentionResult{
		DeletedKeys: make([]string, 0, len(toDelete)),
	}

	for _, backup := range toDelete {
		if err := client.Delete(ctx, backup.Key); err != nil {
			result.Errors = append(result.Errors, err)
			log.Printf("Failed to delete backup %s: %v", backup.Key, err)
		} else {
			result.DeletedCount++
			result.DeletedKeys = append(result.DeletedKeys, backup.Key)
			log.Printf("Deleted old backup: %s", backup.Key)
		}
	}

	return result, nil
}

func determineBackupsToDelete(backups []BackupObject, policy RetentionPolicy) []BackupObject {
	var toDelete []BackupObject
	now := time.Now()

	// Track which backups to keep
	keep := make(map[string]bool)

	// If count policy is set, keep the N most recent
	if policy.Count > 0 && len(backups) > policy.Count {
		// backups are already sorted newest first
		for i := 0; i < policy.Count && i < len(backups); i++ {
			keep[backups[i].Key] = true
		}
	} else if policy.Count > 0 {
		// Keep all if we have fewer than count
		for _, b := range backups {
			keep[b.Key] = true
		}
	}

	// Check each backup
	for _, backup := range backups {
		shouldDelete := false

		// Check age policy
		if policy.Days > 0 {
			age := now.Sub(backup.LastModified)
			maxAge := time.Duration(policy.Days) * 24 * time.Hour
			if age > maxAge {
				shouldDelete = true
			}
		}

		// Check count policy - if not in keep set
		if policy.Count > 0 && !keep[backup.Key] {
			shouldDelete = true
		}

		// Only delete if at least one policy says so
		// (and count policy didn't explicitly keep it)
		if shouldDelete && (policy.Count == 0 || !keep[backup.Key]) {
			toDelete = append(toDelete, backup)
		}
	}

	return toDelete
}

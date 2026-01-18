package storage

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for RetentionPolicy
func TestRetentionPolicy_IsEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		days     int
		count    int
		expected bool
	}{
		{"no retention", 0, 0, false},
		{"days only", 30, 0, true},
		{"count only", 0, 10, true},
		{"both set", 30, 10, true},
		{"negative days", -1, 0, false},
		{"negative count", 0, -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			policy := RetentionPolicy{Days: tt.days, Count: tt.count}
			assert.Equal(t, tt.expected, policy.IsEnabled())
		})
	}
}

// Tests for determineBackupsToDelete
func TestDetermineBackupsToDelete_NoPolicy(t *testing.T) {
	t.Parallel()

	now := time.Now()
	backups := []BackupObject{
		{Key: "backup1", LastModified: now.Add(-24 * time.Hour)},
		{Key: "backup2", LastModified: now.Add(-48 * time.Hour)},
	}

	policy := RetentionPolicy{Days: 0, Count: 0}
	toDelete := determineBackupsToDelete(backups, policy)

	assert.Empty(t, toDelete, "No backups should be deleted when policy is disabled")
}

func TestDetermineBackupsToDelete_DaysPolicy_AllFresh(t *testing.T) {
	t.Parallel()

	now := time.Now()
	backups := []BackupObject{
		{Key: "backup1", LastModified: now.Add(-1 * time.Hour)},
		{Key: "backup2", LastModified: now.Add(-2 * time.Hour)},
		{Key: "backup3", LastModified: now.Add(-12 * time.Hour)},
	}

	policy := RetentionPolicy{Days: 7, Count: 0}
	toDelete := determineBackupsToDelete(backups, policy)

	assert.Empty(t, toDelete, "No backups should be deleted when all are within retention period")
}

func TestDetermineBackupsToDelete_DaysPolicy_SomeOld(t *testing.T) {
	t.Parallel()

	now := time.Now()
	backups := []BackupObject{
		{Key: "fresh1", LastModified: now.Add(-24 * time.Hour)},
		{Key: "fresh2", LastModified: now.Add(-48 * time.Hour)},
		{Key: "old1", LastModified: now.Add(-8 * 24 * time.Hour)},  // 8 days old
		{Key: "old2", LastModified: now.Add(-10 * 24 * time.Hour)}, // 10 days old
	}

	policy := RetentionPolicy{Days: 7, Count: 0}
	toDelete := determineBackupsToDelete(backups, policy)

	assert.Len(t, toDelete, 2)
	keys := make([]string, len(toDelete))
	for i, b := range toDelete {
		keys[i] = b.Key
	}
	assert.Contains(t, keys, "old1")
	assert.Contains(t, keys, "old2")
}

func TestDetermineBackupsToDelete_DaysPolicy_AllOld(t *testing.T) {
	t.Parallel()

	now := time.Now()
	backups := []BackupObject{
		{Key: "old1", LastModified: now.Add(-8 * 24 * time.Hour)},
		{Key: "old2", LastModified: now.Add(-10 * 24 * time.Hour)},
		{Key: "old3", LastModified: now.Add(-30 * 24 * time.Hour)},
	}

	policy := RetentionPolicy{Days: 7, Count: 0}
	toDelete := determineBackupsToDelete(backups, policy)

	assert.Len(t, toDelete, 3, "All old backups should be marked for deletion")
}

func TestDetermineBackupsToDelete_CountPolicy_BelowLimit(t *testing.T) {
	t.Parallel()

	now := time.Now()
	backups := []BackupObject{
		{Key: "backup1", LastModified: now.Add(-1 * time.Hour)},
		{Key: "backup2", LastModified: now.Add(-2 * time.Hour)},
	}

	policy := RetentionPolicy{Days: 0, Count: 5}
	toDelete := determineBackupsToDelete(backups, policy)

	assert.Empty(t, toDelete, "No backups should be deleted when count is below limit")
}

func TestDetermineBackupsToDelete_CountPolicy_AtLimit(t *testing.T) {
	t.Parallel()

	now := time.Now()
	backups := []BackupObject{
		{Key: "backup1", LastModified: now.Add(-1 * time.Hour)},
		{Key: "backup2", LastModified: now.Add(-2 * time.Hour)},
		{Key: "backup3", LastModified: now.Add(-3 * time.Hour)},
	}

	policy := RetentionPolicy{Days: 0, Count: 3}
	toDelete := determineBackupsToDelete(backups, policy)

	assert.Empty(t, toDelete, "No backups should be deleted when count equals limit")
}

func TestDetermineBackupsToDelete_CountPolicy_AboveLimit(t *testing.T) {
	t.Parallel()

	now := time.Now()
	// Backups are sorted newest first (as they would be from ListBackups)
	backups := []BackupObject{
		{Key: "newest", LastModified: now.Add(-1 * time.Hour)},
		{Key: "second", LastModified: now.Add(-2 * time.Hour)},
		{Key: "third", LastModified: now.Add(-3 * time.Hour)},
		{Key: "oldest1", LastModified: now.Add(-4 * time.Hour)},
		{Key: "oldest2", LastModified: now.Add(-5 * time.Hour)},
	}

	policy := RetentionPolicy{Days: 0, Count: 3}
	toDelete := determineBackupsToDelete(backups, policy)

	assert.Len(t, toDelete, 2)
	keys := make([]string, len(toDelete))
	for i, b := range toDelete {
		keys[i] = b.Key
	}
	assert.Contains(t, keys, "oldest1")
	assert.Contains(t, keys, "oldest2")
	assert.NotContains(t, keys, "newest")
	assert.NotContains(t, keys, "second")
	assert.NotContains(t, keys, "third")
}

func TestDetermineBackupsToDelete_BothPolicies_CountProtectsRecent(t *testing.T) {
	t.Parallel()

	now := time.Now()
	// All backups are old (> 7 days) but count policy should protect newest
	backups := []BackupObject{
		{Key: "old_but_newest", LastModified: now.Add(-8 * 24 * time.Hour)},
		{Key: "older", LastModified: now.Add(-10 * 24 * time.Hour)},
		{Key: "oldest", LastModified: now.Add(-15 * 24 * time.Hour)},
	}

	policy := RetentionPolicy{Days: 7, Count: 2}
	toDelete := determineBackupsToDelete(backups, policy)

	// The oldest should be deleted (both old by days AND exceeds count)
	assert.Len(t, toDelete, 1)
	assert.Equal(t, "oldest", toDelete[0].Key)
}

func TestDetermineBackupsToDelete_BothPolicies_Complex(t *testing.T) {
	t.Parallel()

	now := time.Now()
	backups := []BackupObject{
		{Key: "fresh1", LastModified: now.Add(-1 * 24 * time.Hour)}, // Fresh, in count
		{Key: "fresh2", LastModified: now.Add(-2 * 24 * time.Hour)}, // Fresh, in count
		{Key: "fresh3", LastModified: now.Add(-3 * 24 * time.Hour)}, // Fresh, in count
		{Key: "old1", LastModified: now.Add(-10 * 24 * time.Hour)},  // Old, exceeds count
		{Key: "old2", LastModified: now.Add(-20 * 24 * time.Hour)},  // Old, exceeds count
	}

	policy := RetentionPolicy{Days: 7, Count: 3}
	toDelete := determineBackupsToDelete(backups, policy)

	assert.Len(t, toDelete, 2)
	keys := make([]string, len(toDelete))
	for i, b := range toDelete {
		keys[i] = b.Key
	}
	assert.Contains(t, keys, "old1")
	assert.Contains(t, keys, "old2")
}

func TestDetermineBackupsToDelete_EmptyBackupList(t *testing.T) {
	t.Parallel()

	backups := []BackupObject{}

	policy := RetentionPolicy{Days: 7, Count: 5}
	toDelete := determineBackupsToDelete(backups, policy)

	assert.Empty(t, toDelete)
}

func TestDetermineBackupsToDelete_SingleBackup_Fresh(t *testing.T) {
	t.Parallel()

	now := time.Now()
	backups := []BackupObject{
		{Key: "only", LastModified: now.Add(-1 * time.Hour)},
	}

	policy := RetentionPolicy{Days: 7, Count: 1}
	toDelete := determineBackupsToDelete(backups, policy)

	assert.Empty(t, toDelete, "Should not delete the only fresh backup")
}

func TestDetermineBackupsToDelete_SingleBackup_Old(t *testing.T) {
	t.Parallel()

	now := time.Now()
	backups := []BackupObject{
		{Key: "only", LastModified: now.Add(-10 * 24 * time.Hour)},
	}

	// With count=1, the single backup should be protected even if old
	policy := RetentionPolicy{Days: 7, Count: 1}
	toDelete := determineBackupsToDelete(backups, policy)

	assert.Empty(t, toDelete, "Should not delete backup when count policy protects it")
}

func TestDetermineBackupsToDelete_DaysOnly_OldSingleBackup(t *testing.T) {
	t.Parallel()

	now := time.Now()
	backups := []BackupObject{
		{Key: "only", LastModified: now.Add(-10 * 24 * time.Hour)},
	}

	// With only days policy (no count), the old backup should be deleted
	policy := RetentionPolicy{Days: 7, Count: 0}
	toDelete := determineBackupsToDelete(backups, policy)

	assert.Len(t, toDelete, 1)
	assert.Equal(t, "only", toDelete[0].Key)
}

// Tests for BackupObject struct
func TestBackupObject_Fields(t *testing.T) {
	t.Parallel()

	now := time.Now()
	obj := BackupObject{
		Key:          "backups/db-2024.dump.gz",
		Size:         1024 * 1024,
		LastModified: now,
	}

	assert.Equal(t, "backups/db-2024.dump.gz", obj.Key)
	assert.Equal(t, int64(1024*1024), obj.Size)
	assert.Equal(t, now, obj.LastModified)
}

// Tests for RetentionResult struct
func TestRetentionResult_Fields(t *testing.T) {
	t.Parallel()

	result := RetentionResult{
		DeletedCount: 3,
		DeletedKeys:  []string{"key1", "key2", "key3"},
		Errors:       []error{},
	}

	assert.Equal(t, 3, result.DeletedCount)
	assert.Len(t, result.DeletedKeys, 3)
	assert.Empty(t, result.Errors)
}

// Tests for R2Client accessor methods
func TestR2Client_Bucket(t *testing.T) {
	t.Parallel()

	client := &R2Client{bucket: "my-backup-bucket"}
	assert.Equal(t, "my-backup-bucket", client.Bucket())
}

func TestR2Client_Prefix(t *testing.T) {
	t.Parallel()

	client := &R2Client{prefix: "prod/daily/"}
	assert.Equal(t, "prod/daily/", client.Prefix())
}

// Tests for edge cases in retention logic
func TestDetermineBackupsToDelete_ExactlyAtAgeLimit(t *testing.T) {
	t.Parallel()

	now := time.Now()
	// Just under 7 days - should be kept
	justUnderSevenDays := now.Add(-7*24*time.Hour + time.Hour)
	// Just over 7 days - should be deleted
	justOverSevenDays := now.Add(-7*24*time.Hour - time.Hour)

	backups := []BackupObject{
		{Key: "justUnder7days", LastModified: justUnderSevenDays},
		{Key: "justOver7days", LastModified: justOverSevenDays},
	}

	policy := RetentionPolicy{Days: 7, Count: 0}
	toDelete := determineBackupsToDelete(backups, policy)

	// "justOver7days" should be deleted (age > maxAge)
	// "justUnder7days" should be kept
	assert.Len(t, toDelete, 1)
	assert.Equal(t, "justOver7days", toDelete[0].Key)
}

func TestDetermineBackupsToDelete_CountOne(t *testing.T) {
	t.Parallel()

	now := time.Now()
	backups := []BackupObject{
		{Key: "newest", LastModified: now.Add(-1 * time.Hour)},
		{Key: "older1", LastModified: now.Add(-2 * time.Hour)},
		{Key: "older2", LastModified: now.Add(-3 * time.Hour)},
	}

	policy := RetentionPolicy{Days: 0, Count: 1}
	toDelete := determineBackupsToDelete(backups, policy)

	require.Len(t, toDelete, 2)
	keys := make([]string, len(toDelete))
	for i, b := range toDelete {
		keys[i] = b.Key
	}
	assert.Contains(t, keys, "older1")
	assert.Contains(t, keys, "older2")
	assert.NotContains(t, keys, "newest")
}

func TestDetermineBackupsToDelete_LargeBackupCount(t *testing.T) {
	t.Parallel()

	now := time.Now()
	// Create 100 backups with unique keys, 1 hour apart
	backups := make([]BackupObject, 100)
	for i := 0; i < 100; i++ {
		backups[i] = BackupObject{
			Key:          fmt.Sprintf("backup-%03d", i),
			LastModified: now.Add(-time.Duration(i) * time.Hour),
		}
	}

	policy := RetentionPolicy{Days: 0, Count: 10}
	toDelete := determineBackupsToDelete(backups, policy)

	assert.Len(t, toDelete, 90, "Should delete 90 backups keeping only newest 10")

	// Verify that the 10 newest are not in toDelete
	toDeleteKeys := make(map[string]bool)
	for _, b := range toDelete {
		toDeleteKeys[b.Key] = true
	}
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("backup-%03d", i)
		assert.False(t, toDeleteKeys[key], "backup %s should be kept", key)
	}
}

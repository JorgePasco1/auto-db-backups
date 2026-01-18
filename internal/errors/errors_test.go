package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackupError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		dbType      string
		dbName      string
		wrappedErr  error
		expectedMsg string
	}{
		{
			name:        "postgres backup error",
			dbType:      "postgres",
			dbName:      "mydb",
			wrappedErr:  errors.New("connection refused"),
			expectedMsg: "backup failed for postgres database 'mydb': connection refused",
		},
		{
			name:        "mysql backup error",
			dbType:      "mysql",
			dbName:      "production",
			wrappedErr:  errors.New("access denied"),
			expectedMsg: "backup failed for mysql database 'production': access denied",
		},
		{
			name:        "mongodb backup error",
			dbType:      "mongodb",
			dbName:      "analytics",
			wrappedErr:  errors.New("timeout"),
			expectedMsg: "backup failed for mongodb database 'analytics': timeout",
		},
		{
			name:        "empty database name",
			dbType:      "postgres",
			dbName:      "",
			wrappedErr:  errors.New("invalid config"),
			expectedMsg: "backup failed for postgres database '': invalid config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := NewBackupError(tt.dbType, tt.dbName, tt.wrappedErr)
			assert.Equal(t, tt.expectedMsg, err.Error())
		})
	}
}

func TestBackupError_Unwrap(t *testing.T) {
	t.Parallel()

	originalErr := errors.New("original error")
	backupErr := NewBackupError("postgres", "testdb", originalErr)

	unwrapped := backupErr.Unwrap()
	assert.Equal(t, originalErr, unwrapped)

	// Test errors.Is behavior
	assert.True(t, errors.Is(backupErr, originalErr))
}

func TestBackupError_Fields(t *testing.T) {
	t.Parallel()

	err := NewBackupError("mysql", "userdb", errors.New("test"))

	assert.Equal(t, "mysql", err.DatabaseType)
	assert.Equal(t, "userdb", err.DatabaseName)
	require.NotNil(t, err.Err)
}

func TestStorageError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		operation   string
		bucket      string
		key         string
		wrappedErr  error
		expectedMsg string
	}{
		{
			name:        "upload error",
			operation:   "upload",
			bucket:      "my-bucket",
			key:         "backup/db-2024.dump.gz",
			wrappedErr:  errors.New("access denied"),
			expectedMsg: "storage upload failed for bucket 'my-bucket', key 'backup/db-2024.dump.gz': access denied",
		},
		{
			name:        "delete error",
			operation:   "delete",
			bucket:      "backups",
			key:         "old-backup.sql",
			wrappedErr:  errors.New("not found"),
			expectedMsg: "storage delete failed for bucket 'backups', key 'old-backup.sql': not found",
		},
		{
			name:        "list error",
			operation:   "list",
			bucket:      "data",
			key:         "prefix/",
			wrappedErr:  errors.New("timeout"),
			expectedMsg: "storage list failed for bucket 'data', key 'prefix/': timeout",
		},
		{
			name:        "empty key",
			operation:   "upload",
			bucket:      "bucket",
			key:         "",
			wrappedErr:  errors.New("invalid key"),
			expectedMsg: "storage upload failed for bucket 'bucket', key '': invalid key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := NewStorageError(tt.operation, tt.bucket, tt.key, tt.wrappedErr)
			assert.Equal(t, tt.expectedMsg, err.Error())
		})
	}
}

func TestStorageError_Unwrap(t *testing.T) {
	t.Parallel()

	originalErr := errors.New("original storage error")
	storageErr := NewStorageError("upload", "bucket", "key", originalErr)

	unwrapped := storageErr.Unwrap()
	assert.Equal(t, originalErr, unwrapped)

	// Test errors.Is behavior
	assert.True(t, errors.Is(storageErr, originalErr))
}

func TestStorageError_Fields(t *testing.T) {
	t.Parallel()

	err := NewStorageError("delete", "my-bucket", "my-key", errors.New("test"))

	assert.Equal(t, "delete", err.Operation)
	assert.Equal(t, "my-bucket", err.Bucket)
	assert.Equal(t, "my-key", err.Key)
	require.NotNil(t, err.Err)
}

func TestConfigError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		field       string
		message     string
		expectedMsg string
	}{
		{
			name:        "missing required field",
			field:       "database_host",
			message:     "is required",
			expectedMsg: "configuration error for 'database_host': is required",
		},
		{
			name:        "invalid value",
			field:       "encryption_key",
			message:     "must be 32 bytes",
			expectedMsg: "configuration error for 'encryption_key': must be 32 bytes",
		},
		{
			name:        "empty field name",
			field:       "",
			message:     "unknown field",
			expectedMsg: "configuration error for '': unknown field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := NewConfigError(tt.field, tt.message)
			assert.Equal(t, tt.expectedMsg, err.Error())
		})
	}
}

func TestConfigError_Fields(t *testing.T) {
	t.Parallel()

	err := NewConfigError("test_field", "test message")

	assert.Equal(t, "test_field", err.Field)
	assert.Equal(t, "test message", err.Message)
}

func TestSentinelErrors(t *testing.T) {
	t.Parallel()

	// Verify all sentinel errors are defined and distinct
	sentinels := []error{
		ErrBackupFailed,
		ErrUploadFailed,
		ErrEncryptionFailed,
		ErrCompressionFailed,
		ErrConnectionFailed,
		ErrRetentionFailed,
		ErrNotificationFailed,
	}

	// Check they are all non-nil
	for _, err := range sentinels {
		assert.NotNil(t, err)
	}

	// Check they are all distinct
	for i, err1 := range sentinels {
		for j, err2 := range sentinels {
			if i != j {
				assert.NotEqual(t, err1, err2, "sentinel errors should be distinct")
			}
		}
	}
}

func TestErrorWrapping(t *testing.T) {
	t.Parallel()

	// Test that wrapped errors can be detected with errors.As
	t.Run("BackupError with errors.As", func(t *testing.T) {
		t.Parallel()
		originalErr := errors.New("pg_dump failed")
		backupErr := NewBackupError("postgres", "testdb", originalErr)

		var target *BackupError
		assert.True(t, errors.As(backupErr, &target))
		assert.Equal(t, "postgres", target.DatabaseType)
	})

	t.Run("StorageError with errors.As", func(t *testing.T) {
		t.Parallel()
		originalErr := errors.New("s3 error")
		storageErr := NewStorageError("upload", "bucket", "key", originalErr)

		var target *StorageError
		assert.True(t, errors.As(storageErr, &target))
		assert.Equal(t, "upload", target.Operation)
	})

	t.Run("ConfigError with errors.As", func(t *testing.T) {
		t.Parallel()
		configErr := NewConfigError("field", "message")

		var target *ConfigError
		assert.True(t, errors.As(configErr, &target))
		assert.Equal(t, "field", target.Field)
	})
}

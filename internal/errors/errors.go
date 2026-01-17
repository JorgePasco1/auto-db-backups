package errors

import (
	"errors"
	"fmt"
)

var (
	ErrBackupFailed     = errors.New("backup failed")
	ErrUploadFailed     = errors.New("upload failed")
	ErrEncryptionFailed = errors.New("encryption failed")
	ErrCompressionFailed = errors.New("compression failed")
	ErrConnectionFailed = errors.New("database connection failed")
	ErrRetentionFailed  = errors.New("retention cleanup failed")
	ErrNotificationFailed = errors.New("notification failed")
)

type BackupError struct {
	DatabaseType string
	DatabaseName string
	Err          error
}

func (e *BackupError) Error() string {
	return fmt.Sprintf("backup failed for %s database '%s': %v", e.DatabaseType, e.DatabaseName, e.Err)
}

func (e *BackupError) Unwrap() error {
	return e.Err
}

func NewBackupError(dbType, dbName string, err error) *BackupError {
	return &BackupError{
		DatabaseType: dbType,
		DatabaseName: dbName,
		Err:          err,
	}
}

type StorageError struct {
	Operation string
	Bucket    string
	Key       string
	Err       error
}

func (e *StorageError) Error() string {
	return fmt.Sprintf("storage %s failed for bucket '%s', key '%s': %v", e.Operation, e.Bucket, e.Key, e.Err)
}

func (e *StorageError) Unwrap() error {
	return e.Err
}

func NewStorageError(op, bucket, key string, err error) *StorageError {
	return &StorageError{
		Operation: op,
		Bucket:    bucket,
		Key:       key,
		Err:       err,
	}
}

type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("configuration error for '%s': %s", e.Field, e.Message)
}

func NewConfigError(field, message string) *ConfigError {
	return &ConfigError{
		Field:   field,
		Message: message,
	}
}

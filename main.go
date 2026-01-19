package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jorgepascosoto/auto-db-backups/internal/backup"
	"github.com/jorgepascosoto/auto-db-backups/internal/compress"
	"github.com/jorgepascosoto/auto-db-backups/internal/config"
	"github.com/jorgepascosoto/auto-db-backups/internal/encrypt"
	"github.com/jorgepascosoto/auto-db-backups/internal/notify"
	"github.com/jorgepascosoto/auto-db-backups/internal/storage"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Received shutdown signal, canceling...")
		cancel()
	}()

	if err := run(ctx); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run(ctx context.Context) error {
	startTime := time.Now()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	log.Printf("Starting backup for %d database(s)", len(cfg.Databases))

	// Track results for all databases
	var allBackupKeys []string
	var allBackupSizes []int64
	var failedDatabases []string

	// Process each database
	for i, db := range cfg.Databases {
		dbStartTime := time.Now()
		log.Printf("[%d/%d] Backing up %s database: %s", i+1, len(cfg.Databases), db.Type, db.Name)

		// Create summary for this database
		summary := &notify.BackupSummary{
			DatabaseType: string(db.Type),
			DatabaseName: db.Name,
			Compressed:   cfg.Compression,
			Encrypted:    cfg.HasEncryption(),
		}

		// Run the backup for this database
		backupKey, backupSize, err := performBackup(ctx, cfg, &db)
		summary.Duration = time.Since(dbStartTime)

		if err != nil {
			log.Printf("[%d/%d] FAILED: %s - %v", i+1, len(cfg.Databases), db.Name, err)
			summary.Success = false
			summary.Error = err
			failedDatabases = append(failedDatabases, db.Name)

			// Send failure notification for this database
			if err := sendNotifications(ctx, cfg, summary); err != nil {
				log.Printf("Warning: failed to send notifications for %s: %v", db.Name, err)
			}
			continue
		}

		log.Printf("[%d/%d] SUCCESS: %s -> %s (%d bytes)", i+1, len(cfg.Databases), db.Name, backupKey, backupSize)
		summary.Success = true
		summary.BackupKey = backupKey
		summary.BackupSize = backupSize

		allBackupKeys = append(allBackupKeys, backupKey)
		allBackupSizes = append(allBackupSizes, backupSize)

		// Apply retention policy for this database's prefix
		if cfg.HasRetention() {
			r2Client, err := storage.NewR2Client(ctx, cfg, db.BackupPrefix)
			if err != nil {
				log.Printf("Warning: failed to create R2 client for retention (%s): %v", db.Name, err)
			} else {
				result, err := storage.ApplyRetention(ctx, r2Client, storage.RetentionPolicy{
					Days:  cfg.RetentionDays,
					Count: cfg.RetentionCount,
				})
				if err != nil {
					log.Printf("Warning: retention policy failed for %s: %v", db.Name, err)
				} else if result.DeletedCount > 0 {
					log.Printf("[%d/%d] Deleted %d old backup(s) for %s", i+1, len(cfg.Databases), result.DeletedCount, db.Name)
					summary.DeletedBackups = result.DeletedCount
				}
			}
		}

		// Send success notification for this database
		if err := sendNotifications(ctx, cfg, summary); err != nil {
			log.Printf("Warning: failed to send notifications for %s: %v", db.Name, err)
		}
	}

	// Set GitHub Action outputs (aggregate results)
	if len(allBackupKeys) > 0 {
		// For single database, set direct values; for multiple, use first one
		if err := notify.SetGitHubOutput("backup_key", allBackupKeys[0]); err != nil {
			log.Printf("Warning: failed to set backup_key output: %v", err)
		}
		if err := notify.SetGitHubOutput("backup_size", fmt.Sprintf("%d", allBackupSizes[0])); err != nil {
			log.Printf("Warning: failed to set backup_size output: %v", err)
		}
		// Also set count of successful backups
		if err := notify.SetGitHubOutput("backup_count", fmt.Sprintf("%d", len(allBackupKeys))); err != nil {
			log.Printf("Warning: failed to set backup_count output: %v", err)
		}
	}

	totalDuration := time.Since(startTime)
	log.Printf("Completed: %d successful, %d failed (total time: %s)",
		len(allBackupKeys), len(failedDatabases), totalDuration.Round(time.Second))

	// Return error if any database failed
	if len(failedDatabases) > 0 {
		return fmt.Errorf("backup failed for %d database(s): %v", len(failedDatabases), failedDatabases)
	}

	return nil
}

func performBackup(ctx context.Context, cfg *config.Config, db *config.DatabaseConfig) (string, int64, error) {
	// Create database exporter
	exporter, err := backup.NewExporter(db)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create exporter: %w", err)
	}

	// Export database
	log.Printf("  Exporting database...")
	reader, err := exporter.Export(ctx)
	if err != nil {
		return "", 0, fmt.Errorf("failed to export database: %w", err)
	}
	// Note: we don't defer Close() here because we need to check its error
	// after reading all data (it captures pg_dump exit status)

	// Build backup filename
	timestamp := time.Now().UTC().Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s-%s", db.Type, db.Name, timestamp)

	// Add extension based on database type
	switch db.Type {
	case config.DatabaseTypePostgres:
		filename += ".dump"
	case config.DatabaseTypeMySQL:
		filename += ".sql"
	case config.DatabaseTypeMongoDB:
		filename += ".tar"
	}

	var dataReader io.Reader = reader

	// Apply compression if enabled
	if cfg.Compression {
		log.Printf("  Compressing backup...")
		compressor := compress.NewGzipCompressor()
		compressedReader := compressor.Compress(dataReader)
		defer compressedReader.Close()
		dataReader = compressedReader
		filename += compressor.Extension()
	}

	// Apply encryption if enabled
	if cfg.HasEncryption() {
		log.Printf("  Encrypting backup...")
		encryptor, err := encrypt.NewAESEncryptor(cfg.EncryptionKey)
		if err != nil {
			return "", 0, fmt.Errorf("failed to create encryptor: %w", err)
		}
		encryptedReader, err := encryptor.Encrypt(dataReader)
		if err != nil {
			return "", 0, fmt.Errorf("failed to encrypt backup: %w", err)
		}
		defer encryptedReader.Close()
		dataReader = encryptedReader
		filename += encryptor.Extension()
	}

	// Read all data into memory to get size before upload
	// (Required because R2/S3 needs content length for some operations)
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, dataReader); err != nil {
		reader.Close()
		return "", 0, fmt.Errorf("failed to read backup data: %w", err)
	}
	backupSize := int64(buf.Len())

	// Close the reader to capture any errors from the dump command
	// (pg_dump exit status is only available after reading all output)
	if err := reader.Close(); err != nil {
		return "", 0, fmt.Errorf("database export failed: %w", err)
	}

	// Upload to R2
	log.Printf("  Uploading backup to R2...")
	r2Client, err := storage.NewR2Client(ctx, cfg, db.BackupPrefix)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create R2 client: %w", err)
	}

	if err := r2Client.Upload(ctx, filename, &buf); err != nil {
		return "", 0, fmt.Errorf("failed to upload backup: %w", err)
	}

	fullKey := db.BackupPrefix + filename
	return fullKey, backupSize, nil
}

func sendNotifications(ctx context.Context, cfg *config.Config, summary *notify.BackupSummary) error {
	// Write GitHub step summary
	if err := notify.WriteGitHubSummary(summary); err != nil {
		log.Printf("Warning: failed to write GitHub summary: %v", err)
	}

	// Send webhook notification
	if cfg.WebhookURL != "" {
		shouldNotify := (summary.Success && cfg.NotifyOnSuccess) || (!summary.Success && cfg.NotifyOnFailure)
		if shouldNotify {
			notifier := notify.NewWebhookNotifier(cfg.WebhookURL)
			if err := notifier.Notify(ctx, summary); err != nil {
				return fmt.Errorf("webhook notification failed: %w", err)
			}
		}
	}

	return nil
}

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

	log.Printf("Starting backup for %s database: %s", cfg.DatabaseType, cfg.DatabaseName)

	// Create summary for notifications
	summary := &notify.BackupSummary{
		DatabaseType: string(cfg.DatabaseType),
		DatabaseName: cfg.DatabaseName,
		Compressed:   cfg.Compression,
		Encrypted:    cfg.HasEncryption(),
	}

	// Run the backup
	backupKey, backupSize, err := performBackup(ctx, cfg)
	summary.Duration = time.Since(startTime)

	if err != nil {
		summary.Success = false
		summary.Error = err
		if notifyErr := sendNotifications(ctx, cfg, summary); notifyErr != nil {
			log.Printf("Warning: failed to send notifications: %v", notifyErr)
		}
		return err
	}

	summary.Success = true
	summary.BackupKey = backupKey
	summary.BackupSize = backupSize

	// Apply retention policy
	if cfg.HasRetention() {
		r2Client, err := storage.NewR2Client(ctx, cfg)
		if err != nil {
			log.Printf("Warning: failed to create R2 client for retention: %v", err)
		} else {
			result, err := storage.ApplyRetention(ctx, r2Client, storage.RetentionPolicy{
				Days:  cfg.RetentionDays,
				Count: cfg.RetentionCount,
			})
			if err != nil {
				log.Printf("Warning: retention policy failed: %v", err)
			} else if result.DeletedCount > 0 {
				log.Printf("Deleted %d old backup(s)", result.DeletedCount)
				summary.DeletedBackups = result.DeletedCount
			}
		}
	}

	// Send notifications
	if err := sendNotifications(ctx, cfg, summary); err != nil {
		log.Printf("Warning: failed to send notifications: %v", err)
	}

	// Set GitHub Action outputs
	if err := notify.SetGitHubOutput("backup_key", backupKey); err != nil {
		log.Printf("Warning: failed to set backup_key output: %v", err)
	}
	if err := notify.SetGitHubOutput("backup_size", fmt.Sprintf("%d", backupSize)); err != nil {
		log.Printf("Warning: failed to set backup_size output: %v", err)
	}

	log.Printf("Backup completed successfully: %s (%d bytes)", backupKey, backupSize)
	return nil
}

func performBackup(ctx context.Context, cfg *config.Config) (string, int64, error) {
	// Create database exporter
	exporter, err := backup.NewExporter(cfg)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create exporter: %w", err)
	}

	// Export database
	log.Printf("Exporting database...")
	reader, err := exporter.Export(ctx)
	if err != nil {
		return "", 0, fmt.Errorf("failed to export database: %w", err)
	}
	defer reader.Close()

	// Build backup filename
	timestamp := time.Now().UTC().Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s-%s", cfg.DatabaseType, cfg.DatabaseName, timestamp)

	// Add extension based on database type
	switch cfg.DatabaseType {
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
		log.Printf("Compressing backup...")
		compressor := compress.NewGzipCompressor()
		compressedReader := compressor.Compress(dataReader)
		defer compressedReader.Close()
		dataReader = compressedReader
		filename += compressor.Extension()
	}

	// Apply encryption if enabled
	if cfg.HasEncryption() {
		log.Printf("Encrypting backup...")
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
		return "", 0, fmt.Errorf("failed to read backup data: %w", err)
	}
	backupSize := int64(buf.Len())

	// Upload to R2
	log.Printf("Uploading backup to R2...")
	r2Client, err := storage.NewR2Client(ctx, cfg)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create R2 client: %w", err)
	}

	if err := r2Client.Upload(ctx, filename, &buf); err != nil {
		return "", 0, fmt.Errorf("failed to upload backup: %w", err)
	}

	fullKey := cfg.BackupPrefix + filename
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

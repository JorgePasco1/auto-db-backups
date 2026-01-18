package backup

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/jorgepascosoto/auto-db-backups/internal/config"
	"github.com/jorgepascosoto/auto-db-backups/internal/errors"
)

type MongoDBExporter struct {
	db *config.DatabaseConfig
}

func NewMongoDBExporter(db *config.DatabaseConfig) *MongoDBExporter {
	return &MongoDBExporter{db: db}
}

func (e *MongoDBExporter) Export(ctx context.Context) (io.ReadCloser, error) {
	// mongodump writes to a directory, so we need to create a temp dir
	// and then archive it
	tempDir, err := os.MkdirTemp("", "mongodump-*")
	if err != nil {
		return nil, errors.NewBackupError("mongodb", e.db.Name, fmt.Errorf("failed to create temp directory: %w", err))
	}

	outputDir := filepath.Join(tempDir, "dump")

	args := e.buildArgs(outputDir)

	cmd := exec.CommandContext(ctx, "mongodump", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		os.RemoveAll(tempDir)
		return nil, errors.NewBackupError("mongodb", e.db.Name, fmt.Errorf("mongodump failed: %w: %s", err, string(output)))
	}

	// Create archive using tar
	archiveCmd := exec.CommandContext(ctx, "tar", "-cf", "-", "-C", tempDir, "dump")

	stdout, err := archiveCmd.StdoutPipe()
	if err != nil {
		os.RemoveAll(tempDir)
		return nil, errors.NewBackupError("mongodb", e.db.Name, fmt.Errorf("failed to create stdout pipe: %w", err))
	}

	stderrPipe, err := archiveCmd.StderrPipe()
	if err != nil {
		os.RemoveAll(tempDir)
		return nil, errors.NewBackupError("mongodb", e.db.Name, fmt.Errorf("failed to create stderr pipe: %w", err))
	}

	if err := archiveCmd.Start(); err != nil {
		os.RemoveAll(tempDir)
		return nil, errors.NewBackupError("mongodb", e.db.Name, fmt.Errorf("failed to start tar: %w", err))
	}

	return &mongoReadCloser{
		ReadCloser: stdout,
		cmd:        archiveCmd,
		stderr:     stderrPipe,
		tempDir:    tempDir,
		dbName:     e.db.Name,
	}, nil
}

func (e *MongoDBExporter) DatabaseName() string {
	return e.db.Name
}

func (e *MongoDBExporter) DatabaseType() string {
	return "mongodb"
}

func (e *MongoDBExporter) buildArgs(outputDir string) []string {
	// If connection string is provided, use it
	if e.db.ConnectionString != "" {
		return []string{
			"--uri=" + e.db.ConnectionString,
			"--out=" + outputDir,
		}
	}

	args := []string{
		fmt.Sprintf("--host=%s", e.db.Host),
		fmt.Sprintf("--port=%d", e.db.Port),
		fmt.Sprintf("--db=%s", e.db.Name),
		fmt.Sprintf("--out=%s", outputDir),
	}

	if e.db.User != "" {
		args = append(args, fmt.Sprintf("--username=%s", e.db.User))
	}

	if e.db.Password != "" {
		args = append(args, fmt.Sprintf("--password=%s", e.db.Password))
	}

	return args
}

type mongoReadCloser struct {
	io.ReadCloser
	cmd     *exec.Cmd
	stderr  io.ReadCloser
	tempDir string
	dbName  string
}

func (c *mongoReadCloser) Close() error {
	// Always clean up temp directory
	defer os.RemoveAll(c.tempDir)

	// Read any stderr output for error reporting
	stderrBytes, _ := io.ReadAll(c.stderr)

	// Close the stdout pipe first
	if err := c.ReadCloser.Close(); err != nil {
		return err
	}

	// Wait for the command to finish
	if err := c.cmd.Wait(); err != nil {
		stderrMsg := string(stderrBytes)
		if stderrMsg != "" {
			return errors.NewBackupError("mongodb", c.dbName, fmt.Errorf("%w: %s", err, stderrMsg))
		}
		return errors.NewBackupError("mongodb", c.dbName, err)
	}

	return nil
}

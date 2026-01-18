package backup

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/jorgepascosoto/auto-db-backups/internal/config"
	"github.com/jorgepascosoto/auto-db-backups/internal/errors"
)

type PostgresExporter struct {
	db *config.DatabaseConfig
}

func NewPostgresExporter(db *config.DatabaseConfig) *PostgresExporter {
	return &PostgresExporter{db: db}
}

func (e *PostgresExporter) Export(ctx context.Context) (io.ReadCloser, error) {
	args := e.buildArgs()

	cmd := exec.CommandContext(ctx, "pg_dump", args...)
	cmd.Env = e.buildEnv()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.NewBackupError("postgres", e.db.Name, fmt.Errorf("failed to create stdout pipe: %w", err))
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, errors.NewBackupError("postgres", e.db.Name, fmt.Errorf("failed to create stderr pipe: %w", err))
	}

	if err := cmd.Start(); err != nil {
		return nil, errors.NewBackupError("postgres", e.db.Name, fmt.Errorf("failed to start pg_dump: %w", err))
	}

	return &cmdReadCloser{
		ReadCloser: stdout,
		cmd:        cmd,
		stderr:     stderrPipe,
		dbType:     "postgres",
		dbName:     e.db.Name,
	}, nil
}

func (e *PostgresExporter) DatabaseName() string {
	return e.db.Name
}

func (e *PostgresExporter) DatabaseType() string {
	return "postgres"
}

func (e *PostgresExporter) buildArgs() []string {
	// If connection string is provided, use it directly
	if e.db.ConnectionString != "" {
		return []string{e.db.ConnectionString, "--format=custom"}
	}

	args := []string{
		"--format=custom",
		"--no-password",
		fmt.Sprintf("--host=%s", e.db.Host),
		fmt.Sprintf("--port=%d", e.db.Port),
		fmt.Sprintf("--dbname=%s", e.db.Name),
	}

	if e.db.User != "" {
		args = append(args, fmt.Sprintf("--username=%s", e.db.User))
	}

	return args
}

func (e *PostgresExporter) buildEnv() []string {
	env := os.Environ()

	if e.db.Password != "" {
		env = append(env, fmt.Sprintf("PGPASSWORD=%s", e.db.Password))
	}

	return env
}

type cmdReadCloser struct {
	io.ReadCloser
	cmd    *exec.Cmd
	stderr io.ReadCloser
	dbType string
	dbName string
}

func (c *cmdReadCloser) Close() error {
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
			return errors.NewBackupError(c.dbType, c.dbName, fmt.Errorf("%w: %s", err, stderrMsg))
		}
		return errors.NewBackupError(c.dbType, c.dbName, err)
	}

	return nil
}

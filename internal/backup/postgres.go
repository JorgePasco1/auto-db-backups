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
	cfg *config.Config
}

func NewPostgresExporter(cfg *config.Config) *PostgresExporter {
	return &PostgresExporter{cfg: cfg}
}

func (e *PostgresExporter) Export(ctx context.Context) (io.ReadCloser, error) {
	args := e.buildArgs()

	cmd := exec.CommandContext(ctx, "pg_dump", args...)
	cmd.Env = e.buildEnv()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.NewBackupError("postgres", e.cfg.DatabaseName, fmt.Errorf("failed to create stdout pipe: %w", err))
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, errors.NewBackupError("postgres", e.cfg.DatabaseName, fmt.Errorf("failed to create stderr pipe: %w", err))
	}

	if err := cmd.Start(); err != nil {
		return nil, errors.NewBackupError("postgres", e.cfg.DatabaseName, fmt.Errorf("failed to start pg_dump: %w", err))
	}

	return &cmdReadCloser{
		ReadCloser: stdout,
		cmd:        cmd,
		stderr:     stderrPipe,
		dbType:     "postgres",
		dbName:     e.cfg.DatabaseName,
	}, nil
}

func (e *PostgresExporter) DatabaseName() string {
	return e.cfg.DatabaseName
}

func (e *PostgresExporter) DatabaseType() string {
	return "postgres"
}

func (e *PostgresExporter) buildArgs() []string {
	// If connection string is provided, use it directly
	if e.cfg.ConnectionString != "" {
		return []string{e.cfg.ConnectionString, "--format=custom"}
	}

	args := []string{
		"--format=custom",
		"--no-password",
		fmt.Sprintf("--host=%s", e.cfg.DatabaseHost),
		fmt.Sprintf("--port=%d", e.cfg.DatabasePort),
		fmt.Sprintf("--dbname=%s", e.cfg.DatabaseName),
	}

	if e.cfg.DatabaseUser != "" {
		args = append(args, fmt.Sprintf("--username=%s", e.cfg.DatabaseUser))
	}

	return args
}

func (e *PostgresExporter) buildEnv() []string {
	env := os.Environ()

	if e.cfg.DatabasePassword != "" {
		env = append(env, fmt.Sprintf("PGPASSWORD=%s", e.cfg.DatabasePassword))
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

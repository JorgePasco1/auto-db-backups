package backup

import (
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/jorgepascosoto/auto-db-backups/internal/config"
	"github.com/jorgepascosoto/auto-db-backups/internal/errors"
)

type MySQLExporter struct {
	db *config.DatabaseConfig
}

func NewMySQLExporter(db *config.DatabaseConfig) *MySQLExporter {
	return &MySQLExporter{db: db}
}

func (e *MySQLExporter) Export(ctx context.Context) (io.ReadCloser, error) {
	args := e.buildArgs()

	cmd := exec.CommandContext(ctx, "mysqldump", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.NewBackupError("mysql", e.db.Name, fmt.Errorf("failed to create stdout pipe: %w", err))
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, errors.NewBackupError("mysql", e.db.Name, fmt.Errorf("failed to create stderr pipe: %w", err))
	}

	if err := cmd.Start(); err != nil {
		return nil, errors.NewBackupError("mysql", e.db.Name, fmt.Errorf("failed to start mysqldump: %w", err))
	}

	return &cmdReadCloser{
		ReadCloser: stdout,
		cmd:        cmd,
		stderr:     stderrPipe,
		dbType:     "mysql",
		dbName:     e.db.Name,
	}, nil
}

func (e *MySQLExporter) DatabaseName() string {
	return e.db.Name
}

func (e *MySQLExporter) DatabaseType() string {
	return "mysql"
}

func (e *MySQLExporter) buildArgs() []string {
	args := []string{
		"--single-transaction",
		"--routines",
		"--triggers",
		"--events",
		fmt.Sprintf("--host=%s", e.db.Host),
		fmt.Sprintf("--port=%d", e.db.Port),
	}

	if e.db.User != "" {
		args = append(args, fmt.Sprintf("--user=%s", e.db.User))
	}

	if e.db.Password != "" {
		args = append(args, fmt.Sprintf("--password=%s", e.db.Password))
	}

	args = append(args, e.db.Name)

	return args
}

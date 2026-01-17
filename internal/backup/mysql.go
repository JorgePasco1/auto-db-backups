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
	cfg *config.Config
}

func NewMySQLExporter(cfg *config.Config) *MySQLExporter {
	return &MySQLExporter{cfg: cfg}
}

func (e *MySQLExporter) Export(ctx context.Context) (io.ReadCloser, error) {
	args := e.buildArgs()

	cmd := exec.CommandContext(ctx, "mysqldump", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.NewBackupError("mysql", e.cfg.DatabaseName, fmt.Errorf("failed to create stdout pipe: %w", err))
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, errors.NewBackupError("mysql", e.cfg.DatabaseName, fmt.Errorf("failed to create stderr pipe: %w", err))
	}

	if err := cmd.Start(); err != nil {
		return nil, errors.NewBackupError("mysql", e.cfg.DatabaseName, fmt.Errorf("failed to start mysqldump: %w", err))
	}

	return &cmdReadCloser{
		ReadCloser: stdout,
		cmd:        cmd,
		stderr:     stderrPipe,
		dbType:     "mysql",
		dbName:     e.cfg.DatabaseName,
	}, nil
}

func (e *MySQLExporter) DatabaseName() string {
	return e.cfg.DatabaseName
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
		fmt.Sprintf("--host=%s", e.cfg.DatabaseHost),
		fmt.Sprintf("--port=%d", e.cfg.DatabasePort),
	}

	if e.cfg.DatabaseUser != "" {
		args = append(args, fmt.Sprintf("--user=%s", e.cfg.DatabaseUser))
	}

	if e.cfg.DatabasePassword != "" {
		args = append(args, fmt.Sprintf("--password=%s", e.cfg.DatabasePassword))
	}

	args = append(args, e.cfg.DatabaseName)

	return args
}

package backup

import (
	"context"
	"io"
)

type Exporter interface {
	Export(ctx context.Context) (io.ReadCloser, error)
	DatabaseName() string
	DatabaseType() string
}

type ExportResult struct {
	Reader       io.ReadCloser
	DatabaseName string
	DatabaseType string
}

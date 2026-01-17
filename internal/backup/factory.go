package backup

import (
	"fmt"

	"github.com/jorgepascosoto/auto-db-backups/internal/config"
)

func NewExporter(cfg *config.Config) (Exporter, error) {
	switch cfg.DatabaseType {
	case config.DatabaseTypePostgres:
		return NewPostgresExporter(cfg), nil
	case config.DatabaseTypeMySQL:
		return NewMySQLExporter(cfg), nil
	case config.DatabaseTypeMongoDB:
		return NewMongoDBExporter(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.DatabaseType)
	}
}

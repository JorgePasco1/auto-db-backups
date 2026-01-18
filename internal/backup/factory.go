package backup

import (
	"fmt"

	"github.com/jorgepascosoto/auto-db-backups/internal/config"
)

func NewExporter(db *config.DatabaseConfig) (Exporter, error) {
	switch db.Type {
	case config.DatabaseTypePostgres:
		return NewPostgresExporter(db), nil
	case config.DatabaseTypeMySQL:
		return NewMySQLExporter(db), nil
	case config.DatabaseTypeMongoDB:
		return NewMongoDBExporter(db), nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", db.Type)
	}
}

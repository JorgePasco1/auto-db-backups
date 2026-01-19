package config

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type DatabaseType string

const (
	DatabaseTypePostgres DatabaseType = "postgres"
	DatabaseTypeMySQL    DatabaseType = "mysql"
	DatabaseTypeMongoDB  DatabaseType = "mongodb"
)

// DatabaseConfig holds settings for a single database to back up
type DatabaseConfig struct {
	Type             DatabaseType
	Host             string
	Port             int
	Name             string
	User             string
	Password         string
	ConnectionString string
	BackupPrefix     string
}

// Config holds the application configuration
type Config struct {
	// Database settings (multiple databases supported)
	Databases []DatabaseConfig

	// R2 settings (shared across all backups)
	R2AccountID       string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2BucketName      string

	// Backup settings (shared)
	Compression   bool
	EncryptionKey []byte

	// Retention settings (shared)
	RetentionDays  int
	RetentionCount int

	// Notification settings (shared)
	WebhookURL      string
	NotifyOnSuccess bool
	NotifyOnFailure bool
}

func Load() (*Config, error) {
	cfg := &Config{}

	// Determine database type (shared across all connections)
	dbType := getInput("database_type")
	var parsedDBType DatabaseType
	switch strings.ToLower(dbType) {
	case "postgres", "postgresql", "":
		parsedDBType = DatabaseTypePostgres // Default to postgres
	case "mysql", "mariadb":
		parsedDBType = DatabaseTypeMySQL
	case "mongodb", "mongo":
		parsedDBType = DatabaseTypeMongoDB
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	// Load database connections
	databases, err := loadDatabaseConfigs(parsedDBType)
	if err != nil {
		return nil, err
	}
	cfg.Databases = databases

	// R2 settings
	cfg.R2AccountID = getInput("r2_account_id")
	cfg.R2AccessKeyID = getInput("r2_access_key_id")
	cfg.R2SecretAccessKey = getInput("r2_secret_access_key")
	cfg.R2BucketName = getInput("r2_bucket_name")

	// Backup settings
	cfg.Compression = getInputBool("compression", true)

	encKeyStr := getInput("encryption_key")
	if encKeyStr != "" {
		key, err := base64.StdEncoding.DecodeString(encKeyStr)
		if err != nil {
			return nil, fmt.Errorf("invalid encryption key: must be base64 encoded: %w", err)
		}
		if len(key) != 32 {
			return nil, fmt.Errorf("invalid encryption key: must be exactly 32 bytes (256 bits), got %d bytes", len(key))
		}
		cfg.EncryptionKey = key
	}

	// Retention settings
	cfg.RetentionDays = getInputInt("retention_days", 0)
	cfg.RetentionCount = getInputInt("retention_count", 0)

	// Notification settings
	cfg.WebhookURL = getInput("webhook_url")
	cfg.NotifyOnSuccess = getInputBool("notify_on_success", true)
	cfg.NotifyOnFailure = getInputBool("notify_on_failure", true)

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// loadDatabaseConfigs loads multiple database configurations
// It looks for DATABASE_CONNECTION_1, DATABASE_CONNECTION_2, etc.
// Falls back to single CONNECTION_STRING for backward compatibility
func loadDatabaseConfigs(dbType DatabaseType) ([]DatabaseConfig, error) {
	var databases []DatabaseConfig

	// Try numbered connections first: DATABASE_CONNECTION_1, DATABASE_CONNECTION_2, etc.
	for i := 1; ; i++ {
		connStr := getInput(fmt.Sprintf("database_connection_%d", i))
		if connStr == "" {
			break
		}

		// Use custom name if provided, otherwise parse from connection string
		dbName := getInput(fmt.Sprintf("database_name_%d", i))
		if dbName == "" {
			dbName = parseDatabaseNameFromConnectionString(connStr)
		}

		prefix := getInput(fmt.Sprintf("database_prefix_%d", i))
		if prefix == "" {
			prefix = fmt.Sprintf("backups/%s/", dbName)
		}
		if prefix != "" && !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}

		databases = append(databases, DatabaseConfig{
			Type:             dbType,
			ConnectionString: connStr,
			Name:             dbName,
			Port:             defaultPort(dbType),
			BackupPrefix:     prefix,
		})
	}

	// If no numbered connections found, fall back to single connection_string
	if len(databases) == 0 {
		connStr := getInput("connection_string")
		if connStr != "" {
			// Use custom name if provided, otherwise parse from connection string
			dbName := getInput("database_name")
			if dbName == "" {
				dbName = parseDatabaseNameFromConnectionString(connStr)
			}

			prefix := getInput("backup_prefix")
			if prefix != "" && !strings.HasSuffix(prefix, "/") {
				prefix += "/"
			}

			databases = append(databases, DatabaseConfig{
				Type:             dbType,
				ConnectionString: connStr,
				Name:             dbName,
				Port:             defaultPort(dbType),
				BackupPrefix:     prefix,
			})
		}
	}

	// If still no connections, try individual parameters (legacy support)
	if len(databases) == 0 {
		host := getInput("database_host")
		if host != "" {
			prefix := getInput("backup_prefix")
			if prefix != "" && !strings.HasSuffix(prefix, "/") {
				prefix += "/"
			}

			databases = append(databases, DatabaseConfig{
				Type:         dbType,
				Host:         host,
				Port:         getInputInt("database_port", defaultPort(dbType)),
				Name:         getInput("database_name"),
				User:         getInput("database_user"),
				Password:     getInput("database_password"),
				BackupPrefix: prefix,
			})
		}
	}

	if len(databases) == 0 {
		return nil, fmt.Errorf("no database connections configured. Set DATABASE_CONNECTION_1 or CONNECTION_STRING")
	}

	return databases, nil
}

func (c *Config) Validate() error {
	// Validate each database config
	for i, db := range c.Databases {
		if db.ConnectionString == "" {
			if db.Host == "" {
				return fmt.Errorf("database %d: host is required when connection_string is not provided", i+1)
			}
			if db.Name == "" {
				return fmt.Errorf("database %d: name is required when connection_string is not provided", i+1)
			}
		}
	}

	// R2 settings are always required
	if c.R2AccountID == "" {
		return fmt.Errorf("r2_account_id is required")
	}
	if c.R2AccessKeyID == "" {
		return fmt.Errorf("r2_access_key_id is required")
	}
	if c.R2SecretAccessKey == "" {
		return fmt.Errorf("r2_secret_access_key is required")
	}
	if c.R2BucketName == "" {
		return fmt.Errorf("r2_bucket_name is required")
	}

	return nil
}

func (c *Config) HasEncryption() bool {
	return len(c.EncryptionKey) > 0
}

func (c *Config) HasRetention() bool {
	return c.RetentionDays > 0 || c.RetentionCount > 0
}

func getInput(name string) string {
	// First try regular env var (for local development)
	envName := strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
	if val := os.Getenv(envName); val != "" {
		return strings.TrimSpace(val)
	}
	// Fall back to INPUT_ prefixed (GitHub Actions convention)
	return strings.TrimSpace(os.Getenv("INPUT_" + envName))
}

func getInputInt(name string, defaultVal int) int {
	val := getInput(name)
	if val == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return i
}

func getInputBool(name string, defaultVal bool) bool {
	val := strings.ToLower(getInput(name))
	if val == "" {
		return defaultVal
	}
	return val == "true" || val == "yes" || val == "1"
}

func defaultPort(dbType DatabaseType) int {
	switch dbType {
	case DatabaseTypePostgres:
		return 5432
	case DatabaseTypeMySQL:
		return 3306
	case DatabaseTypeMongoDB:
		return 27017
	default:
		return 0
	}
}

func parseDatabaseNameFromConnectionString(connStr string) string {
	// Try to parse as URL
	u, err := url.Parse(connStr)
	if err != nil {
		return ""
	}

	// Database name is typically the path without leading slash
	dbName := strings.TrimPrefix(u.Path, "/")

	// Remove any query parameters (e.g., ?sslmode=require)
	if idx := strings.Index(dbName, "?"); idx >= 0 {
		dbName = dbName[:idx]
	}

	return dbName
}

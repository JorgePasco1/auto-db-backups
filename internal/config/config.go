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

type Config struct {
	// Database settings
	DatabaseType     DatabaseType
	DatabaseHost     string
	DatabasePort     int
	DatabaseName     string
	DatabaseUser     string
	DatabasePassword string
	ConnectionString string

	// R2 settings
	R2AccountID       string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2BucketName      string
	BackupPrefix      string

	// Backup settings
	Compression   bool
	EncryptionKey []byte

	// Retention settings
	RetentionDays  int
	RetentionCount int

	// Notification settings
	WebhookURL      string
	NotifyOnSuccess bool
	NotifyOnFailure bool
}

func Load() (*Config, error) {
	cfg := &Config{}

	// Debug: log environment variables for R2 (redacted)
	fmt.Printf("DEBUG: R2_ACCOUNT_ID length: %d\n", len(getInput("r2_account_id")))
	fmt.Printf("DEBUG: R2_BUCKET_NAME length: %d\n", len(getInput("r2_bucket_name")))

	// Database settings
	dbType := getInput("database_type")
	switch strings.ToLower(dbType) {
	case "postgres", "postgresql":
		cfg.DatabaseType = DatabaseTypePostgres
	case "mysql", "mariadb":
		cfg.DatabaseType = DatabaseTypeMySQL
	case "mongodb", "mongo":
		cfg.DatabaseType = DatabaseTypeMongoDB
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	cfg.DatabaseHost = getInput("database_host")
	cfg.DatabasePort = getInputInt("database_port", defaultPort(cfg.DatabaseType))
	cfg.DatabaseName = getInput("database_name")
	cfg.DatabaseUser = getInput("database_user")
	cfg.DatabasePassword = getInput("database_password")
	cfg.ConnectionString = getInput("connection_string")

	// If using connection string and database name is empty, try to parse it
	if cfg.ConnectionString != "" && cfg.DatabaseName == "" {
		cfg.DatabaseName = parseDatabaseNameFromConnectionString(cfg.ConnectionString)
	}

	// R2 settings
	cfg.R2AccountID = getInput("r2_account_id")
	cfg.R2AccessKeyID = getInput("r2_access_key_id")
	cfg.R2SecretAccessKey = getInput("r2_secret_access_key")
	cfg.R2BucketName = getInput("r2_bucket_name")
	cfg.BackupPrefix = getInput("backup_prefix")

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

func (c *Config) Validate() error {
	// Either connection string or individual params required
	if c.ConnectionString == "" {
		if c.DatabaseHost == "" {
			return fmt.Errorf("database_host is required when connection_string is not provided")
		}
		if c.DatabaseName == "" {
			return fmt.Errorf("database_name is required when connection_string is not provided")
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
	// GitHub Actions passes inputs as INPUT_<NAME> env vars (uppercase, underscores)
	envName := "INPUT_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
	return strings.TrimSpace(os.Getenv(envName))
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

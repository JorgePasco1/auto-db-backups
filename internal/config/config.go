package config

import (
	"encoding/base64"
	"encoding/json"
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

// DatabaseJSONEntry represents a single database in the DATABASES_JSON array
type DatabaseJSONEntry struct {
	Connection string `json:"connection"`
	Name       string `json:"name,omitempty"`
	Prefix     string `json:"prefix,omitempty"`
	Type       string `json:"type,omitempty"`
}

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

	// Determine global database type (used as default)
	dbType := getInput("database_type")
	var globalDBType DatabaseType
	switch strings.ToLower(dbType) {
	case "postgres", "postgresql", "":
		globalDBType = DatabaseTypePostgres // Default to postgres
	case "mysql", "mariadb":
		globalDBType = DatabaseTypeMySQL
	case "mongodb", "mongo":
		globalDBType = DatabaseTypeMongoDB
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	// Load database connections from JSON
	databases, err := loadDatabaseConfigs(globalDBType)
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

// loadDatabaseConfigs loads database configurations from DATABASES_JSON
func loadDatabaseConfigs(globalDBType DatabaseType) ([]DatabaseConfig, error) {
	jsonStr := getInput("databases_json")
	if jsonStr == "" {
		return nil, fmt.Errorf("DATABASES_JSON is required")
	}

	var entries []DatabaseJSONEntry
	if err := json.Unmarshal([]byte(jsonStr), &entries); err != nil {
		return nil, fmt.Errorf("invalid DATABASES_JSON: %w", err)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("DATABASES_JSON must contain at least one database")
	}

	var databases []DatabaseConfig
	for i, entry := range entries {
		if entry.Connection == "" {
			return nil, fmt.Errorf("database %d: connection is required", i+1)
		}

		// Determine database type for this entry
		dbType := globalDBType
		if entry.Type != "" {
			switch strings.ToLower(entry.Type) {
			case "postgres", "postgresql":
				dbType = DatabaseTypePostgres
			case "mysql", "mariadb":
				dbType = DatabaseTypeMySQL
			case "mongodb", "mongo":
				dbType = DatabaseTypeMongoDB
			default:
				return nil, fmt.Errorf("database %d: unsupported type: %s", i+1, entry.Type)
			}
		}

		// Parse connection string to extract components
		parsed, err := parseConnectionString(entry.Connection, dbType)
		if err != nil {
			return nil, fmt.Errorf("database %d: %w", i+1, err)
		}

		// Use custom name if provided, otherwise use parsed name
		dbName := entry.Name
		if dbName == "" {
			dbName = parsed.Name
		}

		// Build backup prefix
		prefix := entry.Prefix
		if prefix == "" {
			prefix = fmt.Sprintf("backups/%s/", dbName)
		}
		if prefix != "" && !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}

		databases = append(databases, DatabaseConfig{
			Type:             dbType,
			Host:             parsed.Host,
			Port:             parsed.Port,
			Name:             dbName,
			User:             parsed.User,
			Password:         parsed.Password,
			ConnectionString: entry.Connection,
			BackupPrefix:     prefix,
		})
	}

	return databases, nil
}

// parsedConnection holds components extracted from a connection string
type parsedConnection struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
}

// parseConnectionString extracts host, port, user, password, and database name from a connection URL
func parseConnectionString(connStr string, dbType DatabaseType) (*parsedConnection, error) {
	u, err := url.Parse(connStr)
	if err != nil {
		return nil, fmt.Errorf("invalid connection string: %w", err)
	}

	parsed := &parsedConnection{
		Port: defaultPort(dbType),
	}

	// Extract host and port
	parsed.Host = u.Hostname()
	if portStr := u.Port(); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			parsed.Port = port
		}
	}

	// Extract user and password
	if u.User != nil {
		parsed.User = u.User.Username()
		if pwd, ok := u.User.Password(); ok {
			parsed.Password = pwd
		}
	}

	// Extract database name from path
	parsed.Name = strings.TrimPrefix(u.Path, "/")
	// Remove any query parameters from database name (shouldn't happen with proper URL parsing, but be safe)
	if idx := strings.Index(parsed.Name, "?"); idx >= 0 {
		parsed.Name = parsed.Name[:idx]
	}

	return parsed, nil
}

func (c *Config) Validate() error {
	// Validate each database config
	for i, db := range c.Databases {
		if db.Name == "" {
			return fmt.Errorf("database %d: name could not be determined from connection string", i+1)
		}
		// For MySQL, we need host to be set since mysqldump doesn't accept connection URLs
		if db.Type == DatabaseTypeMySQL && db.Host == "" {
			return fmt.Errorf("database %d: host could not be parsed from connection string", i+1)
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

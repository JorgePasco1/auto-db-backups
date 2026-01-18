package config

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setTestEnv is a helper to set multiple environment variables for a test
func setTestEnv(t *testing.T, envVars map[string]string) {
	t.Helper()
	for key, value := range envVars {
		t.Setenv(key, value)
	}
}

// minimalValidEnv returns the minimum required environment variables for a valid config
func minimalValidEnv() map[string]string {
	return map[string]string{
		"INPUT_DATABASE_TYPE":        "postgres",
		"INPUT_DATABASE_HOST":        "localhost",
		"INPUT_DATABASE_NAME":        "testdb",
		"INPUT_R2_ACCOUNT_ID":        "account123",
		"INPUT_R2_ACCESS_KEY_ID":     "accesskey",
		"INPUT_R2_SECRET_ACCESS_KEY": "secretkey",
		"INPUT_R2_BUCKET_NAME":       "my-bucket",
	}
}

func TestLoad_MinimalValidConfig(t *testing.T) {
	setTestEnv(t, minimalValidEnv())

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, DatabaseTypePostgres, cfg.DatabaseType)
	assert.Equal(t, "localhost", cfg.DatabaseHost)
	assert.Equal(t, "testdb", cfg.DatabaseName)
	assert.Equal(t, 5432, cfg.DatabasePort) // default port
	assert.Equal(t, "account123", cfg.R2AccountID)
	assert.Equal(t, "accesskey", cfg.R2AccessKeyID)
	assert.Equal(t, "secretkey", cfg.R2SecretAccessKey)
	assert.Equal(t, "my-bucket", cfg.R2BucketName)
	assert.True(t, cfg.Compression)     // default true
	assert.True(t, cfg.NotifyOnSuccess) // default true
	assert.True(t, cfg.NotifyOnFailure) // default true
}

func TestLoad_DatabaseTypeNormalization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected DatabaseType
	}{
		{"postgres lowercase", "postgres", DatabaseTypePostgres},
		{"postgres uppercase", "POSTGRES", DatabaseTypePostgres},
		{"postgresql", "postgresql", DatabaseTypePostgres},
		{"PostgreSQL mixed case", "PostgreSQL", DatabaseTypePostgres},
		{"mysql lowercase", "mysql", DatabaseTypeMySQL},
		{"mysql uppercase", "MYSQL", DatabaseTypeMySQL},
		{"mariadb", "mariadb", DatabaseTypeMySQL},
		{"MariaDB mixed case", "MariaDB", DatabaseTypeMySQL},
		{"mongodb lowercase", "mongodb", DatabaseTypeMongoDB},
		{"mongodb uppercase", "MONGODB", DatabaseTypeMongoDB},
		{"mongo short", "mongo", DatabaseTypeMongoDB},
		{"Mongo mixed case", "Mongo", DatabaseTypeMongoDB},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := minimalValidEnv()
			env["INPUT_DATABASE_TYPE"] = tt.input
			setTestEnv(t, env)

			cfg, err := Load()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.DatabaseType)
		})
	}
}

func TestLoad_UnsupportedDatabaseType(t *testing.T) {
	tests := []string{
		"oracle",
		"sqlite",
		"sqlserver",
		"",
		"invalid",
	}

	for _, dbType := range tests {
		t.Run(dbType, func(t *testing.T) {
			env := minimalValidEnv()
			env["INPUT_DATABASE_TYPE"] = dbType
			setTestEnv(t, env)

			cfg, err := Load()
			assert.Error(t, err)
			assert.Nil(t, cfg)
			assert.Contains(t, err.Error(), "unsupported database type")
		})
	}
}

func TestLoad_DefaultPorts(t *testing.T) {
	tests := []struct {
		name         string
		dbType       string
		expectedPort int
	}{
		{"postgres default port", "postgres", 5432},
		{"mysql default port", "mysql", 3306},
		{"mongodb default port", "mongodb", 27017},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := minimalValidEnv()
			env["INPUT_DATABASE_TYPE"] = tt.dbType
			setTestEnv(t, env)

			cfg, err := Load()
			require.NoError(t, err)
			assert.Equal(t, tt.expectedPort, cfg.DatabasePort)
		})
	}
}

func TestLoad_CustomPort(t *testing.T) {
	env := minimalValidEnv()
	env["INPUT_DATABASE_PORT"] = "15432"
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, 15432, cfg.DatabasePort)
}

func TestLoad_InvalidPort(t *testing.T) {
	env := minimalValidEnv()
	env["INPUT_DATABASE_PORT"] = "not-a-number"
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	// Invalid port should fall back to default
	assert.Equal(t, 5432, cfg.DatabasePort)
}

func TestLoad_ConnectionString(t *testing.T) {
	env := map[string]string{
		"INPUT_DATABASE_TYPE":        "postgres",
		"INPUT_CONNECTION_STRING":    "postgres://user:pass@host:5432/db",
		"INPUT_R2_ACCOUNT_ID":        "account123",
		"INPUT_R2_ACCESS_KEY_ID":     "accesskey",
		"INPUT_R2_SECRET_ACCESS_KEY": "secretkey",
		"INPUT_R2_BUCKET_NAME":       "my-bucket",
	}
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "postgres://user:pass@host:5432/db", cfg.ConnectionString)
	// Host and name not required when connection string is provided
}

func TestLoad_EncryptionKey_Valid(t *testing.T) {
	env := minimalValidEnv()
	// Generate a valid 32-byte key
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	env["INPUT_ENCRYPTION_KEY"] = base64.StdEncoding.EncodeToString(key)
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, key, cfg.EncryptionKey)
	assert.True(t, cfg.HasEncryption())
}

func TestLoad_EncryptionKey_None(t *testing.T) {
	env := minimalValidEnv()
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Empty(t, cfg.EncryptionKey)
	assert.False(t, cfg.HasEncryption())
}

func TestLoad_EncryptionKey_InvalidBase64(t *testing.T) {
	env := minimalValidEnv()
	env["INPUT_ENCRYPTION_KEY"] = "not-valid-base64!!!"
	setTestEnv(t, env)

	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "must be base64 encoded")
}

func TestLoad_EncryptionKey_TooShort(t *testing.T) {
	env := minimalValidEnv()
	shortKey := make([]byte, 16) // Only 16 bytes instead of 32
	env["INPUT_ENCRYPTION_KEY"] = base64.StdEncoding.EncodeToString(shortKey)
	setTestEnv(t, env)

	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "must be exactly 32 bytes")
}

func TestLoad_EncryptionKey_TooLong(t *testing.T) {
	env := minimalValidEnv()
	longKey := make([]byte, 64) // 64 bytes instead of 32
	env["INPUT_ENCRYPTION_KEY"] = base64.StdEncoding.EncodeToString(longKey)
	setTestEnv(t, env)

	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "must be exactly 32 bytes")
}

func TestLoad_CompressionSettings(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"true lowercase", "true", true},
		{"TRUE uppercase", "TRUE", true},
		{"yes", "yes", true},
		{"YES uppercase", "YES", true},
		{"1", "1", true},
		{"false", "false", false},
		{"FALSE uppercase", "FALSE", false},
		{"no", "no", false},
		{"0", "0", false},
		{"empty defaults to true", "", true},
		{"random string", "random", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := minimalValidEnv()
			if tt.value != "" {
				env["INPUT_COMPRESSION"] = tt.value
			}
			setTestEnv(t, env)

			cfg, err := Load()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.Compression)
		})
	}
}

func TestLoad_RetentionSettings_None(t *testing.T) {
	env := minimalValidEnv()
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, 0, cfg.RetentionDays)
	assert.Equal(t, 0, cfg.RetentionCount)
	assert.False(t, cfg.HasRetention())
}

func TestLoad_RetentionSettings_DaysOnly(t *testing.T) {
	env := minimalValidEnv()
	env["INPUT_RETENTION_DAYS"] = "30"
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, 30, cfg.RetentionDays)
	assert.Equal(t, 0, cfg.RetentionCount)
	assert.True(t, cfg.HasRetention())
}

func TestLoad_RetentionSettings_CountOnly(t *testing.T) {
	env := minimalValidEnv()
	env["INPUT_RETENTION_COUNT"] = "10"
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, 0, cfg.RetentionDays)
	assert.Equal(t, 10, cfg.RetentionCount)
	assert.True(t, cfg.HasRetention())
}

func TestLoad_RetentionSettings_Both(t *testing.T) {
	env := minimalValidEnv()
	env["INPUT_RETENTION_DAYS"] = "30"
	env["INPUT_RETENTION_COUNT"] = "10"
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, 30, cfg.RetentionDays)
	assert.Equal(t, 10, cfg.RetentionCount)
	assert.True(t, cfg.HasRetention())
}

func TestLoad_RetentionSettings_InvalidDays(t *testing.T) {
	env := minimalValidEnv()
	env["INPUT_RETENTION_DAYS"] = "invalid"
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, 0, cfg.RetentionDays) // Falls back to default
}

func TestLoad_NotificationSettings_Default(t *testing.T) {
	env := minimalValidEnv()
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Empty(t, cfg.WebhookURL)
	assert.True(t, cfg.NotifyOnSuccess)
	assert.True(t, cfg.NotifyOnFailure)
}

func TestLoad_NotificationSettings_WebhookURL(t *testing.T) {
	env := minimalValidEnv()
	env["INPUT_WEBHOOK_URL"] = "https://hooks.example.com/webhook"
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "https://hooks.example.com/webhook", cfg.WebhookURL)
}

func TestLoad_NotificationSettings_SuccessDisabled(t *testing.T) {
	env := minimalValidEnv()
	env["INPUT_NOTIFY_ON_SUCCESS"] = "false"
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	assert.False(t, cfg.NotifyOnSuccess)
}

func TestLoad_NotificationSettings_FailureDisabled(t *testing.T) {
	env := minimalValidEnv()
	env["INPUT_NOTIFY_ON_FAILURE"] = "false"
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	assert.False(t, cfg.NotifyOnFailure)
}

func TestLoad_BackupPrefix_None(t *testing.T) {
	env := minimalValidEnv()
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Empty(t, cfg.BackupPrefix)
}

func TestLoad_BackupPrefix_WithPrefix(t *testing.T) {
	env := minimalValidEnv()
	env["INPUT_BACKUP_PREFIX"] = "myapp/prod/"
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "myapp/prod/", cfg.BackupPrefix)
}

func TestLoad_AllDatabaseCredentials(t *testing.T) {
	env := minimalValidEnv()
	env["INPUT_DATABASE_USER"] = "dbuser"
	env["INPUT_DATABASE_PASSWORD"] = "dbpass123"
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "dbuser", cfg.DatabaseUser)
	assert.Equal(t, "dbpass123", cfg.DatabasePassword)
}

func TestValidate_MissingR2AccountID(t *testing.T) {
	env := minimalValidEnv()
	delete(env, "INPUT_R2_ACCOUNT_ID")
	setTestEnv(t, env)

	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "r2_account_id is required")
}

func TestValidate_MissingR2AccessKeyID(t *testing.T) {
	env := minimalValidEnv()
	delete(env, "INPUT_R2_ACCESS_KEY_ID")
	setTestEnv(t, env)

	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "r2_access_key_id is required")
}

func TestValidate_MissingR2SecretAccessKey(t *testing.T) {
	env := minimalValidEnv()
	delete(env, "INPUT_R2_SECRET_ACCESS_KEY")
	setTestEnv(t, env)

	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "r2_secret_access_key is required")
}

func TestValidate_MissingR2BucketName(t *testing.T) {
	env := minimalValidEnv()
	delete(env, "INPUT_R2_BUCKET_NAME")
	setTestEnv(t, env)

	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "r2_bucket_name is required")
}

func TestValidate_MissingDatabaseHost(t *testing.T) {
	env := minimalValidEnv()
	delete(env, "INPUT_DATABASE_HOST")
	setTestEnv(t, env)

	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "database_host is required")
}

func TestValidate_MissingDatabaseName(t *testing.T) {
	env := minimalValidEnv()
	delete(env, "INPUT_DATABASE_NAME")
	setTestEnv(t, env)

	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "database_name is required")
}

func TestValidate_ConnectionStringAllowsMissingHostAndName(t *testing.T) {
	env := minimalValidEnv()
	delete(env, "INPUT_DATABASE_HOST")
	delete(env, "INPUT_DATABASE_NAME")
	env["INPUT_CONNECTION_STRING"] = "postgres://user:pass@host:5432/db"
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)
}

func TestGetInput_TrimsWhitespace(t *testing.T) {
	t.Setenv("INPUT_TEST_VALUE", "  trimmed  ")

	result := getInput("test_value")
	assert.Equal(t, "trimmed", result)
}

func TestGetInput_HyphenToUnderscore(t *testing.T) {
	t.Setenv("INPUT_SOME_HYPHENATED_VALUE", "test")

	result := getInput("some-hyphenated-value")
	assert.Equal(t, "test", result)
}

func TestDefaultPort_UnknownType(t *testing.T) {
	t.Parallel()

	// Test with an invalid/unknown database type
	result := defaultPort(DatabaseType("unknown"))
	assert.Equal(t, 0, result)
}

func TestGetInputBool_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		defaultVal bool
		expected   bool
	}{
		{"True capitalized", "True", false, true},
		{"tRuE mixed", "tRuE", false, true},
		{"Yes capitalized", "Yes", false, true},
		{"yEs mixed", "yEs", false, true},
		{"False capitalized", "False", true, false},
		{"No capitalized", "No", true, false},
		{"empty with default true", "", true, true},
		{"empty with default false", "", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				t.Setenv("INPUT_TEST_BOOL", tt.value)
			}

			result := getInputBool("test_bool", tt.defaultVal)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetInputInt_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		defaultVal int
		expected   int
	}{
		{"valid positive", "100", 0, 100},
		{"valid zero", "0", 5, 0},
		{"valid negative", "-1", 0, -1},
		{"invalid float", "1.5", 10, 10},
		{"invalid string", "abc", 10, 10},
		{"empty string", "", 42, 42},
		{"large number", "999999", 0, 999999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				t.Setenv("INPUT_TEST_INT", tt.value)
			}

			result := getInputInt("test_int", tt.defaultVal)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfig_FullConfig(t *testing.T) {
	// Test a complete configuration with all options set
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	env := map[string]string{
		"INPUT_DATABASE_TYPE":        "postgres",
		"INPUT_DATABASE_HOST":        "db.example.com",
		"INPUT_DATABASE_PORT":        "15432",
		"INPUT_DATABASE_NAME":        "mydb",
		"INPUT_DATABASE_USER":        "admin",
		"INPUT_DATABASE_PASSWORD":    "secret123",
		"INPUT_CONNECTION_STRING":    "", // not using connection string
		"INPUT_R2_ACCOUNT_ID":        "cf-account-123",
		"INPUT_R2_ACCESS_KEY_ID":     "cf-access-key",
		"INPUT_R2_SECRET_ACCESS_KEY": "cf-secret-key",
		"INPUT_R2_BUCKET_NAME":       "backups-bucket",
		"INPUT_BACKUP_PREFIX":        "prod/daily/",
		"INPUT_COMPRESSION":          "true",
		"INPUT_ENCRYPTION_KEY":       base64.StdEncoding.EncodeToString(key),
		"INPUT_RETENTION_DAYS":       "30",
		"INPUT_RETENTION_COUNT":      "10",
		"INPUT_WEBHOOK_URL":          "https://hooks.example.com/notify",
		"INPUT_NOTIFY_ON_SUCCESS":    "true",
		"INPUT_NOTIFY_ON_FAILURE":    "true",
	}
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify all fields
	assert.Equal(t, DatabaseTypePostgres, cfg.DatabaseType)
	assert.Equal(t, "db.example.com", cfg.DatabaseHost)
	assert.Equal(t, 15432, cfg.DatabasePort)
	assert.Equal(t, "mydb", cfg.DatabaseName)
	assert.Equal(t, "admin", cfg.DatabaseUser)
	assert.Equal(t, "secret123", cfg.DatabasePassword)
	assert.Empty(t, cfg.ConnectionString)
	assert.Equal(t, "cf-account-123", cfg.R2AccountID)
	assert.Equal(t, "cf-access-key", cfg.R2AccessKeyID)
	assert.Equal(t, "cf-secret-key", cfg.R2SecretAccessKey)
	assert.Equal(t, "backups-bucket", cfg.R2BucketName)
	assert.Equal(t, "prod/daily/", cfg.BackupPrefix)
	assert.True(t, cfg.Compression)
	assert.Equal(t, key, cfg.EncryptionKey)
	assert.True(t, cfg.HasEncryption())
	assert.Equal(t, 30, cfg.RetentionDays)
	assert.Equal(t, 10, cfg.RetentionCount)
	assert.True(t, cfg.HasRetention())
	assert.Equal(t, "https://hooks.example.com/notify", cfg.WebhookURL)
	assert.True(t, cfg.NotifyOnSuccess)
	assert.True(t, cfg.NotifyOnFailure)
}

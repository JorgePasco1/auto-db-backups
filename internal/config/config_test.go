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
		"DATABASES_JSON":       `[{"connection": "postgres://user:pass@localhost:5432/testdb"}]`,
		"R2_ACCOUNT_ID":        "account123",
		"R2_ACCESS_KEY_ID":     "accesskey",
		"R2_SECRET_ACCESS_KEY": "secretkey",
		"R2_BUCKET_NAME":       "my-bucket",
	}
}

func TestLoad_MinimalValidConfig(t *testing.T) {
	setTestEnv(t, minimalValidEnv())

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Len(t, cfg.Databases, 1)
	assert.Equal(t, DatabaseTypePostgres, cfg.Databases[0].Type)
	assert.Equal(t, "testdb", cfg.Databases[0].Name)
	assert.Equal(t, "localhost", cfg.Databases[0].Host)
	assert.Equal(t, 5432, cfg.Databases[0].Port)
	assert.Equal(t, "user", cfg.Databases[0].User)
	assert.Equal(t, "pass", cfg.Databases[0].Password)
	assert.Equal(t, "backups/testdb/", cfg.Databases[0].BackupPrefix)
	assert.Equal(t, "account123", cfg.R2AccountID)
	assert.Equal(t, "accesskey", cfg.R2AccessKeyID)
	assert.Equal(t, "secretkey", cfg.R2SecretAccessKey)
	assert.Equal(t, "my-bucket", cfg.R2BucketName)
	assert.True(t, cfg.Compression)     // default true
	assert.True(t, cfg.NotifyOnSuccess) // default true
	assert.True(t, cfg.NotifyOnFailure) // default true
}

func TestLoad_MultipleConnections(t *testing.T) {
	env := map[string]string{
		"DATABASES_JSON": `[
			{"connection": "postgres://user:pass@host1:5432/db1"},
			{"connection": "postgres://user:pass@host2:5432/db2"},
			{"connection": "postgres://user:pass@host3:5432/db3"}
		]`,
		"R2_ACCOUNT_ID":        "account123",
		"R2_ACCESS_KEY_ID":     "accesskey",
		"R2_SECRET_ACCESS_KEY": "secretkey",
		"R2_BUCKET_NAME":       "my-bucket",
	}
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	require.Len(t, cfg.Databases, 3)

	assert.Equal(t, "db1", cfg.Databases[0].Name)
	assert.Equal(t, "host1", cfg.Databases[0].Host)
	assert.Equal(t, "db2", cfg.Databases[1].Name)
	assert.Equal(t, "host2", cfg.Databases[1].Host)
	assert.Equal(t, "db3", cfg.Databases[2].Name)
	assert.Equal(t, "host3", cfg.Databases[2].Host)
}

func TestLoad_WithCustomNames(t *testing.T) {
	env := map[string]string{
		"DATABASES_JSON": `[
			{"connection": "postgres://user:pass@host:5432/neondb", "name": "users-db"},
			{"connection": "postgres://user:pass@host:5432/neondb", "name": "orders-db"}
		]`,
		"R2_ACCOUNT_ID":        "account123",
		"R2_ACCESS_KEY_ID":     "accesskey",
		"R2_SECRET_ACCESS_KEY": "secretkey",
		"R2_BUCKET_NAME":       "my-bucket",
	}
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	require.Len(t, cfg.Databases, 2)

	assert.Equal(t, "users-db", cfg.Databases[0].Name)
	assert.Equal(t, "backups/users-db/", cfg.Databases[0].BackupPrefix)
	assert.Equal(t, "orders-db", cfg.Databases[1].Name)
	assert.Equal(t, "backups/orders-db/", cfg.Databases[1].BackupPrefix)
}

func TestLoad_WithCustomPrefixes(t *testing.T) {
	env := map[string]string{
		"DATABASES_JSON": `[
			{"connection": "postgres://user:pass@host:5432/db1", "prefix": "prod/daily/"},
			{"connection": "postgres://user:pass@host:5432/db2", "prefix": "staging"}
		]`,
		"R2_ACCOUNT_ID":        "account123",
		"R2_ACCESS_KEY_ID":     "accesskey",
		"R2_SECRET_ACCESS_KEY": "secretkey",
		"R2_BUCKET_NAME":       "my-bucket",
	}
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	require.Len(t, cfg.Databases, 2)

	assert.Equal(t, "prod/daily/", cfg.Databases[0].BackupPrefix)
	assert.Equal(t, "staging/", cfg.Databases[1].BackupPrefix) // Should add trailing slash
}

func TestLoad_WithPerDatabaseTypes(t *testing.T) {
	env := map[string]string{
		"DATABASES_JSON": `[
			{"connection": "postgres://user:pass@host1:5432/db1"},
			{"connection": "mysql://user:pass@host2:3306/db2", "type": "mysql"},
			{"connection": "mongodb://user:pass@host3:27017/db3", "type": "mongodb"}
		]`,
		"R2_ACCOUNT_ID":        "account123",
		"R2_ACCESS_KEY_ID":     "accesskey",
		"R2_SECRET_ACCESS_KEY": "secretkey",
		"R2_BUCKET_NAME":       "my-bucket",
	}
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	require.Len(t, cfg.Databases, 3)

	assert.Equal(t, DatabaseTypePostgres, cfg.Databases[0].Type)
	assert.Equal(t, DatabaseTypeMySQL, cfg.Databases[1].Type)
	assert.Equal(t, DatabaseTypeMongoDB, cfg.Databases[2].Type)
}

func TestLoad_InputPrefixedEnvVars(t *testing.T) {
	// Test that INPUT_ prefixed env vars (GitHub Actions) work
	env := map[string]string{
		"INPUT_DATABASES_JSON":       `[{"connection": "postgres://user:pass@localhost:5432/testdb"}]`,
		"INPUT_R2_ACCOUNT_ID":        "account123",
		"INPUT_R2_ACCESS_KEY_ID":     "accesskey",
		"INPUT_R2_SECRET_ACCESS_KEY": "secretkey",
		"INPUT_R2_BUCKET_NAME":       "my-bucket",
	}
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Databases, 1)
	assert.Equal(t, "testdb", cfg.Databases[0].Name)
}

func TestLoad_RegularEnvTakesPrecedence(t *testing.T) {
	env := map[string]string{
		"DATABASES_JSON":       `[{"connection": "postgres://user:pass@localhost:5432/regular"}]`,
		"INPUT_DATABASES_JSON": `[{"connection": "postgres://user:pass@localhost:5432/input"}]`,
		"R2_ACCOUNT_ID":        "account123",
		"R2_ACCESS_KEY_ID":     "accesskey",
		"R2_SECRET_ACCESS_KEY": "secretkey",
		"R2_BUCKET_NAME":       "my-bucket",
	}
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	require.Len(t, cfg.Databases, 1)
	assert.Equal(t, "regular", cfg.Databases[0].Name)
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
		{"empty defaults to postgres", "", DatabaseTypePostgres},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := minimalValidEnv()
			env["DATABASE_TYPE"] = tt.input
			setTestEnv(t, env)

			cfg, err := Load()
			require.NoError(t, err)
			require.Len(t, cfg.Databases, 1)
			assert.Equal(t, tt.expected, cfg.Databases[0].Type)
		})
	}
}

func TestLoad_UnsupportedDatabaseType(t *testing.T) {
	tests := []string{
		"oracle",
		"sqlite",
		"sqlserver",
		"invalid",
	}

	for _, dbType := range tests {
		t.Run(dbType, func(t *testing.T) {
			env := minimalValidEnv()
			env["DATABASE_TYPE"] = dbType
			setTestEnv(t, env)

			cfg, err := Load()
			assert.Error(t, err)
			assert.Nil(t, cfg)
			assert.Contains(t, err.Error(), "unsupported database type")
		})
	}
}

func TestLoad_UnsupportedPerDatabaseType(t *testing.T) {
	env := map[string]string{
		"DATABASES_JSON": `[
			{"connection": "postgres://user:pass@host:5432/db", "type": "oracle"}
		]`,
		"R2_ACCOUNT_ID":        "account123",
		"R2_ACCESS_KEY_ID":     "accesskey",
		"R2_SECRET_ACCESS_KEY": "secretkey",
		"R2_BUCKET_NAME":       "my-bucket",
	}
	setTestEnv(t, env)

	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "unsupported type")
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
			env := map[string]string{
				"DATABASE_TYPE":        tt.dbType,
				"DATABASES_JSON":       `[{"connection": "` + tt.dbType + `://user:pass@localhost/testdb"}]`,
				"R2_ACCOUNT_ID":        "account123",
				"R2_ACCESS_KEY_ID":     "accesskey",
				"R2_SECRET_ACCESS_KEY": "secretkey",
				"R2_BUCKET_NAME":       "my-bucket",
			}
			setTestEnv(t, env)

			cfg, err := Load()
			require.NoError(t, err)
			require.Len(t, cfg.Databases, 1)
			assert.Equal(t, tt.expectedPort, cfg.Databases[0].Port)
		})
	}
}

func TestLoad_CustomPort(t *testing.T) {
	env := map[string]string{
		"DATABASES_JSON":       `[{"connection": "postgres://user:pass@localhost:15432/testdb"}]`,
		"R2_ACCOUNT_ID":        "account123",
		"R2_ACCESS_KEY_ID":     "accesskey",
		"R2_SECRET_ACCESS_KEY": "secretkey",
		"R2_BUCKET_NAME":       "my-bucket",
	}
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	require.Len(t, cfg.Databases, 1)
	assert.Equal(t, 15432, cfg.Databases[0].Port)
}

func TestLoad_ConnectionStringParsing_Postgres(t *testing.T) {
	env := map[string]string{
		"DATABASES_JSON":       `[{"connection": "postgresql://admin:secret123@db.example.com:5432/mydb?sslmode=require"}]`,
		"R2_ACCOUNT_ID":        "account123",
		"R2_ACCESS_KEY_ID":     "accesskey",
		"R2_SECRET_ACCESS_KEY": "secretkey",
		"R2_BUCKET_NAME":       "my-bucket",
	}
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	require.Len(t, cfg.Databases, 1)

	db := cfg.Databases[0]
	assert.Equal(t, "db.example.com", db.Host)
	assert.Equal(t, 5432, db.Port)
	assert.Equal(t, "admin", db.User)
	assert.Equal(t, "secret123", db.Password)
	assert.Equal(t, "mydb", db.Name)
}

func TestLoad_ConnectionStringParsing_MySQL(t *testing.T) {
	env := map[string]string{
		"DATABASE_TYPE":        "mysql",
		"DATABASES_JSON":       `[{"connection": "mysql://root:password@mysql.example.com:3306/appdb"}]`,
		"R2_ACCOUNT_ID":        "account123",
		"R2_ACCESS_KEY_ID":     "accesskey",
		"R2_SECRET_ACCESS_KEY": "secretkey",
		"R2_BUCKET_NAME":       "my-bucket",
	}
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	require.Len(t, cfg.Databases, 1)

	db := cfg.Databases[0]
	assert.Equal(t, DatabaseTypeMySQL, db.Type)
	assert.Equal(t, "mysql.example.com", db.Host)
	assert.Equal(t, 3306, db.Port)
	assert.Equal(t, "root", db.User)
	assert.Equal(t, "password", db.Password)
	assert.Equal(t, "appdb", db.Name)
}

func TestLoad_ConnectionStringParsing_MongoDB(t *testing.T) {
	env := map[string]string{
		"DATABASE_TYPE":        "mongodb",
		"DATABASES_JSON":       `[{"connection": "mongodb://admin:mongopass@mongo.example.com:27017/analytics"}]`,
		"R2_ACCOUNT_ID":        "account123",
		"R2_ACCESS_KEY_ID":     "accesskey",
		"R2_SECRET_ACCESS_KEY": "secretkey",
		"R2_BUCKET_NAME":       "my-bucket",
	}
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	require.Len(t, cfg.Databases, 1)

	db := cfg.Databases[0]
	assert.Equal(t, DatabaseTypeMongoDB, db.Type)
	assert.Equal(t, "mongo.example.com", db.Host)
	assert.Equal(t, 27017, db.Port)
	assert.Equal(t, "admin", db.User)
	assert.Equal(t, "mongopass", db.Password)
	assert.Equal(t, "analytics", db.Name)
}

func TestLoad_ConnectionStringParsing_MongoDBSRV(t *testing.T) {
	env := map[string]string{
		"DATABASE_TYPE":        "mongodb",
		"DATABASES_JSON":       `[{"connection": "mongodb+srv://admin:mongopass@cluster0.example.mongodb.net/mydb"}]`,
		"R2_ACCOUNT_ID":        "account123",
		"R2_ACCESS_KEY_ID":     "accesskey",
		"R2_SECRET_ACCESS_KEY": "secretkey",
		"R2_BUCKET_NAME":       "my-bucket",
	}
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	require.Len(t, cfg.Databases, 1)

	db := cfg.Databases[0]
	assert.Equal(t, "cluster0.example.mongodb.net", db.Host)
	assert.Equal(t, "mydb", db.Name)
}

func TestLoad_ConnectionStringParsing_NoPort(t *testing.T) {
	env := map[string]string{
		"DATABASES_JSON":       `[{"connection": "postgres://user:pass@localhost/testdb"}]`,
		"R2_ACCOUNT_ID":        "account123",
		"R2_ACCESS_KEY_ID":     "accesskey",
		"R2_SECRET_ACCESS_KEY": "secretkey",
		"R2_BUCKET_NAME":       "my-bucket",
	}
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	require.Len(t, cfg.Databases, 1)
	assert.Equal(t, 5432, cfg.Databases[0].Port) // Default port
}

func TestLoad_ConnectionStringParsing_NoPassword(t *testing.T) {
	env := map[string]string{
		"DATABASES_JSON":       `[{"connection": "postgres://user@localhost:5432/testdb"}]`,
		"R2_ACCOUNT_ID":        "account123",
		"R2_ACCESS_KEY_ID":     "accesskey",
		"R2_SECRET_ACCESS_KEY": "secretkey",
		"R2_BUCKET_NAME":       "my-bucket",
	}
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	require.Len(t, cfg.Databases, 1)
	assert.Equal(t, "user", cfg.Databases[0].User)
	assert.Equal(t, "", cfg.Databases[0].Password)
}

func TestLoad_EncryptionKey_Valid(t *testing.T) {
	env := minimalValidEnv()
	// Generate a valid 32-byte key
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	env["ENCRYPTION_KEY"] = base64.StdEncoding.EncodeToString(key)
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
	env["ENCRYPTION_KEY"] = "not-valid-base64!!!"
	setTestEnv(t, env)

	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "must be base64 encoded")
}

func TestLoad_EncryptionKey_TooShort(t *testing.T) {
	env := minimalValidEnv()
	shortKey := make([]byte, 16) // Only 16 bytes instead of 32
	env["ENCRYPTION_KEY"] = base64.StdEncoding.EncodeToString(shortKey)
	setTestEnv(t, env)

	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "must be exactly 32 bytes")
}

func TestLoad_EncryptionKey_TooLong(t *testing.T) {
	env := minimalValidEnv()
	longKey := make([]byte, 64) // 64 bytes instead of 32
	env["ENCRYPTION_KEY"] = base64.StdEncoding.EncodeToString(longKey)
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
				env["COMPRESSION"] = tt.value
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
	env["RETENTION_DAYS"] = "30"
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, 30, cfg.RetentionDays)
	assert.Equal(t, 0, cfg.RetentionCount)
	assert.True(t, cfg.HasRetention())
}

func TestLoad_RetentionSettings_CountOnly(t *testing.T) {
	env := minimalValidEnv()
	env["RETENTION_COUNT"] = "10"
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, 0, cfg.RetentionDays)
	assert.Equal(t, 10, cfg.RetentionCount)
	assert.True(t, cfg.HasRetention())
}

func TestLoad_RetentionSettings_Both(t *testing.T) {
	env := minimalValidEnv()
	env["RETENTION_DAYS"] = "30"
	env["RETENTION_COUNT"] = "10"
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, 30, cfg.RetentionDays)
	assert.Equal(t, 10, cfg.RetentionCount)
	assert.True(t, cfg.HasRetention())
}

func TestLoad_RetentionSettings_InvalidDays(t *testing.T) {
	env := minimalValidEnv()
	env["RETENTION_DAYS"] = "invalid"
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
	env["WEBHOOK_URL"] = "https://hooks.example.com/webhook"
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "https://hooks.example.com/webhook", cfg.WebhookURL)
}

func TestLoad_NotificationSettings_SuccessDisabled(t *testing.T) {
	env := minimalValidEnv()
	env["NOTIFY_ON_SUCCESS"] = "false"
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	assert.False(t, cfg.NotifyOnSuccess)
}

func TestLoad_NotificationSettings_FailureDisabled(t *testing.T) {
	env := minimalValidEnv()
	env["NOTIFY_ON_FAILURE"] = "false"
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	assert.False(t, cfg.NotifyOnFailure)
}

func TestValidate_MissingR2AccountID(t *testing.T) {
	env := minimalValidEnv()
	delete(env, "R2_ACCOUNT_ID")
	setTestEnv(t, env)

	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "r2_account_id is required")
}

func TestValidate_MissingR2AccessKeyID(t *testing.T) {
	env := minimalValidEnv()
	delete(env, "R2_ACCESS_KEY_ID")
	setTestEnv(t, env)

	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "r2_access_key_id is required")
}

func TestValidate_MissingR2SecretAccessKey(t *testing.T) {
	env := minimalValidEnv()
	delete(env, "R2_SECRET_ACCESS_KEY")
	setTestEnv(t, env)

	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "r2_secret_access_key is required")
}

func TestValidate_MissingR2BucketName(t *testing.T) {
	env := minimalValidEnv()
	delete(env, "R2_BUCKET_NAME")
	setTestEnv(t, env)

	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "r2_bucket_name is required")
}

func TestLoad_MissingDatabasesJSON(t *testing.T) {
	env := map[string]string{
		"R2_ACCOUNT_ID":        "account123",
		"R2_ACCESS_KEY_ID":     "accesskey",
		"R2_SECRET_ACCESS_KEY": "secretkey",
		"R2_BUCKET_NAME":       "my-bucket",
	}
	setTestEnv(t, env)

	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "DATABASES_JSON is required")
}

func TestLoad_EmptyDatabasesJSON(t *testing.T) {
	env := map[string]string{
		"DATABASES_JSON":       `[]`,
		"R2_ACCOUNT_ID":        "account123",
		"R2_ACCESS_KEY_ID":     "accesskey",
		"R2_SECRET_ACCESS_KEY": "secretkey",
		"R2_BUCKET_NAME":       "my-bucket",
	}
	setTestEnv(t, env)

	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "DATABASES_JSON must contain at least one database")
}

func TestLoad_InvalidJSON(t *testing.T) {
	env := map[string]string{
		"DATABASES_JSON":       `not valid json`,
		"R2_ACCOUNT_ID":        "account123",
		"R2_ACCESS_KEY_ID":     "accesskey",
		"R2_SECRET_ACCESS_KEY": "secretkey",
		"R2_BUCKET_NAME":       "my-bucket",
	}
	setTestEnv(t, env)

	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "invalid DATABASES_JSON")
}

func TestLoad_MissingConnectionField(t *testing.T) {
	env := map[string]string{
		"DATABASES_JSON":       `[{"name": "mydb"}]`,
		"R2_ACCOUNT_ID":        "account123",
		"R2_ACCESS_KEY_ID":     "accesskey",
		"R2_SECRET_ACCESS_KEY": "secretkey",
		"R2_BUCKET_NAME":       "my-bucket",
	}
	setTestEnv(t, env)

	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "connection is required")
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

func TestGetInput_RegularEnvTakesPrecedence(t *testing.T) {
	t.Setenv("TEST_VAR", "regular")
	t.Setenv("INPUT_TEST_VAR", "input_prefixed")

	result := getInput("test_var")
	assert.Equal(t, "regular", result)
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
		"DATABASES_JSON": `[
			{"connection": "postgres://admin:secret123@db.example.com:15432/mydb", "name": "prod-db", "prefix": "prod/daily/"}
		]`,
		"R2_ACCOUNT_ID":        "cf-account-123",
		"R2_ACCESS_KEY_ID":     "cf-access-key",
		"R2_SECRET_ACCESS_KEY": "cf-secret-key",
		"R2_BUCKET_NAME":       "backups-bucket",
		"COMPRESSION":          "true",
		"ENCRYPTION_KEY":       base64.StdEncoding.EncodeToString(key),
		"RETENTION_DAYS":       "30",
		"RETENTION_COUNT":      "10",
		"WEBHOOK_URL":          "https://hooks.example.com/notify",
		"NOTIFY_ON_SUCCESS":    "true",
		"NOTIFY_ON_FAILURE":    "true",
	}
	setTestEnv(t, env)

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify database config
	require.Len(t, cfg.Databases, 1)
	db := cfg.Databases[0]
	assert.Equal(t, DatabaseTypePostgres, db.Type)
	assert.Equal(t, "prod-db", db.Name)
	assert.Equal(t, "db.example.com", db.Host)
	assert.Equal(t, 15432, db.Port)
	assert.Equal(t, "admin", db.User)
	assert.Equal(t, "secret123", db.Password)
	assert.Equal(t, "prod/daily/", db.BackupPrefix)

	// Verify R2 config
	assert.Equal(t, "cf-account-123", cfg.R2AccountID)
	assert.Equal(t, "cf-access-key", cfg.R2AccessKeyID)
	assert.Equal(t, "cf-secret-key", cfg.R2SecretAccessKey)
	assert.Equal(t, "backups-bucket", cfg.R2BucketName)

	// Verify backup settings
	assert.True(t, cfg.Compression)
	assert.Equal(t, key, cfg.EncryptionKey)
	assert.True(t, cfg.HasEncryption())

	// Verify retention
	assert.Equal(t, 30, cfg.RetentionDays)
	assert.Equal(t, 10, cfg.RetentionCount)
	assert.True(t, cfg.HasRetention())

	// Verify notifications
	assert.Equal(t, "https://hooks.example.com/notify", cfg.WebhookURL)
	assert.True(t, cfg.NotifyOnSuccess)
	assert.True(t, cfg.NotifyOnFailure)
}

func TestParseConnectionString_ValidPostgres(t *testing.T) {
	parsed, err := parseConnectionString("postgres://user:pass@host:5432/dbname", DatabaseTypePostgres)
	require.NoError(t, err)
	assert.Equal(t, "host", parsed.Host)
	assert.Equal(t, 5432, parsed.Port)
	assert.Equal(t, "user", parsed.User)
	assert.Equal(t, "pass", parsed.Password)
	assert.Equal(t, "dbname", parsed.Name)
}

func TestParseConnectionString_WithQueryParams(t *testing.T) {
	parsed, err := parseConnectionString("postgres://user:pass@host:5432/dbname?sslmode=require&timeout=30", DatabaseTypePostgres)
	require.NoError(t, err)
	assert.Equal(t, "dbname", parsed.Name)
}

func TestParseConnectionString_SpecialCharsInPassword(t *testing.T) {
	// URL-encoded password with special characters
	parsed, err := parseConnectionString("postgres://user:p%40ss%3Aw%2Frd@host:5432/dbname", DatabaseTypePostgres)
	require.NoError(t, err)
	assert.Equal(t, "p@ss:w/rd", parsed.Password)
}

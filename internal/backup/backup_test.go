package backup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jorgepascosoto/auto-db-backups/internal/config"
)

// createTestConfig creates a config for testing with the specified database type
func createTestConfig(dbType config.DatabaseType) *config.Config {
	return &config.Config{
		DatabaseType:      dbType,
		DatabaseHost:      "localhost",
		DatabasePort:      5432,
		DatabaseName:      "testdb",
		DatabaseUser:      "testuser",
		DatabasePassword:  "testpass",
		ConnectionString:  "",
		R2AccountID:       "account",
		R2AccessKeyID:     "accesskey",
		R2SecretAccessKey: "secretkey",
		R2BucketName:      "bucket",
	}
}

// Tests for Factory
func TestNewExporter_Postgres(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(config.DatabaseTypePostgres)
	exporter, err := NewExporter(cfg)

	require.NoError(t, err)
	require.NotNil(t, exporter)

	_, ok := exporter.(*PostgresExporter)
	assert.True(t, ok, "Should return a PostgresExporter")
}

func TestNewExporter_MySQL(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(config.DatabaseTypeMySQL)
	cfg.DatabasePort = 3306
	exporter, err := NewExporter(cfg)

	require.NoError(t, err)
	require.NotNil(t, exporter)

	_, ok := exporter.(*MySQLExporter)
	assert.True(t, ok, "Should return a MySQLExporter")
}

func TestNewExporter_MongoDB(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(config.DatabaseTypeMongoDB)
	cfg.DatabasePort = 27017
	exporter, err := NewExporter(cfg)

	require.NoError(t, err)
	require.NotNil(t, exporter)

	_, ok := exporter.(*MongoDBExporter)
	assert.True(t, ok, "Should return a MongoDBExporter")
}

func TestNewExporter_UnsupportedType(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(config.DatabaseType("oracle"))
	exporter, err := NewExporter(cfg)

	assert.Error(t, err)
	assert.Nil(t, exporter)
	assert.Contains(t, err.Error(), "unsupported database type")
}

// Tests for PostgresExporter
func TestNewPostgresExporter(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(config.DatabaseTypePostgres)
	exporter := NewPostgresExporter(cfg)

	require.NotNil(t, exporter)
	assert.Equal(t, cfg, exporter.cfg)
}

func TestPostgresExporter_DatabaseName(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(config.DatabaseTypePostgres)
	cfg.DatabaseName = "mypostgresdb"
	exporter := NewPostgresExporter(cfg)

	assert.Equal(t, "mypostgresdb", exporter.DatabaseName())
}

func TestPostgresExporter_DatabaseType(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(config.DatabaseTypePostgres)
	exporter := NewPostgresExporter(cfg)

	assert.Equal(t, "postgres", exporter.DatabaseType())
}

func TestPostgresExporter_BuildArgs_WithConnectionString(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(config.DatabaseTypePostgres)
	cfg.ConnectionString = "postgres://user:pass@host:5432/db"
	exporter := NewPostgresExporter(cfg)

	args := exporter.buildArgs()

	assert.Contains(t, args, cfg.ConnectionString)
	assert.Contains(t, args, "--format=custom")
	// Should NOT contain individual params when connection string is used
	for _, arg := range args {
		assert.NotContains(t, arg, "--host=")
	}
}

func TestPostgresExporter_BuildArgs_WithIndividualParams(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(config.DatabaseTypePostgres)
	cfg.DatabaseHost = "db.example.com"
	cfg.DatabasePort = 15432
	cfg.DatabaseName = "proddb"
	cfg.DatabaseUser = "admin"
	exporter := NewPostgresExporter(cfg)

	args := exporter.buildArgs()

	assert.Contains(t, args, "--format=custom")
	assert.Contains(t, args, "--no-password")
	assert.Contains(t, args, "--host=db.example.com")
	assert.Contains(t, args, "--port=15432")
	assert.Contains(t, args, "--dbname=proddb")
	assert.Contains(t, args, "--username=admin")
}

func TestPostgresExporter_BuildArgs_WithoutUser(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(config.DatabaseTypePostgres)
	cfg.DatabaseUser = ""
	exporter := NewPostgresExporter(cfg)

	args := exporter.buildArgs()

	for _, arg := range args {
		assert.NotContains(t, arg, "--username=")
	}
}

func TestPostgresExporter_BuildEnv_WithPassword(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(config.DatabaseTypePostgres)
	cfg.DatabasePassword = "secret123"
	exporter := NewPostgresExporter(cfg)

	env := exporter.buildEnv()

	found := false
	for _, e := range env {
		if e == "PGPASSWORD=secret123" {
			found = true
			break
		}
	}
	assert.True(t, found, "PGPASSWORD should be set in environment")
}

func TestPostgresExporter_BuildEnv_WithoutPassword(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(config.DatabaseTypePostgres)
	cfg.DatabasePassword = ""
	exporter := NewPostgresExporter(cfg)

	env := exporter.buildEnv()

	for _, e := range env {
		assert.NotContains(t, e, "PGPASSWORD=")
	}
}

// Tests for MySQLExporter
func TestNewMySQLExporter(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(config.DatabaseTypeMySQL)
	exporter := NewMySQLExporter(cfg)

	require.NotNil(t, exporter)
	assert.Equal(t, cfg, exporter.cfg)
}

func TestMySQLExporter_DatabaseName(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(config.DatabaseTypeMySQL)
	cfg.DatabaseName = "mymysqldb"
	exporter := NewMySQLExporter(cfg)

	assert.Equal(t, "mymysqldb", exporter.DatabaseName())
}

func TestMySQLExporter_DatabaseType(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(config.DatabaseTypeMySQL)
	exporter := NewMySQLExporter(cfg)

	assert.Equal(t, "mysql", exporter.DatabaseType())
}

func TestMySQLExporter_BuildArgs(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(config.DatabaseTypeMySQL)
	cfg.DatabaseHost = "mysql.example.com"
	cfg.DatabasePort = 3307
	cfg.DatabaseName = "mydb"
	cfg.DatabaseUser = "root"
	cfg.DatabasePassword = "rootpass"
	exporter := NewMySQLExporter(cfg)

	args := exporter.buildArgs()

	// Check standard mysqldump flags
	assert.Contains(t, args, "--single-transaction")
	assert.Contains(t, args, "--routines")
	assert.Contains(t, args, "--triggers")
	assert.Contains(t, args, "--events")
	assert.Contains(t, args, "--host=mysql.example.com")
	assert.Contains(t, args, "--port=3307")
	assert.Contains(t, args, "--user=root")
	assert.Contains(t, args, "--password=rootpass")
	// Database name should be the last argument
	assert.Equal(t, "mydb", args[len(args)-1])
}

func TestMySQLExporter_BuildArgs_WithoutUserAndPassword(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(config.DatabaseTypeMySQL)
	cfg.DatabaseUser = ""
	cfg.DatabasePassword = ""
	exporter := NewMySQLExporter(cfg)

	args := exporter.buildArgs()

	for _, arg := range args {
		assert.NotContains(t, arg, "--user=")
		assert.NotContains(t, arg, "--password=")
	}
}

// Tests for MongoDBExporter
func TestNewMongoDBExporter(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(config.DatabaseTypeMongoDB)
	exporter := NewMongoDBExporter(cfg)

	require.NotNil(t, exporter)
	assert.Equal(t, cfg, exporter.cfg)
}

func TestMongoDBExporter_DatabaseName(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(config.DatabaseTypeMongoDB)
	cfg.DatabaseName = "mymongoDb"
	exporter := NewMongoDBExporter(cfg)

	assert.Equal(t, "mymongoDb", exporter.DatabaseName())
}

func TestMongoDBExporter_DatabaseType(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(config.DatabaseTypeMongoDB)
	exporter := NewMongoDBExporter(cfg)

	assert.Equal(t, "mongodb", exporter.DatabaseType())
}

func TestMongoDBExporter_BuildArgs_WithConnectionString(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(config.DatabaseTypeMongoDB)
	cfg.ConnectionString = "mongodb://user:pass@host:27017/db"
	exporter := NewMongoDBExporter(cfg)

	args := exporter.buildArgs("/tmp/output")

	assert.Contains(t, args, "--uri=mongodb://user:pass@host:27017/db")
	assert.Contains(t, args, "--out=/tmp/output")
	// Should NOT contain individual params when connection string is used
	for _, arg := range args {
		assert.NotContains(t, arg, "--host=")
	}
}

func TestMongoDBExporter_BuildArgs_WithIndividualParams(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(config.DatabaseTypeMongoDB)
	cfg.DatabaseHost = "mongo.example.com"
	cfg.DatabasePort = 27018
	cfg.DatabaseName = "analytics"
	cfg.DatabaseUser = "mongouser"
	cfg.DatabasePassword = "mongopass"
	exporter := NewMongoDBExporter(cfg)

	args := exporter.buildArgs("/var/dump")

	assert.Contains(t, args, "--host=mongo.example.com")
	assert.Contains(t, args, "--port=27018")
	assert.Contains(t, args, "--db=analytics")
	assert.Contains(t, args, "--out=/var/dump")
	assert.Contains(t, args, "--username=mongouser")
	assert.Contains(t, args, "--password=mongopass")
}

func TestMongoDBExporter_BuildArgs_WithoutCredentials(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(config.DatabaseTypeMongoDB)
	cfg.DatabaseUser = ""
	cfg.DatabasePassword = ""
	exporter := NewMongoDBExporter(cfg)

	args := exporter.buildArgs("/tmp/out")

	for _, arg := range args {
		assert.NotContains(t, arg, "--username=")
		assert.NotContains(t, arg, "--password=")
	}
}

// Tests for Exporter interface compliance
func TestExporter_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(config.DatabaseTypePostgres)

	// Verify all exporters implement the Exporter interface
	var _ Exporter = NewPostgresExporter(cfg)
	var _ Exporter = NewMySQLExporter(cfg)
	var _ Exporter = NewMongoDBExporter(cfg)
}

// Tests for cmdReadCloser
func TestCmdReadCloser_Close_Success(t *testing.T) {
	// This test verifies the structure of cmdReadCloser
	// Actual execution tests require mocking exec.Command

	t.Parallel()

	// Verify cmdReadCloser has all expected fields
	rc := &cmdReadCloser{
		dbType: "postgres",
		dbName: "testdb",
	}

	assert.Equal(t, "postgres", rc.dbType)
	assert.Equal(t, "testdb", rc.dbName)
}

// Tests for ExportResult struct
func TestExportResult_Fields(t *testing.T) {
	t.Parallel()

	result := ExportResult{
		DatabaseName: "mydb",
		DatabaseType: "postgres",
	}

	assert.Equal(t, "mydb", result.DatabaseName)
	assert.Equal(t, "postgres", result.DatabaseType)
}

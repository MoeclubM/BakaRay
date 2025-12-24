package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	envKeys := []string{
		"SERVER_HOST", "SERVER_PORT", "SERVER_MODE",
		"DB_TYPE", "DB_PATH", "DB_HOST", "DB_PORT", "DB_USERNAME", "DB_PASSWORD", "DB_NAME",
		"REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "REDIS_DB", "REDIS_POOL_SIZE",
		"SITE_NAME", "SITE_DOMAIN", "NODE_SECRET", "NODE_REPORT_INTERVAL",
		"JWT_SECRET", "JWT_EXPIRATION",
		"CONFIG_FILE",
	}
	original := make(map[string]string, len(envKeys))
	for _, key := range envKeys {
		original[key] = os.Getenv(key)
		os.Unsetenv(key)
	}
	defer func() {
		for key, value := range original {
			if value != "" {
				os.Setenv(key, value)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Mode != "release" {
		t.Errorf("Server.Mode = %v, want release", cfg.Server.Mode)
	}
	if cfg.Database.Type != "sqlite" {
		t.Errorf("Database.Type = %v, want sqlite", cfg.Database.Type)
	}
	if cfg.Database.Path != "data/bakaray.db" {
		t.Errorf("Database.Path = %v, want data/bakaray.db", cfg.Database.Path)
	}
	if cfg.Database.Host != "localhost" {
		t.Errorf("Database.Host = %v, want localhost", cfg.Database.Host)
	}
	if cfg.Database.Port != 3306 {
		t.Errorf("Database.Port = %v, want 3306", cfg.Database.Port)
	}
	if cfg.Database.Username != "root" {
		t.Errorf("Database.Username = %v, want root", cfg.Database.Username)
	}
	if cfg.Site.NodeReportInterval != 300 {
		t.Errorf("Site.NodeReportInterval = %v, want 300", cfg.Site.NodeReportInterval)
	}
	if cfg.JWT.Expiration != 86400 {
		t.Errorf("JWT.Expiration = %v, want 86400", cfg.JWT.Expiration)
	}
}

func TestLoadWithEnv(t *testing.T) {
	os.Setenv("SERVER_HOST", "127.0.0.1")
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("SERVER_MODE", "debug")
	os.Setenv("DB_TYPE", "mysql")
	os.Setenv("DB_HOST", "mysqldb")
	os.Setenv("DB_PORT", "3307")
	os.Setenv("DB_USERNAME", "admin")
	os.Setenv("DB_PASSWORD", "secret")
	os.Setenv("DB_NAME", "testdb")
	defer func() {
		for _, key := range []string{"SERVER_HOST", "SERVER_PORT", "SERVER_MODE", "DB_TYPE", "DB_HOST", "DB_PORT", "DB_USERNAME", "DB_PASSWORD", "DB_NAME"} {
			os.Unsetenv(key)
		}
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Server.Host = %v, want 127.0.0.1", cfg.Server.Host)
	}
	if cfg.Database.Type != "mysql" {
		t.Errorf("Database.Type = %v, want mysql", cfg.Database.Type)
	}
	if cfg.Database.Host != "mysqldb" {
		t.Errorf("Database.Host = %v, want mysqldb", cfg.Database.Host)
	}
	if cfg.Database.Port != 3307 {
		t.Errorf("Database.Port = %v, want 3307", cfg.Database.Port)
	}
	if cfg.Database.Name != "testdb" {
		t.Errorf("Database.Name = %v, want testdb", cfg.Database.Name)
	}
}

func TestLoadWithConfigFile(t *testing.T) {
	content := `
server:
  host: "192.168.1.2"
  port: "8888"
  mode: "release"

database:
  type: "mysql"
  host: "pgdb"
  port: 3306
  username: "pguser"
  password: "pgpass"
  name: "pgdb"

redis:
  host: "redisdb"
  port: 6380
  password: "redispass"
  db: 1
  pool_size: 20

site:
  name: "TestSite"
  domain: "https://test.example.com"
  node_secret: "node-secret"
  node_report_interval: 120

jwt:
  secret: "test-secret"
  expiration: 3600
`
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("CreateTemp error = %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("WriteString error = %v", err)
	}
	tmpFile.Close()

	os.Setenv("CONFIG_FILE", tmpFile.Name())
	defer os.Unsetenv("CONFIG_FILE")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Host != "192.168.1.2" {
		t.Errorf("Server.Host = %v, want 192.168.1.2", cfg.Server.Host)
	}
	if cfg.Database.Username != "pguser" {
		t.Errorf("Database.Username = %v, want pguser", cfg.Database.Username)
	}
	if cfg.Redis.PoolSize != 20 {
		t.Errorf("Redis.PoolSize = %v, want 20", cfg.Redis.PoolSize)
	}
	if cfg.Site.NodeReportInterval != 120 {
		t.Errorf("Site.NodeReportInterval = %v, want 120", cfg.Site.NodeReportInterval)
	}
	if cfg.JWT.Secret != "test-secret" {
		t.Errorf("JWT.Secret = %v, want test-secret", cfg.JWT.Secret)
	}
}

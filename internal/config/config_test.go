package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// 保存原始环境变量
	originalEnv := make(map[string]string)
	for _, key := range []string{
		"SERVER_HOST", "SERVER_PORT", "SERVER_MODE",
		"DB_TYPE", "DB_HOST", "DB_PORT", "DB_USERNAME", "DB_PASSWORD", "DB_NAME",
		"REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "REDIS_DB",
		"SITE_NAME", "SITE_DOMAIN", "NODE_SECRET", "NODE_REPORT_INTERVAL",
		"JWT_SECRET", "JWT_EXPIRATION",
	} {
		originalEnv[key] = os.Getenv(key)
		os.Unsetenv(key)
	}
	defer func() {
		for key, val := range originalEnv {
			if val != "" {
				os.Setenv(key, val)
			}
		}
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// 测试默认值
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host = %v, want 0.0.0.0", cfg.Server.Host)
	}
	if cfg.Server.Port != "8080" {
		t.Errorf("Server.Port = %v, want 8080", cfg.Server.Port)
	}
	if cfg.Server.Mode != "debug" {
		t.Errorf("Server.Mode = %v, want debug", cfg.Server.Mode)
	}
	if cfg.Database.Type != "mysql" {
		t.Errorf("Database.Type = %v, want mysql", cfg.Database.Type)
	}
	if cfg.Database.Port != 3306 {
		t.Errorf("Database.Port = %v, want 3306", cfg.Database.Port)
	}
	if cfg.Redis.Port != 6379 {
		t.Errorf("Redis.Port = %v, want 6379", cfg.Redis.Port)
	}
	if cfg.Site.Name != "BakaRay" {
		t.Errorf("Site.Name = %v, want BakaRay", cfg.Site.Name)
	}
}

func TestLoadWithEnv(t *testing.T) {
	// 设置环境变量
	os.Setenv("SERVER_HOST", "127.0.0.1")
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("DB_HOST", "testdb")
	os.Setenv("DB_PORT", "3307")
	defer func() {
		os.Unsetenv("SERVER_HOST")
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Server.Host = %v, want 127.0.0.1", cfg.Server.Host)
	}
	if cfg.Server.Port != "9090" {
		t.Errorf("Server.Port = %v, want 9090", cfg.Server.Port)
	}
	if cfg.Database.Host != "testdb" {
		t.Errorf("Database.Host = %v, want testdb", cfg.Database.Host)
	}
	if cfg.Database.Port != 3307 {
		t.Errorf("Database.Port = %v, want 3307", cfg.Database.Port)
	}
}

func TestLoadWithConfigFile(t *testing.T) {
	// 创建一个临时配置文件
	configContent := `
server:
  host: "192.168.1.1"
  port: "8888"
  mode: "release"

database:
  type: "postgres"
  host: "pgdb"
  port: 5432
  username: "pguser"
  password: "pgpass"
  name: "pgdb"

redis:
  host: "redisdb"
  port: 6380
  password: "redispass"
  db: 1

site:
  name: "TestSite"
  domain: "https://test.example.com"

jwt:
  secret: "test-secret"
  expiration: 3600
`
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("CreateTemp error = %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("WriteString error = %v", err)
	}
	tmpFile.Close()

	os.Setenv("CONFIG_FILE", tmpFile.Name())
	defer os.Unsetenv("CONFIG_FILE")

	// 清除可能影响测试的环境变量
	for _, key := range []string{"SERVER_HOST", "SERVER_PORT", "DB_HOST", "DB_PORT"} {
		os.Unsetenv(key)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Host != "192.168.1.1" {
		t.Errorf("Server.Host = %v, want 192.168.1.1", cfg.Server.Host)
	}
	if cfg.Server.Port != "8888" {
		t.Errorf("Server.Port = %v, want 8888", cfg.Server.Port)
	}
	if cfg.Server.Mode != "release" {
		t.Errorf("Server.Mode = %v, want release", cfg.Server.Mode)
	}
	if cfg.Database.Type != "postgres" {
		t.Errorf("Database.Type = %v, want postgres", cfg.Database.Type)
	}
	if cfg.Database.Host != "pgdb" {
		t.Errorf("Database.Host = %v, want pgdb", cfg.Database.Host)
	}
	if cfg.Redis.Host != "redisdb" {
		t.Errorf("Redis.Host = %v, want redisdb", cfg.Redis.Host)
	}
	if cfg.Site.Name != "TestSite" {
		t.Errorf("Site.Name = %v, want TestSite", cfg.Site.Name)
	}
	if cfg.JWT.Secret != "test-secret" {
		t.Errorf("JWT.Secret = %v, want test-secret", cfg.JWT.Secret)
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		value     string
		want      string
	}{
		{"existing env", "PATH", "somepath", "somepath"},
		{"non-existing env", "NON_EXISTING_VAR_12345", "", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}
			got := getEnv(tt.key, "default")
			if got != tt.want {
				t.Errorf("getEnv(%q, _) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		value     string
		want      int
	}{
		{"valid int", "TEST_INT", "123", 123},
		{"invalid int - returns default", "TEST_INVALID", "abc", 42},
		{"empty string", "TEST_EMPTY", "", 42},
		{"non-existing", "NON_EXISTING_INT", "", 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}
			got := getEnvInt(tt.key, 42)
			if got != tt.want {
				t.Errorf("getEnvInt(%q, 42) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

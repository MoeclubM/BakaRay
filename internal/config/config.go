package config

import (
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	Site     SiteConfig     `yaml:"site"`
	JWT      JWTConfig      `yaml:"jwt"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
	Mode string `yaml:"mode"`
}

// DatabaseConfig 数据库配置
// 支持 sqlite 和 mysql/mariadb 两种模式
type DatabaseConfig struct {
	Type     string `yaml:"type"`
	Path     string `yaml:"path"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

// SiteConfig 站点配置
type SiteConfig struct {
	Name               string `yaml:"name"`
	Domain             string `yaml:"domain"`
	NodeSecret         string `yaml:"node_secret"`
	NodeReportInterval int    `yaml:"node_report_interval"`
}

// JWTConfig JWT 配置
type JWTConfig struct {
	Secret     string `yaml:"secret"`
	Expiration int    `yaml:"expiration"`
}

// Load 加载配置文件
func Load() (*Config, error) {
	cfg := &Config{}

	// 首先加载默认值
	cfg.Server = ServerConfig{
		Host: "0.0.0.0",
		Port: "8080",
		Mode: "release",
	}
	cfg.Database = DatabaseConfig{
		Type:     "sqlite",
		Path:     "data/bakaray.db",
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Password: "",
		Name:     "bakaray",
	}
	cfg.Redis = RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       0,
		PoolSize: 10,
	}
	cfg.Site = SiteConfig{
		Name:               "BakaRay",
		Domain:             "http://localhost:8080",
		NodeSecret:         "change-this-secret-in-production",
		NodeReportInterval: 30,
	}
	cfg.JWT = JWTConfig{
		Secret:     "change-this-secret-in-production",
		Expiration: 86400,
	}

	// 然后从配置文件加载（会覆盖默认值）
	configFile := getEnv("CONFIG_FILE", "config.yaml")
	if _, err := os.Stat(configFile); err == nil {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return nil, err
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	// 最后用环境变量覆盖（最高优先级）
	applyEnvOverrides(cfg)

	return cfg, nil
}

// applyEnvOverrides 用环境变量覆盖配置
func applyEnvOverrides(cfg *Config) {
	// Server
	if v := os.Getenv("SERVER_HOST"); v != "" {
		cfg.Server.Host = v
	}
	if v := os.Getenv("SERVER_PORT"); v != "" {
		cfg.Server.Port = v
	}
	if v := os.Getenv("SERVER_MODE"); v != "" {
		cfg.Server.Mode = v
	}

	// Database
	if v := os.Getenv("DB_TYPE"); v != "" {
		cfg.Database.Type = v
	}
	if v := os.Getenv("DB_PATH"); v != "" {
		cfg.Database.Path = v
	}
	if v := os.Getenv("DB_HOST"); v != "" {
		cfg.Database.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Database.Port = port
		}
	}
	if v := os.Getenv("DB_USERNAME"); v != "" {
		cfg.Database.Username = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		cfg.Database.Name = v
	}

	// Redis
	if v := os.Getenv("REDIS_HOST"); v != "" {
		cfg.Redis.Host = v
	}
	if v := os.Getenv("REDIS_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Redis.Port = port
		}
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}
	if v := os.Getenv("REDIS_DB"); v != "" {
		if db, err := strconv.Atoi(v); err == nil {
			cfg.Redis.DB = db
		}
	}
	if v := os.Getenv("REDIS_POOL_SIZE"); v != "" {
		if size, err := strconv.Atoi(v); err == nil {
			cfg.Redis.PoolSize = size
		}
	}

	// Site
	if v := os.Getenv("SITE_NAME"); v != "" {
		cfg.Site.Name = v
	}
	if v := os.Getenv("SITE_DOMAIN"); v != "" {
		cfg.Site.Domain = v
	}
	if v := os.Getenv("NODE_SECRET"); v != "" {
		cfg.Site.NodeSecret = v
	}
	if v := os.Getenv("NODE_REPORT_INTERVAL"); v != "" {
		if interval, err := strconv.Atoi(v); err == nil {
			cfg.Site.NodeReportInterval = interval
		}
	}

	// JWT
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.JWT.Secret = v
	}
	if v := os.Getenv("JWT_EXPIRATION"); v != "" {
		if exp, err := strconv.Atoi(v); err == nil {
			cfg.JWT.Expiration = exp
		}
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			return val
		}
	}
	return defaultValue
}

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

// DatabaseConfig 数据库配置（SQLite）
type DatabaseConfig struct {
	Type string `yaml:"type"`
	Path string `yaml:"path"` // SQLite 数据库文件路径
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	Password  string `yaml:"password"`
	DB        int    `yaml:"db"`
	PoolSize  int    `yaml:"pool_size"`
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
	// 优先从环境变量加载
	cfg := &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: getEnv("SERVER_PORT", "8080"),
			Mode: getEnv("SERVER_MODE", "release"),
		},
		Database: DatabaseConfig{
			Type: getEnv("DB_TYPE", "sqlite"),
			Path: getEnv("DB_PATH", "data/bakaray.db"),
		},
		Redis: RedisConfig{
			Host:      getEnv("REDIS_HOST", ""),
			Port:      getEnvInt("REDIS_PORT", 6379),
			Password:  getEnv("REDIS_PASSWORD", ""),
			DB:        getEnvInt("REDIS_DB", 0),
			PoolSize:  getEnvInt("REDIS_POOL_SIZE", 10),
		},
		Site: SiteConfig{
			Name:               getEnv("SITE_NAME", "BakaRay"),
			Domain:             getEnv("SITE_DOMAIN", "http://localhost:8080"),
			NodeSecret:         getEnv("NODE_SECRET", "change-this-secret-in-production"),
			NodeReportInterval: getEnvInt("NODE_REPORT_INTERVAL", 30),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", "change-this-secret-in-production"),
			Expiration: getEnvInt("JWT_EXPIRATION", 86400),
		},
	}

	// 尝试从配置文件加载
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

	return cfg, nil
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

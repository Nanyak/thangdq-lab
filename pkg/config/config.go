package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	MongoDB  MongoDBConfig
	Redis    RedisConfig
	CacheTTL time.Duration
}

type ServerConfig struct {
	Port    string
	BaseURL string
}

type MongoDBConfig struct {
	URI           string
	Database      string
	Collection    string
	MaxRetries    int
	RetryInterval time.Duration
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:    getEnv("PORT", "9808"),
			BaseURL: getEnv("BASE_URL", "http://localhost:9808"),
		},
		MongoDB: MongoDBConfig{
			URI:           getEnv("MONGO_URI", "mongodb://localhost:27017"),
			Database:      getEnv("MONGO_DB", "stuffsy"),
			Collection:    getEnv("MONGO_COLLECTION", "links"),
			MaxRetries:    getEnvInt("MONGO_MAX_RETRIES", 10),
			RetryInterval: getEnvDuration("MONGO_RETRY_INTERVAL", 2*time.Second),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		CacheTTL: getEnvDuration("CACHE_TTL", 6*time.Hour),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.MongoDB.URI == "" {
		return fmt.Errorf("MONGO_URI is required")
	}
	if c.Redis.Addr == "" {
		return fmt.Errorf("REDIS_ADDR is required")
	}
	return nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}

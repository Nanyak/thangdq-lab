package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server         ServerConfig
	MongoDB        MongoDBConfig
	Redis          RedisConfig
	S3             S3Config
	Cognito        CognitoConfig
	CacheTTL            time.Duration
	AIServiceURL        string
	AIInternalKey       string
	EmbedQueueKey       string
	SQSVideoQueueURL    string
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

type S3Config struct {
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string
}

type CognitoConfig struct {
	UserPoolID   string
	TokenUrl     string
	JWTIssuerUrl string
	ClientID     string
	ClientSecret string
	Region       string
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:    getEnv("PORT", "9808"),
			BaseURL: getEnv("BASE_URL", "http://localhost:9808"),
		},
		MongoDB: MongoDBConfig{
			URI:           getEnv("MONGO_URI", "mongodb://localhost:27017"),
			Database:      getEnv("MONGO_DB", "url_shortener"),
			Collection:    getEnv("MONGO_COLLECTION", "links"),
			MaxRetries:    getEnvInt("MONGO_MAX_RETRIES", 10),
			RetryInterval: getEnvDuration("MONGO_RETRY_INTERVAL", 2*time.Second),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		S3: S3Config{
			Region:          getEnv("AWS_REGION", ""),
			Bucket:          getEnv("S3_BUCKET", ""),
			AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
			Endpoint:        getEnv("S3_ENDPOINT", ""),
		},
		Cognito: CognitoConfig{
			UserPoolID:   getEnv("COGNITO_USER_POOL_ID", ""),
			TokenUrl:     getEnv("COGNITO_TOKEN_URL", ""),
			JWTIssuerUrl: getEnv("COGNITO_JWT_ISSUER_URL", ""),
			ClientID:     getEnv("COGNITO_CLIENT_ID", ""),
			ClientSecret: getEnv("COGNITO_CLIENT_SECRET", ""),
			Region:       getEnv("COGNITO_REGION", ""),
		},
		CacheTTL:         getEnvDuration("CACHE_TTL", 6*time.Hour),
		AIServiceURL:     getEnv("AI_SERVICE_URL", "http://localhost:8000"),
		AIInternalKey:    getEnv("AI_INTERNAL_KEY", ""),
		EmbedQueueKey:    getEnv("EMBED_QUEUE_KEY", "stuffsy:embed"),
		SQSVideoQueueURL: getEnv("SQS_VIDEO_QUEUE_URL", ""),
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
	if c.S3.Region == "" {
		return fmt.Errorf("AWS_REGION is required")
	}
	if c.S3.Bucket == "" {
		return fmt.Errorf("S3_BUCKET is required")
	}
	if c.S3.AccessKeyID == "" {
		return fmt.Errorf("AWS_ACCESS_KEY_ID is required")
	}
	if c.S3.SecretAccessKey == "" {
		return fmt.Errorf("AWS_SECRET_ACCESS_KEY is required")
	}
	if c.Cognito.UserPoolID == "" {
		return fmt.Errorf("COGNITO_USER_POOL_ID is required")
	}
	if c.Cognito.ClientID == "" {
		return fmt.Errorf("COGNITO_CLIENT_ID is required")
	}
	if c.Cognito.ClientSecret == "" {
		return fmt.Errorf("COGNITO_CLIENT_SECRET is required")
	}
	if c.Cognito.Region == "" {
		return fmt.Errorf("COGNITO_REGION is required")
	}
	if c.Cognito.TokenUrl == "" {
		return fmt.Errorf("COGNITO_TOKEN_URL is required")
	}
	if c.Cognito.JWTIssuerUrl == "" {
		return fmt.Errorf("COGNITO_JWT_ISSUER_URL is required")
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

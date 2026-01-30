package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoad_DefaultValues(t *testing.T) {
	tests := []struct {
		name           string
		setEnvVars     map[string]string
		expectedConfig func(*Config) bool
	}{
		{
			name:       "all defaults with required S3 fields",
			setEnvVars: map[string]string{},
			expectedConfig: func(cfg *Config) bool {
				return cfg.Server.Port == "9808" &&
					cfg.Server.BaseURL == "http://localhost:9808" &&
					cfg.MongoDB.URI == "mongodb://localhost:27017" &&
					cfg.MongoDB.Database == "url_shortener" &&
					cfg.MongoDB.Collection == "links" &&
					cfg.MongoDB.MaxRetries == 10 &&
					cfg.MongoDB.RetryInterval == 2*time.Second &&
					cfg.Redis.Addr == "localhost:6379" &&
					cfg.Redis.Password == "" &&
					cfg.Redis.DB == 0 &&
					cfg.CacheTTL == 6*time.Hour
			},
		},
		{
			name: "server config with custom port and baseurl",
			setEnvVars: map[string]string{
				"PORT":     "8080",
				"BASE_URL": "https://example.com",
			},
			expectedConfig: func(cfg *Config) bool {
				return cfg.Server.Port == "8080" &&
					cfg.Server.BaseURL == "https://example.com"
			},
		},
		{
			name: "mongodb config with custom values",
			setEnvVars: map[string]string{
				"MONGO_URI":            "mongodb://custom:27017",
				"MONGO_DB":             "custom_db",
				"MONGO_COLLECTION":     "custom_collection",
				"MONGO_MAX_RETRIES":    "5",
				"MONGO_RETRY_INTERVAL": "1s",
			},
			expectedConfig: func(cfg *Config) bool {
				return cfg.MongoDB.URI == "mongodb://custom:27017" &&
					cfg.MongoDB.Database == "custom_db" &&
					cfg.MongoDB.Collection == "custom_collection" &&
					cfg.MongoDB.MaxRetries == 5 &&
					cfg.MongoDB.RetryInterval == 1*time.Second
			},
		},
		{
			name: "redis config with custom values",
			setEnvVars: map[string]string{
				"REDIS_ADDR":     "redis.example.com:6379",
				"REDIS_PASSWORD": "secret123",
				"REDIS_DB":       "2",
			},
			expectedConfig: func(cfg *Config) bool {
				return cfg.Redis.Addr == "redis.example.com:6379" &&
					cfg.Redis.Password == "secret123" &&
					cfg.Redis.DB == 2
			},
		},
		{
			name: "cache ttl with custom value",
			setEnvVars: map[string]string{
				"CACHE_TTL": "24h",
			},
			expectedConfig: func(cfg *Config) bool {
				return cfg.CacheTTL == 24*time.Hour
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			for key, value := range tt.setEnvVars {
				t.Setenv(key, value)
			}

			// Set required S3 fields
			t.Setenv("AWS_REGION", "us-east-1")
			t.Setenv("S3_BUCKET", "test-bucket")
			t.Setenv("AWS_ACCESS_KEY_ID", "test-key")
			t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")

			cfg, err := Load()
			assert.NoError(t, err)
			assert.NotNil(t, cfg)
			assert.True(t, tt.expectedConfig(cfg))
		})
	}
}

func TestLoad_S3ConfigWithEndpoint(t *testing.T) {
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("S3_BUCKET", "test-bucket")
	t.Setenv("AWS_ACCESS_KEY_ID", "test-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")
	t.Setenv("S3_ENDPOINT", "https://s3.example.com")

	cfg, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, "https://s3.example.com", cfg.S3.Endpoint)
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		defaultVal    string
		envValue      string
		expectedValue string
	}{
		{
			name:          "return env value when set",
			key:           "TEST_VAR",
			defaultVal:    "default",
			envValue:      "custom",
			expectedValue: "custom",
		},
		{
			name:          "return default when env is empty",
			key:           "TEST_VAR",
			defaultVal:    "default",
			envValue:      "",
			expectedValue: "default",
		},
		{
			name:          "return default when env is not set",
			key:           "NONEXISTENT_VAR_12345",
			defaultVal:    "default",
			envValue:      "",
			expectedValue: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}

			result := getEnv(tt.key, tt.defaultVal)
			assert.Equal(t, tt.expectedValue, result)
		})
	}
}

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		defaultVal    int
		envValue      string
		expectedValue int
	}{
		{
			name:          "return parsed int when valid",
			key:           "TEST_INT",
			defaultVal:    10,
			envValue:      "42",
			expectedValue: 42,
		},
		{
			name:          "return default when env is empty",
			key:           "TEST_INT",
			defaultVal:    10,
			envValue:      "",
			expectedValue: 10,
		},
		{
			name:          "return default when parse fails",
			key:           "TEST_INT",
			defaultVal:    10,
			envValue:      "not-a-number",
			expectedValue: 10,
		},
		{
			name:          "return zero value when valid",
			key:           "TEST_INT",
			defaultVal:    10,
			envValue:      "0",
			expectedValue: 0,
		},
		{
			name:          "return negative value when valid",
			key:           "TEST_INT",
			defaultVal:    10,
			envValue:      "-5",
			expectedValue: -5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}

			result := getEnvInt(tt.key, tt.defaultVal)
			assert.Equal(t, tt.expectedValue, result)
		})
	}
}

func TestGetEnvDuration(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		defaultVal    time.Duration
		envValue      string
		expectedValue time.Duration
	}{
		{
			name:          "return parsed duration when valid",
			key:           "TEST_DURATION",
			defaultVal:    1 * time.Minute,
			envValue:      "30s",
			expectedValue: 30 * time.Second,
		},
		{
			name:          "return default when env is empty",
			key:           "TEST_DURATION",
			defaultVal:    1 * time.Minute,
			envValue:      "",
			expectedValue: 1 * time.Minute,
		},
		{
			name:          "return default when parse fails",
			key:           "TEST_DURATION",
			defaultVal:    1 * time.Minute,
			envValue:      "invalid-duration",
			expectedValue: 1 * time.Minute,
		},
		{
			name:          "parse hours duration",
			key:           "TEST_DURATION",
			defaultVal:    1 * time.Minute,
			envValue:      "2h",
			expectedValue: 2 * time.Hour,
		},
		{
			name:          "parse complex duration",
			key:           "TEST_DURATION",
			defaultVal:    1 * time.Minute,
			envValue:      "1h30m",
			expectedValue: 1*time.Hour + 30*time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}

			result := getEnvDuration(tt.key, tt.defaultVal)
			assert.Equal(t, tt.expectedValue, result)
		})
	}
}

func TestLoad_WithValidationErrors(t *testing.T) {
	tests := []struct {
		name          string
		envSetup      map[string]string
		expectedError bool
		expectedMsg   string
	}{
		{
			name: "valid config with all required fields",
			envSetup: map[string]string{
				"MONGO_URI":              "mongodb://localhost:27017",
				"REDIS_ADDR":             "localhost:6379",
				"AWS_REGION":             "us-east-1",
				"S3_BUCKET":              "test-bucket",
				"AWS_ACCESS_KEY_ID":      "test-key",
				"AWS_SECRET_ACCESS_KEY":  "test-secret",
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for key, value := range tt.envSetup {
				t.Setenv(key, value)
			}

			cfg, err := Load()

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, cfg)
				assert.Contains(t, err.Error(), tt.expectedMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
			}
		})
	}
}

func TestValidate_DirectCall(t *testing.T) {
	tests := []struct {
		name          string
		config        *Config
		expectedError bool
		expectedMsg   string
	}{
		{
			name: "valid config",
			config: &Config{
				MongoDB: MongoDBConfig{URI: "mongodb://localhost:27017"},
				Redis:   RedisConfig{Addr: "localhost:6379"},
				S3: S3Config{
					Region:          "us-east-1",
					Bucket:          "test-bucket",
					AccessKeyID:     "test-key",
					SecretAccessKey: "test-secret",
				},
			},
			expectedError: false,
		},
		{
			name: "empty MONGO_URI",
			config: &Config{
				MongoDB: MongoDBConfig{URI: ""},
				Redis:   RedisConfig{Addr: "localhost:6379"},
				S3: S3Config{
					Region:          "us-east-1",
					Bucket:          "test-bucket",
					AccessKeyID:     "test-key",
					SecretAccessKey: "test-secret",
				},
			},
			expectedError: true,
			expectedMsg:   "MONGO_URI is required",
		},
		{
			name: "empty REDIS_ADDR",
			config: &Config{
				MongoDB: MongoDBConfig{URI: "mongodb://localhost:27017"},
				Redis:   RedisConfig{Addr: ""},
				S3: S3Config{
					Region:          "us-east-1",
					Bucket:          "test-bucket",
					AccessKeyID:     "test-key",
					SecretAccessKey: "test-secret",
				},
			},
			expectedError: true,
			expectedMsg:   "REDIS_ADDR is required",
		},
		{
			name: "empty AWS_REGION",
			config: &Config{
				MongoDB: MongoDBConfig{URI: "mongodb://localhost:27017"},
				Redis:   RedisConfig{Addr: "localhost:6379"},
				S3: S3Config{
					Region:          "",
					Bucket:          "test-bucket",
					AccessKeyID:     "test-key",
					SecretAccessKey: "test-secret",
				},
			},
			expectedError: true,
			expectedMsg:   "AWS_REGION is required",
		},
		{
			name: "empty S3_BUCKET",
			config: &Config{
				MongoDB: MongoDBConfig{URI: "mongodb://localhost:27017"},
				Redis:   RedisConfig{Addr: "localhost:6379"},
				S3: S3Config{
					Region:          "us-east-1",
					Bucket:          "",
					AccessKeyID:     "test-key",
					SecretAccessKey: "test-secret",
				},
			},
			expectedError: true,
			expectedMsg:   "S3_BUCKET is required",
		},
		{
			name: "empty AWS_ACCESS_KEY_ID",
			config: &Config{
				MongoDB: MongoDBConfig{URI: "mongodb://localhost:27017"},
				Redis:   RedisConfig{Addr: "localhost:6379"},
				S3: S3Config{
					Region:          "us-east-1",
					Bucket:          "test-bucket",
					AccessKeyID:     "",
					SecretAccessKey: "test-secret",
				},
			},
			expectedError: true,
			expectedMsg:   "AWS_ACCESS_KEY_ID is required",
		},
		{
			name: "empty AWS_SECRET_ACCESS_KEY",
			config: &Config{
				MongoDB: MongoDBConfig{URI: "mongodb://localhost:27017"},
				Redis:   RedisConfig{Addr: "localhost:6379"},
				S3: S3Config{
					Region:          "us-east-1",
					Bucket:          "test-bucket",
					AccessKeyID:     "test-key",
					SecretAccessKey: "",
				},
			},
			expectedError: true,
			expectedMsg:   "AWS_SECRET_ACCESS_KEY is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectedError {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedMsg, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoad_InvalidDurationFallback(t *testing.T) {
	t.Setenv("MONGO_RETRY_INTERVAL", "invalid-duration")
	t.Setenv("CACHE_TTL", "also-invalid")
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("S3_BUCKET", "test-bucket")
	t.Setenv("AWS_ACCESS_KEY_ID", "test-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")

	cfg, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, 2*time.Second, cfg.MongoDB.RetryInterval) // default
	assert.Equal(t, 6*time.Hour, cfg.CacheTTL)                // default
}

func TestLoad_InvalidIntFallback(t *testing.T) {
	t.Setenv("MONGO_MAX_RETRIES", "invalid-int")
	t.Setenv("REDIS_DB", "not-a-number")
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("S3_BUCKET", "test-bucket")
	t.Setenv("AWS_ACCESS_KEY_ID", "test-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")

	cfg, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, 10, cfg.MongoDB.MaxRetries) // default
	assert.Equal(t, 0, cfg.Redis.DB)            // default
}

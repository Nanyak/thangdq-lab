package redis

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLinkCache(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ttl := 5 * time.Minute

	cache := NewLinkCache(client, ttl)

	assert.NotNil(t, cache)
	assert.Equal(t, client, cache.client)
	assert.Equal(t, ttl, cache.ttl)
}

func TestLinkCache_Set_KeyFormat(t *testing.T) {
	// Test that keys are formatted correctly
	tests := []struct {
		name        string
		shortCode   string
		expectedKey string
	}{
		{
			name:        "standard alphanumeric",
			shortCode:   "abc123",
			expectedKey: "link:abc123",
		},
		{
			name:        "with special chars",
			shortCode:   "abc-123_XYZ",
			expectedKey: "link:abc-123_XYZ",
		},
		{
			name:        "empty string",
			shortCode:   "",
			expectedKey: "link:",
		},
		{
			name:        "single char",
			shortCode:   "a",
			expectedKey: "link:a",
		},
		{
			name:        "long code",
			shortCode:   "abcdefghijklmnopqrstuvwxyz0123456789",
			expectedKey: "link:abcdefghijklmnopqrstuvwxyz0123456789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualKey := fmt.Sprintf("link:%s", tt.shortCode)
			assert.Equal(t, tt.expectedKey, actualKey)
		})
	}
}

func TestLinkCache_Set_Integration(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()

	// Verify connection
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available - skipping integration test")
	}

	tests := []struct {
		name        string
		shortCode   string
		originalURL string
		ttl         time.Duration
	}{
		{
			name:        "success - set with 5 minute TTL",
			shortCode:   "test123",
			originalURL: "https://google.com",
			ttl:         5 * time.Minute,
		},
		{
			name:        "success - empty URL",
			shortCode:   "test456",
			originalURL: "",
			ttl:         5 * time.Minute,
		},
		{
			name:        "success - long URL",
			shortCode:   "test789",
			originalURL: "https://example.com/very/long/path/with/many/segments?query=param&another=value&more=data",
			ttl:         5 * time.Minute,
		},
		{
			name:        "success - short TTL",
			shortCode:   "testttl",
			originalURL: "https://example.com",
			ttl:         1 * time.Second,
		},
		{
			name:        "success - zero TTL (no expiration)",
			shortCode:   "testnoexp",
			originalURL: "https://example.com/noexpire",
			ttl:         0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewLinkCache(client, tt.ttl)
			key := fmt.Sprintf("link:%s", tt.shortCode)

			// Clean up before test
			client.Del(ctx, key)

			// Test Set
			err := cache.Set(ctx, tt.shortCode, tt.originalURL)
			require.NoError(t, err)

			// Verify it was set correctly
			val, err := client.Get(ctx, key).Result()
			require.NoError(t, err)
			assert.Equal(t, tt.originalURL, val)

			// Verify TTL if not zero
			if tt.ttl > 0 {
				ttlDuration, err := client.TTL(ctx, key).Result()
				require.NoError(t, err)
				assert.True(t, ttlDuration > 0 && ttlDuration <= tt.ttl,
					"TTL should be between 0 and %v, got %v", tt.ttl, ttlDuration)
			}

			// Clean up after test
			client.Del(ctx, key)
		})
	}
}

func TestLinkCache_Set_ErrorCases(t *testing.T) {
	// Test with invalid Redis connection
	client := redis.NewClient(&redis.Options{
		Addr:         "localhost:9999", // Non-existent port
		DialTimeout:  100 * time.Millisecond,
		ReadTimeout:  100 * time.Millisecond,
		WriteTimeout: 100 * time.Millisecond,
	})
	defer client.Close()

	cache := NewLinkCache(client, 5*time.Minute)
	ctx := context.Background()

	err := cache.Set(ctx, "test", "https://example.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redis set failed")
}

func TestLinkCache_Get_Integration(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()

	// Verify connection
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available - skipping integration test")
	}

	cache := NewLinkCache(client, 5*time.Minute)

	tests := []struct {
		name          string
		shortCode     string
		setupURL      string
		shouldSetup   bool
		expectedURL   string
		expectedError bool
		errorContains string
	}{
		{
			name:          "success - cache hit",
			shortCode:     "get123",
			setupURL:      "https://google.com",
			shouldSetup:   true,
			expectedURL:   "https://google.com",
			expectedError: false,
		},
		{
			name:          "success - cache hit with long URL",
			shortCode:     "getlong",
			setupURL:      "https://example.com/very/long/path/with/many/segments?query=param&another=value",
			shouldSetup:   true,
			expectedURL:   "https://example.com/very/long/path/with/many/segments?query=param&another=value",
			expectedError: false,
		},
		{
			name:          "error - cache miss",
			shortCode:     "nonexistent",
			shouldSetup:   false,
			expectedURL:   "",
			expectedError: true,
			errorContains: "cache miss",
		},
		{
			name:          "success - empty URL stored",
			shortCode:     "getempty",
			setupURL:      "",
			shouldSetup:   true,
			expectedURL:   "",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := fmt.Sprintf("link:%s", tt.shortCode)

			// Clean up before test
			client.Del(ctx, key)

			// Setup if needed
			if tt.shouldSetup {
				err := cache.Set(ctx, tt.shortCode, tt.setupURL)
				require.NoError(t, err)
			}

			// Test Get
			url, err := cache.Get(ctx, tt.shortCode)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Equal(t, tt.expectedURL, url)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedURL, url)
			}

			// Clean up after test
			client.Del(ctx, key)
		})
	}
}

func TestLinkCache_Get_ErrorCases(t *testing.T) {
	tests := []struct {
		name          string
		setupClient   func() *redis.Client
		shortCode     string
		errorContains string
	}{
		{
			name: "error - redis connection error",
			setupClient: func() *redis.Client {
				return redis.NewClient(&redis.Options{
					Addr:         "localhost:9999", // Non-existent port
					DialTimeout:  100 * time.Millisecond,
					ReadTimeout:  100 * time.Millisecond,
					WriteTimeout: 100 * time.Millisecond,
				})
			},
			shortCode:     "test",
			errorContains: "redis get failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()
			defer client.Close()

			cache := NewLinkCache(client, 5*time.Minute)
			ctx := context.Background()

			url, err := cache.Get(ctx, tt.shortCode)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorContains)
			assert.Empty(t, url)
		})
	}
}

func TestLinkCache_TTL_Expiration(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()

	// Verify connection
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available - skipping integration test")
	}

	// Use a very short TTL
	ttl := 1 * time.Second
	cache := NewLinkCache(client, ttl)

	shortCode := "expiretest"
	originalURL := "https://example.com/expire"
	key := fmt.Sprintf("link:%s", shortCode)

	// Clean up
	client.Del(ctx, key)

	// Set with short TTL
	err := cache.Set(ctx, shortCode, originalURL)
	require.NoError(t, err)

	// Verify it exists immediately
	url, err := cache.Get(ctx, shortCode)
	require.NoError(t, err)
	assert.Equal(t, originalURL, url)

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Verify it's gone
	_, err = cache.Get(ctx, shortCode)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache miss")

	// Clean up
	client.Del(ctx, key)
}

func TestLinkCache_TTL_Configuration(t *testing.T) {
	tests := []struct {
		name        string
		ttl         time.Duration
		description string
	}{
		{
			name:        "5 minutes",
			ttl:         5 * time.Minute,
			description: "standard cache duration",
		},
		{
			name:        "1 hour",
			ttl:         1 * time.Hour,
			description: "extended cache duration",
		},
		{
			name:        "30 seconds",
			ttl:         30 * time.Second,
			description: "short cache duration",
		},
		{
			name:        "zero TTL",
			ttl:         0,
			description: "no expiration",
		},
		{
			name:        "24 hours",
			ttl:         24 * time.Hour,
			description: "long cache duration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := redis.NewClient(&redis.Options{
				Addr: "localhost:6379",
			})
			defer client.Close()

			cache := NewLinkCache(client, tt.ttl)

			assert.NotNil(t, cache)
			assert.Equal(t, tt.ttl, cache.ttl)
		})
	}
}

func TestLinkCache_ConcurrentAccess(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()

	// Verify connection
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available - skipping integration test")
	}

	cache := NewLinkCache(client, 1*time.Minute)

	// Test concurrent writes and reads
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines*2)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			shortCode := fmt.Sprintf("code%d", index)
			url := fmt.Sprintf("https://example.com/%d", index)
			key := fmt.Sprintf("link:%s", shortCode)

			// Clean up first
			client.Del(ctx, key)

			// Set
			if err := cache.Set(ctx, shortCode, url); err != nil {
				errors <- fmt.Errorf("Set failed for %s: %w", shortCode, err)
			}

			// Get
			retrieved, err := cache.Get(ctx, shortCode)
			if err != nil {
				errors <- fmt.Errorf("Get failed for %s: %w", shortCode, err)
			}
			if retrieved != url {
				errors <- fmt.Errorf("URL mismatch for %s: expected %s, got %s", shortCode, url, retrieved)
			}

			// Clean up
			client.Del(ctx, key)

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	close(errors)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}
}

func TestLinkCache_EmptyShortCode(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()

	// Verify connection
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available - skipping integration test")
	}

	cache := NewLinkCache(client, 5*time.Minute)

	// Test with empty short code
	err := cache.Set(ctx, "", "https://example.com")
	assert.NoError(t, err) // Redis allows empty keys

	// Verify it was set
	url, err := cache.Get(ctx, "")
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com", url)

	// Clean up
	client.Del(ctx, "link:")
}

func TestLinkCache_SpecialCharacters(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()

	// Verify connection
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available - skipping integration test")
	}

	cache := NewLinkCache(client, 5*time.Minute)

	tests := []struct {
		name        string
		shortCode   string
		originalURL string
	}{
		{
			name:        "URL with query params",
			shortCode:   "query123",
			originalURL: "https://example.com?foo=bar&baz=qux",
		},
		{
			name:        "URL with fragment",
			shortCode:   "frag123",
			originalURL: "https://example.com#section",
		},
		{
			name:        "URL with special chars",
			shortCode:   "special123",
			originalURL: "https://example.com/path?q=hello%20world&emoji=😀",
		},
		{
			name:        "Short code with dashes and underscores",
			shortCode:   "abc-123_XYZ",
			originalURL: "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := fmt.Sprintf("link:%s", tt.shortCode)

			// Clean up
			client.Del(ctx, key)

			// Set
			err := cache.Set(ctx, tt.shortCode, tt.originalURL)
			require.NoError(t, err)

			// Get
			url, err := cache.Get(ctx, tt.shortCode)
			require.NoError(t, err)
			assert.Equal(t, tt.originalURL, url)

			// Clean up
			client.Del(ctx, key)
		})
	}
}

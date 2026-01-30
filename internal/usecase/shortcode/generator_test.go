package shortcode

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateShortUrl(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantLen int
	}{
		{
			name:    "success - standard https URL",
			url:     "https://google.com",
			wantLen: 8,
		},
		{
			name:    "success - standard http URL",
			url:     "http://example.com",
			wantLen: 8,
		},
		{
			name:    "success - URL with query parameters",
			url:     "https://example.com/path?foo=bar&baz=qux",
			wantLen: 8,
		},
		{
			name:    "success - URL with fragment",
			url:     "https://example.com/path#section",
			wantLen: 8,
		},
		{
			name:    "success - ftp protocol",
			url:     "ftp://files.example.com/document.pdf",
			wantLen: 8,
		},
		{
			name:    "edge case - empty string",
			url:     "",
			wantLen: 8,
		},
		{
			name:    "edge case - very long URL (1000+ chars)",
			url:     "https://example.com/" + string(make([]byte, 1000)),
			wantLen: 8,
		},
		{
			name:    "edge case - URL with unicode characters",
			url:     "https://example.com/path/日本語",
			wantLen: 8,
		},
		{
			name:    "edge case - URL with spaces",
			url:     "https://example.com/path with spaces",
			wantLen: 8,
		},
		{
			name:    "edge case - URL with special characters",
			url:     "https://example.com/path?q=hello%20world&special=!@#$%^&*()",
			wantLen: 8,
		},
		{
			name:    "edge case - URL with newlines and tabs",
			url:     "https://example.com/path\n\twith\nnewlines",
			wantLen: 8,
		},
		{
			name:    "edge case - single character",
			url:     "a",
			wantLen: 8,
		},
	}

	generator := NewGenerator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shortlink := generator.Generate(tt.url)
			assert.Equal(t, tt.wantLen, len(shortlink), "short code length should be %d", tt.wantLen)
			assert.NotEmpty(t, shortlink, "short code should not be empty")
		})
	}
}

func TestGenerateShortUrlConsistency(t *testing.T) {
	generator := NewGenerator()
	url := "https://example.com/consistent"

	result1 := generator.Generate(url)
	result2 := generator.Generate(url)

	assert.Equal(t, result1, result2, "same URL should produce same short code")
}

func TestGenerateShortUrlUniqueness(t *testing.T) {
	generator := NewGenerator()

	url1 := "https://example.com/first"
	url2 := "https://example.com/second"

	result1 := generator.Generate(url1)
	result2 := generator.Generate(url2)

	assert.NotEqual(t, result1, result2, "different URLs should produce different short codes")
}

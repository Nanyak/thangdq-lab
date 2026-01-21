package shortcode

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateShortUrl(t *testing.T) {
	generator := NewGenerator()
	originalUrl := "https://google.com"
	shortlink := generator.Generate(originalUrl)
	assert.Equal(t, 8, len(shortlink))
}

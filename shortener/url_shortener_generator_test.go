package shortener

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateShortUrl(t *testing.T) {
	originalUrl := "https://google.com"
	shortlink := GenerateShortUrl(originalUrl)
	assert.Equal(t, 8, len(shortlink))
}

package shortener

import (
	"crypto/sha256"

	"github.com/mr-tron/base58"
)

func GenerateShortUrl(originalUrl string) string {
	hash := sha256.Sum256([]byte(originalUrl))

	encoded := base58.Encode(hash[:])

	return encoded[:8]
}

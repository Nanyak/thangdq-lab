package shortcode

import (
	"crypto/sha256"

	"github.com/mr-tron/base58"
)

type Generator struct{}

func NewGenerator() *Generator {
	return &Generator{}
}

func (g *Generator) Generate(url string) string {
	hash := sha256.Sum256([]byte(url))
	encoded := base58.Encode(hash[:])

	if len(encoded) > 8 {
		return encoded[:8]
	}
	return encoded
}

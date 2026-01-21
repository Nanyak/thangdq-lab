package usecase

import (
	"context"
	"github.com/Nanyak/thangdq-lab/internal/entity"
)

// URLRepository defines persistence operations for links
type URLRepository interface {
	Save(ctx context.Context, link *entity.Link) error
	FindByShortCode(ctx context.Context, shortCode string) (*entity.Link, error)
}

// URLCache defines caching operations for links
type URLCache interface {
	Set(ctx context.Context, shortCode string, originalURL string) error
	Get(ctx context.Context, shortCode string) (string, error)
}

// ShortCodeGenerator generates short codes from URLs
type ShortCodeGenerator interface {
	Generate(url string) string
}

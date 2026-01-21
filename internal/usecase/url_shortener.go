package usecase

import (
	"context"
	"fmt"
	"github.com/Nanyak/thangdq-lab/internal/entity"
	"time"
)

type URLShortener struct {
	repo      URLRepository
	cache     URLCache
	generator ShortCodeGenerator
}

func NewURLShortener(repo URLRepository, cache URLCache, generator ShortCodeGenerator) *URLShortener {
	return &URLShortener{
		repo:      repo,
		cache:     cache,
		generator: generator,
	}
}

func (u *URLShortener) CreateShortURL(ctx context.Context, originalURL string) (*entity.Link, error) {
	shortCode := u.generator.Generate(originalURL)

	link := &entity.Link{
		ShortCode:   shortCode,
		OriginalURL: originalURL,
		CreatedAt:   time.Now(),
	}

	// Save to MongoDB first (source of truth)
	if err := u.repo.Save(ctx, link); err != nil {
		return nil, fmt.Errorf("failed to save link: %w", err)
	}

	// Cache in Redis (best effort)
	if err := u.cache.Set(ctx, shortCode, originalURL); err != nil {
		// Log warning but don't fail operation
		fmt.Printf("Warning: failed to cache link: %v\n", err)
	}

	return link, nil
}

func (u *URLShortener) GetOriginalURL(ctx context.Context, shortCode string) (string, error) {
	// Try cache first
	if url, err := u.cache.Get(ctx, shortCode); err == nil {
		return url, nil
	}

	// Cache miss, fetch from MongoDB
	link, err := u.repo.FindByShortCode(ctx, shortCode)
	if err != nil {
		return "", fmt.Errorf("failed to find link: %w", err)
	}

	// Populate cache (best effort)
	if err := u.cache.Set(ctx, shortCode, link.OriginalURL); err != nil {
		fmt.Printf("Warning: failed to populate cache: %v\n", err)
	}

	return link.OriginalURL, nil
}

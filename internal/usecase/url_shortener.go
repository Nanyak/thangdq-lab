package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/Nanyak/thangdq-lab/internal/entity"
	"github.com/Nanyak/thangdq-lab/pkg/errors"
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

	// Check cache first - if exists, return existing link
	if url, err := u.cache.Get(ctx, shortCode); err == nil {
		return &entity.Link{
			ShortCode:   shortCode,
			OriginalURL: url,
		}, nil
	}

	link := &entity.Link{
		ShortCode:   shortCode,
		OriginalURL: originalURL,
		CreatedAt:   time.Now(),
	}

	// Save to MongoDB
	if err := u.repo.Save(ctx, link); err != nil {
		// If duplicate, fetch existing and return it
		if errors.IsDuplicate(err) {
			existing, fetchErr := u.repo.FindByShortCode(ctx, shortCode)
			if fetchErr != nil {
				return nil, fmt.Errorf("failed to fetch existing link: %w", fetchErr)
			}
			// Cache existing link
			_ = u.cache.Set(ctx, shortCode, existing.OriginalURL)
			return existing, nil
		}
		return nil, fmt.Errorf("failed to save link: %w", err)
	}

	// Cache in Redis (best effort)
	if err := u.cache.Set(ctx, shortCode, originalURL); err != nil {
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

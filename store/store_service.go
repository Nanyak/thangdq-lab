package store

import (
	"context"
	"errors"
	"fmt"
)

var ErrShortURLNotFound = errors.New("short URL not found in storage")

type StorageService struct {
	mongoRepo  *MongoRepository
	redisCache *RedisCache
}

func NewStorageService(mongoRepo *MongoRepository, redisCache *RedisCache) *StorageService {
	return &StorageService{
		mongoRepo:  mongoRepo,
		redisCache: redisCache,
	}
}

// SaveUrlMapping saves to MongoDB first (primary store), then caches in Redis.
func (s *StorageService) SaveUrlMapping(
	ctx context.Context,
	shortUrl string,
	originalUrl string,
) error {
	// Save to MongoDB (primary store)
	if err := s.mongoRepo.CreateLink(ctx, shortUrl, originalUrl); err != nil {
		return fmt.Errorf("failed to save to MongoDB: %w", err)
	}

	// Cache in Redis (best effort, don't fail if cache write fails)
	if err := s.redisCache.Set(ctx, shortUrl, originalUrl); err != nil {
		fmt.Printf("Warning: failed to cache in Redis: %v\n", err)
	}

	return nil
}

// RetrieveInitialUrl checks Redis cache first, on miss fetches from MongoDB and populates cache.
func (s *StorageService) RetrieveInitialUrl(
	ctx context.Context,
	shortUrl string,
) (string, error) {
	// Check Redis cache first
	cachedUrl, err := s.redisCache.Get(ctx, shortUrl)
	if err != nil {
		fmt.Printf("Warning: Redis cache lookup failed: %v\n", err)
	}

	if cachedUrl != "" {
		return cachedUrl, nil
	}

	// Cache miss - fetch from MongoDB
	link, err := s.mongoRepo.GetLinkByShortURL(ctx, shortUrl)
	if err != nil {
		if errors.Is(err, ErrLinkNotFound) {
			return "", ErrShortURLNotFound
		}
		return "", fmt.Errorf("failed to fetch from MongoDB: %w", err)
	}

	// Populate cache for future requests (best effort)
	if err := s.redisCache.Set(ctx, shortUrl, link.OriginalURL); err != nil {
		fmt.Printf("Warning: failed to populate Redis cache: %v\n", err)
	}

	return link.OriginalURL, nil
}

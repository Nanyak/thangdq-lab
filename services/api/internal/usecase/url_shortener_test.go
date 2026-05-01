package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Nanyak/thangdq-lab/internal/entity"
	pkgerrors "github.com/Nanyak/thangdq-lab/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockURLRepository is a mock implementation of URLRepository
type MockURLRepository struct {
	mock.Mock
}

func (m *MockURLRepository) Save(ctx context.Context, link *entity.Link) error {
	args := m.Called(ctx, link)
	return args.Error(0)
}

func (m *MockURLRepository) FindByShortCode(ctx context.Context, shortCode string) (*entity.Link, error) {
	args := m.Called(ctx, shortCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Link), args.Error(1)
}

func (m *MockURLRepository) FindByUserID(ctx context.Context, userID string) ([]*entity.Link, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Link), args.Error(1)
}

// MockURLCache is a mock implementation of URLCache
type MockURLCache struct {
	mock.Mock
}

func (m *MockURLCache) Set(ctx context.Context, shortCode string, originalURL string) error {
	args := m.Called(ctx, shortCode, originalURL)
	return args.Error(0)
}

func (m *MockURLCache) Get(ctx context.Context, shortCode string) (string, error) {
	args := m.Called(ctx, shortCode)
	return args.String(0), args.Error(1)
}

// MockShortCodeGenerator is a mock implementation of ShortCodeGenerator
type MockShortCodeGenerator struct {
	mock.Mock
}

func (m *MockShortCodeGenerator) Generate(url string) string {
	args := m.Called(url)
	return args.String(0)
}

func TestNewURLShortener(t *testing.T) {
	mockRepo := new(MockURLRepository)
	mockCache := new(MockURLCache)
	mockGenerator := new(MockShortCodeGenerator)

	shortener := NewURLShortener(mockRepo, mockCache, mockGenerator)

	assert.NotNil(t, shortener)
	assert.Equal(t, mockRepo, shortener.repo)
	assert.Equal(t, mockCache, shortener.cache)
	assert.Equal(t, mockGenerator, shortener.generator)
}

func TestCreateShortURL(t *testing.T) {
	ctx := context.Background()
	originalURL := "https://google.com"
	shortCode := "abc123xy"
	now := time.Now()

	tests := []struct {
		name          string
		inputURL      string
		userID        string
		setupMocks    func(*MockURLRepository, *MockURLCache, *MockShortCodeGenerator)
		expectedLink  *entity.Link
		expectedError bool
	}{
		{
			name:     "success - anonymous cache hit",
			inputURL: originalURL,
			userID:   "",
			setupMocks: func(repo *MockURLRepository, cache *MockURLCache, gen *MockShortCodeGenerator) {
				gen.On("Generate", originalURL).Return(shortCode)
				cache.On("Get", ctx, shortCode).Return(originalURL, nil)
			},
			expectedLink: &entity.Link{
				ShortCode:   shortCode,
				OriginalURL: originalURL,
			},
			expectedError: false,
		},
		{
			name:     "success - cache miss, save to db, cache",
			inputURL: originalURL,
			userID:   "",
			setupMocks: func(repo *MockURLRepository, cache *MockURLCache, gen *MockShortCodeGenerator) {
				gen.On("Generate", originalURL).Return(shortCode)
				cache.On("Get", ctx, shortCode).Return("", errors.New("cache miss"))
				repo.On("Save", ctx, mock.MatchedBy(func(link *entity.Link) bool {
					return link.ShortCode == shortCode && link.OriginalURL == originalURL
				})).Return(nil)
				cache.On("Set", ctx, shortCode, originalURL).Return(nil)
			},
			expectedLink: &entity.Link{
				ShortCode:   shortCode,
				OriginalURL: originalURL,
				CreatedAt:   now,
			},
			expectedError: false,
		},
		{
			name:     "success - authenticated user, cache miss, save to db",
			inputURL: originalURL,
			userID:   "user-123",
			setupMocks: func(repo *MockURLRepository, cache *MockURLCache, gen *MockShortCodeGenerator) {
				gen.On("Generate", originalURL).Return(shortCode)
				repo.On("Save", ctx, mock.MatchedBy(func(link *entity.Link) bool {
					return link.ShortCode == shortCode && link.OriginalURL == originalURL && link.UserID == "user-123"
				})).Return(nil)
				cache.On("Set", ctx, shortCode, originalURL).Return(nil)
			},
			expectedLink: &entity.Link{
				ShortCode:   shortCode,
				OriginalURL: originalURL,
				CreatedAt:   now,
			},
			expectedError: false,
		},
		{
			name:     "success - duplicate error, fetch existing",
			inputURL: originalURL,
			userID:   "",
			setupMocks: func(repo *MockURLRepository, cache *MockURLCache, gen *MockShortCodeGenerator) {
				gen.On("Generate", originalURL).Return(shortCode)
				cache.On("Get", ctx, shortCode).Return("", errors.New("cache miss"))
				repo.On("Save", ctx, mock.Anything).Return(pkgerrors.ErrDuplicateShortCode)
				existingLink := &entity.Link{
					ShortCode:   shortCode,
					OriginalURL: originalURL,
					CreatedAt:   now,
				}
				repo.On("FindByShortCode", ctx, shortCode).Return(existingLink, nil)
				cache.On("Set", ctx, shortCode, originalURL).Return(nil)
			},
			expectedLink: &entity.Link{
				ShortCode:   shortCode,
				OriginalURL: originalURL,
				CreatedAt:   now,
			},
			expectedError: false,
		},
		{
			name:     "success - cache set fails (best effort)",
			inputURL: originalURL,
			userID:   "",
			setupMocks: func(repo *MockURLRepository, cache *MockURLCache, gen *MockShortCodeGenerator) {
				gen.On("Generate", originalURL).Return(shortCode)
				cache.On("Get", ctx, shortCode).Return("", errors.New("cache miss"))
				repo.On("Save", ctx, mock.Anything).Return(nil)
				cache.On("Set", ctx, shortCode, originalURL).Return(errors.New("cache error"))
			},
			expectedLink: &entity.Link{
				ShortCode:   shortCode,
				OriginalURL: originalURL,
				CreatedAt:   now,
			},
			expectedError: false,
		},
		{
			name:     "error - save fails with non-duplicate error",
			inputURL: originalURL,
			userID:   "",
			setupMocks: func(repo *MockURLRepository, cache *MockURLCache, gen *MockShortCodeGenerator) {
				gen.On("Generate", originalURL).Return(shortCode)
				cache.On("Get", ctx, shortCode).Return("", errors.New("cache miss"))
				repo.On("Save", ctx, mock.Anything).Return(errors.New("database error"))
			},
			expectedLink:  nil,
			expectedError: true,
		},
		{
			name:     "error - duplicate error but fetch fails",
			inputURL: originalURL,
			userID:   "",
			setupMocks: func(repo *MockURLRepository, cache *MockURLCache, gen *MockShortCodeGenerator) {
				gen.On("Generate", originalURL).Return(shortCode)
				cache.On("Get", ctx, shortCode).Return("", errors.New("cache miss"))
				repo.On("Save", ctx, mock.Anything).Return(pkgerrors.ErrDuplicateShortCode)
				repo.On("FindByShortCode", ctx, shortCode).Return(nil, errors.New("fetch error"))
			},
			expectedLink:  nil,
			expectedError: true,
		},
		{
			name:     "error - empty originalURL",
			inputURL: "",
			userID:   "",
			setupMocks: func(repo *MockURLRepository, cache *MockURLCache, gen *MockShortCodeGenerator) {
				gen.On("Generate", "").Return("")
				cache.On("Get", ctx, "").Return("", errors.New("cache miss"))
				repo.On("Save", ctx, mock.Anything).Return(errors.New("validation error"))
			},
			expectedLink:  nil,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockURLRepository)
			mockCache := new(MockURLCache)
			mockGenerator := new(MockShortCodeGenerator)

			tt.setupMocks(mockRepo, mockCache, mockGenerator)

			shortener := NewURLShortener(mockRepo, mockCache, mockGenerator)
			link, err := shortener.CreateShortURL(ctx, tt.inputURL, tt.userID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, link)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, link)
				assert.Equal(t, tt.expectedLink.ShortCode, link.ShortCode)
				assert.Equal(t, tt.expectedLink.OriginalURL, link.OriginalURL)
				if !tt.expectedLink.CreatedAt.IsZero() {
					assert.WithinDuration(t, now, link.CreatedAt, 2*time.Second)
				}
			}

			mockRepo.AssertExpectations(t)
			mockCache.AssertExpectations(t)
			mockGenerator.AssertExpectations(t)
		})
	}
}

func TestGetOriginalURL(t *testing.T) {
	ctx := context.Background()
	shortCode := "abc123xy"
	originalURL := "https://google.com"
	now := time.Now()

	tests := []struct {
		name          string
		inputCode     string
		setupMocks    func(*MockURLRepository, *MockURLCache)
		expectedURL   string
		expectedError bool
	}{
		{
			name:      "success - cache hit",
			inputCode: shortCode,
			setupMocks: func(repo *MockURLRepository, cache *MockURLCache) {
				cache.On("Get", ctx, shortCode).Return(originalURL, nil)
			},
			expectedURL:   originalURL,
			expectedError: false,
		},
		{
			name:      "success - cache miss, fetch from db, populate cache",
			inputCode: shortCode,
			setupMocks: func(repo *MockURLRepository, cache *MockURLCache) {
				cache.On("Get", ctx, shortCode).Return("", errors.New("cache miss"))
				link := &entity.Link{
					ShortCode:   shortCode,
					OriginalURL: originalURL,
					CreatedAt:   now,
				}
				repo.On("FindByShortCode", ctx, shortCode).Return(link, nil)
				cache.On("Set", ctx, shortCode, originalURL).Return(nil)
			},
			expectedURL:   originalURL,
			expectedError: false,
		},
		{
			name:      "success - cache miss, fetch from db, cache set fails (best effort)",
			inputCode: shortCode,
			setupMocks: func(repo *MockURLRepository, cache *MockURLCache) {
				cache.On("Get", ctx, shortCode).Return("", errors.New("cache miss"))
				link := &entity.Link{
					ShortCode:   shortCode,
					OriginalURL: originalURL,
					CreatedAt:   now,
				}
				repo.On("FindByShortCode", ctx, shortCode).Return(link, nil)
				cache.On("Set", ctx, shortCode, originalURL).Return(errors.New("cache error"))
			},
			expectedURL:   originalURL,
			expectedError: false,
		},
		{
			name:      "error - cache miss and db not found",
			inputCode: shortCode,
			setupMocks: func(repo *MockURLRepository, cache *MockURLCache) {
				cache.On("Get", ctx, shortCode).Return("", errors.New("cache miss"))
				repo.On("FindByShortCode", ctx, shortCode).Return(nil, pkgerrors.ErrLinkNotFound)
			},
			expectedURL:   "",
			expectedError: true,
		},
		{
			name:      "error - cache miss and db error",
			inputCode: shortCode,
			setupMocks: func(repo *MockURLRepository, cache *MockURLCache) {
				cache.On("Get", ctx, shortCode).Return("", errors.New("cache miss"))
				repo.On("FindByShortCode", ctx, shortCode).Return(nil, errors.New("database error"))
			},
			expectedURL:   "",
			expectedError: true,
		},
		{
			name:      "error - empty shortCode",
			inputCode: "",
			setupMocks: func(repo *MockURLRepository, cache *MockURLCache) {
				cache.On("Get", ctx, "").Return("", errors.New("cache miss"))
				repo.On("FindByShortCode", ctx, "").Return(nil, pkgerrors.ErrLinkNotFound)
			},
			expectedURL:   "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockURLRepository)
			mockCache := new(MockURLCache)

			tt.setupMocks(mockRepo, mockCache)

			shortener := NewURLShortener(mockRepo, mockCache, new(MockShortCodeGenerator))
			url, err := shortener.GetOriginalURL(ctx, tt.inputCode)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Empty(t, url)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedURL, url)
			}

			mockRepo.AssertExpectations(t)
			mockCache.AssertExpectations(t)
		})
	}
}

func TestGetUserLinks(t *testing.T) {
	ctx := context.Background()
	userID := "user-123"

	t.Run("success - returns user links", func(t *testing.T) {
		mockRepo := new(MockURLRepository)
		mockCache := new(MockURLCache)
		links := []*entity.Link{
			{ShortCode: "abc123xy", OriginalURL: "https://google.com", UserID: userID},
		}
		mockRepo.On("FindByUserID", ctx, userID).Return(links, nil)

		shortener := NewURLShortener(mockRepo, mockCache, new(MockShortCodeGenerator))
		result, err := shortener.GetUserLinks(ctx, userID)

		assert.NoError(t, err)
		assert.Len(t, result, 1)
		mockRepo.AssertExpectations(t)
	})

	t.Run("error - repository error", func(t *testing.T) {
		mockRepo := new(MockURLRepository)
		mockCache := new(MockURLCache)
		mockRepo.On("FindByUserID", ctx, userID).Return(nil, errors.New("db error"))

		shortener := NewURLShortener(mockRepo, mockCache, new(MockShortCodeGenerator))
		result, err := shortener.GetUserLinks(ctx, userID)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})
}

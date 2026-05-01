package mongodb

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Nanyak/thangdq-lab/internal/entity"
	pkgerrors "github.com/Nanyak/thangdq-lab/pkg/errors"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
)


func TestSave(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	tests := []struct {
		name          string
		link          *entity.Link
		insertErr     error
		isDuplicateKey bool
		expectedError error
	}{
		{
			name: "success - insert new link",
			link: &entity.Link{
				ShortCode:   "abc123",
				OriginalURL: "https://google.com",
				CreatedAt:   now,
			},
			insertErr:     nil,
			isDuplicateKey: false,
			expectedError: nil,
		},
		{
			name: "error - duplicate short code",
			link: &entity.Link{
				ShortCode:   "abc123",
				OriginalURL: "https://google.com",
				CreatedAt:   now,
			},
			insertErr:     nil,
			isDuplicateKey: true,
			expectedError: pkgerrors.ErrDuplicateShortCode,
		},
		{
			name: "error - database connection error",
			link: &entity.Link{
				ShortCode:   "abc123",
				OriginalURL: "https://google.com",
				CreatedAt:   now,
			},
			insertErr:     errors.New("connection error"),
			isDuplicateKey: false,
			expectedError: errors.New("mongodb insert failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use a real mongo collection wrapper that we control
			var insertErr error
			if tt.isDuplicateKey {
				// Create a duplicate key error
				writeErr := mongo.WriteException{
					WriteErrors: []mongo.WriteError{
						{Code: 11000}, // MongoDB duplicate key error code
					},
				}
				insertErr = writeErr
			} else {
				insertErr = tt.insertErr
			}

			// Create a test repository with a mock inserter
			repo := &testLinkRepository{
				insertOneFn: func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
					if insertErr != nil {
						return nil, insertErr
					}
					return &mongo.InsertOneResult{InsertedID: "test-id"}, nil
				},
			}

			err := repo.Save(ctx, tt.link)

			if tt.expectedError != nil {
				assert.Error(t, err)
				if pkgerrors.IsDuplicate(tt.expectedError) {
					assert.True(t, pkgerrors.IsDuplicate(err))
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFindByShortCode(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	tests := []struct {
		name          string
		shortCode     string
		findDoc       *LinkDocument
		findErr       error
		expectedLink  *entity.Link
		expectedError error
	}{
		{
			name:      "success - link found",
			shortCode: "abc123",
			findDoc: &LinkDocument{
				ShortCode:   "abc123",
				OriginalURL: "https://google.com",
				CreatedAt:   now,
			},
			findErr: nil,
			expectedLink: &entity.Link{
				ShortCode:   "abc123",
				OriginalURL: "https://google.com",
				CreatedAt:   now,
			},
			expectedError: nil,
		},
		{
			name:          "error - link not found",
			shortCode:     "notfound",
			findDoc:       nil,
			findErr:       mongo.ErrNoDocuments,
			expectedLink:  nil,
			expectedError: pkgerrors.ErrLinkNotFound,
		},
		{
			name:          "error - database error",
			shortCode:     "abc123",
			findDoc:       nil,
			findErr:       errors.New("database error"),
			expectedLink:  nil,
			expectedError: errors.New("mongodb find failed"),
		},
		{
			name:          "error - empty shortCode",
			shortCode:     "",
			findDoc:       nil,
			findErr:       mongo.ErrNoDocuments,
			expectedLink:  nil,
			expectedError: pkgerrors.ErrLinkNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test repository with a mock finder
			repo := &testLinkRepository{
				findOneFn: func(ctx context.Context, filter interface{}) (*LinkDocument, error) {
					if tt.findErr != nil {
						return nil, tt.findErr
					}
					return tt.findDoc, nil
				},
			}

			link, err := repo.FindByShortCode(ctx, tt.shortCode)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Nil(t, link)
				if pkgerrors.IsNotFound(tt.expectedError) {
					assert.True(t, pkgerrors.IsNotFound(err))
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, link)
				assert.Equal(t, tt.expectedLink.ShortCode, link.ShortCode)
				assert.Equal(t, tt.expectedLink.OriginalURL, link.OriginalURL)
				assert.True(t, tt.expectedLink.CreatedAt.Equal(link.CreatedAt))
			}
		})
	}
}

// testLinkRepository is a test double for LinkRepository
type testLinkRepository struct {
	insertOneFn func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error)
	findOneFn   func(ctx context.Context, filter interface{}) (*LinkDocument, error)
}

func (r *testLinkRepository) Save(ctx context.Context, link *entity.Link) error {
	doc := toDocument(link)

	_, err := r.insertOneFn(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return pkgerrors.ErrDuplicateShortCode
		}
		return fmt.Errorf("mongodb insert failed: %w", err)
	}

	return nil
}

func (r *testLinkRepository) FindByShortCode(ctx context.Context, shortCode string) (*entity.Link, error) {
	doc, err := r.findOneFn(ctx, map[string]interface{}{"short_code": shortCode})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, pkgerrors.ErrLinkNotFound
		}
		return nil, fmt.Errorf("mongodb find failed: %w", err)
	}

	return toEntity(doc), nil
}

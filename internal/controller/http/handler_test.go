package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Nanyak/thangdq-lab/internal/entity"
	pkgerrors "github.com/Nanyak/thangdq-lab/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockURLShortener is a mock implementation of URLShortener interface
type MockURLShortener struct {
	mock.Mock
}

func (m *MockURLShortener) CreateShortURL(ctx context.Context, originalURL string) (*entity.Link, error) {
	args := m.Called(ctx, originalURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Link), args.Error(1)
}

func (m *MockURLShortener) GetOriginalURL(ctx context.Context, shortCode string) (string, error) {
	args := m.Called(ctx, shortCode)
	return args.String(0), args.Error(1)
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestNewHandler(t *testing.T) {
	mockShortener := new(MockURLShortener)
	baseURL := "http://localhost:8080"

	handler := NewHandler(mockShortener, baseURL)

	assert.NotNil(t, handler)
	assert.Equal(t, mockShortener, handler.shortener)
	assert.Equal(t, baseURL, handler.baseURL)
}

func TestCreateShortURL(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*MockURLShortener)
		expectedStatus int
		validateBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success - valid URL with https",
			requestBody: CreateShortURLRequest{
				LongURL: "https://google.com",
			},
			setupMock: func(m *MockURLShortener) {
				m.On("CreateShortURL", mock.Anything, "https://google.com").Return(&entity.Link{
					ShortCode:   "abc123xy",
					OriginalURL: "https://google.com",
				}, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp CreateShortURLResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "abc123xy", resp.ShortURL)
			},
		},
		{
			name: "success - valid URL with http",
			requestBody: CreateShortURLRequest{
				LongURL: "http://example.com",
			},
			setupMock: func(m *MockURLShortener) {
				m.On("CreateShortURL", mock.Anything, "http://example.com").Return(&entity.Link{
					ShortCode:   "xyz456ab",
					OriginalURL: "http://example.com",
				}, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp CreateShortURLResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "xyz456ab", resp.ShortURL)
			},
		},
		{
			name: "success - URL without protocol prefix",
			requestBody: CreateShortURLRequest{
				LongURL: "google.com",
			},
			setupMock: func(m *MockURLShortener) {
				m.On("CreateShortURL", mock.Anything, "https://google.com").Return(&entity.Link{
					ShortCode:   "def789gh",
					OriginalURL: "https://google.com",
				}, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp CreateShortURLResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "def789gh", resp.ShortURL)
			},
		},
		{
			name: "error - empty URL",
			requestBody: CreateShortURLRequest{
				LongURL: "",
			},
			setupMock:      func(m *MockURLShortener) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Contains(t, resp, "error")
			},
		},
		{
			name:           "error - malformed JSON",
			requestBody:    `{"long_url": invalid}`,
			setupMock:      func(m *MockURLShortener) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Contains(t, resp, "error")
			},
		},
		{
			name: "error - usecase duplicate error",
			requestBody: CreateShortURLRequest{
				LongURL: "https://example.com",
			},
			setupMock: func(m *MockURLShortener) {
				m.On("CreateShortURL", mock.Anything, "https://example.com").
					Return(nil, pkgerrors.ErrDuplicateShortCode)
			},
			expectedStatus: http.StatusConflict,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "Short code already exists", resp["error"])
			},
		},
		{
			name: "error - usecase internal error",
			requestBody: CreateShortURLRequest{
				LongURL: "https://example.com",
			},
			setupMock: func(m *MockURLShortener) {
				m.On("CreateShortURL", mock.Anything, "https://example.com").
					Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "Internal server error", resp["error"])
			},
		},
		{
			name: "success - URL with whitespace trimming",
			requestBody: CreateShortURLRequest{
				LongURL: "  https://example.com  ",
			},
			setupMock: func(m *MockURLShortener) {
				m.On("CreateShortURL", mock.Anything, "https://example.com").Return(&entity.Link{
					ShortCode:   "trim1234",
					OriginalURL: "https://example.com",
				}, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp CreateShortURLResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "trim1234", resp.ShortURL)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockShortener := new(MockURLShortener)
			tt.setupMock(mockShortener)

			handler := NewHandler(mockShortener, "http://localhost:8080")
			router := setupTestRouter()
			router.POST("/shorten", handler.CreateShortURL)

			var body []byte
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				var err error
				body, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.validateBody(t, rec)

			mockShortener.AssertExpectations(t)
		})
	}
}

func TestRedirectURL(t *testing.T) {
	tests := []struct {
		name           string
		shortCode      string
		setupMock      func(*MockURLShortener)
		expectedStatus int
		validateBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:      "success - valid short code redirect",
			shortCode: "abc123xy",
			setupMock: func(m *MockURLShortener) {
				m.On("GetOriginalURL", mock.Anything, "abc123xy").
					Return("https://google.com", nil)
			},
			expectedStatus: http.StatusMovedPermanently,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				location := rec.Header().Get("Location")
				assert.Equal(t, "https://google.com", location)
			},
		},
		{
			name:      "error - invalid short code format (too short)",
			shortCode: "abc123",
			setupMock: func(m *MockURLShortener) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "Invalid short URL format", resp["error"])
			},
		},
		{
			name:      "error - invalid short code format (too long)",
			shortCode: "abc123xyz",
			setupMock: func(m *MockURLShortener) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "Invalid short URL format", resp["error"])
			},
		},
		{
			name:      "error - invalid short code format (special characters)",
			shortCode: "abc@23xy",
			setupMock: func(m *MockURLShortener) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "Invalid short URL format", resp["error"])
			},
		},
		{
			name:      "error - short code not found",
			shortCode: "notfound",
			setupMock: func(m *MockURLShortener) {
				m.On("GetOriginalURL", mock.Anything, "notfound").
					Return("", pkgerrors.ErrLinkNotFound)
			},
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "Short URL not found", resp["error"])
			},
		},
		{
			name:      "error - usecase internal error",
			shortCode: "error123",
			setupMock: func(m *MockURLShortener) {
				m.On("GetOriginalURL", mock.Anything, "error123").
					Return("", errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "Internal server error", resp["error"])
			},
		},
		{
			name:      "success - valid alphanumeric short code",
			shortCode: "aB3Xy9Zq",
			setupMock: func(m *MockURLShortener) {
				m.On("GetOriginalURL", mock.Anything, "aB3Xy9Zq").
					Return("https://example.com/path", nil)
			},
			expectedStatus: http.StatusMovedPermanently,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				location := rec.Header().Get("Location")
				assert.Equal(t, "https://example.com/path", location)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockShortener := new(MockURLShortener)
			tt.setupMock(mockShortener)

			handler := NewHandler(mockShortener, "http://localhost:8080")
			router := setupTestRouter()
			router.GET("/:shortUrl", handler.RedirectURL)

			req := httptest.NewRequest(http.MethodGet, "/"+tt.shortCode, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.validateBody(t, rec)

			mockShortener.AssertExpectations(t)
		})
	}
}

func TestIsValidShortCode(t *testing.T) {
	tests := []struct {
		name      string
		shortCode string
		expected  bool
	}{
		{
			name:      "valid - 8 alphanumeric chars",
			shortCode: "abc123XY",
			expected:  true,
		},
		{
			name:      "valid - all lowercase",
			shortCode: "abcdefgh",
			expected:  true,
		},
		{
			name:      "valid - all uppercase",
			shortCode: "ABCDEFGH",
			expected:  true,
		},
		{
			name:      "valid - all numbers",
			shortCode: "12345678",
			expected:  true,
		},
		{
			name:      "invalid - too short",
			shortCode: "abc123",
			expected:  false,
		},
		{
			name:      "invalid - too long",
			shortCode: "abc123xyz",
			expected:  false,
		},
		{
			name:      "invalid - special characters",
			shortCode: "abc@23xy",
			expected:  false,
		},
		{
			name:      "invalid - with hyphen",
			shortCode: "abc-23xy",
			expected:  false,
		},
		{
			name:      "invalid - with underscore",
			shortCode: "abc_23xy",
			expected:  false,
		},
		{
			name:      "invalid - empty string",
			shortCode: "",
			expected:  false,
		},
		{
			name:      "invalid - with space",
			shortCode: "abc 23xy",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidShortCode(tt.shortCode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

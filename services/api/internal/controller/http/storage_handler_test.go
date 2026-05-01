package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Nanyak/thangdq-lab/internal/entity"
	pkgerrors "github.com/Nanyak/thangdq-lab/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockStorageService is a mock implementation of StorageService interface
type MockStorageService struct {
	mock.Mock
}

func (m *MockStorageService) Upload(ctx context.Context, key string, reader io.Reader, contentType string) error {
	args := m.Called(ctx, key, reader, contentType)
	return args.Error(0)
}

func (m *MockStorageService) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockStorageService) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockStorageService) List(ctx context.Context, prefix string) ([]*entity.File, error) {
	args := m.Called(ctx, prefix)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.File), args.Error(1)
}

func (m *MockStorageService) GetURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	args := m.Called(ctx, key, expiration)
	return args.String(0), args.Error(1)
}

func TestNewStorageHandler(t *testing.T) {
	mockStorage := new(MockStorageService)
	handler := NewStorageHandler(mockStorage)

	assert.NotNil(t, handler)
	assert.Equal(t, mockStorage, handler.storage)
}

func TestUploadFile(t *testing.T) {
	tests := []struct {
		name           string
		setupRequest   func() (*multipart.Writer, *bytes.Buffer)
		setupMock      func(*MockStorageService)
		expectedStatus int
		validateBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success - valid file upload",
			setupRequest: func() (*multipart.Writer, *bytes.Buffer) {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				part, _ := writer.CreateFormFile("file", "test.txt")
				part.Write([]byte("test content"))
				writer.Close()
				return writer, body
			},
			setupMock: func(m *MockStorageService) {
				m.On("Upload", mock.Anything, "test.txt", mock.Anything, "application/octet-stream").Return(nil)
				m.On("GetURL", mock.Anything, "test.txt", time.Hour).Return("https://s3.example.com/test.txt", nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp UploadFileResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "test.txt", resp.Key)
				assert.Equal(t, "https://s3.example.com/test.txt", resp.URL)
			},
		},
		{
			name: "error - missing file",
			setupRequest: func() (*multipart.Writer, *bytes.Buffer) {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				writer.WriteField("other", "value")
				writer.Close()
				return writer, body
			},
			setupMock:      func(m *MockStorageService) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "No file provided", resp["error"])
			},
		},
		{
			name: "error - upload to storage fails",
			setupRequest: func() (*multipart.Writer, *bytes.Buffer) {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				part, _ := writer.CreateFormFile("file", "test.txt")
				part.Write([]byte("test content"))
				writer.Close()
				return writer, body
			},
			setupMock: func(m *MockStorageService) {
				m.On("Upload", mock.Anything, "test.txt", mock.Anything, "application/octet-stream").
					Return(errors.New("s3 error"))
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "Failed to upload file", resp["error"])
			},
		},
		{
			name: "error - GetURL fails after upload",
			setupRequest: func() (*multipart.Writer, *bytes.Buffer) {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				part, _ := writer.CreateFormFile("file", "test.txt")
				part.Write([]byte("test content"))
				writer.Close()
				return writer, body
			},
			setupMock: func(m *MockStorageService) {
				m.On("Upload", mock.Anything, "test.txt", mock.Anything, "application/octet-stream").Return(nil)
				m.On("GetURL", mock.Anything, "test.txt", time.Hour).
					Return("", errors.New("presign error"))
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "Failed to generate URL", resp["error"])
			},
		},
		{
			name: "success - empty file upload",
			setupRequest: func() (*multipart.Writer, *bytes.Buffer) {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				part, _ := writer.CreateFormFile("file", "empty.txt")
				part.Write([]byte(""))
				writer.Close()
				return writer, body
			},
			setupMock: func(m *MockStorageService) {
				m.On("Upload", mock.Anything, "empty.txt", mock.Anything, "application/octet-stream").Return(nil)
				m.On("GetURL", mock.Anything, "empty.txt", time.Hour).Return("https://s3.example.com/empty.txt", nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp UploadFileResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "empty.txt", resp.Key)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := new(MockStorageService)
			tt.setupMock(mockStorage)

			handler := NewStorageHandler(mockStorage)
			router := setupTestRouter()
			router.POST("/upload", handler.UploadFile)

			writer, body := tt.setupRequest()
			req := httptest.NewRequest(http.MethodPost, "/upload", body)
			if writer != nil {
				req.Header.Set("Content-Type", writer.FormDataContentType())
			}
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.validateBody(t, rec)

			mockStorage.AssertExpectations(t)
		})
	}
}

func TestListFiles(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		queryPrefix    string
		setupMock      func(*MockStorageService)
		expectedStatus int
		validateBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:        "success - list all files",
			queryPrefix: "",
			setupMock: func(m *MockStorageService) {
				m.On("List", mock.Anything, "").Return([]*entity.File{
					{
						Key:          "file1.txt",
						Size:         100,
						ContentType:  "text/plain",
						LastModified: baseTime,
					},
					{
						Key:          "file2.jpg",
						Size:         2048,
						ContentType:  "image/jpeg",
						LastModified: baseTime.Add(time.Hour),
					},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp ListFilesResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Len(t, resp.Files, 2)
				assert.Equal(t, "file1.txt", resp.Files[0].Key)
				assert.Equal(t, int64(100), resp.Files[0].Size)
				assert.Equal(t, "text/plain", resp.Files[0].ContentType)
				assert.Equal(t, baseTime.Format(time.RFC3339), resp.Files[0].LastModified)
				assert.Equal(t, "file2.jpg", resp.Files[1].Key)
			},
		},
		{
			name:        "success - list with prefix",
			queryPrefix: "images/",
			setupMock: func(m *MockStorageService) {
				m.On("List", mock.Anything, "images/").Return([]*entity.File{
					{
						Key:          "images/photo.png",
						Size:         4096,
						ContentType:  "image/png",
						LastModified: baseTime,
					},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp ListFilesResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Len(t, resp.Files, 1)
				assert.Equal(t, "images/photo.png", resp.Files[0].Key)
			},
		},
		{
			name:        "success - empty list",
			queryPrefix: "",
			setupMock: func(m *MockStorageService) {
				m.On("List", mock.Anything, "").Return([]*entity.File{}, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp ListFilesResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Len(t, resp.Files, 0)
			},
		},
		{
			name:        "error - usecase error",
			queryPrefix: "",
			setupMock: func(m *MockStorageService) {
				m.On("List", mock.Anything, "").Return(nil, errors.New("s3 error"))
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "Failed to list files", resp["error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := new(MockStorageService)
			tt.setupMock(mockStorage)

			handler := NewStorageHandler(mockStorage)
			router := setupTestRouter()
			router.GET("/files", handler.ListFiles)

			url := "/files"
			if tt.queryPrefix != "" {
				url += "?prefix=" + tt.queryPrefix
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.validateBody(t, rec)

			mockStorage.AssertExpectations(t)
		})
	}
}

func TestDeleteFile(t *testing.T) {
	tests := []struct {
		name           string
		fileKey        string
		setupMock      func(*MockStorageService)
		expectedStatus int
		validateBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:    "success - valid delete",
			fileKey: "test.txt",
			setupMock: func(m *MockStorageService) {
				m.On("Delete", mock.Anything, "test.txt").Return(nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "File deleted successfully", resp["message"])
			},
		},
		{
			name:    "error - empty key",
			fileKey: "",
			setupMock: func(m *MockStorageService) {
				// No mock calls expected - handler returns early
			},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "File key is required", resp["error"])
			},
		},
		{
			name:    "error - file not found (ErrLinkNotFound)",
			fileKey: "nonexistent.txt",
			setupMock: func(m *MockStorageService) {
				m.On("Delete", mock.Anything, "nonexistent.txt").Return(pkgerrors.ErrLinkNotFound)
			},
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "File not found", resp["error"])
			},
		},
		{
			name:    "error - file not found (ErrFileNotFound)",
			fileKey: "other.txt",
			setupMock: func(m *MockStorageService) {
				m.On("Delete", mock.Anything, "other.txt").Return(pkgerrors.ErrFileNotFound)
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "Failed to delete file", resp["error"])
			},
		},
		{
			name:    "error - usecase internal error",
			fileKey: "error.txt",
			setupMock: func(m *MockStorageService) {
				m.On("Delete", mock.Anything, "error.txt").Return(errors.New("s3 error"))
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "Failed to delete file", resp["error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := new(MockStorageService)
			tt.setupMock(mockStorage)

			handler := NewStorageHandler(mockStorage)
			router := setupTestRouter()
			router.DELETE("/files/:key", handler.DeleteFile)

			url := "/files/" + tt.fileKey
			// For empty key test, try with explicit path to test the handler's empty check
			if tt.fileKey == "" {
				// Register a catch-all route for testing empty key validation
				router.DELETE("/files", func(c *gin.Context) {
					// Manually set the key param to empty to simulate the edge case
					c.Params = append(c.Params, gin.Param{Key: "key", Value: ""})
					handler.DeleteFile(c)
				})
				url = "/files"
			}
			req := httptest.NewRequest(http.MethodDelete, url, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.validateBody(t, rec)

			mockStorage.AssertExpectations(t)
		})
	}
}

func TestGetPresignedURL(t *testing.T) {
	tests := []struct {
		name           string
		fileKey        string
		setupMock      func(*MockStorageService)
		expectedStatus int
		validateBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:    "success - valid presigned URL",
			fileKey: "test.txt",
			setupMock: func(m *MockStorageService) {
				m.On("GetURL", mock.Anything, "test.txt", time.Hour).
					Return("https://s3.example.com/test.txt?signature=abc", nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp PresignedURLResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "https://s3.example.com/test.txt?signature=abc", resp.URL)
				assert.NotEmpty(t, resp.ExpiresAt)

				// Validate ExpiresAt is roughly 1 hour from now
				expiresAt, err := time.Parse(time.RFC3339, resp.ExpiresAt)
				assert.NoError(t, err)
				assert.True(t, expiresAt.After(time.Now()))
				assert.True(t, expiresAt.Before(time.Now().Add(2*time.Hour)))
			},
		},
		{
			name:    "error - empty key",
			fileKey: "",
			setupMock: func(m *MockStorageService) {
				// No mock calls expected - handler returns early
			},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "File key is required", resp["error"])
			},
		},
		{
			name:    "error - file not found (ErrLinkNotFound)",
			fileKey: "nonexistent.txt",
			setupMock: func(m *MockStorageService) {
				m.On("GetURL", mock.Anything, "nonexistent.txt", time.Hour).
					Return("", pkgerrors.ErrLinkNotFound)
			},
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "File not found", resp["error"])
			},
		},
		{
			name:    "error - file not found (ErrFileNotFound)",
			fileKey: "other.txt",
			setupMock: func(m *MockStorageService) {
				m.On("GetURL", mock.Anything, "other.txt", time.Hour).
					Return("", pkgerrors.ErrFileNotFound)
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "Failed to generate URL", resp["error"])
			},
		},
		{
			name:    "error - usecase internal error",
			fileKey: "error.txt",
			setupMock: func(m *MockStorageService) {
				m.On("GetURL", mock.Anything, "error.txt", time.Hour).
					Return("", errors.New("presign error"))
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "Failed to generate URL", resp["error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := new(MockStorageService)
			tt.setupMock(mockStorage)

			handler := NewStorageHandler(mockStorage)
			router := setupTestRouter()
			router.GET("/files/:key/url", handler.GetPresignedURL)

			url := "/files/" + tt.fileKey + "/url"
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.validateBody(t, rec)

			mockStorage.AssertExpectations(t)
		})
	}
}

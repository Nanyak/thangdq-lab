package http

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Nanyak/thangdq-lab/internal/entity"
	"github.com/Nanyak/thangdq-lab/pkg/errors"
	"github.com/gin-gonic/gin"
)

// StorageService defines the storage use case interface
type StorageService interface {
	Upload(ctx context.Context, key string, reader io.Reader, contentType string) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) ([]*entity.File, error)
	GetURL(ctx context.Context, key string, expiration time.Duration) (string, error)
}

type StorageHandler struct {
	storage StorageService
}

func NewStorageHandler(storage StorageService) *StorageHandler {
	return &StorageHandler{
		storage: storage,
	}
}

// UploadFile handles file upload via multipart/form-data
func (h *StorageHandler) UploadFile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}
	defer src.Close()

	// Prefix key with userId
	key := userID.(string) + "/" + file.Filename
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if err := h.storage.Upload(c.Request.Context(), key, src, contentType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file"})
		return
	}

	url, err := h.storage.GetURL(c.Request.Context(), key, time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate URL"})
		return
	}

	resp := UploadFileResponse{
		Key: file.Filename, // Return original filename to client
		URL: url,
	}

	c.JSON(http.StatusOK, resp)
}

// ListFiles returns all files for the authenticated user
func (h *StorageHandler) ListFiles(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Always filter by user's prefix
	prefix := userID.(string) + "/"

	files, err := h.storage.List(c.Request.Context(), prefix)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list files"})
		return
	}

	fileInfos := make([]FileInfo, 0, len(files))
	for _, f := range files {
		// Strip userId prefix from key for client
		displayKey := strings.TrimPrefix(f.Key, prefix)
		fileInfos = append(fileInfos, FileInfo{
			Key:          displayKey,
			Size:         f.Size,
			ContentType:  f.ContentType,
			LastModified: f.LastModified.Format(time.RFC3339),
		})
	}

	resp := ListFilesResponse{
		Files: fileInfos,
	}

	c.JSON(http.StatusOK, resp)
}

// DeleteFile removes a file from storage
func (h *StorageHandler) DeleteFile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File key is required"})
		return
	}

	// Construct full key with userId prefix
	fullKey := userID.(string) + "/" + key

	if err := h.storage.Delete(c.Request.Context(), fullKey); err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File deleted successfully"})
}

// GetPresignedURL generates a presigned download URL
func (h *StorageHandler) GetPresignedURL(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File key is required"})
		return
	}

	// Construct full key with userId prefix
	fullKey := userID.(string) + "/" + key

	expiration := time.Hour
	url, err := h.storage.GetURL(c.Request.Context(), fullKey, expiration)
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate URL"})
		return
	}

	expiresAt := time.Now().Add(expiration)
	resp := PresignedURLResponse{
		URL:       url,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, resp)
}

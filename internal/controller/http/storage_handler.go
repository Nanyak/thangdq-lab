package http

import (
	"context"
	"io"
	"log"
	"net/http"
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
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}

	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}
	defer src.Close()

	// Use filename as key
	key := file.Filename
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Upload to storage
	if err := h.storage.Upload(c.Request.Context(), key, src, contentType); err != nil {
		log.Printf("ERROR: Upload failed for key=%q: %v", key, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file"})
		return
	}

	// Generate presigned URL for immediate access (1 hour expiry)
	url, err := h.storage.GetURL(c.Request.Context(), key, time.Hour)
	if err != nil {
		log.Printf("ERROR: GetURL failed for key=%q: %v", key, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate URL"})
		return
	}

	resp := UploadFileResponse{
		Key: key,
		URL: url,
	}

	c.JSON(http.StatusOK, resp)
}

// ListFiles returns all files matching the optional prefix
func (h *StorageHandler) ListFiles(c *gin.Context) {
	prefix := c.Query("prefix")

	files, err := h.storage.List(c.Request.Context(), prefix)
	if err != nil {
		log.Printf("ERROR: ListFiles failed with prefix=%q: %v", prefix, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list files"})
		return
	}

	// Convert entity.File to FileInfo DTOs
	fileInfos := make([]FileInfo, 0, len(files))
	for _, f := range files {
		fileInfos = append(fileInfos, FileInfo{
			Key:          f.Key,
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
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File key is required"})
		return
	}

	if err := h.storage.Delete(c.Request.Context(), key); err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
			return
		}
		log.Printf("ERROR: Delete failed for key=%q: %v", key, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File deleted successfully"})
}

// GetPresignedURL generates a presigned download URL
func (h *StorageHandler) GetPresignedURL(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File key is required"})
		return
	}

	// Generate presigned URL with 1 hour expiration
	expiration := time.Hour
	url, err := h.storage.GetURL(c.Request.Context(), key, expiration)
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
			return
		}
		log.Printf("ERROR: GetPresignedURL failed for key=%q: %v", key, err)
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

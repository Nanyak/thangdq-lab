package http

import (
	"context"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Nanyak/thangdq-lab/internal/entity"
	"github.com/Nanyak/thangdq-lab/internal/service/embed"
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
	storage  StorageService
	embedPub embed.Publisher
}

func NewStorageHandler(storage StorageService, embedPub ...embed.Publisher) *StorageHandler {
	var publisher embed.Publisher
	if len(embedPub) > 0 {
		publisher = embedPub[0]
	}
	return &StorageHandler{
		storage:  storage,
		embedPub: publisher,
	}
}

// UploadFile handles file upload via multipart/form-data.
// Optional form field "path" sets the folder prefix (e.g. "documents/2024").
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

	// Optional folder path; sanitize to prevent path traversal
	folderPath := sanitizePath(strings.TrimSpace(c.PostForm("path")))

	var objectKey string
	if folderPath != "" {
		objectKey = userID.(string) + "/" + folderPath + "/" + file.Filename
	} else {
		objectKey = userID.(string) + "/" + file.Filename
	}

	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if err := h.storage.Upload(c.Request.Context(), objectKey, src, contentType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file"})
		return
	}

	if h.embedPub != nil {
		go func() {
			job := embed.Job{
				Type:     "embed",
				FileID:   objectKey,
				FileName: file.Filename,
				S3Key:    objectKey,
				UserID:   userID.(string),
				Folder:   folderPath,
				MimeType: contentType,
			}
			if err := h.embedPub.Publish(context.Background(), job); err != nil {
				log.Printf("embed publish failed for %s: %v", objectKey, err)
			}
		}()
	}

	url, err := h.storage.GetURL(c.Request.Context(), objectKey, time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate URL"})
		return
	}

	// Return key relative to user prefix
	displayKey := objectKey[len(userID.(string))+1:]
	c.JSON(http.StatusOK, UploadFileResponse{
		Key: displayKey,
		URL: url,
	})
}

// ListFiles returns all files for the authenticated user
func (h *StorageHandler) ListFiles(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	prefix := userID.(string) + "/"

	files, err := h.storage.List(c.Request.Context(), prefix)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list files"})
		return
	}

	fileInfos := make([]FileInfo, 0, len(files))
	for _, f := range files {
		displayKey := strings.TrimPrefix(f.Key, prefix)
		fileInfos = append(fileInfos, FileInfo{
			Key:          displayKey,
			Size:         f.Size,
			ContentType:  f.ContentType,
			LastModified: f.LastModified.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, ListFilesResponse{Files: fileInfos})
}

// DeleteFile removes a file from storage. Uses query param "key".
func (h *StorageHandler) DeleteFile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File key is required"})
		return
	}

	fullKey := userID.(string) + "/" + sanitizePath(key)

	if err := h.storage.Delete(c.Request.Context(), fullKey); err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file"})
		return
	}

	if h.embedPub != nil {
		go func() {
			job := embed.Job{
				Type:   "delete",
				FileID: fullKey,
				UserID: userID.(string),
			}
			if err := h.embedPub.Publish(context.Background(), job); err != nil {
				log.Printf("embed delete publish failed for %s: %v", fullKey, err)
			}
		}()
	}

	c.JSON(http.StatusOK, gin.H{"message": "File deleted successfully"})
}

// GetPresignedURL generates a presigned download URL. Uses query param "key".
func (h *StorageHandler) GetPresignedURL(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File key is required"})
		return
	}

	fullKey := userID.(string) + "/" + sanitizePath(key)
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

	c.JSON(http.StatusOK, PresignedURLResponse{
		URL:       url,
		ExpiresAt: time.Now().Add(expiration).Format(time.RFC3339),
	})
}

// sanitizePath removes leading/trailing slashes and rejects path traversal attempts.
func sanitizePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, "/")
	// Reject traversal
	if strings.Contains(path, "..") {
		return ""
	}
	return path
}

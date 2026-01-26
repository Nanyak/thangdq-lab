package http

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/Nanyak/thangdq-lab/internal/entity"
	"github.com/Nanyak/thangdq-lab/pkg/errors"
	"github.com/gin-gonic/gin"
)

// URLShortener defines the use case interface at consumer side
type URLShortener interface {
	CreateShortURL(ctx context.Context, originalURL string) (*entity.Link, error)
	GetOriginalURL(ctx context.Context, shortCode string) (string, error)
}

type Handler struct {
	shortener URLShortener
	baseURL   string
}

func NewHandler(shortener URLShortener, baseURL string) *Handler {
	return &Handler{
		shortener: shortener,
		baseURL:   baseURL,
	}
}

func (h *Handler) CreateShortURL(c *gin.Context) {
	var req CreateShortURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Format URL with http/https prefix
	longURL := strings.TrimSpace(req.LongURL)
	if !strings.HasPrefix(longURL, "http://") && !strings.HasPrefix(longURL, "https://") {
		longURL = "https://" + longURL
	}

	// Validate URL format
	if _, err := url.Parse(longURL); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid URL format"})
		return
	}

	// Call use case
	link, err := h.shortener.CreateShortURL(c.Request.Context(), longURL)
	if err != nil {
		if errors.IsDuplicate(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "Short code already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Build response - return short code only (matches old behavior)
	resp := CreateShortURLResponse{
		ShortURL: link.ShortCode,
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) RedirectURL(c *gin.Context) {
	shortCode := c.Param("shortUrl")

	// Validate shortCode format: alphanumeric, exactly 8 characters
	if !isValidShortCode(shortCode) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid short URL format"})
		return
	}

	// Call use case
	originalURL, err := h.shortener.GetOriginalURL(c.Request.Context(), shortCode)
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Short URL not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.Redirect(http.StatusMovedPermanently, originalURL)
}

var shortCodePattern = regexp.MustCompile(`^[a-zA-Z0-9]{8}$`)

func isValidShortCode(code string) bool {
	return shortCodePattern.MatchString(code)
}

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
	CreateShortURL(ctx context.Context, originalURL string, userID string) (*entity.Link, error)
	GetOriginalURL(ctx context.Context, shortCode string) (string, error)
	GetUserLinks(ctx context.Context, userID string) ([]*entity.Link, error)
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

	// Extract user_id from context (set by OptionalAuthMiddleware, empty if anonymous)
	userID, _ := c.Get("user_id")
	userIDStr, _ := userID.(string)

	link, err := h.shortener.CreateShortURL(c.Request.Context(), longURL, userIDStr)
	if err != nil {
		if errors.IsDuplicate(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "Short code already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, CreateShortURLResponse{ShortURL: link.ShortCode})
}

func (h *Handler) RedirectURL(c *gin.Context) {
	shortCode := c.Param("shortUrl")

	// Validate shortCode format: alphanumeric, exactly 8 characters
	if !isValidShortCode(shortCode) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid short URL format"})
		return
	}

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

func (h *Handler) GetUserLinks(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userIDStr, _ := userID.(string)

	links, err := h.shortener.GetUserLinks(c.Request.Context(), userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	dtos := make([]LinkDTO, 0, len(links))
	for _, l := range links {
		dtos = append(dtos, LinkDTO{
			ShortCode:   l.ShortCode,
			OriginalURL: l.OriginalURL,
			Title:       l.Title,
			CreatedAt:   l.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, GetUserLinksResponse{Links: dtos})
}

var shortCodePattern = regexp.MustCompile(`^[a-zA-Z0-9]{8}$`)

func isValidShortCode(code string) bool {
	return shortCodePattern.MatchString(code)
}

package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Nanyak/thangdq-lab/shortener"
	"github.com/Nanyak/thangdq-lab/store"
	"github.com/gin-gonic/gin"
)

type UrlCreateRequest struct {
	LongUrl string `json:"long_url" binding:"required"`
}

type UrlCreateResponse struct {
	ShortUrl string `json:"short_url"`
}

func CreateShortUrlHandler(c *gin.Context, storeService *store.StorageService) {
	var createRequest UrlCreateRequest
	if err := c.ShouldBindJSON(&createRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	longUrl := strings.TrimSpace(createRequest.LongUrl)
	if !strings.HasPrefix(longUrl, "http://") && !strings.HasPrefix(longUrl, "https://") {
		longUrl = "https://" + longUrl
	}

	fmt.Printf("Creating short URL for: %s\n", createRequest.LongUrl)
	shortUrl := shortener.GenerateShortUrl(createRequest.LongUrl)
	fmt.Printf("Generated short URL: %s\n", shortUrl)

	err := storeService.SaveUrlMapping(c.Request.Context(), shortUrl, createRequest.LongUrl)

	if err != nil {
		fmt.Printf("Error saving URL mapping: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to save URL mapping",
			"details": err.Error(),
		})
		return
	}

	response := UrlCreateResponse{
		ShortUrl: shortUrl,
	}
	c.JSON(http.StatusOK, response)
}
func RedirectUrlHandler(c *gin.Context, storeService *store.StorageService) {
	shortUrl := c.Param("shortUrl")

	originalUrl, err := storeService.RetrieveInitialUrl(c.Request.Context(), shortUrl)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Short URL not found"})
		return
	}

	c.Redirect(http.StatusFound, originalUrl)
}

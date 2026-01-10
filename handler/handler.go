package handler

import (
	"net/http"

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

	shortUrl := shortener.GenerateShortUrl(createRequest.LongUrl)

	err := storeService.SaveUrlMapping(c.Request.Context(), shortUrl, createRequest.LongUrl)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save URL mapping"})
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

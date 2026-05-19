package http

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AIClient interface {
	Query(question, scope, userID string) (io.ReadCloser, error)
	Chat(message, scope, userID string, history []map[string]string) (io.ReadCloser, error)
}

type AIHandler struct {
	client AIClient
}

func NewAIHandler(client AIClient) *AIHandler {
	return &AIHandler{client: client}
}

func (h *AIHandler) Query(c *gin.Context) {
	question := c.Query("q")
	if question == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "q is required"})
		return
	}
	scope := c.DefaultQuery("scope", "all")

	userID, _ := c.Get("user_id")
	userIDStr, _ := userID.(string)

	body, err := h.client.Query(question, scope, userIDStr)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service unavailable"})
		return
	}
	defer body.Close()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	io.Copy(c.Writer, body) //nolint:errcheck
	c.Writer.Flush()
}

func (h *AIHandler) Chat(c *gin.Context) {
	var req AIChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message is required"})
		return
	}
	if req.Scope == "" {
		req.Scope = "all"
	}

	userID, _ := c.Get("user_id")
	userIDStr, _ := userID.(string)

	history := make([]map[string]string, 0, len(req.History))
	for _, item := range req.History {
		history = append(history, map[string]string{
			"role":    item.Role,
			"content": item.Content,
		})
	}

	body, err := h.client.Chat(req.Message, req.Scope, userIDStr, history)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service unavailable"})
		return
	}
	defer body.Close()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	io.Copy(c.Writer, body) //nolint:errcheck
	c.Writer.Flush()
}

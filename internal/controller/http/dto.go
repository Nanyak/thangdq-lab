package http

type CreateShortURLRequest struct {
	LongURL string `json:"long_url" binding:"required"`
}

type CreateShortURLResponse struct {
	ShortURL string `json:"short_url"`
}

package http

type CreateShortURLRequest struct {
	LongURL string `json:"long_url" binding:"required"`
}

type CreateShortURLResponse struct {
	ShortURL string `json:"short_url"`
}

// Storage DTOs

type UploadFileResponse struct {
	Key string `json:"key"`
	URL string `json:"url"`
}

type FileInfo struct {
	Key          string `json:"key"`
	Size         int64  `json:"size"`
	ContentType  string `json:"content_type"`
	LastModified string `json:"last_modified"`
}

type ListFilesResponse struct {
	Files []FileInfo `json:"files"`
}

type PresignedURLResponse struct {
	URL       string `json:"url"`
	ExpiresAt string `json:"expires_at"`
}

package embed

import "context"

type Job struct {
	Type     string `json:"type"` // "embed" | "delete"
	FileID   string `json:"file_id"`
	FileName string `json:"file_name"`
	S3Key    string `json:"s3_key"`
	UserID   string `json:"user_id"`
	Folder   string `json:"folder"`
	MimeType string `json:"mime_type"`
}

type Publisher interface {
	Publish(ctx context.Context, job Job) error
}

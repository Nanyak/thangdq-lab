package entity

import "time"

// Link represents a URL shortening mapping
type Link struct {
	ShortCode   string
	OriginalURL string
	CreatedAt   time.Time
}

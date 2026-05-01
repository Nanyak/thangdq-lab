package entity

import "time"

// Link represents a URL shortening mapping
type Link struct {
	ShortCode   string
	OriginalURL string
	UserID      string
	Title       string
	CreatedAt   time.Time
}

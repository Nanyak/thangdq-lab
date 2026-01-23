package entity

import "time"

// File represents a cloud storage file object
type File struct {
	Key          string
	Size         int64
	ContentType  string
	LastModified time.Time
}

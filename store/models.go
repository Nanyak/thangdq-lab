package store

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Link struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	ShortURL    string             `bson:"short_url"`
	OriginalURL string             `bson:"original_url"`
	CreatedAt   time.Time          `bson:"created_at"`
}

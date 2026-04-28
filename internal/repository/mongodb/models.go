package mongodb

import (
	"github.com/Nanyak/thangdq-lab/internal/entity"
	"time"
)

// LinkDocument is MongoDB representation of Link entity
type LinkDocument struct {
	ShortCode   string    `bson:"short_code"`
	OriginalURL string    `bson:"original_url"`
	UserID      string    `bson:"user_id,omitempty"`
	Title       string    `bson:"title,omitempty"`
	CreatedAt   time.Time `bson:"created_at"`
}

func toEntity(doc *LinkDocument) *entity.Link {
	return &entity.Link{
		ShortCode:   doc.ShortCode,
		OriginalURL: doc.OriginalURL,
		UserID:      doc.UserID,
		Title:       doc.Title,
		CreatedAt:   doc.CreatedAt,
	}
}

func toDocument(link *entity.Link) *LinkDocument {
	return &LinkDocument{
		ShortCode:   link.ShortCode,
		OriginalURL: link.OriginalURL,
		UserID:      link.UserID,
		Title:       link.Title,
		CreatedAt:   link.CreatedAt,
	}
}

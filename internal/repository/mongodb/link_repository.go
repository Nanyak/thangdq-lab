package mongodb

import (
	"context"
	"fmt"
	"github.com/Nanyak/thangdq-lab/internal/entity"
	"github.com/Nanyak/thangdq-lab/pkg/errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type LinkRepository struct {
	collection *mongo.Collection
}

func NewLinkRepository(db *mongo.Database, collectionName string) (*LinkRepository, error) {
	collection := db.Collection(collectionName)

	// Create indexes
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "short_code", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	if _, err := collection.Indexes().CreateOne(context.Background(), indexModel); err != nil {
		return nil, fmt.Errorf("failed to create index: %w", err)
	}

	return &LinkRepository{collection: collection}, nil
}

func (r *LinkRepository) Save(ctx context.Context, link *entity.Link) error {
	doc := toDocument(link)

	_, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return errors.ErrDuplicateShortCode
		}
		return fmt.Errorf("mongodb insert failed: %w", err)
	}

	return nil
}

func (r *LinkRepository) FindByShortCode(ctx context.Context, shortCode string) (*entity.Link, error) {
	filter := bson.M{"short_code": shortCode}

	var doc LinkDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.ErrLinkNotFound
		}
		return nil, fmt.Errorf("mongodb find failed: %w", err)
	}

	return toEntity(&doc), nil
}

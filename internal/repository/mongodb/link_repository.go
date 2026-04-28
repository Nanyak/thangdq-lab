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

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "short_code", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "user_id", Value: 1}},
		},
	}

	if _, err := collection.Indexes().CreateMany(context.Background(), indexes); err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
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

func (r *LinkRepository) FindByUserID(ctx context.Context, userID string) ([]*entity.Link, error) {
	filter := bson.M{"user_id": userID}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("mongodb find failed: %w", err)
	}
	defer cursor.Close(ctx)

	var links []*entity.Link
	for cursor.Next(ctx) {
		var doc LinkDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("mongodb decode failed: %w", err)
		}
		links = append(links, toEntity(&doc))
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("mongodb cursor error: %w", err)
	}

	return links, nil
}

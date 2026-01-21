package store

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrLinkNotFound = errors.New("link not found in database")

type MongoRepository struct {
	collection *mongo.Collection
}

func NewMongoRepository(collection *mongo.Collection) *MongoRepository {
	return &MongoRepository{collection: collection}
}

func (r *MongoRepository) CreateLink(ctx context.Context, shortURL, originalURL string) error {
	link := Link{
		ShortURL:    shortURL,
		OriginalURL: originalURL,
		CreatedAt:   time.Now(),
	}

	_, err := r.collection.InsertOne(ctx, link)
	return err
}

func (r *MongoRepository) GetLinkByShortURL(ctx context.Context, shortURL string) (*Link, error) {
	var link Link
	filter := bson.M{"short_url": shortURL}

	err := r.collection.FindOne(ctx, filter).Decode(&link)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrLinkNotFound
		}
		return nil, err
	}

	return &link, nil
}

func (r *MongoRepository) EnsureIndexes(ctx context.Context) error {
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "short_url", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	_, err := r.collection.Indexes().CreateOne(ctx, indexModel)
	return err
}

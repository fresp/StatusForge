package repository

import (
	"context"

	"github.com/fresp/StatusForge/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type WebhookChannelRepository interface {
	List(ctx context.Context, page, limit int) ([]models.WebhookChannel, int64, error)
	Insert(ctx context.Context, channel models.WebhookChannel) error
	DeleteByID(ctx context.Context, id primitive.ObjectID) (bool, error)
}

type MongoWebhookChannelRepository struct {
	collection *mongo.Collection
}

func NewMongoWebhookChannelRepository(db *mongo.Database) *MongoWebhookChannelRepository {
	return &MongoWebhookChannelRepository{collection: db.Collection("webhook_channels")}
}

func (r *MongoWebhookChannelRepository) List(ctx context.Context, page, limit int) ([]models.WebhookChannel, int64, error) {
	total, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, err
	}

	skip := int64((page - 1) * limit)
	cursor, err := r.collection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetSkip(skip).SetLimit(int64(limit)))
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var channels []models.WebhookChannel
	if err := cursor.All(ctx, &channels); err != nil {
		return nil, 0, err
	}
	if channels == nil {
		channels = []models.WebhookChannel{}
	}

	return channels, total, nil
}

func (r *MongoWebhookChannelRepository) Insert(ctx context.Context, channel models.WebhookChannel) error {
	_, err := r.collection.InsertOne(ctx, channel)
	return err
}

func (r *MongoWebhookChannelRepository) DeleteByID(ctx context.Context, id primitive.ObjectID) (bool, error) {
	res, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return false, err
	}
	return res.DeletedCount > 0, nil
}

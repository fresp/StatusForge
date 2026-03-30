package repository

import (
	"context"
	"errors"
	"time"

	"github.com/fresp/StatusForge/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SubscriberRepository interface {
	FindByEmail(ctx context.Context, email string) (*models.Subscriber, error)
	Insert(ctx context.Context, sub models.Subscriber) error
	List(ctx context.Context, page, limit int) ([]models.Subscriber, int64, error)
	DeleteByID(ctx context.Context, id primitive.ObjectID) (bool, error)
}

type MongoSubscriberRepository struct {
	collection *mongo.Collection
}

func NewMongoSubscriberRepository(db *mongo.Database) *MongoSubscriberRepository {
	return &MongoSubscriberRepository{collection: db.Collection("subscribers")}
}

func (r *MongoSubscriberRepository) FindByEmail(ctx context.Context, email string) (*models.Subscriber, error) {
	var existing models.Subscriber
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&existing)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &existing, nil
}

func (r *MongoSubscriberRepository) Insert(ctx context.Context, sub models.Subscriber) error {
	_, err := r.collection.InsertOne(ctx, sub)
	return err
}

func (r *MongoSubscriberRepository) List(ctx context.Context, page, limit int) ([]models.Subscriber, int64, error) {
	total, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, err
	}

	skip := int64((page - 1) * limit)
	cursor, err := r.collection.Find(ctx, bson.M{},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetSkip(skip).SetLimit(int64(limit)))
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var subs []models.Subscriber
	if err := cursor.All(ctx, &subs); err != nil {
		return nil, 0, err
	}
	if subs == nil {
		subs = []models.Subscriber{}
	}

	return subs, total, nil
}

func (r *MongoSubscriberRepository) DeleteByID(ctx context.Context, id primitive.ObjectID) (bool, error) {
	res, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return false, err
	}
	return res.DeletedCount > 0, nil
}

func NewSubscriber(email string) models.Subscriber {
	return models.Subscriber{
		ID:        primitive.NewObjectID(),
		Email:     email,
		Verified:  false,
		CreatedAt: time.Now(),
	}
}

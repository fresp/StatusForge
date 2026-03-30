package repository

import (
	"context"

	"github.com/fresp/StatusForge/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SubComponentRepository interface {
	List(ctx context.Context, filter bson.M, page, limit int) ([]models.SubComponent, int64, error)
	Insert(ctx context.Context, sub models.SubComponent) error
	UpdateByID(ctx context.Context, id primitive.ObjectID, setFields bson.M) (models.SubComponent, error)
}

type MongoSubComponentRepository struct {
	collection *mongo.Collection
}

func NewMongoSubComponentRepository(db *mongo.Database) *MongoSubComponentRepository {
	return &MongoSubComponentRepository{collection: db.Collection("subcomponents")}
}

func (r *MongoSubComponentRepository) List(ctx context.Context, filter bson.M, page, limit int) ([]models.SubComponent, int64, error) {
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := int64((page - 1) * limit)
	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}).SetSkip(skip).SetLimit(int64(limit)))
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var subs []models.SubComponent
	if err := cursor.All(ctx, &subs); err != nil {
		return nil, 0, err
	}
	if subs == nil {
		subs = []models.SubComponent{}
	}

	return subs, total, nil
}

func (r *MongoSubComponentRepository) Insert(ctx context.Context, sub models.SubComponent) error {
	_, err := r.collection.InsertOne(ctx, sub)
	return err
}

func (r *MongoSubComponentRepository) UpdateByID(ctx context.Context, id primitive.ObjectID, setFields bson.M) (models.SubComponent, error) {
	var sub models.SubComponent
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	err := r.collection.FindOneAndUpdate(ctx, bson.M{"_id": id}, bson.M{"$set": setFields}, opts).Decode(&sub)
	if err != nil {
		return models.SubComponent{}, err
	}

	return sub, nil
}

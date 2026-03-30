package repository

import (
	"context"
	"time"

	"github.com/fresp/StatusForge/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MaintenanceRepository interface {
	List(ctx context.Context, page, limit int) ([]models.Maintenance, int64, error)
	ListPublic(ctx context.Context, page, limit int) ([]models.Maintenance, int64, error)
	Insert(ctx context.Context, maintenance models.Maintenance) error
	UpdateByID(ctx context.Context, id primitive.ObjectID, setFields bson.M) (models.Maintenance, error)
}

type MongoMaintenanceRepository struct {
	collection *mongo.Collection
}

func NewMongoMaintenanceRepository(db *mongo.Database) *MongoMaintenanceRepository {
	return &MongoMaintenanceRepository{collection: db.Collection("maintenance")}
}

func (r *MongoMaintenanceRepository) List(ctx context.Context, page, limit int) ([]models.Maintenance, int64, error) {
	return r.listByFilter(ctx, bson.M{}, page, limit)
}

func (r *MongoMaintenanceRepository) ListPublic(ctx context.Context, page, limit int) ([]models.Maintenance, int64, error) {
	return r.listByFilter(ctx, bson.M{"status": bson.M{"$ne": models.MaintenanceCompleted}}, page, limit)
}

func (r *MongoMaintenanceRepository) listByFilter(ctx context.Context, filter bson.M, page, limit int) ([]models.Maintenance, int64, error) {
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := int64((page - 1) * limit)
	cursor, err := r.collection.Find(
		ctx,
		filter,
		options.Find().SetSort(bson.D{{Key: "startTime", Value: -1}}).SetSkip(skip).SetLimit(int64(limit)),
	)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var items []models.Maintenance
	if err := cursor.All(ctx, &items); err != nil {
		return nil, 0, err
	}
	if items == nil {
		items = []models.Maintenance{}
	}

	return items, total, nil
}

func (r *MongoMaintenanceRepository) Insert(ctx context.Context, maintenance models.Maintenance) error {
	_, err := r.collection.InsertOne(ctx, maintenance)
	return err
}

func (r *MongoMaintenanceRepository) UpdateByID(ctx context.Context, id primitive.ObjectID, setFields bson.M) (models.Maintenance, error) {
	setFields["updatedAt"] = time.Now()
	var result models.Maintenance
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	err := r.collection.FindOneAndUpdate(ctx, bson.M{"_id": id}, bson.M{"$set": setFields}, opts).Decode(&result)
	if err != nil {
		return models.Maintenance{}, err
	}

	return result, nil
}

package repository

import (
	"context"
	"log"
	"time"

	"github.com/fresp/Statora/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SubComponentRepository interface {
	List(ctx context.Context, filter bson.M, page, limit int) ([]models.SubComponent, int64, error)
	Insert(ctx context.Context, sub models.SubComponent) error
	UpdateByID(ctx context.Context, id primitive.ObjectID, setFields bson.M) (models.SubComponent, error)
	FindByID(ctx context.Context, id primitive.ObjectID) (models.SubComponent, error)
	DeleteByID(ctx context.Context, id primitive.ObjectID) (int64, error)
	CountByComponentID(ctx context.Context, componentID primitive.ObjectID) (int64, error)
	ComponentExists(ctx context.Context, id primitive.ObjectID) (bool, error)
	CleanupReferencesForDeletedSubComponent(ctx context.Context, subComponentID primitive.ObjectID, componentID primitive.ObjectID) error
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
	var before models.SubComponent
	if err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&before); err != nil {
		return models.SubComponent{}, err
	}

	log.Printf("[SUBCOMPONENT_REPO] update request id=%s before=%+v set=%+v", id.Hex(), before, setFields)

	var sub models.SubComponent
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	err := r.collection.FindOneAndUpdate(ctx, bson.M{"_id": id}, bson.M{"$set": setFields}, opts).Decode(&sub)
	if err != nil {
		return models.SubComponent{}, err
	}

	log.Printf("[SUBCOMPONENT_REPO] update result id=%s after=%+v", id.Hex(), sub)

	return sub, nil
}

func (r *MongoSubComponentRepository) FindByID(ctx context.Context, id primitive.ObjectID) (models.SubComponent, error) {
	var sub models.SubComponent
	if err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&sub); err != nil {
		return models.SubComponent{}, err
	}

	return sub, nil
}

func (r *MongoSubComponentRepository) DeleteByID(ctx context.Context, id primitive.ObjectID) (int64, error) {
	res, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return 0, err
	}

	return res.DeletedCount, nil
}

func (r *MongoSubComponentRepository) CountByComponentID(ctx context.Context, componentID primitive.ObjectID) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{"componentId": componentID})
}

func (r *MongoSubComponentRepository) ComponentExists(ctx context.Context, id primitive.ObjectID) (bool, error) {
	count, err := r.collection.Database().Collection("components").CountDocuments(ctx, bson.M{"_id": id})
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *MongoSubComponentRepository) CleanupReferencesForDeletedSubComponent(ctx context.Context, subComponentID primitive.ObjectID, componentID primitive.ObjectID) error {
	if _, err := r.collection.Database().Collection("monitors").UpdateMany(
		ctx,
		bson.M{"subComponentId": subComponentID},
		bson.M{"$set": bson.M{"subComponentId": primitive.NilObjectID, "componentId": componentID, "updatedAt": time.Now()}},
	); err != nil {
		return err
	}

	if _, err := r.collection.Database().Collection("outages").UpdateMany(
		ctx,
		bson.M{"subComponentId": subComponentID},
		bson.M{"$set": bson.M{"subComponentId": primitive.NilObjectID, "componentId": componentID}},
	); err != nil {
		return err
	}

	if _, err := r.collection.Database().Collection("incidents").UpdateMany(
		ctx,
		bson.M{"affectedComponentTargets.subComponentIds": subComponentID},
		bson.M{"$pull": bson.M{"affectedComponentTargets.$[].subComponentIds": subComponentID}, "$set": bson.M{"updatedAt": time.Now()}},
	); err != nil {
		return err
	}

	return nil
}

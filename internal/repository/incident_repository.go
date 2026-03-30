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

type IncidentRepository interface {
	List(ctx context.Context, filter bson.M, page, limit int) ([]models.Incident, int64, error)
	InsertIncident(ctx context.Context, incident models.Incident) error
	UpdateIncidentByID(ctx context.Context, id primitive.ObjectID, setFields bson.M) (models.Incident, error)
	InsertUpdate(ctx context.Context, update models.IncidentUpdate) error
	ApplyIncidentStatus(ctx context.Context, incidentID primitive.ObjectID, status models.IncidentStatus) error
	ListUpdates(ctx context.Context, incidentID primitive.ObjectID) ([]models.IncidentUpdate, error)
	CountComponents(ctx context.Context, ids []primitive.ObjectID) (int64, error)
	CountSubComponentsByComponent(ctx context.Context, componentID primitive.ObjectID, ids []primitive.ObjectID) (int64, error)
}

type MongoIncidentRepository struct {
	incidents       *mongo.Collection
	incidentUpdates *mongo.Collection
	components      *mongo.Collection
	subcomponents   *mongo.Collection
}

func NewMongoIncidentRepository(db *mongo.Database) *MongoIncidentRepository {
	return &MongoIncidentRepository{
		incidents:       db.Collection("incidents"),
		incidentUpdates: db.Collection("incident_updates"),
		components:      db.Collection("components"),
		subcomponents:   db.Collection("subcomponents"),
	}
}

func (r *MongoIncidentRepository) List(ctx context.Context, filter bson.M, page, limit int) ([]models.Incident, int64, error) {
	total, err := r.incidents.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := int64((page - 1) * limit)
	cursor, err := r.incidents.Find(ctx, filter,
		options.Find().
			SetSort(bson.D{{Key: "createdAt", Value: -1}}).
			SetSkip(skip).
			SetLimit(int64(limit)))
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var incidents []models.Incident
	if err := cursor.All(ctx, &incidents); err != nil {
		return nil, 0, err
	}
	if incidents == nil {
		incidents = []models.Incident{}
	}

	return incidents, total, nil
}

func (r *MongoIncidentRepository) InsertIncident(ctx context.Context, incident models.Incident) error {
	_, err := r.incidents.InsertOne(ctx, incident)
	return err
}

func (r *MongoIncidentRepository) UpdateIncidentByID(ctx context.Context, id primitive.ObjectID, setFields bson.M) (models.Incident, error) {
	var incident models.Incident
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	err := r.incidents.FindOneAndUpdate(ctx, bson.M{"_id": id}, bson.M{"$set": setFields}, opts).Decode(&incident)
	if err != nil {
		return models.Incident{}, err
	}
	return incident, nil
}

func (r *MongoIncidentRepository) InsertUpdate(ctx context.Context, update models.IncidentUpdate) error {
	_, err := r.incidentUpdates.InsertOne(ctx, update)
	return err
}

func (r *MongoIncidentRepository) ApplyIncidentStatus(ctx context.Context, incidentID primitive.ObjectID, status models.IncidentStatus) error {
	updateFields := bson.M{
		"status":    status,
		"updatedAt": time.Now(),
	}
	if status == models.IncidentResolved {
		updateFields["resolvedAt"] = time.Now()
	}

	_, err := r.incidents.UpdateOne(ctx, bson.M{"_id": incidentID}, bson.M{"$set": updateFields})
	return err
}

func (r *MongoIncidentRepository) ListUpdates(ctx context.Context, incidentID primitive.ObjectID) ([]models.IncidentUpdate, error) {
	cursor, err := r.incidentUpdates.Find(ctx,
		bson.M{"incidentId": incidentID},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var updates []models.IncidentUpdate
	if err := cursor.All(ctx, &updates); err != nil {
		return nil, err
	}
	if updates == nil {
		updates = []models.IncidentUpdate{}
	}

	return updates, nil
}

func (r *MongoIncidentRepository) CountComponents(ctx context.Context, ids []primitive.ObjectID) (int64, error) {
	return r.components.CountDocuments(ctx, bson.M{"_id": bson.M{"$in": ids}})
}

func (r *MongoIncidentRepository) CountSubComponentsByComponent(ctx context.Context, componentID primitive.ObjectID, ids []primitive.ObjectID) (int64, error) {
	return r.subcomponents.CountDocuments(ctx, bson.M{
		"_id":         bson.M{"$in": ids},
		"componentId": componentID,
	})
}

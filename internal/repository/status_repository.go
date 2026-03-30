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

type StatusRepository interface {
	ListComponents(ctx context.Context) ([]models.Component, error)
	ListComponentsByIDs(ctx context.Context, ids []primitive.ObjectID) ([]models.Component, error)
	ListSubComponentsByComponentIDs(ctx context.Context, componentIDs []primitive.ObjectID) ([]models.SubComponent, error)
	ListAllSubComponents(ctx context.Context) ([]models.SubComponent, error)
	ListSubComponentsByIDs(ctx context.Context, ids []primitive.ObjectID) ([]models.SubComponent, error)
	ListMonitorsByTargets(ctx context.Context, componentIDs []primitive.ObjectID, subComponentIDs []primitive.ObjectID) ([]models.Monitor, error)
	ListDailyUptimeSinceByMonitorIDs(ctx context.Context, monitorIDs []primitive.ObjectID, since time.Time) ([]models.DailyUptime, error)
	ListActiveIncidents(ctx context.Context) ([]models.Incident, error)
	ListActiveMaintenanceAt(ctx context.Context, at time.Time) ([]models.Maintenance, error)
	CountActiveIncidents(ctx context.Context) (int64, error)
	CountActiveMaintenanceAt(ctx context.Context, at time.Time) (int64, error)
	FindLatestIncidentByComponent(ctx context.Context, componentID primitive.ObjectID) (*models.Incident, error)
	ListIncidentsByCreatedAtRange(ctx context.Context, start, end time.Time) ([]models.Incident, error)
	ListResolvedIncidentsSince(ctx context.Context, since time.Time) ([]models.Incident, error)
	ListIncidentsByAffectedComponents(ctx context.Context, affectedIDs []primitive.ObjectID, limit int64) ([]models.Incident, error)
	ListIncidentUpdatesByIncidentIDs(ctx context.Context, incidentIDs []primitive.ObjectID) (map[primitive.ObjectID][]models.IncidentUpdate, error)
}

type MongoStatusRepository struct {
	db *mongo.Database
}

func NewMongoStatusRepository(db *mongo.Database) *MongoStatusRepository {
	return &MongoStatusRepository{db: db}
}

func (r *MongoStatusRepository) ListComponents(ctx context.Context) ([]models.Component, error) {
	cursor, err := r.db.Collection("components").Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var components []models.Component
	if err := cursor.All(ctx, &components); err != nil {
		return nil, err
	}
	if components == nil {
		components = []models.Component{}
	}

	return components, nil
}

func (r *MongoStatusRepository) ListComponentsByIDs(ctx context.Context, ids []primitive.ObjectID) ([]models.Component, error) {
	if len(ids) == 0 {
		return []models.Component{}, nil
	}

	cursor, err := r.db.Collection("components").Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var components []models.Component
	if err := cursor.All(ctx, &components); err != nil {
		return nil, err
	}
	if components == nil {
		components = []models.Component{}
	}

	return components, nil
}

func (r *MongoStatusRepository) ListSubComponentsByComponentIDs(ctx context.Context, componentIDs []primitive.ObjectID) ([]models.SubComponent, error) {
	if len(componentIDs) == 0 {
		return []models.SubComponent{}, nil
	}

	cursor, err := r.db.Collection("subcomponents").Find(
		ctx,
		bson.M{"componentId": bson.M{"$in": componentIDs}},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var subs []models.SubComponent
	if err := cursor.All(ctx, &subs); err != nil {
		return nil, err
	}
	if subs == nil {
		subs = []models.SubComponent{}
	}

	return subs, nil
}

func (r *MongoStatusRepository) ListAllSubComponents(ctx context.Context) ([]models.SubComponent, error) {
	cursor, err := r.db.Collection("subcomponents").Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var subs []models.SubComponent
	if err := cursor.All(ctx, &subs); err != nil {
		return nil, err
	}
	if subs == nil {
		subs = []models.SubComponent{}
	}

	return subs, nil
}

func (r *MongoStatusRepository) ListSubComponentsByIDs(ctx context.Context, ids []primitive.ObjectID) ([]models.SubComponent, error) {
	if len(ids) == 0 {
		return []models.SubComponent{}, nil
	}

	cursor, err := r.db.Collection("subcomponents").Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var subs []models.SubComponent
	if err := cursor.All(ctx, &subs); err != nil {
		return nil, err
	}
	if subs == nil {
		subs = []models.SubComponent{}
	}

	return subs, nil
}

func (r *MongoStatusRepository) ListMonitorsByTargets(ctx context.Context, componentIDs []primitive.ObjectID, subComponentIDs []primitive.ObjectID) ([]models.Monitor, error) {
	filters := make([]bson.M, 0, 2)
	if len(componentIDs) > 0 {
		filters = append(filters, bson.M{"componentId": bson.M{"$in": componentIDs}})
	}
	if len(subComponentIDs) > 0 {
		filters = append(filters, bson.M{"subComponentId": bson.M{"$in": subComponentIDs}})
	}

	if len(filters) == 0 {
		return []models.Monitor{}, nil
	}

	cursor, err := r.db.Collection("monitors").Find(ctx, bson.M{"$or": filters})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var monitors []models.Monitor
	if err := cursor.All(ctx, &monitors); err != nil {
		return nil, err
	}
	if monitors == nil {
		monitors = []models.Monitor{}
	}

	return monitors, nil
}

func (r *MongoStatusRepository) ListDailyUptimeSinceByMonitorIDs(ctx context.Context, monitorIDs []primitive.ObjectID, since time.Time) ([]models.DailyUptime, error) {
	if len(monitorIDs) == 0 {
		return []models.DailyUptime{}, nil
	}

	cursor, err := r.db.Collection("daily_uptime").Find(
		ctx,
		bson.M{
			"monitorId": bson.M{"$in": monitorIDs},
			"date":      bson.M{"$gte": since},
		},
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var records []models.DailyUptime
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	if records == nil {
		records = []models.DailyUptime{}
	}

	return records, nil
}

func (r *MongoStatusRepository) ListActiveIncidents(ctx context.Context) ([]models.Incident, error) {
	cursor, err := r.db.Collection("incidents").Find(
		ctx,
		bson.M{"status": bson.M{"$ne": models.IncidentResolved}},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var incidents []models.Incident
	if err := cursor.All(ctx, &incidents); err != nil {
		return nil, err
	}
	if incidents == nil {
		incidents = []models.Incident{}
	}

	return incidents, nil
}

func (r *MongoStatusRepository) ListActiveMaintenanceAt(ctx context.Context, at time.Time) ([]models.Maintenance, error) {
	cursor, err := r.db.Collection("maintenance").Find(
		ctx,
		bson.M{
			"status":    models.MaintenanceInProgress,
			"startTime": bson.M{"$lte": at},
			"endTime":   bson.M{"$gte": at},
		},
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var maintenance []models.Maintenance
	if err := cursor.All(ctx, &maintenance); err != nil {
		return nil, err
	}
	if maintenance == nil {
		maintenance = []models.Maintenance{}
	}

	return maintenance, nil
}

func (r *MongoStatusRepository) CountActiveIncidents(ctx context.Context) (int64, error) {
	return r.db.Collection("incidents").CountDocuments(ctx, bson.M{"status": bson.M{"$ne": models.IncidentResolved}})
}

func (r *MongoStatusRepository) CountActiveMaintenanceAt(ctx context.Context, at time.Time) (int64, error) {
	return r.db.Collection("maintenance").CountDocuments(
		ctx,
		bson.M{
			"status":    models.MaintenanceInProgress,
			"startTime": bson.M{"$lte": at},
			"endTime":   bson.M{"$gte": at},
		},
	)
}

func (r *MongoStatusRepository) FindLatestIncidentByComponent(ctx context.Context, componentID primitive.ObjectID) (*models.Incident, error) {
	var incident models.Incident
	err := r.db.Collection("incidents").FindOne(
		ctx,
		bson.M{
			"$or": []bson.M{
				{"affectedComponents": bson.M{"$in": []primitive.ObjectID{componentID}}},
				{"affectedComponentTargets.componentId": componentID},
			},
		},
		options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}}),
	).Decode(&incident)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &incident, nil
}

func (r *MongoStatusRepository) ListIncidentsByCreatedAtRange(ctx context.Context, start, end time.Time) ([]models.Incident, error) {
	filter := bson.M{"createdAt": bson.M{"$gte": start, "$lt": end}}
	cursor, err := r.db.Collection("incidents").Find(
		ctx,
		filter,
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var incidents []models.Incident
	if err := cursor.All(ctx, &incidents); err != nil {
		return nil, err
	}
	if incidents == nil {
		incidents = []models.Incident{}
	}

	return incidents, nil
}

func (r *MongoStatusRepository) ListResolvedIncidentsSince(ctx context.Context, since time.Time) ([]models.Incident, error) {
	cursor, err := r.db.Collection("incidents").Find(
		ctx,
		bson.M{
			"status":     models.IncidentResolved,
			"resolvedAt": bson.M{"$gte": since},
		},
		options.Find().SetSort(bson.D{{Key: "resolvedAt", Value: -1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var incidents []models.Incident
	if err := cursor.All(ctx, &incidents); err != nil {
		return nil, err
	}
	if incidents == nil {
		incidents = []models.Incident{}
	}

	return incidents, nil
}

func (r *MongoStatusRepository) ListIncidentsByAffectedComponents(ctx context.Context, affectedIDs []primitive.ObjectID, limit int64) ([]models.Incident, error) {
	if len(affectedIDs) == 0 {
		return []models.Incident{}, nil
	}

	findOptions := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	if limit > 0 {
		findOptions.SetLimit(limit)
	}

	cursor, err := r.db.Collection("incidents").Find(
		ctx,
		bson.M{
			"$or": []bson.M{
				{"affectedComponents": bson.M{"$in": affectedIDs}},
				{"affectedComponentTargets.componentId": bson.M{"$in": affectedIDs}},
				{"affectedComponentTargets.subComponentIds": bson.M{"$in": affectedIDs}},
			},
		},
		findOptions,
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var incidents []models.Incident
	if err := cursor.All(ctx, &incidents); err != nil {
		return nil, err
	}
	if incidents == nil {
		incidents = []models.Incident{}
	}

	return incidents, nil
}

func (r *MongoStatusRepository) ListIncidentUpdatesByIncidentIDs(ctx context.Context, incidentIDs []primitive.ObjectID) (map[primitive.ObjectID][]models.IncidentUpdate, error) {
	updatesByIncident := map[primitive.ObjectID][]models.IncidentUpdate{}
	if len(incidentIDs) == 0 {
		return updatesByIncident, nil
	}

	cursor, err := r.db.Collection("incident_updates").Find(
		ctx,
		bson.M{"incidentId": bson.M{"$in": incidentIDs}},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var updates []models.IncidentUpdate
	if err := cursor.All(ctx, &updates); err != nil {
		return nil, err
	}

	for _, update := range updates {
		updatesByIncident[update.IncidentID] = append(updatesByIncident[update.IncidentID], update)
	}

	return updatesByIncident, nil
}

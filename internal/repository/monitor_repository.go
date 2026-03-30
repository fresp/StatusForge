package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/fresp/StatusForge/internal/models"
)

type MonitorRepository interface {
	Insert(ctx context.Context, monitor models.Monitor) error
	Update(ctx context.Context, id primitive.ObjectID, monitor models.Monitor) (bool, error)
	Delete(ctx context.Context, id primitive.ObjectID) (bool, error)
	List(ctx context.Context, page, limit int) ([]models.Monitor, int64, error)
	ListLogs(ctx context.Context, monitorID primitive.ObjectID, limit int64) ([]models.MonitorLog, error)
	ListUptime(ctx context.Context, monitorID primitive.ObjectID, since time.Time) ([]models.DailyUptime, error)
	ListOutages(ctx context.Context) ([]models.Outage, error)
	ListHistory(ctx context.Context, monitorID primitive.ObjectID, limit int64) ([]models.EnhancedMonitorLog, error)
}

type MongoMonitorRepository struct {
	collection *mongo.Collection
}

func NewMongoMonitorRepository(db *mongo.Database) *MongoMonitorRepository {
	return &MongoMonitorRepository{collection: db.Collection("monitors")}
}

func (r *MongoMonitorRepository) Insert(ctx context.Context, monitor models.Monitor) error {
	_, err := r.collection.InsertOne(ctx, monitor)
	return err
}

func (r *MongoMonitorRepository) Update(ctx context.Context, id primitive.ObjectID, monitor models.Monitor) (bool, error) {
	update := bson.M{
		"$set": bson.M{
			"name":            monitor.Name,
			"type":            monitor.Type,
			"target":          monitor.Target,
			"monitoring":      monitor.Monitoring,
			"sslThresholds":   monitor.SSLThresholds,
			"intervalSeconds": monitor.IntervalSeconds,
			"timeoutSeconds":  monitor.TimeoutSeconds,
			"componentId":     monitor.ComponentID,
			"subComponentId":  monitor.SubComponentID,
			"updatedAt":       time.Now(),
		},
	}

	res, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return false, err
	}

	return res.MatchedCount > 0, nil
}

func (r *MongoMonitorRepository) Delete(ctx context.Context, id primitive.ObjectID) (bool, error) {
	res, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return false, err
	}

	return res.DeletedCount > 0, nil
}

func (r *MongoMonitorRepository) List(ctx context.Context, page, limit int) ([]models.Monitor, int64, error) {
	filter := bson.M{}
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := int64((page - 1) * limit)
	findOptions := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(skip).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var monitors []models.Monitor
	if err := cursor.All(ctx, &monitors); err != nil {
		return nil, 0, err
	}
	if monitors == nil {
		monitors = []models.Monitor{}
	}

	return monitors, total, nil
}

func (r *MongoMonitorRepository) ListLogs(ctx context.Context, monitorID primitive.ObjectID, limit int64) ([]models.MonitorLog, error) {
	cursor, err := r.collection.Database().Collection("monitor_logs").Find(
		ctx,
		bson.M{"monitorId": monitorID},
		options.Find().SetSort(bson.D{{Key: "checkedAt", Value: -1}}).SetLimit(limit),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var logs []models.MonitorLog
	if err := cursor.All(ctx, &logs); err != nil {
		return nil, err
	}
	if logs == nil {
		logs = []models.MonitorLog{}
	}

	return logs, nil
}

func (r *MongoMonitorRepository) ListUptime(ctx context.Context, monitorID primitive.ObjectID, since time.Time) ([]models.DailyUptime, error) {
	cursor, err := r.collection.Database().Collection("daily_uptime").Find(
		ctx,
		bson.M{"monitorId": monitorID, "date": bson.M{"$gte": since}},
		options.Find().SetSort(bson.D{{Key: "date", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var uptime []models.DailyUptime
	if err := cursor.All(ctx, &uptime); err != nil {
		return nil, err
	}
	if uptime == nil {
		uptime = []models.DailyUptime{}
	}

	return uptime, nil
}

func (r *MongoMonitorRepository) ListOutages(ctx context.Context) ([]models.Outage, error) {
	cursor, err := r.collection.Database().Collection("outages").Find(
		ctx,
		bson.M{},
		options.Find().SetSort(bson.D{{Key: "startedAt", Value: -1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var outages []models.Outage
	if err := cursor.All(ctx, &outages); err != nil {
		return nil, err
	}
	if outages == nil {
		outages = []models.Outage{}
	}

	return outages, nil
}

func (r *MongoMonitorRepository) ListHistory(ctx context.Context, monitorID primitive.ObjectID, limit int64) ([]models.EnhancedMonitorLog, error) {
	cursor, err := r.collection.Database().Collection("enhanced_monitor_logs").Find(
		ctx,
		bson.M{"monitorId": monitorID},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(limit),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var logs []models.EnhancedMonitorLog
	if err := cursor.All(ctx, &logs); err != nil {
		return nil, err
	}
	if logs == nil {
		logs = []models.EnhancedMonitorLog{}
	}

	return logs, nil
}

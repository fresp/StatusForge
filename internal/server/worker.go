// cmd/server/worker.go
// Package server provides monitoring worker functionality.
// Wave 2: Contains extracted worker code from apps/worker/main.go.
package server

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/fresp/StatusForge/internal/models"
	"github.com/fresp/StatusForge/internal/utils"
)

// workerCtx and workerCancel are used to signal shutdown to all worker goroutines
var workerCtx context.Context
var workerCancel context.CancelFunc

var workerWG sync.WaitGroup

// StartWorker starts the monitoring worker
func StartWorker(ctx context.Context, db *mongo.Database, rdb *redis.Client) {
	workerCtx, workerCancel = context.WithCancel(ctx)

	log.Println("[WORKER] Monitoring worker started")

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-workerCtx.Done():
			log.Println("[WORKER] Worker shutdown requested, waiting for running checks...")
			ticker.Stop()
			workerWG.Wait()
			log.Println("[WORKER] Worker shutdown complete")
			return
		case <-ticker.C:
			workerWG.Add(1)
			go func() {
				defer workerWG.Done()
				runChecks(db)
				updateMaintenanceStatus(db)
			}()
		}
	}
}

// StopWorker gracefully stops the monitoring worker
func StopWorker() error {
	if workerCancel == nil {
		return nil
	}
	workerCancel()
	return nil
}

func runChecks(db *mongo.Database) {
	ctx, cancel := context.WithTimeout(workerCtx, 30*time.Second)
	defer cancel()

	cursor, err := db.Collection("monitors").Find(ctx, bson.M{})
	if err != nil {
		log.Println("[WORKER] Error fetching monitors:", err)
		return
	}
	defer cursor.Close(ctx)

	var monitors []models.Monitor
	if err := cursor.All(ctx, &monitors); err != nil {
		return
	}

	for _, m := range monitors {
		workerWG.Add(1)
		go func(mon models.Monitor) {
			defer workerWG.Done()
			checkMonitor(db, mon)
		}(m)
	}
}

func checkMonitor(db *mongo.Database, mon models.Monitor) {
	start := time.Now()
	status := models.MonitorUp
	statusCode := 0
	sslWarning := false
	sslDaysRemaining := 0
	sslTriggeredThreshold := 0
	domainWarning := false
	domainDaysRemaining := 0
	domainTriggeredThreshold := 0

	timeout := time.Duration(mon.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	sslThresholds := mon.SSLThresholds
	if len(sslThresholds) == 0 {
		sslThresholds = []int{30, 14, 7}
	}
	advanced := mon.Monitoring.Advanced

	switch mon.Type {
	case models.MonitorHTTP:
		code, err := utils.CheckHTTP(mon.Target, timeout, advanced.IgnoreTLSError)
		statusCode = code
		if err != nil || code >= 500 || code == 0 {
			status = models.MonitorDown
		}

		if advanced.CertExpiry {
			result, sslErr := utils.CheckHTTPSSLCertificate(mon.Target, timeout, sslThresholds)
			if sslErr != nil {
				status = models.MonitorDown
			} else {
				sslWarning = result.Warning
				sslDaysRemaining = result.DaysRemaining
				sslTriggeredThreshold = result.TriggeredThreshold
			}
		}

		if advanced.DomainExpiry {
			result, domainErr := utils.CheckDomain(mon.Target, string(mon.Type), sslThresholds)
			if domainErr != nil {
				status = models.MonitorDown
			} else {
				domainWarning = result.Warning
				domainDaysRemaining = result.DaysRemaining
				domainTriggeredThreshold = result.TriggeredThreshold
			}
		}
	case models.MonitorTCP:
		if err := utils.CheckTCP(mon.Target, timeout); err != nil {
			status = models.MonitorDown
		}
	case models.MonitorDNS:
		if err := utils.CheckDNS(mon.Target, timeout); err != nil {
			status = models.MonitorDown
		}
	case models.MonitorPing:
		if err := utils.CheckPing(mon.Target, timeout); err != nil {
			status = models.MonitorDown
		}
	case models.MonitorSSL:
		result, err := utils.CheckSSL(mon.Target, timeout, sslThresholds)
		if err != nil {
			status = models.MonitorDown
		} else {
			sslWarning = result.Warning
			sslDaysRemaining = result.DaysRemaining
			sslTriggeredThreshold = result.TriggeredThreshold
		}

		if advanced.DomainExpiry {
			result, domainErr := utils.CheckDomain(mon.Target, string(mon.Type), sslThresholds)
			if domainErr != nil {
				status = models.MonitorDown
			} else {
				domainWarning = result.Warning
				domainDaysRemaining = result.DaysRemaining
				domainTriggeredThreshold = result.TriggeredThreshold
			}
		}
	}

	responseTime := time.Since(start).Milliseconds()

	logEntry := models.EnhancedMonitorLog{
		ID:                       primitive.NewObjectID(),
		MonitorID:                mon.ID,
		Status:                   status,
		SSLWarning:               sslWarning,
		SSLDaysRemaining:         sslDaysRemaining,
		SSLTriggeredThreshold:    sslTriggeredThreshold,
		DomainWarning:            domainWarning,
		DomainDaysRemaining:      domainDaysRemaining,
		DomainTriggeredThreshold: domainTriggeredThreshold,
		ResponseTime:             responseTime,
		StatusCode:               statusCode,
		CheckedAt:                time.Now(),
	}

	ctx2, cancel2 := context.WithTimeout(workerCtx, 5*time.Second)
	defer cancel2()

	db.Collection("monitor_logs").InsertOne(ctx2, logEntry)

	db.Collection("monitors").UpdateOne(ctx2,
		bson.M{"_id": mon.ID},
		bson.M{"$set": bson.M{
			"lastStatus":               status,
			"sslWarning":               sslWarning,
			"sslDaysRemaining":         sslDaysRemaining,
			"sslTriggeredThreshold":    sslTriggeredThreshold,
			"domainWarning":            domainWarning,
			"domainDaysRemaining":      domainDaysRemaining,
			"domainTriggeredThreshold": domainTriggeredThreshold,
			"lastCheckedAt":            time.Now(),
		}},
	)

	combinedWarning := sslWarning || domainWarning
	if mon.Type == models.MonitorSSL || advanced.DomainExpiry {
		applySSLWarningStatus(ctx2, db, mon, combinedWarning)
	}

	updateDailyUptime(db, mon.ID, status)
	detectOutage(db, mon, status)
}

func applySSLWarningStatus(ctx context.Context, db *mongo.Database, mon models.Monitor, warning bool) {
	if !mon.ComponentID.IsZero() {
		if warning {
			db.Collection("components").UpdateOne(ctx,
				bson.M{"_id": mon.ComponentID},
				bson.M{"$set": bson.M{"status": models.StatusDegradedPerf, "updatedAt": time.Now()}},
			)
			return
		}

		hasActiveOutage := hasActiveOutageForTarget(ctx, db, mon)
		hasActiveIncident := hasActiveIncidentForTarget(ctx, db, mon)
		if !shouldRestoreOperational(hasActiveOutage, hasActiveIncident) {
			return
		}

		db.Collection("components").UpdateOne(ctx,
			bson.M{"_id": mon.ComponentID, "status": models.StatusDegradedPerf},
			bson.M{"$set": bson.M{"status": models.StatusOperational, "updatedAt": time.Now()}},
		)
		return
	}

	if !mon.SubComponentID.IsZero() {
		if warning {
			db.Collection("subcomponents").UpdateOne(ctx,
				bson.M{"_id": mon.SubComponentID},
				bson.M{"$set": bson.M{"status": models.StatusDegradedPerf, "updated_at": time.Now()}},
			)
			return
		}

		hasActiveOutage := hasActiveOutageForTarget(ctx, db, mon)
		hasActiveIncident := hasActiveIncidentForTarget(ctx, db, mon)
		if !shouldRestoreOperational(hasActiveOutage, hasActiveIncident) {
			return
		}

		db.Collection("subcomponents").UpdateOne(ctx,
			bson.M{"_id": mon.SubComponentID, "status": models.StatusDegradedPerf},
			bson.M{"$set": bson.M{"status": models.StatusOperational, "updated_at": time.Now()}},
		)
	}
}

func shouldRestoreOperational(hasActiveOutage bool, hasActiveIncident bool) bool {
	return !hasActiveOutage && !hasActiveIncident
}

func hasActiveOutageForTarget(ctx context.Context, db *mongo.Database, mon models.Monitor) bool {
	orConditions := make([]bson.M, 0, 2)
	if !mon.ComponentID.IsZero() {
		orConditions = append(orConditions, bson.M{"componentId": mon.ComponentID})
	}
	if !mon.SubComponentID.IsZero() {
		orConditions = append(orConditions, bson.M{"subComponentId": mon.SubComponentID})
	}

	if len(orConditions) == 0 {
		return false
	}

	filter := bson.M{
		"status": models.OutageActive,
		"$or":    orConditions,
	}

	err := db.Collection("outages").FindOne(ctx, filter).Err()
	return err == nil
}

func hasActiveIncidentForTarget(ctx context.Context, db *mongo.Database, mon models.Monitor) bool {
	componentID := mon.ComponentID
	if componentID.IsZero() && !mon.SubComponentID.IsZero() {
		var subComp models.SubComponent
		err := db.Collection("subcomponents").FindOne(ctx, bson.M{"_id": mon.SubComponentID}).Decode(&subComp)
		if err != nil {
			return false
		}
		componentID = subComp.ComponentID
	}

	if componentID.IsZero() {
		return false
	}

	filter := bson.M{
		"affectedComponents": bson.M{"$in": []primitive.ObjectID{componentID}},
		"status":             bson.M{"$ne": models.IncidentResolved},
	}

	err := db.Collection("incidents").FindOne(ctx, filter).Err()
	return err == nil
}

func updateDailyUptime(db *mongo.Database, monitorID primitive.ObjectID, status models.MonitorLogStatus) {
	ctx, cancel := context.WithTimeout(workerCtx, 5*time.Second)
	defer cancel()

	today := time.Now().UTC().Truncate(24 * time.Hour)

	var existing models.DailyUptime
	err := db.Collection("daily_uptime").FindOne(ctx, bson.M{
		"monitorId": monitorID,
		"date":      today,
	}).Decode(&existing)

	if err == mongo.ErrNoDocuments {
		successful := 0
		if status == models.MonitorUp {
			successful = 1
		}
		entry := models.DailyUptime{
			ID:               primitive.NewObjectID(),
			MonitorID:        monitorID,
			Date:             today,
			TotalChecks:      1,
			SuccessfulChecks: successful,
			UptimePercent:    float64(successful) * 100.0,
		}
		db.Collection("daily_uptime").InsertOne(ctx, entry)
		return
	}

	inc := bson.M{"totalChecks": 1}
	if status == models.MonitorUp {
		inc["successfulChecks"] = 1
	}

	newTotal := existing.TotalChecks + 1
	newSuccessful := existing.SuccessfulChecks
	if status == models.MonitorUp {
		newSuccessful++
	}
	pct := float64(newSuccessful) / float64(newTotal) * 100.0

	db.Collection("daily_uptime").UpdateOne(ctx,
		bson.M{"monitorId": monitorID, "date": today},
		bson.M{
			"$inc": inc,
			"$set": bson.M{"uptimePercent": pct},
		},
	)
}

func detectOutage(db *mongo.Database, mon models.Monitor, status models.MonitorLogStatus) {
	ctx, cancel := context.WithTimeout(workerCtx, 10*time.Second)
	defer cancel()

	if status == models.MonitorDown {
		cursor, err := db.Collection("monitor_logs").Find(ctx,
			bson.M{"monitorId": mon.ID},
			options.Find().SetSort(bson.D{{Key: "checkedAt", Value: -1}}).SetLimit(3),
		)
		if err != nil {
			return
		}
		var logs []models.MonitorLog
		cursor.All(ctx, &logs)
		cursor.Close(ctx)

		if len(logs) < 3 {
			return
		}
		for _, l := range logs {
			if l.Status != models.MonitorDown {
				return
			}
		}

		var existingOutage models.Outage
		err = db.Collection("outages").FindOne(ctx, bson.M{
			"monitorId": mon.ID,
			"status":    bson.M{"$eq": models.OutageActive},
		}).Decode(&existingOutage)
		if err == nil {
			return
		}

		outage := models.Outage{
			ID:             primitive.NewObjectID(),
			StartedAt:      time.Now(),
			Status:         models.OutageActive,
			MonitorID:      mon.ID,
			ComponentID:    mon.ComponentID,
			SubComponentID: mon.SubComponentID,
		}
		db.Collection("outages").InsertOne(ctx, outage)
		log.Println("[WORKER] Automatic outage detected for monitor:", mon.Name)

		componentsToAffect := make([]primitive.ObjectID, 0)
		if !mon.ComponentID.IsZero() {
			componentsToAffect = append(componentsToAffect, mon.ComponentID)
			db.Collection("components").UpdateOne(ctx,
				bson.M{"_id": mon.ComponentID},
				bson.M{"$set": bson.M{"status": models.StatusMajorOutage, "updatedAt": time.Now()}},
			)
		} else if !mon.SubComponentID.IsZero() {
			var subComp models.SubComponent
			if err = db.Collection("subcomponents").FindOne(ctx, bson.M{"_id": mon.SubComponentID}).Decode(&subComp); err == nil {
				componentsToAffect = append(componentsToAffect, subComp.ComponentID)
				db.Collection("subcomponents").UpdateOne(ctx,
					bson.M{"_id": mon.SubComponentID},
					bson.M{"$set": bson.M{"status": models.StatusMajorOutage, "updatedAt": time.Now()}},
				)
			}
		}

		if len(componentsToAffect) > 0 {
			var existingIncident models.Incident
			err = db.Collection("incidents").FindOne(ctx, bson.M{
				"affectedComponents": bson.M{"$in": componentsToAffect},
				"status":             bson.M{"$ne": models.IncidentResolved},
			}).Decode(&existingIncident)
			if err != nil {
				incident := models.Incident{
					ID:                 primitive.NewObjectID(),
					Title:              mon.Name + " - Outage Detected",
					Description:        "Automated incident: " + mon.Name + " has failed 3 consecutive checks.",
					Status:             models.IncidentInvestigating,
					Impact:             models.ImpactMajor,
					AffectedComponents: componentsToAffect,
					CreatedAt:          time.Now(),
					UpdatedAt:          time.Now(),
				}
				db.Collection("incidents").InsertOne(ctx, incident)
				log.Println("[WORKER] Auto-incident created for monitor:", mon.Name)
			}
		}

	} else {
		var existingOutage models.Outage
		err := db.Collection("outages").FindOne(ctx, bson.M{
			"monitorId": mon.ID,
			"status":    models.OutageActive,
		}).Decode(&existingOutage)
		if err != nil {
			return
		}

		endTime := time.Now()
		duration := endTime.Sub(existingOutage.StartedAt)
		durationSeconds := int(duration.Seconds())
		updateResult, err := db.Collection("outages").UpdateOne(ctx,
			bson.M{"_id": existingOutage.ID},
			bson.M{"$set": bson.M{
				"endedAt":         endTime,
				"durationSeconds": durationSeconds,
				"status":          models.OutageResolved,
			}},
		)
		if err != nil || updateResult.MatchedCount == 0 {
			log.Println("[WORKER] Error updating outage:", err)
			return
		}
		log.Println("[WORKER] Outage resolved for monitor", mon.Name, "Duration:", durationSeconds)

		var updatedIncident models.Incident
		err = db.Collection("incidents").FindOne(ctx, bson.M{
			"affectedComponents": bson.M{"$in": []primitive.ObjectID{existingOutage.ComponentID, existingOutage.SubComponentID}},
			"status":             bson.M{"$ne": models.IncidentResolved},
		}).Decode(&updatedIncident)

		if err == nil {
			now := time.Now()
			_, err = db.Collection("incidents").UpdateOne(ctx,
				bson.M{"_id": updatedIncident.ID},
				bson.M{"$set": bson.M{
					"status":     models.IncidentResolved,
					"resolvedAt": now,
					"updatedAt":  now,
				}},
			)

			if err != nil {
				log.Println("[WORKER] Error updating incident:", err)
			}

			if !existingOutage.ComponentID.IsZero() {
				db.Collection("components").UpdateOne(ctx,
					bson.M{"_id": existingOutage.ComponentID},
					bson.M{"$set": bson.M{"status": models.StatusOperational, "updatedAt": now}},
				)
			}
			if !existingOutage.SubComponentID.IsZero() {
				db.Collection("subcomponents").UpdateOne(ctx,
					bson.M{"_id": existingOutage.SubComponentID},
					bson.M{"$set": bson.M{"status": models.StatusOperational, "updatedAt": now}},
				)
			}
			log.Println("[WORKER] Auto-resolved incident for monitor:", mon.Name)
		}
	}
}

func updateMaintenanceStatus(db *mongo.Database) {
	ctx, cancel := context.WithTimeout(workerCtx, 10*time.Second)
	defer cancel()

	now := time.Now()

	db.Collection("maintenance").UpdateMany(ctx,
		bson.M{"status": models.MaintenanceScheduled, "startTime": bson.M{"$lte": now}},
		bson.M{"$set": bson.M{"status": models.MaintenanceInProgress}},
	)

	db.Collection("maintenance").UpdateMany(ctx,
		bson.M{"status": models.MaintenanceInProgress, "endTime": bson.M{"$lte": now}},
		bson.M{"$set": bson.M{"status": models.MaintenanceCompleted}},
	)
}

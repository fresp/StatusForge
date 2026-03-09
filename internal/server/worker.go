// cmd/server/worker.go
// Package server provides monitoring worker functionality.
// Wave 2: Contains extracted worker code from apps/worker/main.go.
package server

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"

	"status-platform/internal/models"
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

	timeout := time.Duration(mon.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	switch mon.Type {
	case models.MonitorHTTP:
		code, err := checkHTTP(mon.Target, timeout)
		statusCode = code
		if err != nil || code >= 500 || code == 0 {
			status = models.MonitorDown
		}
	case models.MonitorTCP:
		if err := checkTCP(mon.Target, timeout); err != nil {
			status = models.MonitorDown
		}
	case models.MonitorDNS:
		if err := checkDNS(mon.Target, timeout); err != nil {
			status = models.MonitorDown
		}
	case models.MonitorPing:
		if err := checkPing(mon.Target, timeout); err != nil {
			status = models.MonitorDown
		}
	}

	responseTime := time.Since(start).Milliseconds()

	logEntry := models.EnhancedMonitorLog{
		ID:           primitive.NewObjectID(),
		MonitorID:    mon.ID,
		Status:       status,
		ResponseTime: responseTime,
		StatusCode:   statusCode,
		CheckedAt:    time.Now(),
	}

	ctx2, cancel2 := context.WithTimeout(workerCtx, 5*time.Second)
	defer cancel2()

	db.Collection("monitor_logs").InsertOne(ctx2, logEntry)
	updateDailyUptime(db, mon.ID, status)
	detectOutage(db, mon, status)
}

func checkHTTP(target string, timeout time.Duration) (int, error) {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(target)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

func checkTCP(target string, timeout time.Duration) error {
	conn, err := net.DialTimeout("tcp", target, timeout)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

func checkDNS(target string, timeout time.Duration) error {
	resolver := &net.Resolver{}
	ctx, cancel := context.WithTimeout(workerCtx, timeout)
	defer cancel()
	_, err := resolver.LookupHost(ctx, target)
	return err
}

func checkPing(target string, timeout time.Duration) error {
	conn, err := icmp.ListenPacket("udp4", "")
	if err != nil {
		c, err2 := net.DialTimeout("tcp", target+":80", timeout)
		if err2 != nil {
			return err2
		}
		c.Close()
		return nil
	}
	defer conn.Close()

	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{ID: os.Getpid() & 0xffff, Seq: 1, Data: []byte("ping")},
	}
	b, _ := msg.Marshal(nil)
	conn.SetDeadline(time.Now().Add(timeout))

	dst, err := net.ResolveIPAddr("ip4", target)
	if err != nil {
		return err
	}
	if _, err := conn.WriteTo(b, dst); err != nil {
		return err
	}

	reply := make([]byte, 1500)
	if _, _, err := conn.ReadFrom(reply); err != nil {
		return err
	}
	return nil
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

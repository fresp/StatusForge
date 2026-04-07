//go:build ignore

package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/fresp/Statora/internal/models"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	godotenv.Load()

	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://root:strongpassword@host.docker.internal:27017/admin?authSource=admin"
	}
	dbName := os.Getenv("MONGODB_DB")
	if dbName == "" {
		dbName = "statusplatform"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("MongoDB connect failed: %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database(dbName)
	log.Printf("Connected to MongoDB: %s / %s", mongoURI, dbName)

	// --- Components ---
	hostedPagesID := primitive.NewObjectID()
	publicAPIID := primitive.NewObjectID()

	components := []interface{}{
		models.Component{
			ID:          hostedPagesID,
			Name:        "Hosted Pages",
			Description: "Customer-facing hosted status and landing pages",
			Status:      "operational",
			Order:       1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		models.Component{
			ID:          publicAPIID,
			Name:        "Public API",
			Description: "REST API endpoints for programmatic access",
			Status:      "operational",
			Order:       2,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	if _, err := db.Collection("components").InsertMany(ctx, components); err != nil {
		log.Printf("Warning seeding components (may already exist): %v", err)
	} else {
		log.Println("✓ Components seeded")
	}

	// --- SubComponents ---
	subComponents := []interface{}{
		models.SubComponent{
			ID:          primitive.NewObjectID(),
			ComponentID: hostedPagesID,
			Name:        "HTTP Pages",
			Description: "HTTP hosted page delivery",
			Status:      "operational",
			Order:       1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		models.SubComponent{
			ID:          primitive.NewObjectID(),
			ComponentID: hostedPagesID,
			Name:        "HTTPS Pages",
			Description: "HTTPS hosted page delivery",
			Status:      "operational",
			Order:       2,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		models.SubComponent{
			ID:          primitive.NewObjectID(),
			ComponentID: publicAPIID,
			Name:        "Auth API",
			Description: "Authentication and token endpoints",
			Status:      "operational",
			Order:       1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		models.SubComponent{
			ID:          primitive.NewObjectID(),
			ComponentID: publicAPIID,
			Name:        "Messaging API",
			Description: "Messaging and notification delivery endpoints",
			Status:      "operational",
			Order:       2,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	if _, err := db.Collection("subcomponents").InsertMany(ctx, subComponents); err != nil {
		log.Printf("Warning seeding subcomponents: %v", err)
	} else {
		log.Println("✓ SubComponents seeded")
	}

	// --- Sample resolved incident ---
	incidentID := primitive.NewObjectID()
	resolvedAt := time.Now().Add(-2 * time.Hour)
	incident := models.Incident{
		ID:                 incidentID,
		Title:              "Elevated API error rates",
		Description:        "We observed elevated error rates on the Public API affecting a subset of requests.",
		Status:             "resolved",
		Impact:             "minor",
		AffectedComponents: []primitive.ObjectID{publicAPIID},
		CreatedAt:          time.Now().Add(-4 * time.Hour),
		UpdatedAt:          resolvedAt,
		ResolvedAt:         &resolvedAt,
	}

	if _, err := db.Collection("incidents").InsertOne(ctx, incident); err != nil {
		log.Printf("Warning seeding incident: %v", err)
	} else {
		log.Println("✓ Sample incident seeded")
	}

	// --- Incident updates ---
	incidentUpdates := []interface{}{
		models.IncidentUpdate{
			ID:         primitive.NewObjectID(),
			IncidentID: incidentID,
			Message:    "We are investigating reports of elevated error rates on the Public API.",
			Status:     "investigating",
			CreatedAt:  time.Now().Add(-4 * time.Hour),
		},
		models.IncidentUpdate{
			ID:         primitive.NewObjectID(),
			IncidentID: incidentID,
			Message:    "The issue has been identified as a misconfigured rate limiter deployed in the last release.",
			Status:     "identified",
			CreatedAt:  time.Now().Add(-3 * time.Hour),
		},
		models.IncidentUpdate{
			ID:         primitive.NewObjectID(),
			IncidentID: incidentID,
			Message:    "A fix has been deployed. We are monitoring the situation to confirm full recovery.",
			Status:     "monitoring",
			CreatedAt:  time.Now().Add(-150 * time.Minute),
		},
		models.IncidentUpdate{
			ID:         primitive.NewObjectID(),
			IncidentID: incidentID,
			Message:    "All systems are operating normally. The incident is resolved.",
			Status:     "resolved",
			CreatedAt:  resolvedAt,
		},
	}

	if _, err := db.Collection("incident_updates").InsertMany(ctx, incidentUpdates); err != nil {
		log.Printf("Warning seeding incident updates: %v", err)
	} else {
		log.Println("✓ Incident updates seeded")
	}

	log.Println("Seed complete.")
}

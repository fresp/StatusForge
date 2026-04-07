package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/fresp/Statora/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// WebhookEvent represents the payload structure for webhook deliveries.
type WebhookEvent struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// loadEnabledWebhookChannels retrieves all enabled webhook channels from the database.
func loadEnabledWebhookChannels(db *mongo.Database) ([]models.WebhookChannel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := db.Collection("webhook_channels").Find(ctx, bson.M{"enabled": true})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var channels []models.WebhookChannel
	if err := cursor.All(ctx, &channels); err != nil {
		return nil, err
	}
	if channels == nil {
		channels = []models.WebhookChannel{}
	}
	return channels, nil
}

// DispatchWebhookEvent sends a webhook event to all enabled webhook channels.
// This function is best-effort and never returns an error. Failures are logged.
func DispatchWebhookEvent(db *mongo.Database, eventType string, data interface{}) {
	channels, err := loadEnabledWebhookChannels(db)
	if err != nil {
		log.Printf("[Webhook] Failed to load enabled channels: %v", err)
		return
	}

	if len(channels) == 0 {
		return // No enabled channels to dispatch to
	}

	eventPayload := WebhookEvent{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}

	jsonPayload, err := json.Marshal(eventPayload)
	if err != nil {
		log.Printf("[Webhook] Failed to marshal event payload for type %s: %v", eventType, err)
		return
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	for _, channel := range channels {
		req, err := http.NewRequest(http.MethodPost, channel.URL, bytes.NewBuffer(jsonPayload))
		if err != nil {
			log.Printf("[Webhook] Failed to create request for channel %s (%s): %v", channel.Name, channel.URL, err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[Webhook] Failed to dispatch event to channel %s (%s): %v", channel.Name, channel.URL, err)
			continue
		}

		if _, drainErr := io.Copy(io.Discard, resp.Body); drainErr != nil {
			log.Printf("[Webhook] Failed to drain response body for channel %s (%s): %v", channel.Name, channel.URL, drainErr)
		}
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("[Webhook] Failed to close response body for channel %s (%s): %v", channel.Name, channel.URL, closeErr)
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			log.Printf("[Webhook] Received non-2xx response for channel %s (%s): %s", channel.Name, channel.URL, resp.Status)
		}
	}
}

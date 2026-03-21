package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type WebhookChannel struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name      string             `bson:"name" json:"name"`
	URL       string             `bson:"url" json:"url"`
	Enabled   bool               `bson:"enabled" json:"enabled"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
}

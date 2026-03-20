package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserInvitation struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TokenHash  string             `bson:"tokenHash" json:"-"`
	Email      string             `bson:"email" json:"email"`
	Role       string             `bson:"role" json:"role"`
	ExpiresAt  time.Time          `bson:"expiresAt" json:"expiresAt"`
	CreatedBy  primitive.ObjectID `bson:"createdBy" json:"createdBy"`
	AcceptedAt *time.Time         `bson:"acceptedAt,omitempty" json:"acceptedAt,omitempty"`
	RevokedAt  *time.Time         `bson:"revokedAt,omitempty" json:"revokedAt,omitempty"`
	CreatedAt  time.Time          `bson:"createdAt" json:"createdAt"`
}

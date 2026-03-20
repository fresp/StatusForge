package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID                   primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	Username             string              `bson:"username" json:"username"`
	Email                string              `bson:"email" json:"email"`
	Role                 string              `bson:"role,omitempty" json:"role,omitempty"`
	Status               string              `bson:"status,omitempty" json:"status,omitempty"`
	MFAEnabled           bool                `bson:"mfaEnabled,omitempty" json:"mfaEnabled,omitempty"`
	MFASecretEnc         string              `bson:"mfaSecretEnc,omitempty" json:"-"`
	MFARecoveryCodesHash []string            `bson:"mfaRecoveryCodesHash,omitempty" json:"-"`
	LastLoginAt          *time.Time          `bson:"lastLoginAt,omitempty" json:"lastLoginAt,omitempty"`
	InvitedBy            *primitive.ObjectID `bson:"invitedBy,omitempty" json:"invitedBy,omitempty"`
	PasswordHash         string              `bson:"passwordHash" json:"-"`
	CreatedAt            time.Time           `bson:"createdAt" json:"createdAt"`
}

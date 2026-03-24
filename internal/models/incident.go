package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type IncidentWithUpdates struct {
	Incident
	Updates 		   []IncidentUpdate `json:"updates"`
	AffectedComponents []Component `json:"affectedComponents"`
}

type IncidentStatus string

const (
	IncidentInvestigating IncidentStatus = "investigating"
	IncidentIdentified    IncidentStatus = "identified"
	IncidentMonitoring    IncidentStatus = "monitoring"
	IncidentResolved      IncidentStatus = "resolved"
)

type IncidentImpact string

const (
	ImpactNone     IncidentImpact = "none"
	ImpactMinor    IncidentImpact = "minor"
	ImpactMajor    IncidentImpact = "major"
	ImpactCritical IncidentImpact = "critical"
)

type Incident struct {
	ID                 primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Title              string               `bson:"title" json:"title"`
	Description        string               `bson:"description" json:"description"`
	Status             IncidentStatus       `bson:"status" json:"status"`
	Impact             IncidentImpact       `bson:"impact" json:"impact"`
	CreatorID          *primitive.ObjectID  `bson:"creatorId,omitempty" json:"creatorId,omitempty"`
	CreatorUsername    string               `bson:"creatorUsername,omitempty" json:"creatorUsername,omitempty"`
	AffectedComponents []primitive.ObjectID `bson:"affectedComponents" json:"affectedComponents"`
	CreatedAt          time.Time            `bson:"createdAt" json:"createdAt"`
	UpdatedAt          time.Time            `bson:"updatedAt" json:"updatedAt"`
	ResolvedAt         *time.Time           `bson:"resolvedAt,omitempty" json:"resolvedAt,omitempty"`
}

type IncidentUpdate struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	IncidentID primitive.ObjectID `bson:"incidentId" json:"incidentId"`
	Message    string             `bson:"message" json:"message"`
	Status     IncidentStatus     `bson:"status" json:"status"`
	CreatedAt  time.Time          `bson:"createdAt" json:"createdAt"`
}

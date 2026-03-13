package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MonitorType string

const (
	MonitorHTTP MonitorType = "http"
	MonitorTCP  MonitorType = "tcp"
	MonitorDNS  MonitorType = "dns"
	MonitorPing MonitorType = "ping"
)

type MonitorLogStatus string

const (
	MonitorUp   MonitorLogStatus = "up"
	MonitorDown MonitorLogStatus = "down"
)

type Monitor struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name            string             `bson:"name" json:"name"`
	Type            MonitorType        `bson:"type" json:"type"`
	Target          string             `bson:"target" json:"target"`
	IntervalSeconds int                `bson:"intervalSeconds" json:"intervalSeconds"`
	TimeoutSeconds  int                `bson:"timeoutSeconds" json:"timeoutSeconds"`
	ComponentID     primitive.ObjectID `bson:"componentId,omitempty" json:"componentId,omitempty"`
	SubComponentID  primitive.ObjectID `bson:"subComponentId,omitempty" json:"subComponentId,omitempty"`
	LastStatus      MonitorLogStatus   `bson:"lastStatus,omitempty" json:"lastStatus,omitempty"`
	LastCheckedAt   time.Time          `bson:"lastCheckedAt,omitempty" json:"lastCheckedAt,omitempty"`
	CreatedAt       time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time          `bson:"updatedAt,omitempty" json:"updatedAt,omitempty"`
}


type EnhancedMonitorLog struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	MonitorID     primitive.ObjectID `bson:"monitorId" json:"monitorId"`
	Status        MonitorLogStatus   `bson:"status" json:"status"`
	ResponseTime  int64              `bson:"responseTime" json:"responseTime"`
	StatusCode    int                `bson:"statusCode" json:"statusCode"`
	CheckedAt     time.Time          `bson:"checkedAt" json:"checkedAt"`
	StartedAt     time.Time          `bson:"startedAt,omitempty" json:"startedAt,omitempty"`
	EndedAt       time.Time          `bson:"endedAt,omitempty" json:"endedAt,omitempty"`
	DurationSeconds int              `bson:"durationSeconds,omitempty" json:"durationSeconds,omitempty"`
}


type OutageStatus string

const (
	OutageActive OutageStatus = "active"
	OutageResolved OutageStatus = "resolved"
)

type Outage struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	MonitorID       primitive.ObjectID `bson:"monitorId" json:"monitorId"`
	ComponentID     primitive.ObjectID `bson:"componentId,omitempty" json:"componentId,omitempty"`
	SubComponentID  primitive.ObjectID `bson:"subComponentId,omitempty" json:"subComponentId,omitempty"`
	StartedAt       time.Time          `bson:"startedAt" json:"startedAt"`
	EndedAt         time.Time          `bson:"endedAt,omitempty" json:"endedAt,omitempty"`
	DurationSeconds int               `bson:"durationSeconds,omitempty" json:"durationSeconds,omitempty"`
	Status          OutageStatus       `bson:"status" json:"status"`
}


// Legacy model for backward compatibility
// Used by old monitor logs in "monitor_logs" collection
type MonitorLog struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	MonitorID    primitive.ObjectID `bson:"monitorId" json:"monitorId"`
	Status       MonitorLogStatus   `bson:"status" json:"status"`
	ResponseTime int64              `bson:"responseTime" json:"responseTime"`
	StatusCode   int                `bson:"statusCode" json:"statusCode"`
	CheckedAt    time.Time          `bson:"checkedAt" json:"checkedAt"`
}

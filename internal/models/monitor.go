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
	MonitorSSL  MonitorType = "ssl"
)

type MonitorLogStatus string

const (
	MonitorUp   MonitorLogStatus = "up"
	MonitorDown MonitorLogStatus = "down"
)

type Monitor struct {
	ID                       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name                     string             `bson:"name" json:"name"`
	Type                     MonitorType        `bson:"type" json:"type"`
	Target                   string             `bson:"target" json:"target"`
	Monitoring               MonitorConfig      `bson:"monitoring,omitempty" json:"monitoring,omitempty"`
	SSLThresholds            []int              `bson:"sslThresholds,omitempty" json:"sslThresholds,omitempty"`
	IntervalSeconds          int                `bson:"intervalSeconds" json:"intervalSeconds"`
	TimeoutSeconds           int                `bson:"timeoutSeconds" json:"timeoutSeconds"`
	ComponentID              primitive.ObjectID `bson:"componentId,omitempty" json:"componentId,omitempty"`
	SubComponentID           primitive.ObjectID `bson:"subComponentId,omitempty" json:"subComponentId,omitempty"`
	LastStatus               MonitorLogStatus   `bson:"lastStatus,omitempty" json:"lastStatus,omitempty"`
	SSLWarning               bool               `bson:"sslWarning,omitempty" json:"sslWarning,omitempty"`
	SSLDaysRemaining         int                `bson:"sslDaysRemaining,omitempty" json:"sslDaysRemaining,omitempty"`
	SSLTriggeredThreshold    int                `bson:"sslTriggeredThreshold,omitempty" json:"sslTriggeredThreshold,omitempty"`
	DomainWarning            bool               `bson:"domainWarning,omitempty" json:"domainWarning,omitempty"`
	DomainDaysRemaining      int                `bson:"domainDaysRemaining,omitempty" json:"domainDaysRemaining,omitempty"`
	DomainTriggeredThreshold int                `bson:"domainTriggeredThreshold,omitempty" json:"domainTriggeredThreshold,omitempty"`
	LastCheckedAt            time.Time          `bson:"lastCheckedAt,omitempty" json:"lastCheckedAt,omitempty"`
	CreatedAt                time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt                time.Time          `bson:"updatedAt,omitempty" json:"updatedAt,omitempty"`
}

type MonitorConfig struct {
	Advanced MonitorAdvancedOptions `bson:"advanced,omitempty" json:"advanced,omitempty"`
}

type MonitorAdvancedOptions struct {
	DomainExpiry   bool `bson:"domain_expiry,omitempty" json:"domain_expiry,omitempty"`
	CertExpiry     bool `bson:"cert_expiry,omitempty" json:"cert_expiry,omitempty"`
	IgnoreTLSError bool `bson:"ignore_tls_error,omitempty" json:"ignore_tls_error,omitempty"`
}

type EnhancedMonitorLog struct {
	ID                       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	MonitorID                primitive.ObjectID `bson:"monitorId" json:"monitorId"`
	Status                   MonitorLogStatus   `bson:"status" json:"status"`
	SSLWarning               bool               `bson:"sslWarning,omitempty" json:"sslWarning,omitempty"`
	SSLDaysRemaining         int                `bson:"sslDaysRemaining,omitempty" json:"sslDaysRemaining,omitempty"`
	SSLTriggeredThreshold    int                `bson:"sslTriggeredThreshold,omitempty" json:"sslTriggeredThreshold,omitempty"`
	DomainWarning            bool               `bson:"domainWarning,omitempty" json:"domainWarning,omitempty"`
	DomainDaysRemaining      int                `bson:"domainDaysRemaining,omitempty" json:"domainDaysRemaining,omitempty"`
	DomainTriggeredThreshold int                `bson:"domainTriggeredThreshold,omitempty" json:"domainTriggeredThreshold,omitempty"`
	ResponseTime             int64              `bson:"responseTime" json:"responseTime"`
	StatusCode               int                `bson:"statusCode" json:"statusCode"`
	CheckedAt                time.Time          `bson:"checkedAt" json:"checkedAt"`
	StartedAt                time.Time          `bson:"startedAt,omitempty" json:"startedAt,omitempty"`
	EndedAt                  time.Time          `bson:"endedAt,omitempty" json:"endedAt,omitempty"`
	DurationSeconds          int                `bson:"durationSeconds,omitempty" json:"durationSeconds,omitempty"`
}

type OutageStatus string

const (
	OutageActive   OutageStatus = "active"
	OutageResolved OutageStatus = "resolved"
)

type Outage struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	MonitorID       primitive.ObjectID `bson:"monitorId" json:"monitorId"`
	ComponentID     primitive.ObjectID `bson:"componentId,omitempty" json:"componentId,omitempty"`
	SubComponentID  primitive.ObjectID `bson:"subComponentId,omitempty" json:"subComponentId,omitempty"`
	StartedAt       time.Time          `bson:"startedAt" json:"startedAt"`
	EndedAt         time.Time          `bson:"endedAt,omitempty" json:"endedAt,omitempty"`
	DurationSeconds int                `bson:"durationSeconds,omitempty" json:"durationSeconds,omitempty"`
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

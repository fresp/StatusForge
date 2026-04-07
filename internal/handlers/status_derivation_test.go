package handlers

import (
	"testing"

	"github.com/fresp/Statora/internal/models"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestDeriveStatusesFallsBackToManualWhenNoActiveImpact(t *testing.T) {
	componentID := primitive.NewObjectID()
	subID := primitive.NewObjectID()

	components := []models.Component{{
		ID:     componentID,
		Status: models.StatusOperational,
	}}
	subs := []models.SubComponent{{
		ID:          subID,
		ComponentID: componentID,
		Status:      models.StatusDegradedPerf,
	}}

	result, err := deriveStatuses(components, subs, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, models.StatusOperational, result.ComponentStatus[componentID])
	assert.Equal(t, models.StatusDegradedPerf, result.SubStatus[subID])
}

func TestDeriveStatusesMapsIncidentSeverityAndPartialImpact(t *testing.T) {
	componentID := primitive.NewObjectID()
	subA := primitive.NewObjectID()
	subB := primitive.NewObjectID()

	components := []models.Component{{
		ID:     componentID,
		Status: models.StatusOperational,
	}}
	subs := []models.SubComponent{
		{ID: subA, ComponentID: componentID, Status: models.StatusOperational},
		{ID: subB, ComponentID: componentID, Status: models.StatusOperational},
	}

	incidents := []models.Incident{{
		Impact: models.ImpactCritical,
		AffectedComponentTargets: []models.IncidentAffectedComponent{{
			ComponentID:     componentID,
			SubComponentIDs: []primitive.ObjectID{subA},
		}},
	}}

	result, err := deriveStatuses(components, subs, incidents, nil)
	assert.NoError(t, err)
	assert.Equal(t, models.StatusPartialOutage, result.ComponentStatus[componentID])
	assert.Equal(t, models.StatusMajorOutage, result.SubStatus[subA])
	assert.Equal(t, models.StatusOperational, result.SubStatus[subB])
}

func TestDeriveStatusesDirectComponentImpactOverridesToWorst(t *testing.T) {
	componentID := primitive.NewObjectID()
	subA := primitive.NewObjectID()

	components := []models.Component{{
		ID:     componentID,
		Status: models.StatusOperational,
	}}
	subs := []models.SubComponent{{
		ID:          subA,
		ComponentID: componentID,
		Status:      models.StatusOperational,
	}}

	incidents := []models.Incident{{
		Impact: models.ImpactMajor,
		AffectedComponentTargets: []models.IncidentAffectedComponent{{
			ComponentID: componentID,
		}},
	}}

	maintenance := []models.Maintenance{{
		Components: []primitive.ObjectID{componentID},
	}}

	result, err := deriveStatuses(components, subs, incidents, maintenance)
	assert.NoError(t, err)
	assert.Equal(t, models.StatusPartialOutage, result.ComponentStatus[componentID])
	assert.Equal(t, models.StatusPartialOutage, result.SubStatus[subA])
}

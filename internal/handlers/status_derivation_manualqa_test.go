package handlers

import (
	"testing"

	"github.com/fresp/Statora/internal/models"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestManualQADerivedStatusesScenarios(t *testing.T) {
	componentID := primitive.NewObjectID()
	subA := primitive.NewObjectID()
	subB := primitive.NewObjectID()

	components := []models.Component{{ID: componentID, Status: models.StatusOperational}}
	subs := []models.SubComponent{
		{ID: subA, ComponentID: componentID, Status: models.StatusOperational},
		{ID: subB, ComponentID: componentID, Status: models.StatusOperational},
	}

	t.Run("no_active_incidents_or_maintenance_falls_back_to_manual", func(t *testing.T) {
		manualComponents := []models.Component{{ID: componentID, Status: models.StatusDegradedPerf}}
		manualSubs := []models.SubComponent{{ID: subA, ComponentID: componentID, Status: models.StatusMaintenance}}
		result, err := deriveStatuses(manualComponents, manualSubs, nil, nil)
		require.NoError(t, err)
		require.Equal(t, models.StatusDegradedPerf, result.ComponentStatus[componentID])
		require.Equal(t, models.StatusMaintenance, result.SubStatus[subA])
		t.Logf("fallback component=%s subA=%s", result.ComponentStatus[componentID], result.SubStatus[subA])
	})

	t.Run("partial_subcomponent_critical_impact_yields_partial_outage_component", func(t *testing.T) {
		incidents := []models.Incident{{
			Impact: models.ImpactCritical,
			AffectedComponentTargets: []models.IncidentAffectedComponent{{
				ComponentID:     componentID,
				SubComponentIDs: []primitive.ObjectID{subA},
			}},
		}}
		result, err := deriveStatuses(components, subs, incidents, nil)
		require.NoError(t, err)
		require.Equal(t, models.StatusPartialOutage, result.ComponentStatus[componentID])
		require.Equal(t, models.StatusMajorOutage, result.SubStatus[subA])
		require.Equal(t, models.StatusOperational, result.SubStatus[subB])
		t.Logf("partial-impact component=%s subA=%s subB=%s", result.ComponentStatus[componentID], result.SubStatus[subA], result.SubStatus[subB])
	})

	t.Run("direct_component_major_impact_applies_to_all_subcomponents", func(t *testing.T) {
		incidents := []models.Incident{{
			Impact: models.ImpactMajor,
			AffectedComponentTargets: []models.IncidentAffectedComponent{{
				ComponentID: componentID,
			}},
		}}
		result, err := deriveStatuses(components, subs, incidents, nil)
		require.NoError(t, err)
		require.Equal(t, models.StatusPartialOutage, result.ComponentStatus[componentID])
		require.Equal(t, models.StatusPartialOutage, result.SubStatus[subA])
		require.Equal(t, models.StatusPartialOutage, result.SubStatus[subB])
		t.Logf("direct-impact component=%s subA=%s subB=%s", result.ComponentStatus[componentID], result.SubStatus[subA], result.SubStatus[subB])
	})
}

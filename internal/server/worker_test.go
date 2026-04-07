package server

import (
	"bytes"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fresp/Statora/configs"
	"github.com/fresp/Statora/internal/models"
)

func TestShouldRunMonitor_FirstRun(t *testing.T) {
	mon := models.Monitor{}

	assert.True(t, shouldRunMonitor(mon))
}

func TestShouldRunMonitor_DueByInterval(t *testing.T) {
	mon := models.Monitor{
		IntervalSeconds: 60,
		LastCheckedAt:   time.Now().Add(-61 * time.Second),
	}

	assert.True(t, shouldRunMonitor(mon))
}

func TestShouldRunMonitor_NotDueByInterval(t *testing.T) {
	mon := models.Monitor{
		IntervalSeconds: 60,
		LastCheckedAt:   time.Now().Add(-30 * time.Second),
	}

	assert.False(t, shouldRunMonitor(mon))
}

func TestShouldRunMonitor_ZeroIntervalFallsBackToDefault(t *testing.T) {
	mon := models.Monitor{
		IntervalSeconds: 0,
		LastCheckedAt:   time.Now().Add(-30 * time.Second),
	}

	assert.False(t, shouldRunMonitor(mon))

	mon.LastCheckedAt = time.Now().Add(-(time.Duration(defaultMonitorIntervalSeconds) + 1) * time.Second)
	assert.True(t, shouldRunMonitor(mon))
}

func TestEffectiveIntervalSeconds_DefaultFallback(t *testing.T) {
	assert.Equal(t, defaultMonitorIntervalSeconds, effectiveIntervalSeconds(0))
	assert.Equal(t, defaultMonitorIntervalSeconds, effectiveIntervalSeconds(-1))
	assert.Equal(t, 120, effectiveIntervalSeconds(120))
}

func TestDueMonitors_LogsExecuteAndSkip(t *testing.T) {
	var buf bytes.Buffer
	origWriter := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(origWriter)

	notDue := models.Monitor{
		Name:            "not-due-monitor",
		IntervalSeconds: 60,
		LastCheckedAt:   time.Now().Add(-10 * time.Second),
	}
	due := models.Monitor{
		Name:            "due-monitor",
		IntervalSeconds: 0,
		LastCheckedAt:   time.Now().Add(-(time.Duration(defaultMonitorIntervalSeconds) + 1) * time.Second),
	}

	result := dueMonitors([]models.Monitor{notDue, due})

	if assert.Len(t, result, 1) {
		assert.Equal(t, "due-monitor", result[0].Name)
	}

	output := buf.String()
	assert.Contains(t, output, "[WORKER] Skipping monitor (not due): not-due-monitor")
	assert.Contains(t, output, "[WORKER] Executing monitor: due-monitor interval: 60")
	t.Log(output)
}

func TestWorkerCycleOverlapGuard(t *testing.T) {
	workerCycleRunning.Store(false)
	assert.True(t, workerCycleRunning.CompareAndSwap(false, true))
	assert.False(t, workerCycleRunning.CompareAndSwap(false, true))
	workerCycleRunning.Store(false)
	assert.True(t, workerCycleRunning.CompareAndSwap(false, true))
	workerCycleRunning.Store(false)
}

// TestWorkerModeToggle tests that the ENABLE_WORKER flag works correctly
func TestWorkerModeToggle(t *testing.T) {
	// Test with worker enabled (default)
	cfgWithWorker := configs.Load()
	assert.True(t, cfgWithWorker.EnableWorker, "EnableWorker should be true by default")

	// Test with worker disabled
	err := os.Setenv("ENABLE_WORKER", "false")
	if err != nil {
		t.Fatal(err) // If we can't set env var, test is invalidated
	}
	defer func() {
		// Restore the environment variable after test
		os.Setenv("ENABLE_WORKER", "true")
	}()

	cfgNoWorker := configs.Load()
	assert.False(t, cfgNoWorker.EnableWorker, "EnableWorker should be false when env var is false")

	// Test with worker explicitly enabled
	err = os.Setenv("ENABLE_WORKER", "true")
	if err != nil {
		t.Fatal(err)
	}
	cfgWorkerEnabled := configs.Load()
	assert.True(t, cfgWorkerEnabled.EnableWorker, "EnableWorker should be true when env var is true")
}

// TestRunServerWorkerEnabled tests that worker goroutine is launched when enabled
func TestRunServerWorkerEnabled(t *testing.T) {
	// This test is more integration-focused as we need to test the actual RunServer function

	// For now, we'll create a partial test focusing on the configuration logic
	// The actual RunServer function launches a full server which is hard to test in isolation

	// Test is to validate that given EnableWorker=true, worker functions get called
	// This requires a more complex mocking setup, so we'll defer this to a more comprehensive test framework
	t.Skip("Full RunServer integration test deferred to end-to-end testing")
}

// TestRunServerWorkerDisabled tests behavior when the worker is disabled
func TestRunServerWorkerDisabled(t *testing.T) {
	// Similar to the worker enabled test, this involves complex integration
	// So we'll focus on a unit-level validation

	// The key aspect to test is that when EnableWorker=config.Load() = false,
	// the worker codepath isn't executed

	// This validates the toggle functionality in a broader sense
	origEnv := os.Getenv("ENABLE_WORKER")
	defer os.Setenv("ENABLE_WORKER", origEnv) // Restore original value

	// Set environment to disable worker
	os.Setenv("ENABLE_WORKER", "false")
	cfg := configs.Load()
	assert.False(t, cfg.EnableWorker, "Configuration should reflect disabled worker setting")

	// Restore in defer above
}

package observability

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskEventRepository_WriteAndQuery(t *testing.T) {
	tmpFile := t.TempDir() + "/test.db"
	repo, err := NewTaskEventRepository(tmpFile, 10)
	require.NoError(t, err)
	defer repo.Close()

	// Create test events
	traceID := "trace-123abc"

	logEvent := &TaskEvent{
		Timestamp:       time.Now(),
		TraceID:         traceID,
		SpanID:          "span-456",
		OrchestrationID: "orch-1",
		EventType:       "log",
		Activity:        "payment:charge",
		Payload:         []byte(`{"msg":"charging payment","severity":"INFO"}`),
	}

	metricEvent := &TaskEvent{
		Timestamp:   time.Now(),
		TraceID:     traceID,
		EventType:   "metric",
		Payload:     []byte(`{"metric_name":"duration","metric_value":145,"metric_unit":"ms"}`),
	}

	traceEvent := &TaskEvent{
		Timestamp:       time.Now(),
		TraceID:         traceID,
		SpanID:          "span-456",
		OrchestrationID: "orch-1",
		EventType:       "trace",
		Activity:        "payment:charge",
		Payload:         []byte(`{"span_name":"payment:charge","latency_ms":1450}`),
	}

	// Write events
	err = repo.WriteEvent(logEvent)
	require.NoError(t, err)
	err = repo.WriteEvent(metricEvent)
	require.NoError(t, err)
	err = repo.WriteEvent(traceEvent)
	require.NoError(t, err)

	// Force flush
	err = repo.FlushBatch()
	require.NoError(t, err)

	// Query by trace ID
	events, err := repo.QueryByTraceID(traceID)
	require.NoError(t, err)
	assert.Equal(t, 3, len(events))
	assert.Equal(t, "log", events[0].EventType)
	assert.Equal(t, "metric", events[1].EventType)
	assert.Equal(t, "trace", events[2].EventType)
}

func TestTaskEventRepository_OrchestrationTimeline(t *testing.T) {
	tmpFile := t.TempDir() + "/test.db"
	repo, err := NewTaskEventRepository(tmpFile, 10)
	require.NoError(t, err)
	defer repo.Close()

	orchID := "ORD-123"

	// Create timeline of events
	events := []*TaskEvent{
		{
			Timestamp:       time.Now(),
			TraceID:         "trace-1",
			OrchestrationID: orchID,
			EventType:       "log",
			Activity:        "inventory:check",
			Payload:         []byte(`{"msg":"checking availability"}`),
		},
		{
			Timestamp:       time.Now().Add(100 * time.Millisecond),
			TraceID:         "trace-1",
			OrchestrationID: orchID,
			EventType:       "trace",
			Activity:        "inventory:check",
			Payload:         []byte(`{"latency_ms":45}`),
		},
		{
			Timestamp:       time.Now().Add(200 * time.Millisecond),
			TraceID:         "trace-1",
			OrchestrationID: orchID,
			EventType:       "log",
			Activity:        "inventory:reserve",
			Payload:         []byte(`{"msg":"reserving items"}`),
		},
	}

	for _, event := range events {
		repo.WriteEvent(event)
	}
	repo.FlushBatch()

	// Query by orchestration ID
	result, err := repo.QueryByOrchestrationID(orchID)
	require.NoError(t, err)
	assert.Equal(t, 3, len(result))

	// Verify order
	assert.Equal(t, "inventory:check", result[0].Activity)
	assert.Equal(t, "inventory:check", result[1].Activity)
	assert.Equal(t, "inventory:reserve", result[2].Activity)
}

func TestTaskEventRepository_ActivityPerformance(t *testing.T) {
	tmpFile := t.TempDir() + "/test.db"
	repo, err := NewTaskEventRepository(tmpFile, 10)
	require.NoError(t, err)
	defer repo.Close()

	// Create trace events with durations
	activities := []struct {
		activity  string
		latencyMs int64
	}{
		{"payment:charge", 150},
		{"payment:charge", 200},
		{"inventory:reserve", 50},
		{"inventory:check", 30},
	}

	for _, a := range activities {
		payload := map[string]interface{}{
			"span_name":  a.activity,
			"latency_ms": a.latencyMs,
		}
		payloadBytes, _ := json.Marshal(payload)

		event := &TaskEvent{
			Timestamp: time.Now(),
			TraceID:   "trace-perf",
			EventType: "trace",
			Activity:  a.activity,
			Payload:   payloadBytes,
		}
		repo.WriteEvent(event)
	}
	repo.FlushBatch()

	// Query slow activities (> 100ms)
	results, err := repo.QueryActivityPerformance(100)
	require.NoError(t, err)

	// Should find payment:charge (avg 175ms)
	assert.True(t, len(results) > 0)
	found := false
	for _, r := range results {
		if activity, ok := r["activity"].(string); ok && activity == "payment:charge" {
			found = true
			assert.Equal(t, int64(2), r["count"])
		}
	}
	assert.True(t, found, "should find payment:charge in slow activities")
}

func TestTaskEventRepository_ErrorEvents(t *testing.T) {
	tmpFile := t.TempDir() + "/test.db"
	repo, err := NewTaskEventRepository(tmpFile, 10)
	require.NoError(t, err)
	defer repo.Close()

	// Create events with errors
	errorPayload := map[string]interface{}{
		"msg":   "payment timed out",
		"error": "PAYMENT_TIMEOUT",
	}
	errorBytes, _ := json.Marshal(errorPayload)

	event := &TaskEvent{
		Timestamp: time.Now(),
		TraceID:   "trace-error",
		EventType: "log",
		Activity:  "payment:charge",
		Payload:   errorBytes,
	}
	repo.WriteEvent(event)
	repo.FlushBatch()

	// Query error events
	errors, err := repo.QueryErrorEvents(10)
	require.NoError(t, err)
	assert.Equal(t, 1, len(errors))
	assert.Equal(t, "payment:charge", errors[0].Activity)
}

func TestTaskEventRepository_BatchPerformance(t *testing.T) {
	tmpFile := t.TempDir() + "/test.db"
	repo, err := NewTaskEventRepository(tmpFile, 100)
	require.NoError(t, err)
	defer repo.Close()

	// Write 500 events (should result in 5 batch flushes)
	start := time.Now()
	for i := 0; i < 500; i++ {
		event := &TaskEvent{
			Timestamp: time.Now(),
			TraceID:   "trace-batch",
			EventType: "log",
			Payload:   []byte(`{"msg":"test"}`),
		}
		repo.WriteEvent(event)
	}
	repo.FlushBatch()
	elapsed := time.Since(start)

	// Should complete in reasonable time
	assert.Less(t, elapsed, 5*time.Second, "batch write should be fast")

	// Verify all events were written
	events, err := repo.QueryByTraceID("trace-batch")
	require.NoError(t, err)
	assert.Equal(t, 500, len(events))
}

func TestTaskEventRepository_PruneOldEvents(t *testing.T) {
	tmpFile := t.TempDir() + "/test.db"
	repo, err := NewTaskEventRepository(tmpFile, 10)
	require.NoError(t, err)
	defer repo.Close()

	// Create an event
	event := &TaskEvent{
		Timestamp: time.Now(),
		TraceID:   "trace-prune",
		EventType: "log",
		Payload:   []byte(`{"msg":"test"}`),
	}
	repo.WriteEvent(event)
	repo.FlushBatch()

	// Verify it exists
	events, err := repo.QueryByTraceID("trace-prune")
	require.NoError(t, err)
	assert.Equal(t, 1, len(events))

	// Prune (should not delete recent)
	deleted, err := repo.PruneOldEvents(1 * time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(0), deleted)

	// Verify still exists
	events, err = repo.QueryByTraceID("trace-prune")
	require.NoError(t, err)
	assert.Equal(t, 1, len(events))
}

func TestNewLogEvent(t *testing.T) {
	timestamp := time.Now()
	attributes := map[string]interface{}{
		"order_id":       "ORD-123",
		"orchestration_id": "orch-1",
		"activity":       "payment:charge",
	}

	event := NewLogEvent("trace-123", "span-456", timestamp, "payment processed", "INFO", attributes)

	assert.Equal(t, "trace-123", event.TraceID)
	assert.Equal(t, "span-456", event.SpanID)
	assert.Equal(t, "log", event.EventType)
	assert.Equal(t, "orch-1", event.OrchestrationID)
	assert.Equal(t, "payment:charge", event.Activity)

	var payload EventPayload
	json.Unmarshal(event.Payload, &payload)
	assert.Equal(t, "payment processed", payload.Message)
	assert.Equal(t, "INFO", payload.Severity)
}

func TestNewMetricEvent(t *testing.T) {
	timestamp := time.Now()
	attributes := map[string]interface{}{
		"status": "success",
	}

	event := NewMetricEvent("trace-123", timestamp, "activity_duration", 145.5, "ms", attributes)

	assert.Equal(t, "metric", event.EventType)

	var payload EventPayload
	json.Unmarshal(event.Payload, &payload)
	assert.Equal(t, "activity_duration", payload.MetricName)
	assert.Equal(t, 145.5, payload.MetricValue)
	assert.Equal(t, "ms", payload.MetricUnit)
}

func TestNewTraceEvent(t *testing.T) {
	timestamp := time.Now()
	attributes := map[string]interface{}{
		"order_id":          "ORD-123",
		"orchestration_id":  "orch-1",
		"activity":          "payment:charge",
	}

	event := NewTraceEvent("trace-123", "span-456", "payment:charge", timestamp, 1450, "OK", attributes)

	assert.Equal(t, "trace", event.EventType)
	assert.Equal(t, "payment:charge", event.Activity)
	assert.Equal(t, "orch-1", event.OrchestrationID)

	var payload EventPayload
	json.Unmarshal(event.Payload, &payload)
	assert.Equal(t, "payment:charge", payload.SpanName)
	assert.Equal(t, int64(1450), payload.LatencyMs)
	assert.Equal(t, "OK", payload.SpanStatus)
}

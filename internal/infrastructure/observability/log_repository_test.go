package observability

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogRepository_WriteAndQuery(t *testing.T) {
	// Create temporary database
	tmpFile := t.TempDir() + "/test.db"
	repo, err := NewLogRepository(tmpFile, 10)
	require.NoError(t, err)
	defer repo.Close()

	// Create test logs
	traceID := "trace-123"
	record1 := NewLogRecord(LogLevelInfo, traceID, "test message 1").
		WithOrchestrationID("orch-1").
		WithActivity("activity:test").
		WithDuration(100 * time.Millisecond).
		WithInput([]byte(`{"key":"value"}`))

	record2 := NewLogRecord(LogLevelError, traceID, "test error").
		WithOrchestrationID("orch-1").
		WithActivity("activity:test").
		WithError("TEST_ERROR: something went wrong")

	// Write logs
	err = repo.WriteLog(record1)
	require.NoError(t, err)

	err = repo.WriteLog(record2)
	require.NoError(t, err)

	// Force flush
	err = repo.FlushBatch()
	require.NoError(t, err)

	// Query by trace ID
	logs, err := repo.QueryByTraceID(traceID)
	require.NoError(t, err)
	assert.Equal(t, 2, len(logs))
	assert.Equal(t, "test message 1", logs[0].Message)
	assert.Equal(t, "test error", logs[1].Message)
}

func TestLogRepository_ErrorGrouping(t *testing.T) {
	tmpFile := t.TempDir() + "/test.db"
	repo, err := NewLogRepository(tmpFile, 10)
	require.NoError(t, err)
	defer repo.Close()

	// Create multiple logs with same error type (different messages)
	errorHash := hashError("PAYMENT_FAILED")

	for i := 0; i < 3; i++ {
		record := NewLogRecord(LogLevelError, "trace-"+string(rune(i)), "payment failed").
			WithActivity("payment:charge").
			WithError("PAYMENT_FAILED: timeout" + string(rune(i)))

		repo.WriteLog(record)
	}

	repo.FlushBatch()

	// Query by error hash
	logs, err := repo.QueryErrorsByHash(errorHash)
	require.NoError(t, err)
	assert.True(t, len(logs) >= 1, "should find logs with same error hash")
}

func TestLogRepository_SlowActivities(t *testing.T) {
	tmpFile := t.TempDir() + "/test.db"
	repo, err := NewLogRepository(tmpFile, 10)
	require.NoError(t, err)
	defer repo.Close()

	// Create logs with varying durations
	activities := []struct {
		name     string
		duration time.Duration
	}{
		{"activity:fast", 10 * time.Millisecond},
		{"activity:slow", 100 * time.Millisecond},
		{"activity:slow", 150 * time.Millisecond},
		{"activity:very_slow", 500 * time.Millisecond},
	}

	for i, a := range activities {
		record := NewLogRecord(LogLevelInfo, "trace-"+string(rune(i)), "test").
			WithActivity(a.name).
			WithDuration(a.duration)

		repo.WriteLog(record)
	}

	repo.FlushBatch()

	// Query slow activities (threshold > 50ms)
	results, err := repo.QuerySlowActivities(50, 10)
	require.NoError(t, err)

	assert.True(t, len(results) > 0)
	// Verify slowest activity is included
	found := false
	for _, r := range results {
		if activity, ok := r["activity"].(string); ok && activity == "activity:very_slow" {
			found = true
		}
	}
	assert.True(t, found, "very_slow activity should be in results")
}

func TestLogRepository_ErrorFrequency(t *testing.T) {
	tmpFile := t.TempDir() + "/test.db"
	repo, err := NewLogRepository(tmpFile, 10)
	require.NoError(t, err)
	defer repo.Close()

	// Create logs with different errors
	errors := []string{
		"PAYMENT_FAILED: timeout",
		"PAYMENT_FAILED: invalid_card",
		"INVENTORY_FAILED: out_of_stock",
		"PAYMENT_FAILED: network_error",
	}

	for i, e := range errors {
		record := NewLogRecord(LogLevelError, "trace-"+string(rune(i)), "error").
			WithError(e)

		repo.WriteLog(record)
	}

	repo.FlushBatch()

	// Query error frequency
	results, err := repo.QueryErrorFrequency(10)
	require.NoError(t, err)

	assert.True(t, len(results) > 0)
	// PAYMENT_FAILED should be most frequent (3 occurrences)
	if len(results) > 0 {
		mostFrequent := results[0]
		assert.Equal(t, int64(3), mostFrequent["frequency"])
	}
}

func TestLogRepository_InputOutputHashing(t *testing.T) {
	tmpFile := t.TempDir() + "/test.db"
	repo, err := NewLogRepository(tmpFile, 10)
	require.NoError(t, err)
	defer repo.Close()

	inputData := []byte(`{"order_id":"123","amount":100}`)
	outputData := []byte(`{"payment_id":"pay-456","status":"completed"}`)

	record := NewLogRecord(LogLevelInfo, "trace-1", "payment processed").
		WithInput(inputData).
		WithOutput(outputData)

	repo.WriteLog(record)
	repo.FlushBatch()

	// Query and verify hashes are present
	logs, err := repo.QueryByTraceID("trace-1")
	require.NoError(t, err)
	require.Equal(t, 1, len(logs))

	assert.NotEmpty(t, logs[0].InputHash)
	assert.NotEmpty(t, logs[0].OutputHash)
	// Verify hashes are deterministic
	assert.Equal(t, hashData(inputData), logs[0].InputHash)
	assert.Equal(t, hashData(outputData), logs[0].OutputHash)
}

func TestLogRepository_Batch_Performance(t *testing.T) {
	tmpFile := t.TempDir() + "/test.db"
	repo, err := NewLogRepository(tmpFile, 100)
	require.NoError(t, err)
	defer repo.Close()

	// Write 500 logs (should result in 5 batch inserts)
	start := time.Now()
	for i := 0; i < 500; i++ {
		record := NewLogRecord(LogLevelInfo, "trace-batch", "test").
			WithActivity("activity:test").
			WithDuration(time.Duration(i) * time.Millisecond)

		repo.WriteLog(record)
	}
	repo.FlushBatch()
	elapsed := time.Since(start)

	// Should complete in reasonable time (batching makes this fast)
	assert.Less(t, elapsed, 5*time.Second, "batch write should be fast")

	// Verify all logs were written
	logs, err := repo.QueryByTraceID("trace-batch")
	require.NoError(t, err)
	assert.Equal(t, 500, len(logs))
}

func TestLogRepository_PruneOldLogs(t *testing.T) {
	tmpFile := t.TempDir() + "/test.db"
	repo, err := NewLogRepository(tmpFile, 10)
	require.NoError(t, err)
	defer repo.Close()

	// Create a log
	record := NewLogRecord(LogLevelInfo, "trace-1", "test").
		WithActivity("activity:test")

	repo.WriteLog(record)
	repo.FlushBatch()

	// Verify it exists
	logs, err := repo.QueryByTraceID("trace-1")
	require.NoError(t, err)
	assert.Equal(t, 1, len(logs))

	// Prune logs older than 1 minute (should not delete recent log)
	deleted, err := repo.PruneOldLogs(1 * time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(0), deleted)

	// Verify log still exists
	logs, err = repo.QueryByTraceID("trace-1")
	require.NoError(t, err)
	assert.Equal(t, 1, len(logs))
}

func TestHashError(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"PAYMENT_FAILED: timeout", hashError("PAYMENT_FAILED: timeout")},
		{"PAYMENT_FAILED: network", hashError("PAYMENT_FAILED: timeout")}, // Same code = same hash
		{"INVENTORY_FAILED: out_of_stock", hashError("INVENTORY_FAILED: out_of_stock")},
	}

	for _, tt := range tests {
		result := hashError(tt.input)
		assert.Equal(t, tt.expected, result)
	}

	// Verify that same error code produces same hash
	hash1 := hashError("PAYMENT_FAILED: reason1")
	hash2 := hashError("PAYMENT_FAILED: reason2")
	assert.Equal(t, hash1, hash2, "same error code should produce same hash")
}

func TestHashData(t *testing.T) {
	input1 := []byte(`{"key":"value"}`)
	input2 := []byte(`{"key":"value"}`)
	input3 := []byte(`{"key":"other"}`)

	hash1 := hashData(input1)
	hash2 := hashData(input2)
	hash3 := hashData(input3)

	assert.Equal(t, hash1, hash2, "identical data should have identical hashes")
	assert.NotEqual(t, hash1, hash3, "different data should have different hashes")
}

func TestLogRecord_Builder(t *testing.T) {
	traceID := "trace-123"
	record := NewLogRecord(LogLevelInfo, traceID, "test message").
		WithOrchestrationID("orch-1").
		WithActivity("payment:charge").
		WithDuration(100 * time.Millisecond).
		WithInput([]byte(`{"amount":100}`)).
		WithOutput([]byte(`{"id":"pay-1"}`)).
		WithError("TEST_ERROR: something")

	assert.Equal(t, traceID, record.TraceID)
	assert.Equal(t, "orch-1", record.OrchestrationID)
	assert.Equal(t, "payment:charge", record.Activity)
	assert.Equal(t, int64(100), record.DurationMs)
	assert.NotEmpty(t, record.InputHash)
	assert.NotEmpty(t, record.OutputHash)
	assert.Equal(t, "TEST_ERROR: something", record.ErrorMessage)
	assert.NotEmpty(t, record.ErrorHash)
}

package observability

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// TaskEventRepository handles persistence of task events to SQLite
type TaskEventRepository struct {
	db        *sql.DB
	mu        sync.Mutex
	batch     []*TaskEvent
	batchSize int
	flushTick *time.Ticker
	done      chan struct{}
}

// NewTaskEventRepository creates a new repository
func NewTaskEventRepository(dbPath string, batchSize int) (*TaskEventRepository, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	repo := &TaskEventRepository{
		db:        db,
		batch:     make([]*TaskEvent, 0, batchSize),
		batchSize: batchSize,
		flushTick: time.NewTicker(5 * time.Second),
		done:      make(chan struct{}),
	}

	// Initialize schema
	if err := repo.initSchema(); err != nil {
		return nil, err
	}

	// Start background flush worker
	go repo.flushWorker()

	return repo, nil
}

// initSchema creates the task_events table and indexes
func (r *TaskEventRepository) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS task_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		trace_id TEXT NOT NULL,
		span_id TEXT,
		orchestration_id TEXT,
		event_type TEXT NOT NULL,
		activity TEXT,
		payload JSON NOT NULL
	);

	-- PRIMARY INDEX for trace correlation
	CREATE INDEX IF NOT EXISTS idx_trace_id ON task_events(trace_id);

	-- SECONDARY INDEX for orchestration tracking
	CREATE INDEX IF NOT EXISTS idx_orchestration_id ON task_events(orchestration_id);

	-- COMPOSITE INDEX for common query patterns
	CREATE INDEX IF NOT EXISTS idx_trace_activity
		ON task_events(trace_id, activity, timestamp);

	-- TIME-BASED queries
	CREATE INDEX IF NOT EXISTS idx_timestamp ON task_events(timestamp);

	-- EVENT TYPE filtering
	CREATE INDEX IF NOT EXISTS idx_event_type ON task_events(event_type);

	-- Orchestration timeline
	CREATE INDEX IF NOT EXISTS idx_orchestration_timestamp
		ON task_events(orchestration_id, timestamp);
	`

	_, err := r.db.Exec(schema)
	return err
}

// WriteEvent adds an event to the batch
func (r *TaskEventRepository) WriteEvent(event *TaskEvent) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	r.mu.Lock()
	r.batch = append(r.batch, event)
	shouldFlush := len(r.batch) >= r.batchSize
	r.mu.Unlock()

	if shouldFlush {
		return r.FlushBatch()
	}

	return nil
}

// FlushBatch writes all batched events to the database in a single transaction
func (r *TaskEventRepository) FlushBatch() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.batch) == 0 {
		return nil
	}

	// Start transaction
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO task_events (
			timestamp, trace_id, span_id, orchestration_id,
			event_type, activity, payload
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Execute all inserts within transaction
	for _, event := range r.batch {
		_, err := stmt.Exec(
			event.Timestamp,
			event.TraceID,
			event.SpanID,
			event.OrchestrationID,
			event.EventType,
			event.Activity,
			string(event.Payload),
		)
		if err != nil {
			return fmt.Errorf("failed to insert event: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Clear batch after successful flush
	r.batch = r.batch[:0]
	return nil
}

// flushWorker periodically flushes events to the database
func (r *TaskEventRepository) flushWorker() {
	for {
		select {
		case <-r.flushTick.C:
			r.FlushBatch()
		case <-r.done:
			r.FlushBatch() // Final flush on shutdown
			return
		}
	}
}

// Close flushes remaining events and closes the database
func (r *TaskEventRepository) Close() error {
	r.flushTick.Stop()
	close(r.done)
	<-time.After(100 * time.Millisecond) // Wait for worker to finish

	if err := r.FlushBatch(); err != nil {
		return err
	}

	return r.db.Close()
}

// QueryByTraceID retrieves all events for a given trace ID
func (r *TaskEventRepository) QueryByTraceID(traceID string) ([]*TaskEvent, error) {
	rows, err := r.db.Query(`
		SELECT id, timestamp, trace_id, span_id, orchestration_id,
		       event_type, activity, payload
		FROM task_events
		WHERE trace_id = ?
		ORDER BY timestamp ASC
	`, traceID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// QueryByOrchestrationID retrieves all events for a given orchestration
func (r *TaskEventRepository) QueryByOrchestrationID(orchID string) ([]*TaskEvent, error) {
	rows, err := r.db.Query(`
		SELECT id, timestamp, trace_id, span_id, orchestration_id,
		       event_type, activity, payload
		FROM task_events
		WHERE orchestration_id = ?
		ORDER BY timestamp ASC
	`, orchID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// QueryByEventType retrieves all events of a specific type
func (r *TaskEventRepository) QueryByEventType(eventType string) ([]*TaskEvent, error) {
	rows, err := r.db.Query(`
		SELECT id, timestamp, trace_id, span_id, orchestration_id,
		       event_type, activity, payload
		FROM task_events
		WHERE event_type = ?
		ORDER BY timestamp DESC
		LIMIT 1000
	`, eventType)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// QueryActivityPerformance retrieves performance metrics for activities
func (r *TaskEventRepository) QueryActivityPerformance(thresholdMs int64) ([]map[string]interface{}, error) {
	rows, err := r.db.Query(`
		SELECT
			activity,
			COUNT(*) as count,
			AVG(json_extract(payload, '$.latency_ms')) as avg_latency_ms,
			MAX(json_extract(payload, '$.latency_ms')) as max_latency_ms,
			MIN(json_extract(payload, '$.latency_ms')) as min_latency_ms
		FROM task_events
		WHERE event_type = 'trace' AND timestamp > datetime('now', '-1 hour')
		GROUP BY activity
		HAVING MAX(json_extract(payload, '$.latency_ms')) > ?
		ORDER BY avg_latency_ms DESC
	`, thresholdMs)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		var activity string
		var count, avgLatency, maxLatency, minLatency sql.NullInt64

		if err := rows.Scan(&activity, &count, &avgLatency, &maxLatency, &minLatency); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		results = append(results, map[string]interface{}{
			"activity":           activity,
			"count":              count.Int64,
			"avg_latency_ms":     avgLatency.Int64,
			"max_latency_ms":     maxLatency.Int64,
			"min_latency_ms":     minLatency.Int64,
		})
	}

	return results, rows.Err()
}

// QueryErrorEvents retrieves error events from logs and traces
func (r *TaskEventRepository) QueryErrorEvents(limit int) ([]*TaskEvent, error) {
	rows, err := r.db.Query(`
		SELECT id, timestamp, trace_id, span_id, orchestration_id,
		       event_type, activity, payload
		FROM task_events
		WHERE json_extract(payload, '$.error') IS NOT NULL
		   OR json_extract(payload, '$.span_status') = 'ERROR'
		ORDER BY timestamp DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// PruneOldEvents deletes events older than the specified duration
func (r *TaskEventRepository) PruneOldEvents(olderThan time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-olderThan)

	result, err := r.db.Exec(`
		DELETE FROM task_events
		WHERE timestamp < ?
	`, cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("delete failed: %w", err)
	}

	return result.RowsAffected()
}

// scanRows scans database rows into TaskEvent structs
func (r *TaskEventRepository) scanRows(rows *sql.Rows) ([]*TaskEvent, error) {
	events := make([]*TaskEvent, 0)

	for rows.Next() {
		var id int64
		var timestamp time.Time
		var traceID, spanID, orchID, eventType, activity string
		var payload string

		err := rows.Scan(
			&id, &timestamp, &traceID, &spanID, &orchID,
			&eventType, &activity, &payload,
		)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		event := &TaskEvent{
			ID:              id,
			Timestamp:       timestamp,
			TraceID:         traceID,
			SpanID:          spanID,
			OrchestrationID: orchID,
			EventType:       eventType,
			Activity:        activity,
			Payload:         []byte(payload),
		}

		events = append(events, event)
	}

	return events, rows.Err()
}

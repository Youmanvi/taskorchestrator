package observability

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// LogRepository handles persistence of logs to SQLite
type LogRepository struct {
	db        *sql.DB
	mu        sync.Mutex
	batch     []*LogRecord
	batchSize int
	flushTick *time.Ticker
	done      chan struct{}
}

// NewLogRepository creates a new log repository
func NewLogRepository(dbPath string, batchSize int) (*LogRepository, error) {
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

	repo := &LogRepository{
		db:        db,
		batch:     make([]*LogRecord, 0, batchSize),
		batchSize: batchSize,
		flushTick: time.NewTicker(5 * time.Second),
		done:      make(chan struct{}),
	}

	// Initialize schema
	if err := repo.initSchema(); err != nil {
		return nil, err
	}

	// Start background flush ticker
	go repo.flushWorker()

	return repo, nil
}

// initSchema creates the necessary tables and indexes
func (r *LogRepository) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		level TEXT NOT NULL,
		trace_id TEXT NOT NULL,
		span_id TEXT,
		orchestration_id TEXT,
		activity TEXT,
		message TEXT NOT NULL,
		duration_ms INTEGER,
		input_hash TEXT,
		output_hash TEXT,
		error_message TEXT,
		error_hash TEXT,
		raw_json TEXT
	);

	-- PRIMARY INDEX for efficient trace correlation
	CREATE INDEX IF NOT EXISTS idx_trace_id ON logs(trace_id);

	-- SECONDARY INDEX for orchestration correlation
	CREATE INDEX IF NOT EXISTS idx_orchestration_id ON logs(orchestration_id);

	-- COMPOSITE INDEX for common query patterns
	CREATE INDEX IF NOT EXISTS idx_trace_activity
		ON logs(trace_id, activity, timestamp);

	-- ERROR deduplication and grouping
	CREATE INDEX IF NOT EXISTS idx_error_hash ON logs(error_hash);

	-- Time-based queries and cleanup
	CREATE INDEX IF NOT EXISTS idx_timestamp ON logs(timestamp);

	-- Activity performance analysis
	CREATE INDEX IF NOT EXISTS idx_activity_timestamp
		ON logs(activity, timestamp DESC);
	`

	_, err := r.db.Exec(schema)
	return err
}

// WriteLog adds a log record to the batch
func (r *LogRepository) WriteLog(log *LogRecord) error {
	if log == nil {
		return fmt.Errorf("log record cannot be nil")
	}

	r.mu.Lock()
	r.batch = append(r.batch, log)
	shouldFlush := len(r.batch) >= r.batchSize
	r.mu.Unlock()

	if shouldFlush {
		return r.FlushBatch()
	}

	return nil
}

// FlushBatch writes all batched logs to the database in a single transaction
func (r *LogRepository) FlushBatch() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.batch) == 0 {
		return nil
	}

	// Start transaction for atomic write
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO logs (
			timestamp, level, trace_id, span_id, orchestration_id,
			activity, message, duration_ms, input_hash, output_hash,
			error_message, error_hash, raw_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Execute all inserts within transaction
	for _, log := range r.batch {
		rawJSON, _ := log.Marshal()

		_, err := stmt.Exec(
			log.Timestamp,
			log.Level,
			log.TraceID,
			log.SpanID,
			log.OrchestrationID,
			log.Activity,
			log.Message,
			log.DurationMs,
			log.InputHash,
			log.OutputHash,
			log.ErrorMessage,
			log.ErrorHash,
			string(rawJSON),
		)
		if err != nil {
			return fmt.Errorf("failed to insert log: %w", err)
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

// flushWorker periodically flushes logs to the database
func (r *LogRepository) flushWorker() {
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

// Close flushes remaining logs and closes the database connection
func (r *LogRepository) Close() error {
	r.flushTick.Stop()
	close(r.done)
	<-time.After(100 * time.Millisecond) // Wait for worker to finish

	if err := r.FlushBatch(); err != nil {
		return err
	}

	return r.db.Close()
}

// QueryByTraceID retrieves all logs for a given trace ID
func (r *LogRepository) QueryByTraceID(traceID string) ([]*LogRecord, error) {
	rows, err := r.db.Query(`
		SELECT id, timestamp, level, trace_id, span_id, orchestration_id,
		       activity, message, duration_ms, input_hash, output_hash,
		       error_message, error_hash
		FROM logs
		WHERE trace_id = ?
		ORDER BY timestamp ASC
	`, traceID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// QueryByOrchestrationID retrieves all logs for a given orchestration
func (r *LogRepository) QueryByOrchestrationID(orchID string) ([]*LogRecord, error) {
	rows, err := r.db.Query(`
		SELECT id, timestamp, level, trace_id, span_id, orchestration_id,
		       activity, message, duration_ms, input_hash, output_hash,
		       error_message, error_hash
		FROM logs
		WHERE orchestration_id = ?
		ORDER BY timestamp ASC
	`, orchID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// QueryErrorsByHash retrieves all logs with a specific error hash
func (r *LogRepository) QueryErrorsByHash(errorHash string) ([]*LogRecord, error) {
	rows, err := r.db.Query(`
		SELECT id, timestamp, level, trace_id, span_id, orchestration_id,
		       activity, message, duration_ms, input_hash, output_hash,
		       error_message, error_hash
		FROM logs
		WHERE error_hash = ?
		ORDER BY timestamp DESC
		LIMIT 1000
	`, errorHash)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// QuerySlowActivities retrieves activities that took longer than threshold
func (r *LogRepository) QuerySlowActivities(thresholdMs int64, limit int) ([]map[string]interface{}, error) {
	rows, err := r.db.Query(`
		SELECT activity,
		       COUNT(*) as count,
		       AVG(duration_ms) as avg_duration_ms,
		       MAX(duration_ms) as max_duration_ms,
		       MIN(duration_ms) as min_duration_ms
		FROM logs
		WHERE duration_ms > 0 AND timestamp > datetime('now', '-1 hour')
		GROUP BY activity
		HAVING MAX(duration_ms) > ?
		ORDER BY avg_duration_ms DESC
		LIMIT ?
	`, thresholdMs, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		var activity string
		var count, avgDurationMs, maxDurationMs, minDurationMs int64

		if err := rows.Scan(&activity, &count, &avgDurationMs, &maxDurationMs, &minDurationMs); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		results = append(results, map[string]interface{}{
			"activity":          activity,
			"count":             count,
			"avg_duration_ms":   avgDurationMs,
			"max_duration_ms":   maxDurationMs,
			"min_duration_ms":   minDurationMs,
		})
	}

	return results, rows.Err()
}

// QueryErrorFrequency returns error distribution
func (r *LogRepository) QueryErrorFrequency(limit int) ([]map[string]interface{}, error) {
	rows, err := r.db.Query(`
		SELECT error_hash, error_message, COUNT(*) as frequency
		FROM logs
		WHERE error_message IS NOT NULL AND error_hash IS NOT NULL
		GROUP BY error_hash
		ORDER BY frequency DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		var errorHash, errorMessage string
		var frequency int64

		if err := rows.Scan(&errorHash, &errorMessage, &frequency); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		results = append(results, map[string]interface{}{
			"error_hash":    errorHash,
			"error_message": errorMessage,
			"frequency":     frequency,
		})
	}

	return results, rows.Err()
}

// PruneOldLogs deletes logs older than the specified duration
func (r *LogRepository) PruneOldLogs(olderThan time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-olderThan)

	result, err := r.db.Exec(`
		DELETE FROM logs
		WHERE timestamp < ?
	`, cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("delete failed: %w", err)
	}

	return result.RowsAffected()
}

// scanRows scans database rows into LogRecord structs
func (r *LogRepository) scanRows(rows *sql.Rows) ([]*LogRecord, error) {
	records := make([]*LogRecord, 0)

	for rows.Next() {
		var id int64
		var timestamp time.Time
		var level, traceID, spanID, orchID, activity, message string
		var durationMs sql.NullInt64
		var inputHash, outputHash, errorMsg, errorHash sql.NullString

		err := rows.Scan(
			&id, &timestamp, &level, &traceID, &spanID, &orchID,
			&activity, &message, &durationMs, &inputHash, &outputHash,
			&errorMsg, &errorHash,
		)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		record := &LogRecord{
			ID:              id,
			Timestamp:       timestamp,
			Level:           LogLevel(level),
			TraceID:         traceID,
			SpanID:          spanID,
			OrchestrationID: orchID,
			Activity:        activity,
			Message:         message,
		}

		if durationMs.Valid {
			record.DurationMs = durationMs.Int64
		}
		if inputHash.Valid {
			record.InputHash = inputHash.String
		}
		if outputHash.Valid {
			record.OutputHash = outputHash.String
		}
		if errorMsg.Valid {
			record.ErrorMessage = errorMsg.String
		}
		if errorHash.Valid {
			record.ErrorHash = errorHash.String
		}

		records = append(records, record)
	}

	return records, rows.Err()
}

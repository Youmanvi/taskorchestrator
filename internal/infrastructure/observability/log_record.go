package observability

import (
	"crypto/sha256"
	"encoding/json"
	"encoding/hex"
	"strings"
	"time"
)

// LogLevel represents the severity level of a log
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// LogRecord represents a structured log entry to be persisted
type LogRecord struct {
	ID              int64           `json:"id,omitempty"`
	Timestamp       time.Time       `json:"timestamp"`
	Level           LogLevel        `json:"level"`
	TraceID         string          `json:"trace_id"`
	SpanID          string          `json:"span_id,omitempty"`
	OrchestrationID string          `json:"orchestration_id,omitempty"`
	Activity        string          `json:"activity,omitempty"`
	Message         string          `json:"message"`
	DurationMs      int64           `json:"duration_ms,omitempty"`
	InputHash       string          `json:"input_hash,omitempty"`
	OutputHash      string          `json:"output_hash,omitempty"`
	ErrorMessage    string          `json:"error,omitempty"`
	ErrorHash       string          `json:"error_hash,omitempty"`
	RawJSON         json.RawMessage `json:"raw_json,omitempty"`
}

// NewLogRecord creates a new log record
func NewLogRecord(level LogLevel, traceID, message string) *LogRecord {
	return &LogRecord{
		Timestamp: time.Now(),
		Level:     level,
		TraceID:   traceID,
		Message:   message,
	}
}

// WithOrchestrationID adds orchestration context
func (lr *LogRecord) WithOrchestrationID(id string) *LogRecord {
	lr.OrchestrationID = id
	return lr
}

// WithActivity adds activity context
func (lr *LogRecord) WithActivity(name string) *LogRecord {
	lr.Activity = name
	return lr
}

// WithDuration adds execution duration
func (lr *LogRecord) WithDuration(d time.Duration) *LogRecord {
	lr.DurationMs = d.Milliseconds()
	return lr
}

// WithInput adds and hashes input data
func (lr *LogRecord) WithInput(data []byte) *LogRecord {
	if len(data) > 0 {
		lr.InputHash = hashData(data)
	}
	return lr
}

// WithOutput adds and hashes output data
func (lr *LogRecord) WithOutput(data []byte) *LogRecord {
	if len(data) > 0 {
		lr.OutputHash = hashData(data)
	}
	return lr
}

// WithError adds error information
func (lr *LogRecord) WithError(errMsg string) *LogRecord {
	if errMsg != "" {
		lr.ErrorMessage = errMsg
		lr.ErrorHash = hashError(errMsg)
	}
	return lr
}

// hashData creates a SHA256 hash of data
func hashData(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// hashError creates a truncated hash of error for grouping
// Hashes only the error code (first part before colon), not the full message
func hashError(errMsg string) string {
	if errMsg == "" {
		return ""
	}

	// Extract error code (e.g., "PAYMENT_FAILED" from "PAYMENT_FAILED: timeout")
	parts := strings.SplitN(errMsg, ":", 2)
	errorCode := parts[0]

	hash := sha256.Sum256([]byte(errorCode))
	// Truncate to 16 hex chars (64 bits) for readability
	return hex.EncodeToString(hash[:8])
}

// Marshal serializes the log record to JSON
func (lr *LogRecord) Marshal() ([]byte, error) {
	return json.Marshal(lr)
}

// String returns a string representation
func (lr *LogRecord) String() string {
	data, _ := json.Marshal(lr)
	return string(data)
}

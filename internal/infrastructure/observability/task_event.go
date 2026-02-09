package observability

import (
	"encoding/json"
	"time"
)

// TaskEvent represents a telemetry event (log, metric, trace) stored in SQLite
type TaskEvent struct {
	ID              int64             `json:"id,omitempty"`
	Timestamp       time.Time         `json:"timestamp"`
	TraceID         string            `json:"trace_id"`
	SpanID          string            `json:"span_id,omitempty"`
	OrchestrationID string            `json:"orchestration_id,omitempty"`
	EventType       string            `json:"event_type"` // log, metric, trace
	Activity        string            `json:"activity,omitempty"`
	Payload         json.RawMessage   `json:"payload"`
}

// EventPayload is the structure of the JSON payload
type EventPayload struct {
	// Common fields
	Message       string                 `json:"msg,omitempty"`
	Severity      string                 `json:"severity,omitempty"`
	Error         string                 `json:"error,omitempty"`

	// For metrics
	MetricName    string                 `json:"metric_name,omitempty"`
	MetricValue   float64                `json:"metric_value,omitempty"`
	MetricUnit    string                 `json:"metric_unit,omitempty"`

	// For traces/spans
	SpanName      string                 `json:"span_name,omitempty"`
	SpanKind      string                 `json:"span_kind,omitempty"`
	SpanStatus    string                 `json:"span_status,omitempty"`
	LatencyMs     int64                  `json:"latency_ms,omitempty"`

	// Common attributes
	Attributes    map[string]interface{} `json:"attributes,omitempty"`
	Status        string                 `json:"status,omitempty"`
	Input         map[string]interface{} `json:"input,omitempty"`
	Output        map[string]interface{} `json:"output,omitempty"`
}

// NewLogEvent creates a task event from a log
func NewLogEvent(traceID, spanID string, timestamp time.Time, message string, severity string, attributes map[string]interface{}) *TaskEvent {
	payload := EventPayload{
		Message:    message,
		Severity:   severity,
		Attributes: attributes,
		EventType:  "log",
	}

	// Extract orchestration_id from attributes if present
	orchID := ""
	if val, ok := attributes["orchestration_id"].(string); ok {
		orchID = val
	}

	// Extract activity from attributes if present
	activity := ""
	if val, ok := attributes["activity"].(string); ok {
		activity = val
	}

	payloadBytes, _ := json.Marshal(payload)

	return &TaskEvent{
		Timestamp:       timestamp,
		TraceID:         traceID,
		SpanID:          spanID,
		OrchestrationID: orchID,
		EventType:       "log",
		Activity:        activity,
		Payload:         payloadBytes,
	}
}

// NewMetricEvent creates a task event from a metric
func NewMetricEvent(traceID string, timestamp time.Time, metricName string, value float64, unit string, attributes map[string]interface{}) *TaskEvent {
	payload := EventPayload{
		MetricName:  metricName,
		MetricValue: value,
		MetricUnit:  unit,
		Attributes:  attributes,
	}

	payloadBytes, _ := json.Marshal(payload)

	return &TaskEvent{
		Timestamp:   timestamp,
		TraceID:     traceID,
		EventType:   "metric",
		Payload:     payloadBytes,
	}
}

// NewTraceEvent creates a task event from a span
func NewTraceEvent(traceID, spanID, spanName string, timestamp time.Time, latencyMs int64, status string, attributes map[string]interface{}) *TaskEvent {
	payload := EventPayload{
		SpanName:   spanName,
		SpanStatus: status,
		LatencyMs:  latencyMs,
		Attributes: attributes,
	}

	// Extract orchestration_id and activity from attributes
	orchID := ""
	activity := ""
	if val, ok := attributes["orchestration_id"].(string); ok {
		orchID = val
	}
	if val, ok := attributes["activity"].(string); ok {
		activity = val
	}

	payloadBytes, _ := json.Marshal(payload)

	return &TaskEvent{
		Timestamp:       timestamp,
		TraceID:         traceID,
		SpanID:          spanID,
		OrchestrationID: orchID,
		EventType:       "trace",
		Activity:        activity,
		Payload:         payloadBytes,
	}
}

package observability

import (
	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	collectorlogs "go.opentelemetry.io/proto/otlpv1/collector/logs"
	collectormetrics "go.opentelemetry.io/proto/otlpv1/collector/metrics"
	collectortraces "go.opentelemetry.io/proto/otlpv1/collector/traces"
	commonpb "go.opentelemetry.io/proto/otlpv1/common"
	logspb "go.opentelemetry.io/proto/otlpv1/logs"
	metricspb "go.opentelemetry.io/proto/otlpv1/metrics"
	tracespb "go.opentelemetry.io/proto/otlpv1/traces"
)

// OTLPReceiver receives OTLP gRPC messages and writes to SQLite
type OTLPReceiver struct {
	eventRepo *TaskEventRepository
	logger    *Logger
	server    *grpc.Server
	listener  net.Listener
}

// NewOTLPReceiver creates a new OTLP receiver
func NewOTLPReceiver(eventRepo *TaskEventRepository, logger *Logger) (*OTLPReceiver, error) {
	return &OTLPReceiver{
		eventRepo: eventRepo,
		logger:    logger,
	}, nil
}

// Start starts the gRPC server on localhost:4317
func (r *OTLPReceiver) Start(ctx context.Context) error {
	var err error
	r.listener, err = net.Listen("tcp", "localhost:4317")
	if err != nil {
		return fmt.Errorf("failed to listen on :4317: %w", err)
	}

	r.server = grpc.NewServer()

	// Register OTLP services
	collectorlogs.RegisterLogsServiceServer(r.server, r)
	collectormetrics.RegisterMetricsServiceServer(r.server, r)
	collectortraces.RegisterTracesServiceServer(r.server, r)

	// Start server in background
	go func() {
		if err := r.server.Serve(r.listener); err != nil && err != grpc.ErrServerStopped {
			r.logger.Logger.Error().Err(err).Msg("OTLP receiver server error")
		}
	}()

	r.logger.Logger.Info().Msg("OTLP receiver started on localhost:4317")
	return nil
}

// Stop gracefully shuts down the receiver
func (r *OTLPReceiver) Stop() {
	if r.server != nil {
		r.server.GracefulStop()
	}
}

// Export implements the Logs service
func (r *OTLPReceiver) Export(ctx context.Context, req *collectorlogs.ExportLogsServiceRequest) (*collectorlogs.ExportLogsServiceResponse, error) {
	for _, resourceLogs := range req.GetResourceLogs() {
		for _, scopeLogs := range resourceLogs.GetScopeLogs() {
			for _, logRecord := range scopeLogs.GetLogRecords() {
				event := r.logRecordToEvent(logRecord)
				if err := r.eventRepo.WriteEvent(event); err != nil {
					r.logger.Logger.Error().Err(err).Msg("failed to write log event")
				}
			}
		}
	}

	return &collectorlogs.ExportLogsServiceResponse{
		PartialSuccess: nil,
	}, nil
}

// Export implements the Metrics service
func (r *OTLPReceiver) Export(ctx context.Context, req *collectormetrics.ExportMetricsServiceRequest) (*collectormetrics.ExportMetricsServiceResponse, error) {
	for _, resourceMetrics := range req.GetResourceMetrics() {
		for _, scopeMetrics := range resourceMetrics.GetScopeMetrics() {
			for _, metric := range scopeMetrics.GetMetrics() {
				events := r.metricToEvents(metric)
				for _, event := range events {
					if err := r.eventRepo.WriteEvent(event); err != nil {
						r.logger.Logger.Error().Err(err).Msg("failed to write metric event")
					}
				}
			}
		}
	}

	return &collectormetrics.ExportMetricsServiceResponse{
		PartialSuccess: nil,
	}, nil
}

// Export implements the Traces service
func (r *OTLPReceiver) Export(ctx context.Context, req *collectortraces.ExportTracesServiceRequest) (*collectortraces.ExportTracesServiceResponse, error) {
	for _, resourceSpans := range req.GetResourceSpans() {
		for _, scopeSpans := range resourceSpans.GetScopeSpans() {
			for _, span := range scopeSpans.GetSpans() {
				event := r.spanToEvent(span)
				if err := r.eventRepo.WriteEvent(event); err != nil {
					r.logger.Logger.Error().Err(err).Msg("failed to write trace event")
				}
			}
		}
	}

	return &collectortraces.ExportTracesServiceResponse{
		PartialSuccess: nil,
	}, nil
}

// logRecordToEvent converts an OTLP LogRecord to a TaskEvent
func (r *OTLPReceiver) logRecordToEvent(logRecord *logspb.LogRecord) *TaskEvent {
	timestamp := time.Now()
	if logRecord.TimeUnixNano > 0 {
		timestamp = time.UnixMilli(int64(logRecord.TimeUnixNano / 1_000_000))
	}

	traceID := fmt.Sprintf("%032x", logRecord.TraceId)
	spanID := fmt.Sprintf("%016x", logRecord.SpanId)

	// Extract attributes
	attributes := attributesToMap(logRecord.GetAttributes())

	// Get severity
	severity := logRecord.GetSeverityText()

	// Get message body
	message := logRecord.GetBody().GetStringValue()

	return NewLogEvent(traceID, spanID, timestamp, message, severity, attributes)
}

// metricToEvents converts an OTLP Metric to TaskEvents
func (r *OTLPReceiver) metricToEvents(metric *metricspb.Metric) []*TaskEvent {
	events := make([]*TaskEvent, 0)

	// Handle different metric types
	switch data := metric.Data.(type) {
	case *metricspb.Metric_Sum:
		for _, dp := range data.Sum.GetDataPoints() {
			traceID := attributeValueToString(dp.GetAttributes(), "trace_id", "unknown")
			timestamp := time.UnixMilli(int64(dp.TimeUnixNano / 1_000_000))

			var value float64
			switch val := dp.GetValue().(type) {
			case *metricspb.NumberDataPoint_AsInt:
				value = float64(val.AsInt)
			case *metricspb.NumberDataPoint_AsDouble:
				value = val.AsDouble
			}

			attributes := attributesToMap(dp.GetAttributes())
			event := NewMetricEvent(traceID, timestamp, metric.GetName(), value, metric.GetUnit(), attributes)
			events = append(events, event)
		}

	case *metricspb.Metric_Gauge:
		for _, dp := range data.Gauge.GetDataPoints() {
			traceID := attributeValueToString(dp.GetAttributes(), "trace_id", "unknown")
			timestamp := time.UnixMilli(int64(dp.TimeUnixNano / 1_000_000))

			var value float64
			switch val := dp.GetValue().(type) {
			case *metricspb.NumberDataPoint_AsInt:
				value = float64(val.AsInt)
			case *metricspb.NumberDataPoint_AsDouble:
				value = val.AsDouble
			}

			attributes := attributesToMap(dp.GetAttributes())
			event := NewMetricEvent(traceID, timestamp, metric.GetName(), value, metric.GetUnit(), attributes)
			events = append(events, event)
		}

	case *metricspb.Metric_Histogram:
		for _, dp := range data.Histogram.GetDataPoints() {
			traceID := attributeValueToString(dp.GetAttributes(), "trace_id", "unknown")
			timestamp := time.UnixMilli(int64(dp.TimeUnixNano / 1_000_000))

			attributes := attributesToMap(dp.GetAttributes())
			event := NewMetricEvent(traceID, timestamp, metric.GetName()+"_count", float64(dp.GetCount()), metric.GetUnit(), attributes)
			events = append(events, event)
		}
	}

	return events
}

// spanToEvent converts an OTLP Span to a TaskEvent
func (r *OTLPReceiver) spanToEvent(span *tracespb.Span) *TaskEvent {
	traceID := fmt.Sprintf("%032x", span.TraceId)
	spanID := fmt.Sprintf("%016x", span.SpanId)

	timestamp := time.UnixMilli(int64(span.EndTimeUnixNano / 1_000_000))
	startTime := time.UnixMilli(int64(span.StartTimeUnixNano / 1_000_000))
	latencyMs := int64(timestamp.Sub(startTime).Milliseconds())

	status := "OK"
	if span.GetStatus() != nil {
		status = span.GetStatus().GetMessage()
		if status == "" {
			status = span.GetStatus().GetCode().String()
		}
	}

	attributes := attributesToMap(span.GetAttributes())

	return NewTraceEvent(traceID, spanID, span.GetName(), timestamp, latencyMs, status, attributes)
}

// attributesToMap converts OTLP attributes to a map
func attributesToMap(attrs []*commonpb.KeyValue) map[string]interface{} {
	result := make(map[string]interface{})
	for _, attr := range attrs {
		result[attr.GetKey()] = attributeValueToInterface(attr.GetValue())
	}
	return result
}

// attributeValueToInterface converts an OTLP attribute value to an interface
func attributeValueToInterface(av *commonpb.AnyValue) interface{} {
	if av == nil {
		return nil
	}

	switch v := av.Value.(type) {
	case *commonpb.AnyValue_StringValue:
		return v.StringValue
	case *commonpb.AnyValue_IntValue:
		return v.IntValue
	case *commonpb.AnyValue_DoubleValue:
		return v.DoubleValue
	case *commonpb.AnyValue_BoolValue:
		return v.BoolValue
	default:
		return nil
	}
}

// attributeValueToString extracts a string attribute value with fallback
func attributeValueToString(attrs []*commonpb.KeyValue, key string, fallback string) string {
	for _, attr := range attrs {
		if attr.GetKey() == key {
			if sv, ok := attr.GetValue().Value.(*commonpb.AnyValue_StringValue); ok {
				return sv.StringValue
			}
		}
	}
	return fallback
}

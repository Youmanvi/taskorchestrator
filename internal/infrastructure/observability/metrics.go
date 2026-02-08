package observability

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	OrchestrationStarted    prometheus.Counter
	OrchestrationCompleted  prometheus.Counter
	OrchestrationFailed     prometheus.Counter
	OrchestrationDuration   prometheus.Histogram
	ActivityExecutions      prometheus.Counter
	ActivityDuration        prometheus.Histogram
	ActivityErrors          prometheus.Counter
	CompensationExecutions  prometheus.Counter
	CompensationDuration    prometheus.Histogram
	CompensationErrors      prometheus.Counter
}

// NewMetrics creates a new metrics collector
func NewMetrics() *Metrics {
	return &Metrics{
		OrchestrationStarted: promauto.NewCounter(prometheus.CounterOpts{
			Name: "orchestration_started_total",
			Help: "Total number of orchestrations started",
		}),
		OrchestrationCompleted: promauto.NewCounter(prometheus.CounterOpts{
			Name: "orchestration_completed_total",
			Help: "Total number of orchestrations completed successfully",
		}),
		OrchestrationFailed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "orchestration_failed_total",
			Help: "Total number of orchestrations failed",
		}),
		OrchestrationDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name: "orchestration_duration_seconds",
			Help: "Orchestration execution duration in seconds",
			Buckets: []float64{.1, .5, 1, 2, 5, 10, 30, 60},
		}),
		ActivityExecutions: promauto.NewCounter(prometheus.CounterOpts{
			Name: "activity_executions_total",
			Help: "Total number of activity executions",
		}),
		ActivityDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name: "activity_duration_seconds",
			Help: "Activity execution duration in seconds",
			Buckets: []float64{.01, .05, .1, .5, 1, 5, 10},
		}),
		ActivityErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "activity_errors_total",
			Help: "Total number of activity errors",
		}),
		CompensationExecutions: promauto.NewCounter(prometheus.CounterOpts{
			Name: "compensation_executions_total",
			Help: "Total number of compensation executions",
		}),
		CompensationDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name: "compensation_duration_seconds",
			Help: "Compensation execution duration in seconds",
			Buckets: []float64{.01, .05, .1, .5, 1, 5},
		}),
		CompensationErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "compensation_errors_total",
			Help: "Total number of compensation errors",
		}),
	}
}

// RecordOrchestrationStart records orchestration start
func (m *Metrics) RecordOrchestrationStart() {
	m.OrchestrationStarted.Inc()
}

// RecordOrchestrationCompleted records successful orchestration completion
func (m *Metrics) RecordOrchestrationCompleted(duration time.Duration) {
	m.OrchestrationCompleted.Inc()
	m.OrchestrationDuration.Observe(duration.Seconds())
}

// RecordOrchestrationFailed records orchestration failure
func (m *Metrics) RecordOrchestrationFailed(duration time.Duration) {
	m.OrchestrationFailed.Inc()
	m.OrchestrationDuration.Observe(duration.Seconds())
}

// RecordActivityExecution records activity execution
func (m *Metrics) RecordActivityExecution(duration time.Duration, err error) {
	m.ActivityExecutions.Inc()
	m.ActivityDuration.Observe(duration.Seconds())
	if err != nil {
		m.ActivityErrors.Inc()
	}
}

// RecordCompensation records compensation execution
func (m *Metrics) RecordCompensation(duration time.Duration, err error) {
	m.CompensationExecutions.Inc()
	m.CompensationDuration.Observe(duration.Seconds())
	if err != nil {
		m.CompensationErrors.Inc()
	}
}

# Task Orchestrator - Production-Ready Durable Task Framework

A comprehensive implementation of a distributed task orchestration system in Go using the `durabletask-go` framework. This project demonstrates production-ready patterns for building fault-tolerant workflows with automatic compensation, retry logic, and comprehensive observability.

## Features

- **Domain-Driven Architecture**: Clean separation of concerns with domain models, activities, and workflows
- **Real-World Use Case**: Complete order processing pipeline demonstrating compensation patterns
- **Middleware Pattern**: Composable cross-cutting concerns (retry, timeout, circuit breaker, logging)
- **Error Classification**: Distinguish transient vs permanent errors for intelligent retry logic
- **OTLP-Based Observability**: OpenTelemetry Protocol receiver for persistent telemetry, queryable via SQL
- **SQLite Persistence**: All data stored persistently for audit trails and analysis
- **Comprehensive Testing**: Integration tests with temporary SQLite for deterministic testing
- **Graceful Shutdown**: Proper signal handling and resource cleanup
- **Horizontal Scaling**: Worker-only mode for distributed deployments

## Architecture

```
taskorchestrator/
├── cmd/
│   ├── orchestrator/      # Main app with client + worker
│   └── worker/            # Worker-only mode for scaling
├── internal/
│   ├── domain/            # Order, Payment, Inventory entities
│   ├── workflows/         # Orchestration logic
│   ├── activities/        # Activity implementations
│   ├── middleware/        # Retry, timeout, circuit breaker
│   ├── infrastructure/    # Config, logger, metrics, tracing, backend
│   └── pkg/errors/        # Custom error types
├── test/
│   ├── integration/       # End-to-end tests
│   └── fixtures/          # Test data
└── configs/               # Configuration files
```

## Order Processing Workflow

The main orchestration demonstrates a complete order processing pipeline:

1. **Check Availability** - Verify items are in stock
2. **Reserve Inventory** - Reserve items (tracked for compensation)
3. **Charge Payment** - Process payment (tracked for compensation)
4. **Send Confirmation** - Notify customer of successful order
5. **On Failure** - Automatically release inventory and compensate

```
Order Received
    ↓
Check Availability
    ├─ No → FAIL (send failure email)
    └─ Yes ↓
Reserve Inventory [saves reservation ID for compensation]
    ├─ Fail → COMPENSATE + FAIL
    └─ Success ↓
Charge Payment [saves payment ID for compensation]
    ├─ Fail → RELEASE INVENTORY + COMPENSATE + FAIL
    └─ Success ↓
Send Confirmation Email
    ↓
SUCCESS (order confirmed)
```

## Getting Started

### Prerequisites

- Go 1.22+
- Docker (optional, for Zipkin tracing)

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd taskorchestrator
```

2. Install dependencies:
```bash
go mod tidy
```

3. Build the application:
```bash
go build -o orchestrator ./cmd/orchestrator
```

### Running

#### Orchestrator (Client + Worker)

```bash
./orchestrator -config configs/dev.yaml
```

This will:
- Start the task hub worker
- Schedule a sample order
- Wait for completion
- Exit gracefully

#### Worker Mode (for distributed deployments)

```bash
go run ./cmd/worker -config configs/dev.yaml
```

This runs a standalone worker that processes orchestrations scheduled by clients on other machines.

### Configuration

Configuration can be set via:

1. **YAML file** (default: `configs/dev.yaml`):
```yaml
app:
  name: task-orchestrator
  port: 8080

backend:
  sqliteFile: data/orchestration.db  # Only SQLite is supported

observability:
  logLevel: debug
  tracingEnabled: true
  zipkinEndpoint: http://localhost:9411/api/v2/spans
```

2. **Environment variables** (override YAML):
```bash
APP_BACKEND_SQLITE_FILE=/var/log/orchestrator/execution.db
APP_LOG_LEVEL=info
APP_TRACING_ENABLED=true
```

## Development

### Running Tests

```bash
# Run all tests with coverage
go test -v -cover ./...

# Run integration tests only
go test -v ./test/integration/...

# Run with coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Observability

#### Metrics

Prometheus metrics are exported on `http://localhost:9090/metrics`:

```bash
curl http://localhost:9090/metrics | grep orchestration
```

Key metrics:
- `orchestration_started_total` - Orders started
- `orchestration_completed_total` - Orders completed successfully
- `orchestration_failed_total` - Orders failed
- `orchestration_duration_seconds` - Histogram of execution time
- `activity_executions_total` - Activity execution count
- `activity_errors_total` - Activity errors

#### Tracing with Zipkin

Start Zipkin:
```bash
docker run -d -p 9411:9411 openzipkin/zipkin
```

Enable tracing:
```bash
APP_TRACING_ENABLED=true ./orchestrator
```

Visit http://localhost:9411 to view traces.

#### Logging

Structured logging with zerolog outputs to stdout:

```json
{
  "level": "info",
  "activity": "payment:charge",
  "orchestration_id": "ORD-123",
  "duration_ms": 145,
  "message": "activity completed"
}
```

## Adding New Activities

1. Define input/output structs:
```go
type MyActivityInput struct {
    ID   string
    Data string
}

type MyActivityOutput struct {
    Result string
}
```

2. Implement the activity function:
```go
func MyActivity(ctx context.Context, input []byte) ([]byte, error) {
    var inp MyActivityInput
    json.Unmarshal(input, &inp)

    // Process...

    output := MyActivityOutput{Result: "ok"}
    return json.Marshal(output)
}
```

3. Register with middleware in `internal/activities/registry.go`:
```go
registerActivity(registry, "domain:action", MyActivity, deps)
```

4. Call from orchestrator:
```go
inputBytes, _ := json.Marshal(input)
result := ctx.CallActivity("domain:action", api.WithActivityInput(inputBytes))
var output MyActivityOutput
result.Await(&output)
```

## Adding New Workflows

1. Implement the orchestrator function:
```go
func MyWorkflow(ctx *api.OrchestrationContext, input []byte) ([]byte, error) {
    // Implement workflow steps
    // Call activities with ctx.CallActivity()
    // Handle compensation on failure
}
```

2. Register in `internal/workflows/registry.go`:
```go
registry.AddOrchestratorN("my_workflow", MyWorkflow)
```

3. Schedule from client:
```go
execution, err := client.ScheduleNewOrchestration(
    ctx,
    "my_workflow",
    api.WithInstanceID(id),
    api.WithInput(inputBytes),
)
```

## Error Handling

Errors are classified as:

- **Transient**: Network timeouts, temporary service unavailability
  - Automatically retried with exponential backoff
  - Example: `errors.NewTransientError(...)`

- **Permanent**: Invalid input, business logic errors
  - Not retried, fails immediately
  - Example: `errors.NewPermanentError(...)`

- **Timeout**: Activity execution exceeds timeout
  - Treated as transient and retried
  - Example: `errors.NewTimeoutError(...)`

Custom retry policy:
```go
policy := middleware.RetryPolicy{
    MaxAttempts:      5,
    InitialBackoff:   100 * time.Millisecond,
    MaxBackoff:       30 * time.Second,
    BackoffMultiplier: 2.0,
}
```

## Middleware

Activities are automatically wrapped with:

1. **Logging** - Log start/end with duration
2. **Timeout** - Configurable per-activity timeout
3. **Retry** - Exponential backoff for transient errors
4. **Circuit Breaker** - Prevent cascade failures

Middleware is composable and applied in order:
```go
ApplyMiddleware(activity,
    WithLogging(logger, name),
    WithTimeout(30*time.Second),
    WithRetry(logger, policy),
    WithCircuitBreaker(name, threshold, timeout),
)
```

## Production Deployment

### Scaling

Run multiple worker instances:

```bash
# Terminal 1: Client/scheduler
./orchestrator -config configs/prod.yaml

# Terminal 2+: Workers
./worker -config configs/prod.yaml
./worker -config configs/prod.yaml
./worker -config configs/prod.yaml
```

All workers process from the same backend database.

### Monitoring

1. **Health checks**:
```bash
curl http://localhost:8080/health
```

2. **Metrics collection**:
```bash
# Prometheus scrape config
scrape_configs:
  - job_name: 'task-orchestrator'
    static_configs:
      - targets: ['localhost:9090']
```

3. **Tracing**:
Connect Zipkin to see distributed traces across orchestrations.

### Database

SQLite (dev/testing):
```bash
sqlite3 data/orchestration.db
SELECT * FROM sqlite_master;
```

For production, use PostgreSQL backend (extend `internal/infrastructure/backend`).

## Common Patterns

### Parallel Activities

Use `task.WhenAll()`:
```go
results := ctx.WhenAll(
    ctx.CallActivity("activity:1", ...),
    ctx.CallActivity("activity:2", ...),
)
```

### Conditional Logic

```go
if condition {
    ctx.CallActivity("activity:a", ...)
} else {
    ctx.CallActivity("activity:b", ...)
}
```

### Compensation/Saga Pattern

Track compensation activities:
```go
// Forward: reserve inventory
reserveResult := ctx.CallActivity("inventory:reserve", ...)
reserveOutput := ...
reserveResult.Await(&reserveOutput)

// If later step fails, compensate
if paymentFailed {
    // Release inventory (compensation)
    ctx.CallActivity("inventory:release", ...)
}
```

### Retry with Specific Policy

Per-activity retry in the middleware composition:
```go
retryPolicy := middleware.RetryPolicy{
    MaxAttempts: 10,
    InitialBackoff: 50 * time.Millisecond,
}
```

## Troubleshooting

### Activities not running

1. Check worker is started
2. Verify activity is registered in registry
3. Check logs for errors

### Tracing not working

1. Start Zipkin: `docker run -d -p 9411:9411 openzipkin/zipkin`
2. Set `APP_TRACING_ENABLED=true`
3. Check logs for tracer initialization

### Slow execution

1. Check activity durations in metrics
2. Review retry counts
3. Consider increasing timeout if legitimate

## Testing

### Unit Tests

Test individual activities:
```go
func TestChargePayment(t *testing.T) {
    input := ChargePaymentInput{...}
    inputBytes, _ := json.Marshal(input)
    output, err := ChargePaymentActivity(mockGateway)(ctx, inputBytes)
    assert.NoError(t, err)
}
```

### Integration Tests

Test complete workflows:
```go
harness, _ := NewTestHarness()
harness.Start(ctx)
execution, _ := harness.ScheduleOrder(ctx, &input)
result, _ := harness.WaitForOrchestration(ctx, execution, 5*time.Second)
assert.True(t, result.IsSuccessful())
```

## Contributing

When adding features:

1. Follow the layered architecture
2. Add comprehensive error handling
3. Write integration tests
4. Update README with examples
5. Ensure observability (logging/metrics)

## License

MIT

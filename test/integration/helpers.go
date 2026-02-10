package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/microsoft/durabletask-go/api"
	"github.com/Youmanvi/taskorchestrator/internal/activities"
	"github.com/Youmanvi/taskorchestrator/internal/activities/inventory"
	"github.com/Youmanvi/taskorchestrator/internal/activities/notification"
	"github.com/Youmanvi/taskorchestrator/internal/activities/payment"
	"github.com/Youmanvi/taskorchestrator/internal/infrastructure/backend"
	"github.com/Youmanvi/taskorchestrator/internal/infrastructure/config"
	"github.com/Youmanvi/taskorchestrator/internal/infrastructure/observability"
	"github.com/Youmanvi/taskorchestrator/internal/middleware"
	"github.com/Youmanvi/taskorchestrator/internal/workflows"
)

// TestHarness provides utilities for integration testing
type TestHarness struct {
	Backend         api.Backend
	Client          api.TaskHubClient
	Worker          api.TaskHubWorker
	Logger          *observability.Logger
	Metrics         *observability.Metrics
	PaymentGateway  *payment.MockPaymentGateway
	InventoryMgr    *inventory.MockInventoryManager
	EmailService    *notification.MockEmailService
	DBFile          string
}

// NewTestHarness creates a new test harness with SQLite backend
func NewTestHarness() (*TestHarness, error) {
	// Create temporary SQLite database for testing
	dbFile := fmt.Sprintf("%s/test-orchestrator-%d.db", os.TempDir(), time.Now().UnixNano())

	cfg := &config.BackendConfig{
		SQLiteFile:    dbFile,
		MaxConnection: 10,
	}

	be, err := backend.NewSQLiteBackend(cfg)
	if err != nil {
		return nil, err
	}

	// Create logger
	logger := observability.NewLogger(&observability.ObservabilityConfig{
		LogLevel:  "debug",
		LogFormat: "text",
	})

	// Create metrics
	metrics := observability.NewMetrics()

	// Create mock dependencies
	paymentGateway := payment.NewMockPaymentGateway()
	inventoryMgr := inventory.NewMockInventoryManager()
	emailService := notification.NewMockEmailService()

	// Create activity dependencies
	activityDeps := &activities.ActivityDeps{
		Logger:          logger,
		Metrics:         metrics,
		PaymentGateway:  paymentGateway,
		InventoryMgr:    inventoryMgr,
		EmailService:    emailService,
		RetryPolicy:     middleware.DefaultRetryPolicy(3),
		TimeoutDuration: 30 * time.Second,
	}

	// Create registries
	activityRegistry := activities.NewActivityRegistry(activityDeps)
	workflowRegistry := workflows.NewWorkflowRegistry()

	// Create client and worker
	client, err := api.NewTaskHubClient(be)
	if err != nil {
		return nil, err
	}

	worker, err := api.NewTaskHubWorker(be, workflowRegistry, activityRegistry)
	if err != nil {
		return nil, err
	}

	return &TestHarness{
		Backend:        be,
		Client:         client,
		Worker:         worker,
		Logger:         logger,
		Metrics:        metrics,
		PaymentGateway: paymentGateway,
		InventoryMgr:   inventoryMgr,
		EmailService:   emailService,
		DBFile:         dbFile,
	}, nil
}

// Start starts the worker
func (h *TestHarness) Start(ctx context.Context) error {
	go h.Worker.Start(ctx)
	// Give worker time to start
	time.Sleep(100 * time.Millisecond)
	return nil
}

// Stop stops the worker and cleans up temporary database
func (h *TestHarness) Stop(ctx context.Context) error {
	err := h.Worker.Stop(ctx)
	// Clean up temporary database file
	os.Remove(h.DBFile)
	return err
}

// ScheduleOrder schedules an order processing orchestration
func (h *TestHarness) ScheduleOrder(ctx context.Context, input *workflows.OrderProcessingInput) (api.OrchestrationExecution, error) {
	inputBytes, _ := json.Marshal(input)
	return h.Client.ScheduleNewOrchestration(
		ctx,
		"order_processing",
		api.WithInstanceID(input.Order.ID),
		api.WithInput(inputBytes),
	)
}

// WaitForOrchestration waits for an orchestration to complete
func (h *TestHarness) WaitForOrchestration(ctx context.Context, execution api.OrchestrationExecution, timeout time.Duration) (*api.OrchestrationExecutionResult, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return execution.WaitForCompletion(ctx)
}

// GetOrderOutput parses the orchestration output as OrderProcessingOutput
func GetOrderOutput(result *api.OrchestrationExecutionResult) (*workflows.OrderProcessingOutput, error) {
	var output workflows.OrderProcessingOutput
	if result.Output != nil {
		if err := json.Unmarshal(result.Output, &output); err != nil {
			return nil, err
		}
	}
	return &output, nil
}

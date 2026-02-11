package activities

import (
	"time"

	"github.com/microsoft/durabletask-go/task"
	"github.com/Youmanvi/taskorchestrator/internal/activities/inventory"
	"github.com/Youmanvi/taskorchestrator/internal/activities/notification"
	"github.com/Youmanvi/taskorchestrator/internal/activities/payment"
	"github.com/Youmanvi/taskorchestrator/internal/infrastructure/observability"
	"github.com/Youmanvi/taskorchestrator/internal/middleware"
)

// ActivityDeps contains dependencies for all activities
type ActivityDeps struct {
	Logger          *observability.Logger
	Metrics         *observability.Metrics
	PaymentGateway  payment.PaymentGateway
	InventoryMgr    inventory.InventoryManager
	EmailService    notification.EmailService
	RetryPolicy     middleware.RetryPolicy
	TimeoutDuration time.Duration
}

// NewActivityRegistry creates and registers all activities with middleware
func NewActivityRegistry(deps *ActivityDeps) *task.TaskRegistry {
	registry := task.NewTaskRegistry()

	// Payment activities
	registerActivity(registry, "payment:charge",
		payment.ChargePaymentActivity(deps.PaymentGateway),
		deps,
	)
	registerActivity(registry, "payment:refund",
		payment.RefundPaymentActivity(deps.PaymentGateway),
		deps,
	)
	registerActivity(registry, "payment:verify",
		payment.VerifyPaymentActivity(deps.PaymentGateway),
		deps,
	)

	// Inventory activities
	registerActivity(registry, "inventory:reserve",
		inventory.ReserveInventoryActivity(deps.InventoryMgr),
		deps,
	)
	registerActivity(registry, "inventory:release",
		inventory.ReleaseInventoryActivity(deps.InventoryMgr),
		deps,
	)
	registerActivity(registry, "inventory:check",
		inventory.CheckAvailabilityActivity(deps.InventoryMgr),
		deps,
	)

	// Notification activities
	registerActivity(registry, "notification:order_confirmation",
		notification.SendOrderConfirmationActivity(deps.EmailService),
		deps,
	)
	registerActivity(registry, "notification:order_failure",
		notification.SendOrderFailureActivity(deps.EmailService),
		deps,
	)
	registerActivity(registry, "notification:refund",
		notification.SendRefundNotificationActivity(deps.EmailService),
		deps,
	)

	return registry
}

// registerActivity registers an activity with middleware
func registerActivity(registry *task.TaskRegistry, name string, activity middleware.ActivityFunc, deps *ActivityDeps) {
	// Apply middleware chain (order matters - innermost to outermost)
	wrapped := middleware.ApplyMiddleware(
		activity,
		middleware.WithLogging(deps.Logger, name),
		middleware.WithTimeout(deps.TimeoutDuration),
		// gRPC error handling BEFORE retry so transient errors are classified correctly
		middleware.WithGRPCErrorHandling(),
		middleware.WithRetry(deps.Logger, deps.RetryPolicy),
	)

	// Adapt middleware.ActivityFunc to task.Activity
	taskActivity := func(ctx task.ActivityContext) (any, error) {
		// Serialize input
		var input []byte
		if err := ctx.GetInput(&input); err != nil {
			return nil, err
		}

		// Call the middleware-wrapped activity
		output, err := wrapped(ctx.Context(), input)
		return output, err
	}

	registry.AddActivityN(name, taskActivity)
}

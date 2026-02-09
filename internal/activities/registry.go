package activities

import (
	"time"

	"github.com/microsoft/durabletask-go/api"
	"github.com/vihan/taskorchestrator/internal/activities/inventory"
	"github.com/vihan/taskorchestrator/internal/activities/notification"
	"github.com/vihan/taskorchestrator/internal/activities/payment"
	"github.com/vihan/taskorchestrator/internal/infrastructure/observability"
	"github.com/vihan/taskorchestrator/internal/middleware"
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
func NewActivityRegistry(deps *ActivityDeps) *api.TaskActivityRegistry {
	registry := api.NewTaskActivityRegistry()

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
func registerActivity(registry *api.TaskActivityRegistry, name string, activity middleware.ActivityFunc, deps *ActivityDeps) {
	// Apply middleware chain (order matters - innermost to outermost)
	wrapped := middleware.ApplyMiddleware(
		activity,
		middleware.WithLogging(deps.Logger, name),
		middleware.WithTimeout(deps.TimeoutDuration),
		// gRPC error handling BEFORE retry so transient errors are classified correctly
		middleware.WithGRPCErrorHandling(),
		middleware.WithRetry(deps.Logger, deps.RetryPolicy),
	)

	registry.AddActivityN(name, wrapped)
}

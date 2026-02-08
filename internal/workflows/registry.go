package workflows

import (
	"github.com/microsoft/durabletask-go/api"
)

// NewWorkflowRegistry creates and registers all workflow orchestrators
func NewWorkflowRegistry() *api.TaskOrchestratorRegistry {
	registry := api.NewTaskOrchestratorRegistry()

	registry.AddOrchestratorN("order_processing", OrderProcessingOrchestrator)

	return registry
}

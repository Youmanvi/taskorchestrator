package workflows

import (
	"github.com/microsoft/durabletask-go/task"
)

// NewWorkflowRegistry creates and registers all workflow orchestrators
func NewWorkflowRegistry() *task.TaskRegistry {
	registry := task.NewTaskRegistry()

	registry.AddOrchestratorN("order_processing", OrderProcessingOrchestrator)

	return registry
}

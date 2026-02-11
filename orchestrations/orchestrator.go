package orchestrations

import (
	"fmt"

	"github.com/microsoft/durabletask-go/task"

	"github.com/Youmanvi/taskorchestrator/internal/activities"
)

// OrchestrationInput represents input to the orchestrator
type OrchestrationInput struct {
	Items []activities.ItemInput `json:"items"`
}

// OrchestrationOutput represents output from the orchestrator
type OrchestrationOutput struct {
	TotalItems   int           `json:"total_items"`
	SuccessCount int           `json:"success_count"`
	FailureCount int           `json:"failure_count"`
	Results      []interface{} `json:"results"`
	Errors       []string      `json:"errors"`
}

// SequenceOrchestrator orchestrates a sequence of activities
// for processing items. This is a template that can be customized
// for different business workflows.
//
// The default sequence is:
// 1. Validate input
// 2. Process items sequentially
// 3. Transform results
// 4. Return aggregated output
//
// Customization points:
// - Modify the activity sequence
// - Add conditional branching based on results
// - Change parallelization strategy (parallel vs sequential)
// - Add retry logic for failed activities
func SequenceOrchestrator(ctx *task.OrchestrationContext) (interface{}, error) {
	var input OrchestrationInput
	if err := ctx.GetInput(&input); err != nil {
		return nil, fmt.Errorf("failed to deserialize orchestration input: %w", err)
	}

	output := OrchestrationOutput{
		TotalItems: len(input.Items),
		Results:    []interface{}{},
		Errors:     []string{},
	}

	// Step 1: Validate input
	validationInput := activities.ValidationInput{
		Data: map[string]interface{}{
			"count": len(input.Items),
		},
	}

	var validationResult activities.ValidationResult
	if err := ctx.CallActivity("ValidateInputActivity", task.WithActivityInput(validationInput)).Await(&validationResult); err != nil {
		return nil, fmt.Errorf("validation activity failed: %w", err)
	}

	if !validationResult.Valid {
		return OrchestrationOutput{
			TotalItems:   output.TotalItems,
			SuccessCount: 0,
			FailureCount: output.TotalItems,
			Errors:       validationResult.Errors,
		}, nil
	}

	// Step 2: Process items sequentially
	// Template: Can be customized for parallel processing using ctx.WhenAll()
	processedItems := []interface{}{}
	for _, item := range input.Items {
		var itemResult activities.ItemResult
		if err := ctx.CallActivity("ProcessItemActivity", task.WithActivityInput(item)).Await(&itemResult); err != nil {
			output.FailureCount++
			output.Errors = append(output.Errors, fmt.Sprintf("failed to process item %s: %v", item.ID, err))
			continue
		}

		if !itemResult.Success {
			output.FailureCount++
			output.Errors = append(output.Errors, fmt.Sprintf("item %s processing failed: %s", item.ID, itemResult.Error))
			continue
		}

		output.SuccessCount++
		processedItems = append(processedItems, itemResult.Result)
	}

	// Step 3: Transform results
	transformInput := activities.TransformInput{
		Data: map[string]interface{}{
			"processed": true,
			"count":     output.SuccessCount,
		},
	}

	var transformResult activities.TransformResult
	if err := ctx.CallActivity("TransformDataActivity", task.WithActivityInput(transformInput)).Await(&transformResult); err != nil {
		return nil, fmt.Errorf("transform activity failed: %w", err)
	}

	// Step 4: Aggregate and return output
	output.Results = processedItems

	return output, nil
}

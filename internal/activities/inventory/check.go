package inventory

import (
	"context"
	"encoding/json"

	"github.com/Youmanvi/taskorchestrator/internal/domain"
	"github.com/Youmanvi/taskorchestrator/internal/pkg/errors"
)

// CheckAvailabilityInput is the input for checking inventory availability
type CheckAvailabilityInput struct {
	Items []domain.OrderItem
}

// CheckAvailabilityOutput is the output of checking availability
type CheckAvailabilityOutput struct {
	Available           bool
	UnavailableItems    []string
}

// CheckAvailabilityActivity checks if inventory is available for items
func CheckAvailabilityActivity(manager InventoryManager) func(ctx context.Context, input []byte) ([]byte, error) {
	return func(ctx context.Context, input []byte) ([]byte, error) {
		var inp CheckAvailabilityInput
		if err := json.Unmarshal(input, &inp); err != nil {
			return nil, errors.NewPermanentError("INVALID_INPUT", "failed to unmarshal check input", err)
		}

		// For demo purposes, all items are available
		// In a real implementation, this would check a database
		output := CheckAvailabilityOutput{
			Available:        len(inp.Items) > 0,
			UnavailableItems: []string{},
		}

		result, err := json.Marshal(output)
		if err != nil {
			return nil, errors.NewPermanentError("SERIALIZATION_ERROR", "failed to marshal check output", err)
		}

		return result, nil
	}
}

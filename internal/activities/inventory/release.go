package inventory

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Youmanvi/taskorchestrator/internal/pkg/errors"
)

// ReleaseInventoryInput is the input for releasing inventory
type ReleaseInventoryInput struct {
	ReservationID string
}

// ReleaseInventoryOutput is the output of releasing inventory
type ReleaseInventoryOutput struct {
	Status string
}

// ReleaseInventoryActivity releases a reserved inventory
func ReleaseInventoryActivity(manager InventoryManager) func(ctx context.Context, input []byte) ([]byte, error) {
	return func(ctx context.Context, input []byte) ([]byte, error) {
		var inp ReleaseInventoryInput
		if err := json.Unmarshal(input, &inp); err != nil {
			return nil, errors.NewPermanentError("INVALID_INPUT", "failed to unmarshal release input", err)
		}

		if inp.ReservationID == "" {
			return nil, errors.NewPermanentError("MISSING_RESERVATION_ID", "reservation ID is required", nil)
		}

		if err := manager.Release(ctx, inp.ReservationID); err != nil {
			return nil, errors.NewTransientError("RELEASE_FAILED", fmt.Sprintf("failed to release inventory: %v", err), err)
		}

		output := ReleaseInventoryOutput{
			Status: "released",
		}

		result, err := json.Marshal(output)
		if err != nil {
			return nil, errors.NewPermanentError("SERIALIZATION_ERROR", "failed to marshal release output", err)
		}

		return result, nil
	}
}

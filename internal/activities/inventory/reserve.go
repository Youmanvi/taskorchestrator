package inventory

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/vihan/taskorchestrator/internal/domain"
	"github.com/vihan/taskorchestrator/internal/pkg/errors"
)

// ReserveInventoryInput is the input for reserving inventory
type ReserveInventoryInput struct {
	OrderID string
	Items   []domain.OrderItem
}

// ReserveInventoryOutput is the output of reserving inventory
type ReserveInventoryOutput struct {
	ReservationID string
	Status        string
}

// InventoryManager manages inventory reservations
type InventoryManager interface {
	Reserve(ctx context.Context, orderID string, items []domain.OrderItem) (string, error)
	Release(ctx context.Context, reservationID string) error
}

// ReserveInventoryActivity reserves inventory for an order
func ReserveInventoryActivity(manager InventoryManager) func(ctx context.Context, input []byte) ([]byte, error) {
	return func(ctx context.Context, input []byte) ([]byte, error) {
		var inp ReserveInventoryInput
		if err := json.Unmarshal(input, &inp); err != nil {
			return nil, errors.NewPermanentError("INVALID_INPUT", "failed to unmarshal reserve input", err)
		}

		if inp.OrderID == "" {
			return nil, errors.NewPermanentError("MISSING_ORDER_ID", "order ID is required", nil)
		}

		if len(inp.Items) == 0 {
			return nil, errors.NewPermanentError("EMPTY_ITEMS", "items list cannot be empty", nil)
		}

		reservationID, err := manager.Reserve(ctx, inp.OrderID, inp.Items)
		if err != nil {
			// Classify error
			return nil, errors.NewPermanentError("RESERVATION_FAILED", fmt.Sprintf("failed to reserve inventory: %v", err), err)
		}

		output := ReserveInventoryOutput{
			ReservationID: reservationID,
			Status:        "reserved",
		}

		result, err := json.Marshal(output)
		if err != nil {
			return nil, errors.NewPermanentError("SERIALIZATION_ERROR", "failed to marshal reserve output", err)
		}

		return result, nil
	}
}

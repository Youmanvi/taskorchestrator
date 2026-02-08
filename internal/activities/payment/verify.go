package payment

import (
	"context"
	"encoding/json"

	"github.com/shopspring/decimal"
	"github.com/vihan/taskorchestrator/internal/pkg/errors"
)

// VerifyPaymentInput is the input for verifying a payment
type VerifyPaymentInput struct {
	PaymentID string
}

// VerifyPaymentOutput is the output of verifying a payment
type VerifyPaymentOutput struct {
	PaymentID string
	Status    string
	Amount    decimal.Decimal
}

// VerifyPaymentActivity verifies the status of a payment
func VerifyPaymentActivity(gateway PaymentGateway) func(ctx context.Context, input []byte) ([]byte, error) {
	return func(ctx context.Context, input []byte) ([]byte, error) {
		var inp VerifyPaymentInput
		if err := json.Unmarshal(input, &inp); err != nil {
			return nil, errors.NewPermanentError("INVALID_INPUT", "failed to unmarshal verify input", err)
		}

		if inp.PaymentID == "" {
			return nil, errors.NewPermanentError("MISSING_PAYMENT_ID", "payment ID is required", nil)
		}

		// Simulate verification
		output := VerifyPaymentOutput{
			PaymentID: inp.PaymentID,
			Status:    "completed",
			Amount:    decimal.Zero,
		}

		result, err := json.Marshal(output)
		if err != nil {
			return nil, errors.NewPermanentError("SERIALIZATION_ERROR", "failed to marshal verify output", err)
		}

		return result, nil
	}
}

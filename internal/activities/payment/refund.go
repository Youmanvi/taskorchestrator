package payment

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/Youmanvi/taskorchestrator/internal/pkg/errors"
)

// RefundPaymentInput is the input for refunding a payment
type RefundPaymentInput struct {
	PaymentID string
	Amount    decimal.Decimal
}

// RefundPaymentOutput is the output of refunding a payment
type RefundPaymentOutput struct {
	RefundID string
	Status   string
}

// RefundPaymentActivity refunds a previously charged payment
func RefundPaymentActivity(gateway PaymentGateway) func(ctx context.Context, input []byte) ([]byte, error) {
	return func(ctx context.Context, input []byte) ([]byte, error) {
		var inp RefundPaymentInput
		if err := json.Unmarshal(input, &inp); err != nil {
			return nil, errors.NewPermanentError("INVALID_INPUT", "failed to unmarshal refund input", err)
		}

		if inp.PaymentID == "" {
			return nil, errors.NewPermanentError("MISSING_PAYMENT_ID", "payment ID is required", nil)
		}

		// Simulate refund processing
		refundID := fmt.Sprintf("REFUND_%s", inp.PaymentID)

		output := RefundPaymentOutput{
			RefundID: refundID,
			Status:   "completed",
		}

		result, err := json.Marshal(output)
		if err != nil {
			return nil, errors.NewPermanentError("SERIALIZATION_ERROR", "failed to marshal refund output", err)
		}

		return result, nil
	}
}

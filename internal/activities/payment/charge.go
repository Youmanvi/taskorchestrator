package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/shopspring/decimal"
	"github.com/vihan/taskorchestrator/internal/domain"
	"github.com/vihan/taskorchestrator/internal/pkg/errors"
)

// ChargePaymentInput is the input for charging a payment
type ChargePaymentInput struct {
	OrderID       string
	Amount        decimal.Decimal
	PaymentMethod domain.PaymentMethod
	CustomerID    string
}

// ChargePaymentOutput is the output of charging a payment
type ChargePaymentOutput struct {
	PaymentID     string
	TransactionID string
	Status        string
}

// PaymentGateway simulates an external payment processor
type PaymentGateway interface {
	Charge(ctx context.Context, amount decimal.Decimal, method domain.PaymentMethod) (string, error)
}

// ChargePaymentActivity charges a payment for an order
func ChargePaymentActivity(gateway PaymentGateway) func(ctx context.Context, input []byte) ([]byte, error) {
	return func(ctx context.Context, input []byte) ([]byte, error) {
		var inp ChargePaymentInput
		if err := json.Unmarshal(input, &inp); err != nil {
			return nil, errors.NewPermanentError("INVALID_INPUT", "failed to unmarshal payment input", err)
		}

		// Simulate occasional payment gateway failures
		if rand.Float64() < 0.1 { // 10% chance of transient failure
			return nil, errors.NewTransientError(
				"PAYMENT_GATEWAY_UNAVAILABLE",
				"payment gateway temporarily unavailable",
				fmt.Errorf("network timeout"),
			)
		}

		transactionID, err := gateway.Charge(ctx, inp.Amount, inp.PaymentMethod)
		if err != nil {
			// Classify error based on type
			return nil, errors.NewTransientError(
				"PAYMENT_PROCESSING_ERROR",
				fmt.Sprintf("failed to process payment: %v", err),
				err,
			)
		}

		output := ChargePaymentOutput{
			PaymentID:     fmt.Sprintf("PAY_%s", inp.OrderID),
			TransactionID: transactionID,
			Status:        "completed",
		}

		result, err := json.Marshal(output)
		if err != nil {
			return nil, errors.NewPermanentError("SERIALIZATION_ERROR", "failed to marshal payment output", err)
		}

		return result, nil
	}
}

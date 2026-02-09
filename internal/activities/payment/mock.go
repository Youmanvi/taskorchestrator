package payment

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/Youmanvi/taskorchestrator/internal/domain"
)

// MockPaymentGateway is a mock implementation of PaymentGateway for testing
type MockPaymentGateway struct {
	transactions map[string]decimal.Decimal
}

// NewMockPaymentGateway creates a new mock payment gateway
func NewMockPaymentGateway() *MockPaymentGateway {
	return &MockPaymentGateway{
		transactions: make(map[string]decimal.Decimal),
	}
}

// Charge simulates charging a payment
func (m *MockPaymentGateway) Charge(ctx context.Context, amount decimal.Decimal, method domain.PaymentMethod) (string, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return "", fmt.Errorf("invalid amount")
	}

	transactionID := fmt.Sprintf("TXN_%d", len(m.transactions)+1)
	m.transactions[transactionID] = amount

	return transactionID, nil
}

// GetTransaction retrieves a transaction
func (m *MockPaymentGateway) GetTransaction(txnID string) (decimal.Decimal, bool) {
	amount, exists := m.transactions[txnID]
	return amount, exists
}

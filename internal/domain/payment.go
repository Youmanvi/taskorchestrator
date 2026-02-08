package domain

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// PaymentMethod represents the payment method
type PaymentMethod string

const (
	PaymentMethodCard   PaymentMethod = "card"
	PaymentMethodBank   PaymentMethod = "bank"
	PaymentMethodWallet PaymentMethod = "wallet"
)

// PaymentStatus represents the status of a payment
type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusProcessing PaymentStatus = "processing"
	PaymentStatusCompleted  PaymentStatus = "completed"
	PaymentStatusFailed     PaymentStatus = "failed"
	PaymentStatusRefunded   PaymentStatus = "refunded"
)

// Payment represents a payment transaction
type Payment struct {
	ID              string
	OrderID         string
	Amount          decimal.Decimal
	Method          PaymentMethod
	Status          PaymentStatus
	TransactionID   string
	FailureReason   string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	ProcessingError error `json:"-"`
}

// NewPayment creates a new payment
func NewPayment(id, orderID string, amount decimal.Decimal, method PaymentMethod) (*Payment, error) {
	if id == "" {
		return nil, fmt.Errorf("payment ID cannot be empty")
	}
	if orderID == "" {
		return nil, fmt.Errorf("order ID cannot be empty")
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("amount must be greater than zero")
	}

	now := time.Now()
	return &Payment{
		ID:        id,
		OrderID:   orderID,
		Amount:    amount,
		Method:    method,
		Status:    PaymentStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// MarkProcessing marks the payment as processing
func (p *Payment) MarkProcessing() {
	p.Status = PaymentStatusProcessing
	p.UpdatedAt = time.Now()
}

// MarkCompleted marks the payment as completed with transaction ID
func (p *Payment) MarkCompleted(transactionID string) {
	p.Status = PaymentStatusCompleted
	p.TransactionID = transactionID
	p.UpdatedAt = time.Now()
}

// MarkFailed marks the payment as failed with reason
func (p *Payment) MarkFailed(reason string, err error) {
	p.Status = PaymentStatusFailed
	p.FailureReason = reason
	p.ProcessingError = err
	p.UpdatedAt = time.Now()
}

// MarkRefunded marks the payment as refunded
func (p *Payment) MarkRefunded() {
	p.Status = PaymentStatusRefunded
	p.UpdatedAt = time.Now()
}

// CanBeRefunded checks if payment can be refunded
func (p *Payment) CanBeRefunded() bool {
	return p.Status == PaymentStatusCompleted
}

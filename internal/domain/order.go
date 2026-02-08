package domain

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// OrderStatus represents the status of an order
type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusConfirmed  OrderStatus = "confirmed"
	OrderStatusFailed     OrderStatus = "failed"
	OrderStatusRefunded   OrderStatus = "refunded"
	OrderStatusCancelled  OrderStatus = "cancelled"
)

// OrderItem represents a single item in an order
type OrderItem struct {
	SKU      string
	Quantity int32
	Price    decimal.Decimal
}

// Order represents a customer order
type Order struct {
	ID            string
	CustomerID    string
	Items         []OrderItem
	TotalAmount   decimal.Decimal
	Status        OrderStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
	PaymentID     string
	ReservationID string
	FailureReason string
}

// NewOrder creates a new order
func NewOrder(id, customerID string, items []OrderItem) (*Order, error) {
	if id == "" {
		return nil, fmt.Errorf("order ID cannot be empty")
	}
	if customerID == "" {
		return nil, fmt.Errorf("customer ID cannot be empty")
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("order must contain at least one item")
	}

	// Calculate total amount
	total := decimal.NewFromInt(0)
	for _, item := range items {
		if item.Quantity <= 0 {
			return nil, fmt.Errorf("quantity must be greater than zero for SKU %s", item.SKU)
		}
		lineTotal := item.Price.Mul(decimal.NewFromInt(int64(item.Quantity)))
		total = total.Add(lineTotal)
	}

	now := time.Now()
	return &Order{
		ID:         id,
		CustomerID: customerID,
		Items:      items,
		TotalAmount: total,
		Status:     OrderStatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

// IsValid validates the order state
func (o *Order) IsValid() error {
	if o.ID == "" {
		return fmt.Errorf("order ID cannot be empty")
	}
	if o.CustomerID == "" {
		return fmt.Errorf("customer ID cannot be empty")
	}
	if len(o.Items) == 0 {
		return fmt.Errorf("order must contain at least one item")
	}
	if o.TotalAmount.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("total amount must be greater than zero")
	}
	return nil
}

// MarkConfirmed marks the order as confirmed
func (o *Order) MarkConfirmed(paymentID, reservationID string) {
	o.Status = OrderStatusConfirmed
	o.PaymentID = paymentID
	o.ReservationID = reservationID
	o.UpdatedAt = time.Now()
}

// MarkFailed marks the order as failed
func (o *Order) MarkFailed(reason string) {
	o.Status = OrderStatusFailed
	o.FailureReason = reason
	o.UpdatedAt = time.Now()
}

// MarkRefunded marks the order as refunded
func (o *Order) MarkRefunded() {
	o.Status = OrderStatusRefunded
	o.UpdatedAt = time.Now()
}

// CanBeConfirmed checks if order can be confirmed
func (o *Order) CanBeConfirmed() bool {
	return o.Status == OrderStatusPending
}

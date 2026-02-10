package fixtures

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/Youmanvi/taskorchestrator/internal/domain"
)

// CreateValidOrder creates a valid test order
func CreateValidOrder() domain.Order {
	items := []domain.OrderItem{
		{
			SKU:      "ITEM-001",
			Quantity: 2,
			Price:    decimal.NewFromInt(29) + decimal.NewFromFloat(0.99),
		},
		{
			SKU:      "ITEM-002",
			Quantity: 1,
			Price:    decimal.NewFromInt(49) + decimal.NewFromFloat(0.99),
		},
	}

	order, _ := domain.NewOrder(
		fmt.Sprintf("ORD-%d", time.Now().UnixNano()),
		"CUST-12345",
		items,
	)

	return *order
}

// CreateOrderWithItems creates an order with specific items
func CreateOrderWithItems(customerID string, items []domain.OrderItem) domain.Order {
	order, _ := domain.NewOrder(
		fmt.Sprintf("ORD-%d", time.Now().UnixNano()),
		customerID,
		items,
	)

	return *order
}

// CreateSingleItemOrder creates an order with a single item
func CreateSingleItemOrder() domain.Order {
	items := []domain.OrderItem{
		{
			SKU:      "ITEM-001",
			Quantity: 1,
			Price:    decimal.NewFromInt(99) + decimal.NewFromFloat(0.99),
		},
	}

	order, _ := domain.NewOrder(
		fmt.Sprintf("ORD-%d", time.Now().UnixNano()),
		"CUST-12345",
		items,
	)

	return *order
}

// CreateLargeOrder creates an order with multiple items
func CreateLargeOrder() domain.Order {
	items := []domain.OrderItem{
		{
			SKU:      "ITEM-001",
			Quantity: 5,
			Price:    decimal.NewFromInt(25),
		},
		{
			SKU:      "ITEM-002",
			Quantity: 3,
			Price:    decimal.NewFromInt(50),
		},
		{
			SKU:      "ITEM-003",
			Quantity: 2,
			Price:    decimal.NewFromInt(75),
		},
	}

	order, _ := domain.NewOrder(
		fmt.Sprintf("ORD-%d", time.Now().UnixNano()),
		"CUST-12345",
		items,
	)

	return *order
}

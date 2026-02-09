package inventory

import (
	"context"
	"fmt"
	"sync"

	"github.com/Youmanvi/taskorchestrator/internal/domain"
)

// MockInventoryManager is a mock implementation of InventoryManager for testing
type MockInventoryManager struct {
	mu           sync.RWMutex
	reservations map[string]*domain.InventoryReservation
}

// NewMockInventoryManager creates a new mock inventory manager
func NewMockInventoryManager() *MockInventoryManager {
	return &MockInventoryManager{
		reservations: make(map[string]*domain.InventoryReservation),
	}
}

// Reserve simulates reserving inventory
func (m *MockInventoryManager) Reserve(ctx context.Context, orderID string, items []domain.OrderItem) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	reservationID := fmt.Sprintf("RES_%s", orderID)

	// Convert OrderItems to ReservedItems
	reservedItems := make([]domain.ReservedItem, len(items))
	for i, item := range items {
		reservedItems[i] = domain.ReservedItem{
			SKU:      item.SKU,
			Quantity: item.Quantity,
		}
	}

	res, err := domain.NewInventoryReservation(reservationID, orderID, reservedItems)
	if err != nil {
		return "", err
	}

	m.reservations[reservationID] = res
	return reservationID, nil
}

// Release simulates releasing a reservation
func (m *MockInventoryManager) Release(ctx context.Context, reservationID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	res, exists := m.reservations[reservationID]
	if !exists {
		return fmt.Errorf("reservation not found: %s", reservationID)
	}

	res.MarkReleased()
	return nil
}

// GetReservation retrieves a reservation
func (m *MockInventoryManager) GetReservation(reservationID string) (*domain.InventoryReservation, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	res, exists := m.reservations[reservationID]
	return res, exists
}

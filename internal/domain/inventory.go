package domain

import (
	"fmt"
	"time"
)

// InventoryReservation represents a reservation of inventory items
type InventoryReservation struct {
	ID        string
	OrderID   string
	Items     []ReservedItem
	Status    ReservationStatus
	CreatedAt time.Time
	ExpiresAt time.Time
}

// ReservedItem represents a reserved inventory item
type ReservedItem struct {
	SKU      string
	Quantity int32
}

// ReservationStatus represents the status of a reservation
type ReservationStatus string

const (
	ReservationStatusActive   ReservationStatus = "active"
	ReservationStatusReleased ReservationStatus = "released"
	ReservationStatusExpired  ReservationStatus = "expired"
)

// NewInventoryReservation creates a new inventory reservation
func NewInventoryReservation(id, orderID string, items []ReservedItem) (*InventoryReservation, error) {
	if id == "" {
		return nil, fmt.Errorf("reservation ID cannot be empty")
	}
	if orderID == "" {
		return nil, fmt.Errorf("order ID cannot be empty")
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("reservation must contain at least one item")
	}

	now := time.Now()
	return &InventoryReservation{
		ID:        id,
		OrderID:   orderID,
		Items:     items,
		Status:    ReservationStatusActive,
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour), // 24 hour expiration
	}, nil
}

// MarkReleased marks the reservation as released
func (r *InventoryReservation) MarkReleased() {
	r.Status = ReservationStatusReleased
}

// MarkExpired marks the reservation as expired
func (r *InventoryReservation) MarkExpired() {
	r.Status = ReservationStatusExpired
}

// IsExpired checks if the reservation has expired
func (r *InventoryReservation) IsExpired() bool {
	return time.Now().After(r.ExpiresAt) || r.Status == ReservationStatusExpired
}

// IsActive checks if the reservation is active
func (r *InventoryReservation) IsActive() bool {
	return r.Status == ReservationStatusActive && !r.IsExpired()
}

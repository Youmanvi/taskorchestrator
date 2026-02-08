package backend

import (
	"github.com/microsoft/durabletask-go/backend"
	"github.com/microsoft/durabletask-go/backend/memory"
)

// NewInMemoryBackend creates a new in-memory backend for testing
func NewInMemoryBackend() backend.Backend {
	return memory.NewMemoryBackend()
}

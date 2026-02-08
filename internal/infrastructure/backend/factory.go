package backend

import (
	"fmt"
	"strings"

	"github.com/microsoft/durabletask-go/backend"
	"github.com/vihan/taskorchestrator/internal/infrastructure/config"
)

// NewBackend creates a new backend based on configuration
func NewBackend(cfg *config.BackendConfig) (backend.Backend, error) {
	backendType := strings.ToLower(cfg.Type)

	switch backendType {
	case "sqlite":
		return NewSQLiteBackend(cfg)
	case "memory":
		return NewInMemoryBackend(), nil
	default:
		return nil, fmt.Errorf("unsupported backend type: %s", cfg.Type)
	}
}

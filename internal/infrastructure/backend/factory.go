package backend

import (
	"github.com/microsoft/durabletask-go/backend"
	"github.com/vihan/taskorchestrator/internal/infrastructure/config"
)

// NewBackend creates a new SQLite backend
// In-memory backends are not supported as all analysis requires persistent SQLite storage
func NewBackend(cfg *config.BackendConfig) (backend.Backend, error) {
	return NewSQLiteBackend(cfg)
}

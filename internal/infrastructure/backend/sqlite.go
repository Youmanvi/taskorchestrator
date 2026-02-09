package backend

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/microsoft/durabletask-go/backend"
	"github.com/microsoft/durabletask-go/backend/sqlite"
	"github.com/Youmanvi/taskorchestrator/internal/infrastructure/config"
)

// NewSQLiteBackend creates a new SQLite backend
func NewSQLiteBackend(cfg *config.BackendConfig) (backend.Backend, error) {
	// Ensure directory exists
	dir := filepath.Dir(cfg.SQLiteFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Create SQLite backend
	sqliteBackend := sqlite.NewSqliteBackend(cfg.SQLiteFile)
	if sqliteBackend == nil {
		return nil, fmt.Errorf("failed to create SQLite backend at %s", cfg.SQLiteFile)
	}

	return sqliteBackend, nil
}

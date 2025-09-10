package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type ReplaceableDB struct {
	*sqlx.DB
}

func NewVaultDB(db *sqlx.DB) *ReplaceableDB {
	return &ReplaceableDB{DB: db}
}

// ReplaceDB replaces the current database connection with a new one.
// It ensures that the new connection is valid and pings it to check its readiness.
// If the new connection is valid, it closes the old connection and replaces it with the new one.
// This is useful for scenarios where the database connection needs to be refreshed or replaced.
func (d *ReplaceableDB) ReplaceDB(ctx context.Context, newDB *sqlx.DB) error {
	if newDB == nil {
		return errors.New("new database connection is nil")
	}

	// Ping the new database to ensure it's ready before making any changes
	if err := pingDBWithRetry(ctx, newDB); err != nil {
		return fmt.Errorf("failed to ping new database: %w", err)
	}

	// Store old connection for cleanup
	oldDB := d.DB
	d.DB = newDB

	// Close old connection after successful replacement
	// Don't fail the replacement if old connection fails to close gracefully
	if oldDB != nil {
		if err := oldDB.Close(); err != nil {
			// Log the error but don't fail the replacement
			// In a real application, you'd use a proper logger here
			fmt.Printf("Warning: failed to close old database connection: %v\n", err)
		}
	}

	return nil
}

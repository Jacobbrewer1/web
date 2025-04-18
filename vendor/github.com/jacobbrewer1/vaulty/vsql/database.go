package vsql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

var (
	newDbPingTimeout  = 7 * time.Second
	newDbPingInterval = 500 * time.Millisecond
)

type Database struct {
	*sqlx.DB
}

// NewDatabase establishes a database connection with the given Vault credentials
func NewDatabase(db *sqlx.DB) *Database {
	dbConn := &Database{
		DB: db,
	}

	return dbConn
}

func (d *Database) ReplaceDB(ctx context.Context, newDB *sqlx.DB) error {
	// Ensure that the new database connection is valid
	if newDB == nil {
		return errors.New("new database connection is nil")
	}

	// Ping the new database to ensure it's ready
	if err := pingDB(ctx, newDB); err != nil {
		return fmt.Errorf("failed to ping new database: %w", err)
	}

	if err := d.closeAndReplaceDB(newDB); err != nil {
		return fmt.Errorf("failed to close and replace database: %w", err)
	}

	return nil
}

func (d *Database) closeAndReplaceDB(newDB *sqlx.DB) error {
	// Close the old database connection
	if err := d.Close(); err != nil {
		return fmt.Errorf("failed to close old database connection: %w", err)
	}

	// Replace the old database connection with the new one
	d.DB = newDB

	return nil
}

func pingDB(ctx context.Context, db *sqlx.DB) error {
	ctx, cancel := context.WithTimeout(ctx, newDbPingTimeout)
	defer cancel()

	for {
		err := db.PingContext(ctx)
		if err == nil {
			break
		}
		select {
		case <-time.After(newDbPingInterval):
			continue
		case <-ctx.Done():
			return fmt.Errorf("failed to successfully ping database before context timeout: %w", err)
		}
	}

	return nil
}

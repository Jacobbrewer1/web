package database

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

const (
	// newDbPingTimeout is the maximum duration to wait for a successful ping to the new database.
	newDbPingTimeout = 7 * time.Second

	// newDbPingInterval is the interval between ping attempts to the new database.
	newDbPingInterval = 500 * time.Millisecond
)

// MySQLConnectionStringFromVaultSecret constructs a MySQL connection string from the given parameters and Vault secret data.
func MySQLConnectionStringFromVaultSecret(
	username, password, host string, port int, dbName string,
) string {
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true&multiStatements=true",
		username,
		password,
		net.JoinHostPort(host, strconv.Itoa(port)),
		dbName,
	)
}

// OpenDBConnection opens a database connection using the provided connection string and pings it to ensure it's valid.
func OpenDBConnection(ctx context.Context, connectionString string) (*sqlx.DB, error) {
	db, err := sqlx.Open("mysql", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	if err := pingDBWithRetry(ctx, db); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// pingDBWithRetry pings the database with retry logic and proper context handling
func pingDBWithRetry(ctx context.Context, db *sqlx.DB) error {
	pingCtx, cancel := context.WithTimeout(ctx, newDbPingTimeout)
	defer cancel()

	ticker := time.NewTicker(newDbPingInterval)
	defer ticker.Stop()

	// Try immediate ping first
	if err := db.PingContext(pingCtx); err == nil {
		return nil
	}

	// Retry with ticker
	for {
		select {
		case <-ticker.C:
			if err := db.PingContext(pingCtx); err == nil {
				return nil
			}
		case <-pingCtx.Done():
			// Try one final ping to get the actual error
			if err := db.PingContext(context.Background()); err != nil {
				return fmt.Errorf("database ping failed after timeout: %w", err)
			}
			return pingCtx.Err()
		}
	}
}

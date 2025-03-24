package repositories

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	hashiVault "github.com/hashicorp/vault/api"
	"github.com/jacobbrewer1/vaulty"
	"github.com/jmoiron/sqlx"
	"github.com/spf13/viper"
)

type DatabaseConnector interface {
	ConnectDB() (*Database, error)
}

type databaseConnector struct {
	ctx            context.Context
	l              *slog.Logger
	client         vaulty.Client
	vip            *viper.Viper
	currentSecrets *hashiVault.Secret
}

func NewDatabaseConnector(opts ...ConnectorOption) (DatabaseConnector, error) {
	c := &databaseConnector{
		ctx: context.Background(),
		l:   slog.Default(),
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.ctx == nil {
		c.ctx = context.Background()
	}

	switch {
	case c.client == nil:
		return nil, errors.New("no vault client provided")
	case c.vip == nil:
		return nil, errors.New("no viper configuration provided")
	case c.currentSecrets == nil:
		return nil, errors.New("no current secrets provided")
	}

	return c, nil
}

// ConnectDB connects to the database
func (d *databaseConnector) ConnectDB() (*Database, error) {
	if !d.vip.IsSet("vault") {
		return nil, errors.New("no vault configuration found")
	}

	d.vip.Set("database.connection_string", generateConnectionStr(d.vip, d.currentSecrets))
	sqlxDb, err := createConnection(d.vip)
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	d.l.Info("Database connection established")

	db := NewDatabase(sqlxDb,
		WithDatabaseLogger(d.l),
	)

	go func() {
		err := vaulty.RenewLease(d.ctx, d.l, d.client, fmt.Sprintf("%s/%s", d.vip.GetString("vault.database.path"), d.vip.GetString("vault.database.role")), d.currentSecrets, func() (*hashiVault.Secret, error) {
			d.l.Warn("Vault lease expired, reconnecting to database")

			vs, err := d.client.Path(
				d.vip.GetString("vault.database.role"),
				vaulty.WithPrefix(d.vip.GetString("vault.database.path")),
			).GetSecret(d.ctx)
			if err != nil {
				return nil, fmt.Errorf("error getting secrets from vault: %w", err)
			}

			dbConnectionString := generateConnectionStr(d.vip, vs)
			d.vip.Set("database.connection_string", dbConnectionString)

			newDb, err := createConnection(d.vip)
			if err != nil {
				return nil, fmt.Errorf("error connecting to database: %w", err)
			}

			if err := db.Reconnect(d.ctx, newDb); err != nil {
				return nil, fmt.Errorf("error reconnecting to database: %w", err)
			}

			d.l.Info("Database reconnected")

			return vs, nil
		})
		if err != nil { // nolint:revive // Traditional error handling
			d.l.Error("Error renewing vault lease", slog.String(loggingKeyError, err.Error()))
			os.Exit(1) // Forces new credentials to be fetched
		}
	}()

	d.l.Info("Database connection established with vault")
	return db, nil
}

func createConnection(v *viper.Viper) (*sqlx.DB, error) {
	connectionString := v.GetString("database.connection_string")
	if connectionString == "" {
		return nil, errors.New("no database connection string provided")
	}

	db, err := sqlx.Open("mysql", connectionString)
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	// Test the connection.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	return db, nil
}

func generateConnectionStr(v *viper.Viper, vs *hashiVault.Secret) string {
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?timeout=90s&multiStatements=true&parseTime=true",
		vs.Data["username"],
		vs.Data["password"],
		v.GetString("database.host"),
		v.GetString("database.schema"),
	)
}

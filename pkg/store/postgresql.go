package store

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// NewPostgres creates a new PostgreSQL connection using sqlx and verifies it.
func NewPostgres(dsn string) (store SQLX, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	store.Client, err = sqlx.ConnectContext(ctx, "postgres", dsn)
	if err != nil {
		return
	}

	// Optional: set max idle and open connections
	store.Client.SetMaxOpenConns(25)
	store.Client.SetMaxIdleConns(25)
	store.Client.SetConnMaxLifetime(5 * time.Minute)

	return
}

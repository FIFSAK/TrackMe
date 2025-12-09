package store

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
)

type ClickHouse struct {
	Conn clickhouse.Conn
}

// NewClickHouse connects to ClickHouse and returns a store.ClickHouse
func NewClickHouse(addr, userName, password, db string) (ClickHouse, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: db,
			Username: userName,
			Password: password,
		},
		Protocol: clickhouse.Native,
		TLS:      nil,
	})
	if err != nil {
		return ClickHouse{}, err
	}

	if err := conn.Ping(context.Background()); err != nil {
		return ClickHouse{}, fmt.Errorf("failed to ping clickhouse: %w", err)
	}

	// Run migration right after connection
	if err := MigrateClickHouse(context.Background(), conn); err != nil {
		return ClickHouse{}, fmt.Errorf("clickhouse migration failed: %w", err)
	}

	return ClickHouse{Conn: conn}, nil
}

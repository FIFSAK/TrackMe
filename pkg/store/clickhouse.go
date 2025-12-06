package store

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
)

type ClickHouse struct {
	Conn clickhouse.Conn
}

// NewClickHouse connects to ClickHouse and returns a store.ClickHouse
func NewClickHouse() (ClickHouse, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{"qipzxoq7h7.europe-west4.gcp.clickhouse.cloud:9440"},
		Auth: clickhouse.Auth{
			Username: "default",
			Password: "Vvo87NNck_O~D",
		},
		Protocol: clickhouse.Native,
		TLS:      &tls.Config{},
	})
	if err != nil {
		return ClickHouse{}, err
	}

	// Run migration right after connection
	if err := MigrateClickHouse(context.Background(), conn); err != nil {
		return ClickHouse{}, fmt.Errorf("clickhouse migration failed: %w", err)
	}

	return ClickHouse{Conn: conn}, nil
}

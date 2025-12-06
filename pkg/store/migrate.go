package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mongodb"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func Migrate(dataSourceName string) (err error) {
	if !strings.Contains(dataSourceName, "://") {
		err = errors.New("store: undefined data source name " + dataSourceName)
		return
	}
	driverName := strings.ToLower(strings.Split(dataSourceName, "://")[0])
	driverName = strings.TrimSuffix(driverName, "+srv")

	migrations, err := migrate.New(fmt.Sprintf("file://migrations/%s", driverName), dataSourceName)
	if err != nil {
		return
	}

	if err = migrations.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		return err
	}

	return
}

// MigrateClickHouse ensures required tables exist
func MigrateClickHouse(ctx context.Context, conn clickhouse.Conn) error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id UUID,
			name String,
			email String,
			password String,
			role String,
			created_at DateTime ,
			updated_at DateTime
		) ENGINE = ReplacingMergeTree(updated_at)
		ORDER BY id`,

		`CREATE TABLE IF NOT EXISTS clients (
			id UUID,
			name String,
			email String,
			current_stage String,
			last_updated DateTime DEFAULT now(),
			is_active UInt8,
			source String,
			channel String,
			app String,
			last_login DateTime
		) ENGINE = ReplacingMergeTree(last_updated)
		ORDER BY id`,

		`CREATE TABLE IF NOT EXISTS metrics (
			id UUID,
			client_id UUID,
			metric_type String,
			value Float64,
			timestamp DateTime DEFAULT now()
		) ENGINE = MergeTree()
		ORDER BY (client_id, timestamp)`,
	}

	for _, query := range tables {
		if err := conn.Exec(ctx, query); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	fmt.Println("ClickHouse migration completed")
	return nil
}

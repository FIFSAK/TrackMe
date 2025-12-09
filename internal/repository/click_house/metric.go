package clickhouse

import (
	"TrackMe/internal/domain/metric"
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
)

type MetricRepository struct {
	conn clickhouse.Conn
}

func NewMetricRepository(conn clickhouse.Conn) *MetricRepository {
	return &MetricRepository{conn: conn}
}

// Add inserts a new metric
func (r *MetricRepository) Add(ctx context.Context, data metric.Entity) (string, error) {
	if data.ID == "" {
		data.ID = uuid.New().String()
	}

	fmt.Println(data.ID) // Should be like: "f47ac10b-58cc-4372-a567-0e02b2c3d479"
	if data.CreatedAt.IsZero() {
		t := time.Now()
		data.CreatedAt = &t
	}

	// Use double quotes for ClickHouse
	query := `
		INSERT INTO metrics (id, "type", value, "interval", created_at, metadata)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	err := r.conn.Exec(ctx, query,
		data.ID,
		data.Type,
		data.Value,
		data.Interval,
		data.CreatedAt,
		data.Metadata,
	)
	if err != nil {
		return "", err
	}
	return data.ID, nil
}

// List retrieves metrics with optional filters
func (r *MetricRepository) List(ctx context.Context, filters metric.Filters) ([]metric.Entity, error) {
	// Use ? for ClickHouse positional parameters
	query := `SELECT id, type, value, interval, created_at, metadata 
			  FROM metrics 
			  WHERE 1=1 
			  AND type = ? 
			  AND interval = ?`

	rows, err := r.conn.Query(ctx, query, filters.Type, filters.Interval)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []metric.Entity
	for rows.Next() {
		var m metric.Entity
		if err := rows.Scan(
			&m.ID,
			&m.Type,
			&m.Value,
			&m.Interval,
			&m.CreatedAt,
			&m.Metadata,
		); err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}
	return metrics, nil
}

// Update inserts a new version of the metric (ClickHouse ReplacingMergeTree)
func (r *MetricRepository) Update(ctx context.Context, id string, data metric.Entity) (metric.Entity, error) {
	data.ID = id
	_, err := r.Add(ctx, data)
	if err != nil {
		return metric.Entity{}, err
	}
	return data, nil
}

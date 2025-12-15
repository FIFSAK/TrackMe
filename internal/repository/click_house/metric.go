package clickhouse

import (
	"TrackMe/internal/domain/metric"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
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
	query := `
  SELECT
   id,
   type,
   argMax(value, created_at) as value,
   argMax(interval, created_at) as metric_interval,
   argMax(toDate(created_at), created_at) as created_date,
   argMax(metadata, created_at) as metadata
  FROM metrics
  WHERE 1=1
 `

	var args []interface{}
	var conditions []string

	if filters.Type != "" {
		conditions = append(conditions, "type = ?")
		args = append(args, filters.Type)
	}

	if filters.Interval != "" {
		conditions = append(conditions, "interval = ?")
		args = append(args, filters.Interval)
	}

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	query += " GROUP BY id, type"
	query += " ORDER BY created_date DESC"

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func(rows driver.Rows) {
		cerr := rows.Close()
		if cerr != nil {
			log.Printf("rows.Close error: %v", cerr)
		}
	}(rows)

	var metrics []metric.Entity
	for rows.Next() {
		var m metric.Entity
		var metricType string
		var value float64
		var interval string
		var createdDate time.Time
		var metadataMap map[string]string

		if err := rows.Scan(
			&m.ID,
			&metricType,
			&value,
			&interval,
			&createdDate,
			&metadataMap,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		t := metric.Type(metricType)
		m.Type = &t
		m.Value = &value
		m.Interval = &interval
		m.CreatedAt = &createdDate
		m.Metadata = metadataMap

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

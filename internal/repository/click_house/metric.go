package clickhouse

import (
	"TrackMe/internal/domain/metric"
	"context"
	"database/sql"
	"fmt"
	"strings"
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
	query := `
        SELECT 
            id, 
            type, 
            value, 
            interval, 
            created_at, 
            metadata,
            client_id
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

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []metric.Entity
	for rows.Next() {
		var m metric.Entity

		// Temporary variables for scanning
		var metricType string
		var value float64
		var interval sql.NullString
		var createdAt time.Time
		var metadataMap map[string]string // Changed from string to map
		var clientID string

		if err := rows.Scan(
			&m.ID,
			&metricType,
			&value,
			&interval,
			&createdAt,
			&metadataMap, // Scan directly as map
			&clientID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert string to metric.Type
		t := metric.Type(metricType)
		m.Type = &t
		m.Value = &value

		if interval.Valid {
			intervalStr := interval.String
			m.Interval = &intervalStr
		}

		m.CreatedAt = &createdAt
		m.Metadata = metadataMap // Assign the map directly

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

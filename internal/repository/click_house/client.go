package clickhouse

import (
	"TrackMe/internal/domain/client"
	"TrackMe/pkg/store"
	"context"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
)

type ClientRepository struct {
	conn clickhouse.Conn
}

func GenerateUUID() string {
	return uuid.New().String()
}

func NewClientRepository(conn clickhouse.Conn) *ClientRepository {
	return &ClientRepository{conn: conn}
}

// Create inserts a new client
func (r *ClientRepository) Create(ctx context.Context, data client.Entity) (client.Entity, error) {
	if data.ID == "" {
		data.ID = GenerateUUID()
	}
	query := `
		INSERT INTO clients (id, name, email, current_stage, last_updated, is_active, source, channel, app, last_login)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	err := r.conn.Exec(ctx, query,
		data.ID,
		data.Name,
		data.Email,
		data.CurrentStage,
		time.Now(),
		data.IsActive,
		data.Source,
		data.Channel,
		data.App,
		data.LastLogin,
	)
	if err != nil {
		return client.Entity{}, err
	}
	return data, nil
}

// Get retrieves a client by ID
func (r *ClientRepository) Get(ctx context.Context, id string) (client.Entity, error) {
	query := `
		SELECT id, name, email, current_stage, last_updated, is_active, source, channel, app, last_login
		FROM clients 
		WHERE id = ?
	`
	var c client.Entity
	row := r.conn.QueryRow(ctx, query, id)
	if err := row.Scan(
		&c.ID,
		&c.Name,
		&c.Email,
		&c.CurrentStage,
		&c.LastUpdated,
		&c.IsActive,
		&c.Source,
		&c.Channel,
		&c.App,
		&c.LastLogin,
	); err != nil {
		// Check if no rows returned
		if err.Error() == "no rows in result set" {
			return c, store.ErrorNotFound
		}
		return c, err
	}
	return c, nil
}

// List retrieves clients with optional filters
func (r *ClientRepository) List(ctx context.Context, filters client.Filters, limit, offset int) ([]client.Entity, int, error) {
	query := "SELECT id, name, email, current_stage, last_updated, is_active, source, channel, app, last_login FROM clients WHERE 1=1"
	args := []interface{}{}

	if filters.Stage != "" {
		query += " AND current_stage = ?"
		args = append(args, filters.Stage)
	}
	if filters.Source != "" {
		query += " AND source = ?"
		args = append(args, filters.Source)
	}
	if filters.Channel != "" {
		query += " AND channel = ?"
		args = append(args, filters.Channel)
	}
	if filters.AppStatus != "" {
		query += " AND app = ?"
		args = append(args, filters.AppStatus)
	}

	query += " ORDER BY last_updated DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer func(rows driver.Rows) {
		cerr := rows.Close()
		if cerr != nil {
			log.Printf("rows.Close error: %v", cerr)
		}
	}(rows)

	var clients []client.Entity
	for rows.Next() {
		var c client.Entity
		if err := rows.Scan(
			&c.ID,
			&c.Name,
			&c.Email,
			&c.CurrentStage,
			&c.LastUpdated,
			&c.IsActive,
			&c.Source,
			&c.Channel,
			&c.App,
			&c.LastLogin,
		); err != nil {
			return nil, 0, err
		}
		clients = append(clients, c)
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM clients WHERE 1=1"
	countArgs := args[:len(args)-2] // remove LIMIT & OFFSET
	var total int
	if err := r.conn.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	return clients, total, nil
}

// Update replaces an existing client (ClickHouse usually uses ReplacingMergeTree)
func (r *ClientRepository) Update(ctx context.Context, id string, data client.Entity) (client.Entity, error) {
	// We just insert a new version
	data.ID = id
	t := time.Now()
	data.LastUpdated = &t
	return r.Create(ctx, data)
}

// Delete is tricky in ClickHouse, usually mark as inactive
func (r *ClientRepository) Delete(ctx context.Context, id string) error {
	query := "ALTER TABLE clients UPDATE is_active = 0 WHERE id = ?"
	return r.conn.Exec(ctx, query, id)
}

// Count returns the total number of client entities matching the filter.
func (r *ClientRepository) Count(ctx context.Context, filter bson.M) (int64, error) {
	query := "SELECT COUNT(*) FROM clients WHERE 1=1"
	args := []interface{}{}
	for k, v := range filter {
		query += " AND " + k + " = ?"
		args = append(args, v)
	}

	var count int64
	if err := r.conn.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *ClientRepository) GetByEmail(ctx context.Context, email string) (client.Entity, error) {
	query := "SELECT id, name, email, current_stage, last_updated, is_active, source, channel, app, last_login FROM clients WHERE email = ? LIMIT 1"
	var c client.Entity
	err := r.conn.QueryRow(ctx, query, email).Scan(
		&c.ID,
		&c.Name,
		&c.Email,
		&c.CurrentStage,
		&c.LastUpdated,
		&c.IsActive,
		&c.Source,
		&c.Channel,
		&c.App,
		&c.LastLogin,
	)
	if err != nil {
		return c, store.ErrorNotFound
	}
	return c, nil
}

package clickhouse

import (
	"TrackMe/internal/domain/user"
	"TrackMe/pkg/store"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// UserRepository implements user.Repository for ClickHouse
type UserRepository struct {
	conn clickhouse.Conn
}

// NewUserRepository creates a new UserRepository
func NewUserRepository(conn clickhouse.Conn) *UserRepository {
	return &UserRepository{conn: conn}
}

// Create inserts a new user into ClickHouse
func (r *UserRepository) Create(ctx context.Context, data user.Entity) (user.Entity, error) {
	if data.ID == "" {
		data.ID = GenerateUUID()
	}
	now := time.Now()
	data.CreatedAt = now
	data.UpdatedAt = now

	query := `
		INSERT INTO users (id, name, email,password, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	fmt.Println(data.Role + "______________")

	if err := r.conn.Exec(ctx, query,
		data.ID,
		data.Name,
		data.Email,
		data.Password,
		data.Role,
		data.CreatedAt,
		data.UpdatedAt,
	); err != nil {
		return user.Entity{}, err
	}

	return data, nil
}

// Get retrieves a user by ID
func (r *UserRepository) Get(ctx context.Context, id string) (user.Entity, error) {
	query := "SELECT id, name, email, role, created_at, updated_at FROM users WHERE id = ?"
	var u user.Entity
	err := r.conn.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.Role,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return u, err
	}
	return u, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (user.Entity, error) {
	query := "SELECT id, name, email,password, role, created_at, updated_at FROM users WHERE email = ? LIMIT 1"
	var u user.Entity
	err := r.conn.QueryRow(ctx, query, email).Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.Password,
		&u.Role,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return u, store.ErrorNotFound // <-- return your standard "not found"
		}
		return u, err
	}
	return u, nil
}

// List retrieves users with pagination
func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]user.Entity, int, error) {
	if limit <= 0 {
		limit = 50
	}

	query := "SELECT id, name, email, role, created_at, updated_at FROM users ORDER BY created_at DESC LIMIT ? OFFSET ?"
	rows, err := r.conn.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func(rows driver.Rows) {
		cerr := rows.Close()
		if cerr != nil {
			log.Printf("rows.Close error: %v", cerr)
		}
	}(rows)

	var users []user.Entity
	for rows.Next() {
		var u user.Entity
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}

	// Total count
	countQuery := "SELECT COUNT(*) FROM users"
	var total int
	if err := r.conn.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// Update replaces an existing user (ClickHouse uses ReplacingMergeTree)
func (r *UserRepository) Update(ctx context.Context, id string, data user.Entity) (user.Entity, error) {
	data.ID = id
	data.UpdatedAt = time.Now()

	query := `
		INSERT INTO users (id, name, email,password, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?,?)
	`
	if err := r.conn.Exec(ctx, query,
		data.ID,
		data.Name,
		data.Email,
		data.Password,
		data.Role,
		data.CreatedAt,
		data.UpdatedAt,
	); err != nil {
		return user.Entity{}, err
	}

	return data, nil
}

// Delete marks a user as deleted
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	query := "ALTER TABLE users UPDATE role = 'deleted' WHERE id = ?"
	return r.conn.Exec(ctx, query, id)
}

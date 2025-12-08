package postgres

import (
	"TrackMe/internal/domain/user"
	"TrackMe/pkg/store"
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// UserRepository handles CRUD operations for users in PostgreSQL.
type UserRepository struct {
	db *sqlx.DB
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

// List retrieves all users with pagination.
func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]user.Entity, int, error) {
	if limit <= 0 {
		limit = 50
	}

	// Count total users
	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, name, email, role, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []user.Entity
	for rows.Next() {
		var u user.Entity
		if err := rows.Scan(
			&u.ID,
			&u.Name,
			&u.Email,
			&u.Role,
			&u.CreatedAt,
			&u.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}

	return users, total, nil
}

// Create inserts a new user into the database.
func (r *UserRepository) Create(ctx context.Context, data user.Entity) (user.Entity, error) {
	now := time.Now()
	data.CreatedAt = now
	data.UpdatedAt = now
	data.ID = uuid.NewString()

	query := `
    INSERT INTO users (id, name, email, password, role, created_at, updated_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7)
    RETURNING id
`
	err := r.db.QueryRowContext(ctx, query,
		data.ID,
		data.Name,
		data.Email,
		data.Password, // <--- include password
		data.Role,
		data.CreatedAt,
		data.UpdatedAt,
	).Scan(&data.ID)

	if err != nil {
		return user.Entity{}, err
	}

	return data, nil
}

// Get retrieves a user by ID from the database.
func (r *UserRepository) Get(ctx context.Context, id string) (user.Entity, error) {
	query := `
		SELECT id, name, email, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	var u user.Entity
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.Role,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return u, store.ErrorNotFound
		}
		return u, err
	}
	return u, nil
}

// GetByEmail retrieves a user by email from the database.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (user.Entity, error) {
	query := `
		SELECT id, name,password, email, role, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	var u user.Entity
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&u.ID,
		&u.Name,
		&u.Password,
		&u.Email,
		&u.Role,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return u, store.ErrorNotFound
		}
		return u, err
	}
	return u, nil
}

// Update modifies an existing user in the database.
func (r *UserRepository) Update(ctx context.Context, id string, data user.Entity) (user.Entity, error) {
	data.UpdatedAt = time.Now()

	query := `
		UPDATE users SET
			name = $1,
			email = $2,
			role = $3,
			updated_at = $4
		WHERE id = $5
		RETURNING id, name, email, role, created_at, updated_at
	`
	var updated user.Entity
	err := r.db.QueryRowContext(ctx, query,
		data.Name,
		data.Email,
		data.Role,
		data.UpdatedAt,
		id,
	).Scan(
		&updated.ID,
		&updated.Name,
		&updated.Email,
		&updated.Role,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user.Entity{}, store.ErrorNotFound
		}
		return user.Entity{}, err
	}

	return updated, nil
}

// Delete removes a user from the database.
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return store.ErrorNotFound
	}
	return nil
}

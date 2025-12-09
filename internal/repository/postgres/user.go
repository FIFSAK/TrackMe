package postgres

import (
	"TrackMe/internal/domain/user"
	"TrackMe/pkg/store"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepository handles CRUD operations for users in PostgreSQL.
type UserRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// List retrieves all users with pagination.
func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]user.Entity, int, error) {
	if limit <= 0 {
		limit = 50
	}

	// Count total users
	var total int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	query := `
        SELECT id, name, email, role, created_at, updated_at
        FROM users
        ORDER BY created_at DESC
        LIMIT $1 OFFSET $2
    `

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []user.Entity

	for rows.Next() {
		var u user.Entity
		err := rows.Scan(
			&u.ID,
			&u.Name,
			&u.Email,
			&u.Role,
			&u.CreatedAt,
			&u.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, u)
	}

	return users, total, rows.Err()
}

// Create inserts a new user into the database.
func (r *UserRepository) Create(ctx context.Context, data user.Entity) (user.Entity, error) {
	now := time.Now()
	data.CreatedAt = now
	data.UpdatedAt = now

	if data.ID == "" {
		data.ID = uuid.NewString()
	}

	query := `
        INSERT INTO users (id, name, email, password, role, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id
    `

	var insertedID string
	err := r.db.QueryRow(ctx, query,
		data.ID,
		data.Name,
		data.Email,
		data.Password,
		data.Role,
		data.CreatedAt,
		data.UpdatedAt,
	).Scan(&insertedID)

	if err != nil {
		// Detect unique violation
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			if pgErr.ConstraintName == "users_email_key" {
				return user.Entity{}, store.ErrorNotFound
			}
		}
		return user.Entity{}, fmt.Errorf("failed to create user: %w", err)
	}

	data.ID = insertedID
	return data, nil
}

// Get retrieves a user by ID.
func (r *UserRepository) Get(ctx context.Context, id string) (user.Entity, error) {
	query := `
        SELECT id, name, email, password, role, created_at, updated_at
        FROM users
        WHERE id = $1
    `

	var u user.Entity
	err := r.db.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.Password,
		&u.Role,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return u, store.ErrorNotFound
		}
		return u, fmt.Errorf("failed to get user: %w", err)
	}

	return u, nil
}

// GetByEmail retrieves a user by email.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (user.Entity, error) {
	query := `
        SELECT id, name, email, password, role, created_at, updated_at
        FROM users
        WHERE email = $1
    `

	var u user.Entity
	err := r.db.QueryRow(ctx, query, email).Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.Password,
		&u.Role,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return u, store.ErrorNotFound
		}
		return u, fmt.Errorf("failed to get user by email: %w", err)
	}

	return u, nil
}

// Update modifies an existing user.
func (r *UserRepository) Update(ctx context.Context, id string, data user.Entity) (user.Entity, error) {
	data.UpdatedAt = time.Now()

	query := `
        UPDATE users SET
            name = $1,
            email = $2,
            role = $3,
            updated_at = $4
        WHERE id = $5
        RETURNING id, name, email, password, role, created_at, updated_at
    `

	var updated user.Entity
	err := r.db.QueryRow(ctx, query,
		data.Name,
		data.Email,
		data.Role,
		data.UpdatedAt,
		id,
	).Scan(
		&updated.ID,
		&updated.Name,
		&updated.Email,
		&updated.Password,
		&updated.Role,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return user.Entity{}, store.ErrorNotFound
		}

		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			if pgErr.ConstraintName == "users_email_key" {
				return user.Entity{}, store.ErrorNotFound
			}
		}

		return user.Entity{}, fmt.Errorf("failed to update user: %w", err)
	}

	return updated, nil
}

// UpdateWithPassword updates user including password.
func (r *UserRepository) UpdateWithPassword(ctx context.Context, id string, data user.Entity) (user.Entity, error) {
	data.UpdatedAt = time.Now()

	query := `
        UPDATE users SET
            name = $1,
            email = $2,
            password = $3,
            role = $4,
            updated_at = $5
        WHERE id = $6
        RETURNING id, name, email, password, role, created_at, updated_at
    `

	var updated user.Entity
	err := r.db.QueryRow(ctx, query,
		data.Name,
		data.Email,
		data.Password,
		data.Role,
		data.UpdatedAt,
		id,
	).Scan(
		&updated.ID,
		&updated.Name,
		&updated.Email,
		&updated.Password,
		&updated.Role,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return user.Entity{}, store.ErrorNotFound
		}

		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			if pgErr.ConstraintName == "users_email_key" {
				return user.Entity{}, store.ErrorNotFound
			}
		}

		return user.Entity{}, fmt.Errorf("failed to update user with password: %w", err)
	}

	return updated, nil
}

// Delete removes a user from the database.
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = $1`

	cmd, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return store.ErrorNotFound
	}

	return nil
}

// EmailExists checks if a user with the given email exists.
func (r *UserRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if email exists: %w", err)
	}

	return exists, nil
}

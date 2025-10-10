package user

import (
	"context"
)

// Repository defines the interface for user repository operations.
type Repository interface {
	// List retrieves all user entities.
	List(ctx context.Context, limit, offset int) ([]Entity, int, error)

	// Create creates a new user entity.
	Create(ctx context.Context, data Entity) (Entity, error)

	// Get retrieves a user entity by its ID.
	Get(ctx context.Context, id string) (Entity, error)

	// GetByEmail retrieves a user entity by email.
	GetByEmail(ctx context.Context, email string) (Entity, error)

	// Update modifies an existing user entity by its ID.
	Update(ctx context.Context, id string, data Entity) (Entity, error)

	// Delete removes a user entity by its ID.
	Delete(ctx context.Context, id string) error
}

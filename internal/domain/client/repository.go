package client

import (
	"context"
)

// Repository defines the interface for client repository operations.
type Repository interface {
	// List retrieves all client entities.
	List(ctx context.Context, filters Filters, limit, offset int) ([]Entity, int, error)

	// Add inserts a new client entity and returns its ID.
	Add(ctx context.Context, data Entity) (string, error)

	// Get retrieves a client entity by its ID.
	Get(ctx context.Context, id string) (Entity, error)

	// Update modifies an existing client entity by its ID.
	Update(ctx context.Context, id string, data Entity) (Entity, error)

	// Delete removes a client entity by its ID.
	Delete(ctx context.Context, id string) error
}

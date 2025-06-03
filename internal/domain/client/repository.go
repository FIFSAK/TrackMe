package client

import (
	"context"
)

// Repository defines the interface for client repository operations.
type Repository interface {
	// List retrieves all client entities.
	List(ctx context.Context) ([]Entity, error)

	// Add inserts a new client entity and returns its ID.
	Add(ctx context.Context, data Entity) (string, error)

	// Get retrieves a client entity by its ID.
	Get(ctx context.Context, id string) (Entity, error)

	// Update modifies an existing client entity by its ID.
	Update(ctx context.Context, id string, data Entity) error

	// Delete removes a client entity by its ID.
	Delete(ctx context.Context, id string) error
}

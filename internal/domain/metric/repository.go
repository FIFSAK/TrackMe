package metric

import (
	"context"
)

// Repository defines the interface for client repository operations.
type Repository interface {
	// List retrieves all client entities.
	List(ctx context.Context, filters Filters) ([]Entity, error)

	// Add inserts a new client entity and returns its ID.
	Add(ctx context.Context, data Entity) (string, error)

	// Update modifies an existing client entity by its ID.
	Update(ctx context.Context, id string, data Entity) (Entity, error)
}

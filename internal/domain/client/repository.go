package client

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
)

// Repository defines the interface for client repository operations.
type Repository interface {
	// List retrieves all client entities.
	List(ctx context.Context, filters Filters, limit, offset int) ([]Entity, int, error)

	// Create creates a new client entity.
	Create(ctx context.Context, data Entity) (Entity, error)

	// Get retrieves a client entity by its ID.
	Get(ctx context.Context, id string) (Entity, error)

	// GetByEmail retrieves a client entity by its email.
	GetByEmail(ctx context.Context, email string) (Entity, error)

	// Update modifies an existing client entity by its ID.
	Update(ctx context.Context, id string, data Entity) (Entity, error)

	// Count returns the total number of client entities matching the filter.
	Count(ctx context.Context, filter bson.M) (int64, error)

	// Delete removes a client entity by its ID.
	Delete(ctx context.Context, id string) error
}

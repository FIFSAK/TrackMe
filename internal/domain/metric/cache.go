package metric

import "context"

// Cache defines the interface for client cache operations.
type Cache interface {
	// Get retrieves a client entity by its ID from the cache.
	// Returns the entity and an error if the operation fails.
	Get(ctx context.Context, id string) (Entity, error)

	// Set stores a client entity in the cache.
	// Returns an error if the operation fails.
	Set(ctx context.Context, id string, entity Entity) error
}

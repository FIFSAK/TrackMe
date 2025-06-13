package metric

import "context"

// Cache defines the interface for client cache operations.
type Cache interface {
	Set(ctx context.Context, id string, entity Entity) error
	List(ctx context.Context, filters Filters) ([]Entity, error)
	StoreList(ctx context.Context, filters Filters, entities []Entity) error
	InvalidateListCache(ctx context.Context, filters Filters) error
}

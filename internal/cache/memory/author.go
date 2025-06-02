package memory

import (
	"context"
	"time"

	"github.com/patrickmn/go-cache"

	"TrackMe/internal/domain/client"
)

// AuthorCache handles caching of client entities in memory.
type AuthorCache struct {
	cache      *cache.Cache
	repository client.Repository
}

// NewAuthorCache creates a new AuthorCache.
func NewAuthorCache(r client.Repository) *AuthorCache {
	c := cache.New(5*time.Minute, 10*time.Minute) // Cache with 5 minutes expiration and 10 minutes cleanup interval
	return &AuthorCache{
		cache:      c,
		repository: r,
	}
}

// Get retrieves an client entity by its ID from the cache or repository.
func (c *AuthorCache) Get(ctx context.Context, id string) (client.Entity, error) {
	// Check if data is available in the cache
	if data, found := c.cache.Get(id); found {
		// Data found in the cache, return it
		return data.(client.Entity), nil
	}

	// Data not found in the cache, retrieve it from the repository
	entity, err := c.repository.Get(ctx, id)
	if err != nil {
		return client.Entity{}, err
	}

	// Store the retrieved data in the cache for future use
	c.cache.Set(id, entity, cache.DefaultExpiration)

	return entity, nil
}

// Set stores an client entity in the cache.
func (c *AuthorCache) Set(ctx context.Context, id string, entity client.Entity) error {
	c.cache.Set(id, entity, cache.DefaultExpiration)
	return nil
}

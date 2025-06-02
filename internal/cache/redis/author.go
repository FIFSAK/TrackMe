package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"

	"TrackMe/internal/domain/client"
)

// AuthorCache handles caching of client entities in Redis.
type AuthorCache struct {
	cache      *redis.Client
	repository client.Repository
}

// NewAuthorCache creates a new AuthorCache.
func NewAuthorCache(c *redis.Client, r client.Repository) *AuthorCache {
	return &AuthorCache{
		cache:      c,
		repository: r,
	}
}

// Get retrieves an client entity by its ID from the cache or repository.
func (c *AuthorCache) Get(ctx context.Context, id string) (client.Entity, error) {
	// Check if data is available in Redis cache
	data, err := c.cache.Get(ctx, id).Result()
	if err == nil {
		// Data found in cache, unmarshal JSON into struct
		var entity client.Entity
		if err = json.Unmarshal([]byte(data), &entity); err != nil {
			return client.Entity{}, err
		}
		return entity, nil
	}

	// Data not found in cache, retrieve it from the repository
	entity, err := c.repository.Get(ctx, id)
	if err != nil {
		return client.Entity{}, err
	}

	// Marshal struct data into JSON and store it in Redis cache
	payload, err := json.Marshal(entity)
	if err != nil {
		return client.Entity{}, err
	}

	if err = c.cache.Set(ctx, id, payload, 5*time.Minute).Err(); err != nil {
		return client.Entity{}, err
	}

	return entity, nil
}

// Set stores an client entity in the cache.
func (c *AuthorCache) Set(ctx context.Context, id string, entity client.Entity) error {
	// Marshal struct data into JSON and store it in Redis cache
	payload, err := json.Marshal(entity)
	if err != nil {
		return err
	}

	if err = c.cache.Set(ctx, id, payload, 5*time.Minute).Err(); err != nil {
		return err
	}

	return nil
}

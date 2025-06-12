package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"TrackMe/internal/domain/metric"
)

// MetricCache handles caching of author entities in Redis.
type MetricCache struct {
	cache      *redis.Client
	repository metric.Repository
}

// NewMetricCache creates a new MetricCache.
func NewMetricCache(c *redis.Client, r metric.Repository) *MetricCache {
	return &MetricCache{
		cache:      c,
		repository: r,
	}
}

// Get retrieves an author entity by its ID from the cache or repository.
func (c *MetricCache) Get(ctx context.Context, id string) (metric.Entity, error) {
	// Check if data is available in Redis cache
	data, err := c.cache.Get(ctx, id).Result()
	if err == nil {
		// Data found in cache, unmarshal JSON into struct
		var entity metric.Entity
		if err = json.Unmarshal([]byte(data), &entity); err != nil {
			return metric.Entity{}, err
		}
		return entity, nil
	}

	// Data not found in cache, retrieve it from the repository
	entity, err := c.repository.Get(ctx, id)
	if err != nil {
		return metric.Entity{}, err
	}

	// Marshal struct data into JSON and store it in Redis cache
	payload, err := json.Marshal(entity)
	if err != nil {
		return metric.Entity{}, err
	}

	if err = c.cache.Set(ctx, id, payload, 5*time.Minute).Err(); err != nil {
		return metric.Entity{}, err
	}

	return entity, nil
}

// Set stores an author entity in the cache.
func (c *MetricCache) Set(ctx context.Context, id string, entity metric.Entity) error {
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

// List retrieves metrics from cache based on filters, falling back to repository
func (c *MetricCache) List(ctx context.Context, filters metric.Filters) ([]metric.Entity, error) {
	// Create a cache key based on the filters
	cacheKey := fmt.Sprintf("metrics:list:%s:%s", filters.Type, filters.Interval)

	// Try to get from cache first
	data, err := c.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		// Data found in cache, unmarshal JSON into struct slice
		var entities []metric.Entity
		if err = json.Unmarshal([]byte(data), &entities); err != nil {
			return nil, err
		}
		return entities, nil
	}

	// If not in cache or error, get from repository
	entities, err := c.repository.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	// Cache the results for future use
	payload, err := json.Marshal(entities)
	if err != nil {
		return nil, err
	}

	// Store in Redis with TTL
	if err = c.cache.Set(ctx, cacheKey, payload, 5*time.Minute).Err(); err != nil {
		// Log the error but don't fail the request
		// Just continue with returning the data from repository
	}

	return entities, nil
}

// InvalidateListCache clears the cache for a specific list query
func (c *MetricCache) InvalidateListCache(ctx context.Context, filters metric.Filters) error {
	cacheKey := fmt.Sprintf("metrics:list:%s:%s", filters.Type, filters.Interval)
	return c.cache.Del(ctx, cacheKey).Err()
}

// StoreList caches a collection of metrics with the given filters
func (c *MetricCache) StoreList(ctx context.Context, filters metric.Filters, entities []metric.Entity) error {
	// Create a cache key based on the filters
	cacheKey := fmt.Sprintf("metrics:list:%s:%s", filters.Type, filters.Interval)

	// Marshal the entities to JSON
	payload, err := json.Marshal(entities)
	if err != nil {
		return err
	}

	// Store in Redis with TTL
	return c.cache.Set(ctx, cacheKey, payload, 5*time.Minute).Err()
}

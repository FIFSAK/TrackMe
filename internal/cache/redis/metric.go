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

// Set stores an author entity in the cache.
func (c *MetricCache) Set(ctx context.Context, id string, entity metric.Entity) error {
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
	cacheKey := fmt.Sprintf("metrics:list:%s:%s", filters.Type, filters.Interval)

	data, err := c.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		// Data found in cache, unmarshal JSON into struct slice
		var entities []metric.Entity
		if err = json.Unmarshal([]byte(data), &entities); err != nil {
			return nil, err
		}
		return entities, nil
	}

	entities, err := c.repository.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(entities)
	if err != nil {
		return nil, err
	}

	if err = c.cache.Set(ctx, cacheKey, payload, 5*time.Minute).Err(); err != nil {
		return nil, err
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
	cacheKey := fmt.Sprintf("metrics:list:%s:%s", filters.Type, filters.Interval)

	payload, err := json.Marshal(entities)
	if err != nil {
		return err
	}

	return c.cache.Set(ctx, cacheKey, payload, 5*time.Minute).Err()
}

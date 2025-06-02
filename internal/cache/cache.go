package cache

import (
	"TrackMe/internal/cache/memory"
	"TrackMe/internal/cache/redis"
	"TrackMe/internal/domain/book"
	"TrackMe/internal/domain/client"
	"TrackMe/pkg/store"
)

type Dependencies struct {
	AuthorRepository client.Repository
	BookRepository   book.Repository
}

// Configuration is an alias for a function that will take in a pointer to a Cache and modify it
type Configuration func(r *Cache) error

// Cache is an implementation of the Cache
type Cache struct {
	dependencies Dependencies
	redis        store.Redis

	Author client.Cache
	Book   book.Cache
}

// New takes a variable amount of Configuration functions and returns a new Cache
// Each Configuration will be called in the order they are passed in
func New(d Dependencies, configs ...Configuration) (s *Cache, err error) {
	// Create the cache
	s = &Cache{
		dependencies: d,
	}

	// Apply all Configurations passed in
	for _, cfg := range configs {
		// Pass the cache into the configuration function
		if err = cfg(s); err != nil {
			return
		}
	}

	return
}

// Close closes the cache and prevents new queries from starting.
// Close then waits for all queries that have started processing on the server to finish.
func (r *Cache) Close() {
	if r.redis.Connection != nil {
		r.redis.Connection.Close()
	}
}

// WithMemoryStore applies a memory database to the Cache
func WithMemoryStore() Configuration {
	return func(s *Cache) (err error) {
		// Create the memory database, if we needed parameters, such as connection strings they could be inputted here
		s.Author = memory.NewAuthorCache(s.dependencies.AuthorRepository)
		s.Book = memory.NewBookCache(s.dependencies.BookRepository)

		return
	}
}

// WithRedisStore applies a redis store to the Cache
func WithRedisStore(url string) Configuration {
	return func(s *Cache) (err error) {
		// Create the redis store, if we needed parameters, such as connection strings they could be inputted here
		s.redis, err = store.NewRedis(url)
		if err != nil {
			return
		}

		s.Author = redis.NewAuthorCache(s.redis.Connection, s.dependencies.AuthorRepository)
		s.Book = redis.NewBookCache(s.redis.Connection, s.dependencies.BookRepository)

		return
	}
}

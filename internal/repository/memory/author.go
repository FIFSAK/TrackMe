package memory

import (
	"context"
	"database/sql"
	"sync"

	"github.com/google/uuid"

	"TrackMe/internal/domain/client"
)

// AuthorRepository handles CRUD operations for authors in an in-memory database.
type AuthorRepository struct {
	db map[string]client.Entity
	sync.RWMutex
}

// NewAuthorRepository creates a new AuthorRepository.
func NewAuthorRepository() *AuthorRepository {
	return &AuthorRepository{db: make(map[string]client.Entity)}
}

// List retrieves all authors from the in-memory database.
func (r *AuthorRepository) List(ctx context.Context) ([]client.Entity, error) {
	r.RLock()
	defer r.RUnlock()

	authors := make([]client.Entity, 0, len(r.db))
	for _, data := range r.db {
		authors = append(authors, data)
	}
	return authors, nil
}

// Add inserts a new client into the in-memory database.
func (r *AuthorRepository) Add(ctx context.Context, data client.Entity) (string, error) {
	r.Lock()
	defer r.Unlock()

	id := uuid.New().String()
	data.ID = id
	r.db[id] = data
	return id, nil
}

// Get retrieves an client by ID from the in-memory database.
func (r *AuthorRepository) Get(ctx context.Context, id string) (client.Entity, error) {
	r.RLock()
	defer r.RUnlock()

	data, ok := r.db[id]
	if !ok {
		return client.Entity{}, sql.ErrNoRows
	}
	return data, nil
}

// Update modifies an existing client in the in-memory database.
func (r *AuthorRepository) Update(ctx context.Context, id string, data client.Entity) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.db[id]; !ok {
		return sql.ErrNoRows
	}
	r.db[id] = data
	return nil
}

// Delete removes an client by ID from the in-memory database.
func (r *AuthorRepository) Delete(ctx context.Context, id string) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.db[id]; !ok {
		return sql.ErrNoRows
	}
	delete(r.db, id)
	return nil
}

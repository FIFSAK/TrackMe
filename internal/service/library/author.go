package library

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"TrackMe/internal/domain/client"
	"TrackMe/pkg/log"
	"TrackMe/pkg/store"
)

// ListAuthors retrieves all authors from the repository.
func (s *Service) ListAuthors(ctx context.Context) ([]client.Response, error) {
	logger := log.LoggerFromContext(ctx).Named("list_authors")

	// Retrieve authors from the repository
	authors, err := s.authorRepository.List(ctx)
	if err != nil {
		logger.Error("failed to list authors", zap.Error(err))
		return nil, err
	}
	return client.ParseFromEntities(authors), nil
}

// AddAuthor adds a new client to the repository.
func (s *Service) AddAuthor(ctx context.Context, req client.Request) (client.Response, error) {
	logger := log.LoggerFromContext(ctx).Named("add_author").With(zap.Any("client", req))

	// Create a new client entity from the request
	newAuthor := client.New(req)

	// Add the new client to the repository
	id, err := s.authorRepository.Add(ctx, newAuthor)
	if err != nil {
		logger.Error("failed to add client", zap.Error(err))
		return client.Response{}, err
	}
	newAuthor.ID = id

	// Cache the newly created client
	if err := s.authorCache.Set(ctx, id, newAuthor); err != nil {
		logger.Warn("failed to cache new client", zap.Error(err))
	}

	return client.ParseFromEntity(newAuthor), nil
}

// GetAuthor retrieves an client by ID from the cache or repository.
func (s *Service) GetAuthor(ctx context.Context, id string) (client.Response, error) {
	logger := log.LoggerFromContext(ctx).Named("get_author").With(zap.String("id", id))

	// Try to get the client from the cache
	cachedAuthor, err := s.authorCache.Get(ctx, id)
	if err == nil {
		return client.ParseFromEntity(cachedAuthor), nil
	}

	// If not found in cache, get from the repository
	repoAuthor, err := s.authorRepository.Get(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrorNotFound) {
			logger.Warn("client not found", zap.Error(err))
			return client.Response{}, err
		}
		logger.Error("failed to get client", zap.Error(err))
		return client.Response{}, err
	}

	// Store the retrieved client in the cache
	if cacheErr := s.authorCache.Set(ctx, id, repoAuthor); cacheErr != nil {
		logger.Warn("failed to cache client", zap.Error(cacheErr))
	}

	return client.ParseFromEntity(repoAuthor), nil
}

// UpdateAuthor updates an existing client in the repository.
func (s *Service) UpdateAuthor(ctx context.Context, id string, req client.Request) error {
	logger := log.LoggerFromContext(ctx).Named("update_author").With(zap.String("id", id), zap.Any("client", req))

	// Create an updated client entity from the request
	updatedAuthor := client.New(req)

	// Update the client in the repository
	err := s.authorRepository.Update(ctx, id, updatedAuthor)
	if err != nil {
		if errors.Is(err, store.ErrorNotFound) {
			logger.Warn("client not found", zap.Error(err))
			return err
		}
		logger.Error("failed to update client", zap.Error(err))
		return err
	}

	// Update the cache with the new client data
	if err := s.authorCache.Set(ctx, id, updatedAuthor); err != nil {
		logger.Warn("failed to update cache for client", zap.Error(err))
	}

	return nil
}

// DeleteAuthor deletes an client by ID from the repository.
func (s *Service) DeleteAuthor(ctx context.Context, id string) error {
	logger := log.LoggerFromContext(ctx).Named("delete_author").With(zap.String("id", id))

	// Delete the client from the repository
	err := s.authorRepository.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrorNotFound) {
			logger.Warn("client not found", zap.Error(err))
			return err
		}
		logger.Error("failed to delete client", zap.Error(err))
		return err
	}

	// Remove the client from the cache
	if err := s.authorCache.Set(ctx, id, client.Entity{}); err != nil {
		logger.Warn("failed to remove client from cache", zap.Error(err))
	}

	return nil
}

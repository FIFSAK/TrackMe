package track

import (
	"TrackMe/internal/domain/client"
	"TrackMe/pkg/log"
	"TrackMe/pkg/store"
	"context"
	"errors"
	"go.uber.org/zap"
)

// ListClients retrieves all clients from the repository.
func (s *Service) ListClients(ctx context.Context, filters client.Filters, limit, offset int) ([]client.Response, int, error) {
	logger := zap.L().Named("service.client").With(
		zap.Any("filters", filters),
		zap.Int("limit", limit),
		zap.Int("offset", offset),
	)

	entities, total, err := s.clientRepository.List(ctx, filters, limit, offset)
	if err != nil {
		logger.Error("failed to list clients", zap.Error(err))
		return nil, 0, err
	}

	responses := client.ParseFromEntities(entities)

	return responses, total, nil
}

// AddClient adds a new client to the repository.
func (s *Service) AddClient(ctx context.Context, req client.Request) (client.Response, error) {
	logger := log.LoggerFromContext(ctx).Named("add_client").With(zap.Any("client", req))

	// Create a new client entity from the request
	newClient := client.New(req)

	// Add the new client to the repository
	id, err := s.clientRepository.Add(ctx, newClient)
	if err != nil {
		logger.Error("failed to add client", zap.Error(err))
		return client.Response{}, err
	}
	newClient.ID = id

	// Cache the newly created client
	if err := s.clientCache.Set(ctx, id, newClient); err != nil {
		logger.Warn("failed to cache new client", zap.Error(err))
	}

	return client.ParseFromEntity(newClient), nil
}

// GetClient retrieves an client by ID from the cache or repository.
func (s *Service) GetClient(ctx context.Context, id string) (client.Response, error) {
	logger := log.LoggerFromContext(ctx).Named("get_client").With(zap.String("id", id))

	// Try to get the client from the cache
	cachedClient, err := s.clientCache.Get(ctx, id)
	if err == nil {
		return client.ParseFromEntity(cachedClient), nil
	}

	// If not found in cache, get from the repository
	repoClient, err := s.clientRepository.Get(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrorNotFound) {
			logger.Warn("client not found", zap.Error(err))
			return client.Response{}, err
		}
		logger.Error("failed to get client", zap.Error(err))
		return client.Response{}, err
	}

	// Store the retrieved client in the cache
	if cacheErr := s.clientCache.Set(ctx, id, repoClient); cacheErr != nil {
		logger.Warn("failed to cache client", zap.Error(cacheErr))
	}

	return client.ParseFromEntity(repoClient), nil
}

// UpdateClient updates an existing client in the repository.
func (s *Service) UpdateClient(ctx context.Context, id string, req client.Request) (client.Response, error) {
	logger := log.LoggerFromContext(ctx).Named("update_client_stage").With(
		zap.String("client_id", id),
		zap.Any("request", req),
	)

	// Get existing client or initialize a new one
	var existingClient client.Entity
	var err error

	// Try to get existing client
	existingClient, err = s.clientRepository.Get(ctx, id)
	if err != nil && !errors.Is(err, store.ErrorNotFound) {
		logger.Error("failed to get client", zap.Error(err))
		return client.Response{}, err
	}

	// Apply updates from request to the entity
	updatedClient := client.New(req)
	updatedClient.ID = id

	// Preserve fields that shouldn't be overwritten
	if !existingClient.RegistrationDate.IsZero() {
		updatedClient.RegistrationDate = existingClient.RegistrationDate
	}

	// Update the client in the repository
	updatedClient, err = s.clientRepository.Update(ctx, id, updatedClient)
	if err != nil {
		logger.Error("failed to update client", zap.Error(err))
		return client.Response{}, err
	}

	return client.ParseFromEntity(updatedClient), nil
}

// DeleteClient deletes a client by ID from the repository.
func (s *Service) DeleteClient(ctx context.Context, id string) error {
	logger := log.LoggerFromContext(ctx).Named("delete_client").With(zap.String("id", id))

	// Delete the client from the repository
	err := s.clientRepository.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrorNotFound) {
			logger.Warn("client not found", zap.Error(err))
			return err
		}
		logger.Error("failed to delete client", zap.Error(err))
		return err
	}

	// Remove the client from the cache
	if err := s.clientCache.Set(ctx, id, client.Entity{}); err != nil {
		logger.Warn("failed to remove client from cache", zap.Error(err))
	}

	return nil
}

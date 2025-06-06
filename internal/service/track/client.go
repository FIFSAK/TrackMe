package track

import (
	"TrackMe/internal/domain/client"
	"TrackMe/pkg/log"
	"TrackMe/pkg/store"
	"context"
	"errors"
	"github.com/google/uuid"
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

	newClient := client.New(req)

	id, err := s.clientRepository.Add(ctx, newClient)
	if err != nil {
		logger.Error("failed to add client", zap.Error(err))
		return client.Response{}, err
	}
	newClient.ID = id

	return client.ParseFromEntity(newClient), nil
}

// GetClient retrieves a client by ID from the cache or repository.
func (s *Service) GetClient(ctx context.Context, id string) (client.Response, error) {
	logger := log.LoggerFromContext(ctx).Named("get_client").With(zap.String("id", id))

	repoClient, err := s.clientRepository.Get(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrorNotFound) {
			logger.Warn("client not found", zap.Error(err))
			return client.Response{}, err
		}
		logger.Error("failed to get client", zap.Error(err))
		return client.Response{}, err
	}

	if cacheErr := s.clientCache.Set(ctx, id, repoClient); cacheErr != nil {
		logger.Warn("failed to cache client", zap.Error(cacheErr))
	}

	return client.ParseFromEntity(repoClient), nil
}

// UpdateClient updates an existing client in the repository.
func (s *Service) UpdateClient(ctx context.Context, id string, req client.Request) (client.Response, error) {
	logger := log.LoggerFromContext(ctx).Named("update_client").With(
		zap.String("client_id", id),
		zap.Any("request", req),
	)

	existing, err := s.clientRepository.Get(ctx, id)
	if err != nil && !errors.Is(err, store.ErrorNotFound) {
		logger.Error("failed to get client", zap.Error(err))
		return client.Response{}, err
	}

	if len(req.Contracts) > 0 {
		for i, contract := range req.Contracts {
			if contract.ID == "" {
				req.Contracts[i].ID = uuid.New().String()
			}
		}
	}

	updated := client.New(req)
	updated.ID = id

	if existing.RegistrationDate != nil {
		updated.RegistrationDate = existing.RegistrationDate
	}

	newStage, err := s.StageRepository.UpdateStage(ctx, *updated.CurrentStage, req.Stage)
	if err != nil {
		logger.Error("invalid stage transition",
			zap.String("from", *updated.CurrentStage),
			zap.String("direction", req.Stage),
			zap.Error(err))
		return client.Response{}, errors.New("invalid stage transition: " + err.Error())
	}

	updated.CurrentStage = &newStage

	result, err := s.clientRepository.Update(ctx, id, updated)
	if err != nil {
		logger.Error("failed to update client", zap.Error(err))
		return client.Response{}, err
	}

	return client.ParseFromEntity(result), nil
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

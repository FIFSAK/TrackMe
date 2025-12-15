package track

import (
	"TrackMe/internal/domain/client"
	"TrackMe/pkg/log"
	"TrackMe/pkg/store"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type ClientTrackService interface {
	ListClients(ctx context.Context, filters client.Filters, limit, offset int) ([]client.Response, int, error)
	CreateClient(ctx context.Context, req client.Request) (client.Response, error)
	UpdateClient(ctx context.Context, id string, req client.Request) (client.Response, error)
	DeleteClient(ctx context.Context, id string) error
}

// ListClients retrieves all clients from the repository.
func (s *Service) ListClients(ctx context.Context, filters client.Filters, limit, offset int) ([]client.Response, int, error) {
	logger := log.LoggerFromContext(ctx).With().
		Interface("filters", filters).
		Int("limit", limit).
		Int("offset", offset).
		Str("component", "service.client").
		Logger()

	entities, total, err := s.clientRepository.List(ctx, filters, limit, offset)
	if err != nil {
		logger.Error().Err(err).Msg("failed to list clients")
		return nil, 0, err
	}

	responses := client.ParseFromEntities(entities)

	return responses, total, nil
}

// CreateClient creates a new client in the repository.
func (s *Service) CreateClient(ctx context.Context, req client.Request) (client.Response, error) {
	logger := log.LoggerFromContext(ctx).With().
		Interface("request", req).
		Str("component", "create_client").
		Logger()

	// Check if client with this email already exists
	existingClient, err := s.clientRepository.GetByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, store.ErrorNotFound) {
		logger.Error().Err(err).Msg("failed to check existing client by email")
		return client.Response{}, err
	}
	if existingClient.ID != "" {
		logger.Warn().Str("email", req.Email).Msg("client with this email already exists")
		return client.Response{}, errors.New("client with this email already exists")
	}

	if len(req.Contracts) > 0 {
		for i, contract := range req.Contracts {
			if contract.ID == "" {
				req.Contracts[i].ID = uuid.New().String()
			}
		}
	}

	newClient := client.New(req)
	now := time.Now()
	newClient.RegistrationDate = &now
	newClient.LastUpdated = &now

	// Validate stage transition from empty
	if req.Stage != "" {
		_, err := s.StageRepository.UpdateStage(ctx, "", req.Stage)
		if err != nil {
			logger.Error().
				Str("stage", req.Stage).
				Err(err).
				Msg("invalid initial stage")
			return client.Response{}, errors.New("invalid initial stage: " + err.Error())
		}
	}

	if newClient.IsActive != nil {
		*newClient.IsActive = true
	}

	result, err := s.clientRepository.Create(ctx, newClient)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create client")
		return client.Response{}, err
	}

	logger.Info().Str("client_id", result.ID).Msg("client created successfully")
	return client.ParseFromEntity(result), nil
}

// UpdateClient updates an existing client in the repository.
func (s *Service) UpdateClient(ctx context.Context, id string, req client.Request) (client.Response, error) {
	logger := log.LoggerFromContext(ctx).With().
		Str("client_id", id).
		Interface("request", req).
		Str("component", "update_client").
		Logger()

	existing, err := s.clientRepository.Get(ctx, id)
	if err != nil && !errors.Is(err, store.ErrorNotFound) {
		logger.Error().Err(err).Msg("failed to get client")
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
	now := time.Now()

	updated.RegistrationDate = &now

	if existing.RegistrationDate != nil {
		updated.RegistrationDate = existing.RegistrationDate
	}
	if existing.CurrentStage == nil {
		emptyStage := ""
		existing.CurrentStage = &emptyStage
	}

	newStage, err := s.StageRepository.UpdateStage(ctx, *existing.CurrentStage, req.Stage)
	if err != nil {
		logger.Error().
			Str("from", *updated.CurrentStage).
			Str("direction", req.Stage).
			Err(err).
			Msg("invalid stage transition")
		return client.Response{}, errors.New("invalid stage transition: " + err.Error())
	}

	updated.CurrentStage = &newStage
	if req.Stage == "prev" {
		err := s.calculateRollbackCount(ctx, now)
		if err != nil {
			return client.Response{}, err
		}
	}

	if updated.IsActive != nil {
		*updated.IsActive = true
	}
	if *updated.Name == "" {
		*updated.Name = "Guest_" + updated.ID
	}

	updated.LastUpdated = &now

	result, err := s.clientRepository.Update(ctx, id, updated)
	if err != nil {
		logger.Error().Err(err).Msg("failed to update client")
		return client.Response{}, err
	}

	return client.ParseFromEntity(result), nil
}

// DeleteClient removes a client from the repository.
func (s *Service) DeleteClient(ctx context.Context, id string) error {
	logger := log.LoggerFromContext(ctx).With().
		Str("client_id", id).
		Str("component", "delete_client").
		Logger()

	err := s.clientRepository.Delete(ctx, id)
	if err != nil {
		logger.Error().Err(err).Msg("failed to delete client")
		return err
	}

	logger.Info().Msg("client deleted successfully")
	return nil
}

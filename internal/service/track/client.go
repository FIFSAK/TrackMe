package track

import (
	"TrackMe/internal/domain/client"
	"TrackMe/pkg/log"
	"TrackMe/pkg/store"
	"context"
	"errors"
	"github.com/google/uuid"
	"time"
)

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
		err := s.calculateRollbackCount(ctx, time.Now())
		if err != nil {
			return client.Response{}, err
		}
	}

	result, err := s.clientRepository.Update(ctx, id, updated)
	if err != nil {
		logger.Error().Err(err).Msg("failed to update client")
		return client.Response{}, err
	}

	return client.ParseFromEntity(result), nil
}

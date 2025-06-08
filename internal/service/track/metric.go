package track

import (
	"TrackMe/internal/domain/metric"
	"TrackMe/pkg/store"
	"context"
	"go.uber.org/zap"
)

// ListMetrics retrieves all metric from the repository.
func (s *Service) ListMetrics(ctx context.Context, filters metric.Filters) ([]metric.Response, error) {
	logger := zap.L().Named("service.client").With(
		zap.Any("filters", filters),
	)
	if s.MetricRepository == nil {
		logger.Error("metric repository is not initialized")
		return nil, store.ErrorNotFound
	}
	entities, err := s.MetricRepository.List(ctx, filters)
	if err != nil {
		logger.Error("failed to list clients", zap.Error(err))
		return nil, err
	}

	responses := metric.ParseFromEntities(entities)

	return responses, nil
}

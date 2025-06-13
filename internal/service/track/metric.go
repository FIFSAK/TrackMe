package track

import (
	"TrackMe/internal/domain/client"
	"TrackMe/internal/domain/metric"
	"TrackMe/pkg/log"
	"TrackMe/pkg/store"
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

// ListMetrics retrieves all metric from the repository.
func (s *Service) ListMetrics(ctx context.Context, filters metric.Filters) ([]metric.Response, error) {
	logger := log.LoggerFromContext(ctx).With().
		Interface("filters", filters).
		Str("component", "service.client").
		Logger()

	var entities []metric.Entity
	var err error

	// Try to get metrics from cache if available
	if s.MetricCache != nil {
		entities, err = s.MetricCache.List(ctx, filters)
		if err == nil {
			logger.Debug().Msg("metrics retrieved from cache")
			responses := metric.ParseFromEntities(entities)
			return responses, nil
		}
		// Log cache miss but continue with repository
		logger.Debug().Err(err).Msg("cache miss, fetching from repository")
	}

	entities, err = s.MetricRepository.List(ctx, filters)
	if err != nil {
		logger.Error().Err(err).Msg("failed to list clients")
		return nil, err
	}

	if s.MetricCache != nil && len(entities) > 0 {
		go func(ctx context.Context, filters metric.Filters, entitiesToCache []metric.Entity) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			if err := s.MetricCache.StoreList(ctx, filters, entitiesToCache); err != nil {
				logger.Warn().Err(err).Msg("failed to update metrics list cache")
			}
		}(context.Background(), filters, entities)
	}

	responses := metric.ParseFromEntities(entities)
	return responses, nil
}

func (s *Service) CalculateAllMetrics(ctx context.Context, interval string) error {
	logger := log.LoggerFromContext(ctx).With().Str("component", "service.track.metric").Logger()
	now := time.Now()

	if err := s.calculateClientsPerStage(ctx, now); err != nil {
		logger.Error().Err(err).Msg("failed to calculate clients per stage")
		return err
	}

	if err := s.calculateStageDuration(ctx, now); err != nil {
		logger.Error().Err(err).Msg("failed to calculate stage duration")
		return err
	}

	if interval != "day" {
		if err := s.aggregateRollBackCount(ctx, now, interval); err != nil {
			logger.Error().Err(err).Msg("failed to aggregate rollback count")
			return err
		}
	}

	if err := s.calculateDropout(ctx, now, interval); err != nil {
		logger.Error().Err(err).Msg("failed to calculate dropout")
		return err
	}

	if err := s.calculateConversion(ctx, now, interval); err != nil {
		logger.Error().Err(err).Msg("failed to calculate conversion")
		return err
	}

	if err := s.calculateTotalDuration(ctx, now); err != nil {
		logger.Error().Err(err).Msg("failed to calculate total duration")
		return err
	}

	if err := s.calculateStatusUpdates(ctx, now, interval); err != nil {
		logger.Error().Err(err).Msg("failed to calculate status updates")
		return err
	}

	if interval == "day" {
		if err := s.calculateDAU(ctx, now); err != nil {
			logger.Error().Err(err).Msg("failed to calculate DAU")
			return err
		}
	}
	if interval == "month" {
		if err := s.calculateMAU(ctx, now); err != nil {
			logger.Error().Err(err).Msg("failed to calculate MAU")
			return err
		}
	}

	if err := s.calculateSourceConversion(ctx, now, interval); err != nil {
		logger.Error().Err(err).Msg("failed to calculate source conversion")
		return err
	}

	if err := s.calculateChannelConversion(ctx, now, interval); err != nil {
		logger.Error().Err(err).Msg("failed to calculate channel conversion")
		return err
	}

	if err := s.calculateAppInstallRate(ctx, now); err != nil {
		logger.Error().Err(err).Msg("failed to calculate app install rate")
		return err
	}

	if err := s.calculateAutoPaymentRate(ctx, now); err != nil {
		logger.Error().Err(err).Msg("failed to calculate auto payment rate")
		return err
	}

	// At the end of CalculateAllMetrics function
	if s.MetricCache != nil {
		// Define metric types and intervals that need invalidation
		metricsToInvalidate := []struct {
			Type     string
			Interval string
		}{
			{string(metric.RollbackCount), interval},
			{string(metric.ClientsPerStage), ""},
			{string(metric.StageDuration), ""},
			{string(metric.Dropout), interval},
			{string(metric.Conversion), interval},
			{string(metric.TotalDuration), ""},
			{string(metric.StatusUpdates), interval},
			{string(metric.SourceConversion), interval},
			{string(metric.ChannelConversion), interval},
			{string(metric.AppInstallRate), ""},
			{string(metric.AutoPaymentRate), ""},
		}

		// Add DAU/MAU based on interval
		if interval == "day" {
			metricsToInvalidate = append(metricsToInvalidate, struct {
				Type     string
				Interval string
			}{string(metric.DAU), "day"})
		} else if interval == "month" {
			metricsToInvalidate = append(metricsToInvalidate, struct {
				Type     string
				Interval string
			}{string(metric.MAU), "month"})
		}

		// Invalidate all affected caches
		for _, m := range metricsToInvalidate {
			if err := s.MetricCache.InvalidateListCache(ctx, metric.Filters{
				Type:     m.Type,
				Interval: m.Interval,
			}); err != nil {
				fromContext := log.LoggerFromContext(ctx)
				fromContext.Warn().
					Str("type", m.Type).
					Str("interval", m.Interval).
					Err(err).
					Msg("failed to invalidate metrics cache")
			}
		}
	}

	return nil
}

func (s *Service) calculateDAU(ctx context.Context, timestamp time.Time) error {
	startOfDay := time.Date(timestamp.Year(), timestamp.Month(), timestamp.Day(), 0, 0, 0, 0, timestamp.Location())

	count, err := s.clientRepository.Count(ctx, bson.M{
		"last_login.date": bson.M{
			"$gte": startOfDay,
		},
	})
	if err != nil {
		return err
	}
	metric, err := s.createMetric("", metric.DAU, float64(count), "day", timestamp, nil)
	if err != nil {
		return err
	}

	_, err = s.MetricRepository.Add(ctx, metric)

	if err != nil && !errors.Is(err, store.ErrorNotFound) {
		return err
	}

	s.PrometheusMetrics.DAU.Set(float64(count))

	return nil
}

func (s *Service) calculateMAU(ctx context.Context, timestamp time.Time) error {
	startOfMonth := time.Now().Add(-time.Hour * 24 * 30)

	count, err := s.clientRepository.Count(ctx, bson.M{
		"last_login.date": bson.M{
			"$gte": startOfMonth,
		},
	})
	if err != nil {
		return err
	}
	metric, err := s.createMetric("", metric.MAU, float64(count), "day", timestamp, nil)
	if err != nil {
		return err
	}
	_, err = s.MetricRepository.Add(ctx, metric)
	if err != nil && !errors.Is(err, store.ErrorNotFound) {
		return err
	}

	s.PrometheusMetrics.MAU.Set(float64(count))

	return nil
}

func (s *Service) calculateClientsPerStage(ctx context.Context, timestamp time.Time) error {
	stages, err := s.StageRepository.List(ctx)
	if err != nil {
		return err
	}

	for _, stage := range stages {
		count, err := s.clientRepository.Count(ctx, bson.M{
			"current_stage": stage.ID,
		})
		if err != nil {
			return err
		}
		metric, err := s.createMetric("", metric.ClientsPerStage, float64(count), "", timestamp, map[string]string{"stage": stage.ID})
		if err != nil {
			return err
		}
		_, err = s.MetricRepository.Add(ctx, metric)

		if err != nil {
			return err
		}
		s.PrometheusMetrics.ClientsPerStage.WithLabelValues(stage.ID).Set(float64(count))

	}

	return nil
}

func (s *Service) calculateStageDuration(ctx context.Context, timestamp time.Time) error {
	clients, _, err := s.clientRepository.List(ctx, client.Filters{IsActive: ptr(true)}, 0, 0)
	if err != nil {
		return err
	}

	stageDurations := make(map[string][]time.Duration)
	for _, c := range clients {
		if c.CurrentStage == nil || c.RegistrationDate == nil || c.LastUpdated == nil {
			continue
		}
		duration := timestamp.Sub(*c.LastUpdated)
		stageDurations[*c.CurrentStage] = append(stageDurations[*c.CurrentStage], duration)
	}

	for stageID, durations := range stageDurations {
		var total time.Duration
		for _, d := range durations {
			total += d
		}
		avgDuration := total / time.Duration(len(durations))

		metric, err := s.createMetric("", metric.StageDuration, avgDuration.Hours(), "", timestamp, map[string]string{"stage": stageID})
		if err != nil {
			return err
		}

		if _, err := s.MetricRepository.Add(ctx, metric); err != nil {
			return err
		}

		s.PrometheusMetrics.StageDuration.WithLabelValues(stageID).Set(avgDuration.Hours())

	}

	return nil
}

func (s *Service) aggregateRollBackCount(ctx context.Context, timestamp time.Time, interval string) error {
	logger := log.LoggerFromContext(ctx).With().
		Str("timestamp", timestamp.Format(time.RFC3339)).
		Str("interval", interval).
		Str("component", "service.track.metric").
		Logger()

	// Verify the interval is valid
	if interval != "week" && interval != "month" {
		return fmt.Errorf("invalid interval: %s (valid values: week, month)", interval)
	}

	var startTime, endTime time.Time

	if interval == "week" {
		// Get the start and end of the week
		year, week := timestamp.ISOWeek()
		startTime = firstDayOfISOWeek(year, week, timestamp.Location())
		endTime = startTime.AddDate(0, 0, 7)
	} else {
		// Get the start and end of the month
		startTime = time.Date(timestamp.Year(), timestamp.Month(), 1, 0, 0, 0, 0, timestamp.Location())
		endTime = startTime.AddDate(0, 1, 0)
	}

	logger.Info().
		Time("startTime", startTime).
		Time("endTime", endTime).
		Msg("Aggregating rollback count")

	// Get all rollback count metrics for the period
	dailyMetrics, err := s.MetricRepository.List(ctx, metric.Filters{
		Type:     string(metric.RollbackCount),
		Interval: "day",
	})
	if err != nil && !errors.Is(err, store.ErrorNotFound) {
		return fmt.Errorf("failed to get rollback count metrics: %w", err)
	}

	// Filter metrics for the period
	periodMetrics := make([]metric.Entity, 0)
	for _, m := range dailyMetrics {
		if !m.CreatedAt.Before(startTime) && m.CreatedAt.Before(endTime) {
			periodMetrics = append(periodMetrics, m)
		}
	}

	var totalRollbacks float64
	for _, m := range periodMetrics {
		totalRollbacks += *m.Value
	}

	logger.Info().
		Int("metrics_count", len(periodMetrics)).
		Float64("total_rollbacks", totalRollbacks).
		Msg("Rollback count aggregation results")

	// Create and store aggregated metric
	aggregatedMetric, err := s.createMetric("",
		metric.RollbackCount,
		totalRollbacks,
		interval,
		endTime,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create aggregated rollback count metric: %w", err)
	}

	// Check if we already have a metric for this period
	existingMetrics, err := s.MetricRepository.List(ctx, metric.Filters{
		Type:     string(metric.RollbackCount),
		Interval: interval,
	})
	if err != nil && !errors.Is(err, store.ErrorNotFound) {
		return fmt.Errorf("failed to check existing aggregated metrics: %w", err)
	}

	var found bool
	for _, m := range existingMetrics {
		var isInSamePeriod bool

		if interval == "week" {
			existingYear, existingWeek := m.CreatedAt.ISOWeek()
			currentYear, currentWeek := timestamp.ISOWeek()
			isInSamePeriod = existingYear == currentYear && existingWeek == currentWeek
		} else {
			isInSamePeriod = m.CreatedAt.Year() == timestamp.Year() &&
				m.CreatedAt.Month() == timestamp.Month()
		}

		if isInSamePeriod {
			found = true
			if _, err := s.MetricRepository.Update(ctx, m.ID, aggregatedMetric); err != nil {
				return fmt.Errorf("failed to update existing %s rollback count metric: %w", interval, err)
			}
			logger.Info().Msgf("Updated existing %s rollback count metric", interval)
			break
		}
	}

	if !found {
		if _, err := s.MetricRepository.Add(ctx, aggregatedMetric); err != nil {
			return fmt.Errorf("failed to store %s rollback count metric: %w", interval, err)
		}
		logger.Info().Msgf("Created new %s rollback count metric", interval)
	}
	return nil
}

// Helper function to get the first day of an ISO week
func firstDayOfISOWeek(year, week int, loc *time.Location) time.Time {
	// Get January 1 for the year
	jan1 := time.Date(year, 1, 1, 0, 0, 0, 0, loc)

	// Get the day of the week for January 1
	dayOfWeek := int(jan1.Weekday())
	if dayOfWeek == 0 {
		// Sunday is 0, but ISO considers it 7
		dayOfWeek = 7
	}

	// Days to add to get to the Monday of week 1
	daysToAdd := 1 - dayOfWeek

	// Monday of the first week
	firstMonday := jan1.AddDate(0, 0, daysToAdd)

	// If January 1 is after Thursday, it's part of week 1
	// If not, it's part of the last week of the previous year
	if dayOfWeek > 4 {
		firstMonday = firstMonday.AddDate(0, 0, 7)
	}

	// Add the required number of weeks
	return firstMonday.AddDate(0, 0, 7*(week-1))
}

func (s *Service) calculateRollbackCount(ctx context.Context, timestamp time.Time) error {
	fromContext := log.LoggerFromContext(ctx)
	todayDate := timestamp.Format("2006-01-02")
	rollBackCountMetrics, err := s.ListMetrics(ctx, metric.Filters{
		Type:     string(metric.RollbackCount),
		Interval: "day",
	})
	if err != nil && !errors.Is(err, store.ErrorNotFound) {
		return err
	}

	var found bool
	metricType := metric.RollbackCount

	for _, m := range rollBackCountMetrics {
		if m.CreatedAt.Format("2006-01-02") == todayDate {
			found = true
			newValue := m.Value + 1.0

			interval := "day"
			// Create a new entity without ID - the ID will be set by MongoDB update operation
			updated := metric.Entity{
				ID:        m.ID,
				Type:      &metricType,
				Value:     &newValue,
				Interval:  &interval,
				CreatedAt: &timestamp,
				Metadata:  nil,
			}

			if _, err = s.MetricRepository.Update(ctx, updated.ID, updated); err != nil {
				return fmt.Errorf("failed to update rollback count: %w", err)
			}

			// Invalidate the cached list so future queries get fresh data
			if s.MetricCache != nil {
				if err = s.MetricCache.InvalidateListCache(ctx, metric.Filters{
					Type:     string(metric.RollbackCount),
					Interval: "day",
				}); err != nil {
					// Log error but don't fail the operation
					fromContext.Warn().Err(err).Msg("failed to invalidate metrics cache")
				}
			}

			s.PrometheusMetrics.RollbackCount.Inc()

			return nil
		}
	}

	if !found {
		newMetric, err := s.createMetric("", metricType, 1.0, "day", timestamp, nil)
		if err != nil {
			return fmt.Errorf("failed to create rollback metric: %w", err)
		}

		if _, err = s.MetricRepository.Add(ctx, newMetric); err != nil {
			return fmt.Errorf("failed to store rollback metric: %w", err)
		}

		// Invalidate the cached list after adding a new metric
		if s.MetricCache != nil {
			if err = s.MetricCache.InvalidateListCache(ctx, metric.Filters{
				Type:     string(metric.RollbackCount),
				Interval: "day",
			}); err != nil {
				fromContext.Warn().Err(err).Msg("failed to invalidate metrics cache")
			}
		}
	}

	s.PrometheusMetrics.RollbackCount.Inc()

	return nil
}

func (s *Service) calculateDropout(ctx context.Context, timestamp time.Time, interval string) error {
	var inactivePeriod int
	switch interval {
	case "week":
		inactivePeriod = 7
	case "month":
		inactivePeriod = 30
	}

	cutoffDate := timestamp.AddDate(0, 0, -inactivePeriod)

	count, err := s.clientRepository.Count(ctx, bson.M{
		"last_updated": bson.M{"$lt": cutoffDate},
		"is_active":    true,
	})
	if err != nil {
		return err
	}

	m, err := s.createMetric("", metric.Dropout, float64(count), interval, timestamp, nil)

	_, err = s.MetricRepository.Add(ctx, m)
	if err != nil && !errors.Is(err, store.ErrorNotFound) {
		return err
	}
	s.PrometheusMetrics.Dropout.Set(float64(count))

	return err
}

func (s *Service) calculateConversion(ctx context.Context, timestamp time.Time, interval string) error {
	logger := log.LoggerFromContext(ctx).With().Str("component", "service.track.metric.conversion").Logger()

	stages, err := s.StageRepository.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list stages: %w", err)
	}

	if len(stages) < 1 {
		return nil
	}

	lastStage := stages[len(stages)-1].ID

	// Calculate time period based on interval
	var startDate time.Time
	switch interval {
	case "day":
		startDate = time.Date(timestamp.Year(), timestamp.Month(), timestamp.Day(), 0, 0, 0, 0, timestamp.Location())
	case "week":
		year, week := timestamp.ISOWeek()
		startDate = firstDayOfISOWeek(year, week, timestamp.Location())
	case "month":
		startDate = time.Date(timestamp.Year(), timestamp.Month(), 1, 0, 0, 0, 0, timestamp.Location())
	default:
		return fmt.Errorf("invalid interval: %s", interval)
	}

	logger.Info().
		Time("start_date", startDate).
		Time("end_date", timestamp).
		Str("interval", interval).
		Msg("Calculating conversion rate")

	// Clients who reached last stage within the interval period
	lastStageRecentCount, err := s.clientRepository.Count(ctx, bson.M{
		"current_stage": lastStage,
		"last_updated": bson.M{
			"$gte": startDate,
			"$lte": timestamp,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to count clients on last stage with recent updates: %w", err)
	}

	// Total number of clients active in this period
	totalClientsCount, err := s.clientRepository.Count(ctx, bson.M{
		"last_updated": bson.M{
			"$gte": startDate,
			"$lte": timestamp,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to count total active clients: %w", err)
	}

	var conversionRate float64
	if totalClientsCount > 0 {
		conversionRate = float64(lastStageRecentCount) / float64(totalClientsCount)
	}

	logger.Info().
		Int64("last_stage_count", lastStageRecentCount).
		Int64("total_count", totalClientsCount).
		Float64("conversion_rate", conversionRate).
		Msg("Conversion calculation results")

	m, err := s.createMetric("",
		metric.Conversion,
		conversionRate,
		interval,
		timestamp,
		nil,
	)
	if err != nil {
		return err
	}

	if _, err = s.MetricRepository.Add(ctx, m); err != nil {
		return err
	}

	s.PrometheusMetrics.Conversion.Set(conversionRate)

	return nil
}

func (s *Service) calculateTotalDuration(ctx context.Context, timestamp time.Time) error {
	stages, err := s.StageRepository.List(ctx)
	if err != nil {
		return err
	}

	if len(stages) == 0 {
		return nil
	}

	lastStage := stages[len(stages)-1].ID

	clients, _, err := s.clientRepository.List(ctx, client.Filters{
		Stage:    lastStage,
		IsActive: ptr(true),
	}, 0, 0)
	if err != nil {
		return err
	}

	var totalDuration time.Duration
	var count int

	for _, c := range clients {
		if c.RegistrationDate != nil && c.LastUpdated != nil {
			totalDuration += c.LastUpdated.Sub(*c.RegistrationDate)
			count++
		}
	}

	var avgDurationDays float64
	if count > 0 {
		avgDurationDays = totalDuration.Hours() / 24 / float64(count)
	}

	m, err := s.createMetric("", metric.TotalDuration, avgDurationDays, "", timestamp, nil)

	_, err = s.MetricRepository.Add(ctx, m)
	if err != nil && !errors.Is(err, store.ErrorNotFound) {
		return err
	}

	s.PrometheusMetrics.TotalDuration.Set(avgDurationDays)

	return nil
}

func (s *Service) calculateStatusUpdates(ctx context.Context, timestamp time.Time, interval string) error {
	logger := log.LoggerFromContext(ctx).With().Str("component", "service.track.metric.status_updates").Logger()

	// Calculate time period based on interval
	var startDate time.Time
	switch interval {
	case "day":
		startDate = time.Date(timestamp.Year(), timestamp.Month(), timestamp.Day(), 0, 0, 0, 0, timestamp.Location())
	case "week":
		year, week := timestamp.ISOWeek()
		startDate = firstDayOfISOWeek(year, week, timestamp.Location())
	case "month":
		startDate = time.Date(timestamp.Year(), timestamp.Month(), 1, 0, 0, 0, 0, timestamp.Location())
	default:
		return fmt.Errorf("invalid interval: %s", interval)
	}

	logger.Info().
		Time("start_date", startDate).
		Time("end_date", timestamp).
		Str("interval", interval).
		Msg("Calculating status updates")

	// Count clients updated within the specified period
	count, err := s.clientRepository.Count(ctx, bson.M{
		"last_updated": bson.M{
			"$gte": startDate,
			"$lte": timestamp,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to count status updates: %w", err)
	}

	logger.Info().
		Int64("count", count).
		Str("interval", interval).
		Msg("Status updates calculation results")

	// Create and store the metric
	m, err := s.createMetric("", metric.StatusUpdates, float64(count), interval, timestamp, nil)
	if err != nil {
		return fmt.Errorf("failed to create status updates metric: %w", err)
	}

	if _, err = s.MetricRepository.Add(ctx, m); err != nil {
		return fmt.Errorf("failed to store status updates metric: %w", err)
	}

	s.PrometheusMetrics.StatusUpdates.Set(float64(count))

	return nil
}

func (s *Service) calculateSourceConversion(ctx context.Context, timestamp time.Time, interval string) error {
	logger := log.LoggerFromContext(ctx).With().Str("component", "service.track.metric.source_conversion").Logger()

	stages, err := s.StageRepository.List(ctx)
	if err != nil {
		return err
	}

	if len(stages) < 2 {
		return nil
	}

	lastStage := stages[len(stages)-1].ID

	// Calculate time period based on interval
	var startDate time.Time
	switch interval {
	case "day":
		startDate = time.Date(timestamp.Year(), timestamp.Month(), timestamp.Day(), 0, 0, 0, 0, timestamp.Location())
	case "week":
		year, week := timestamp.ISOWeek()
		startDate = firstDayOfISOWeek(year, week, timestamp.Location())
	case "month":
		startDate = time.Date(timestamp.Year(), timestamp.Month(), 1, 0, 0, 0, 0, timestamp.Location())
	default:
		return err
	}

	logger.Info().
		Time("start_date", startDate).
		Time("end_date", timestamp).
		Str("interval", interval).
		Msg("Calculating source conversion")

	// Get clients active within the time period
	clients, _, err := s.clientRepository.List(ctx, client.Filters{}, 0, 0)
	if err != nil {
		return err
	}

	// Collect unique sources
	sourceMap := make(map[string]bool)
	for _, c := range clients {
		if c.Source != nil && *c.Source != "" && c.LastUpdated != nil {
			if !c.LastUpdated.Before(startDate) && !c.LastUpdated.After(timestamp) {
				sourceMap[*c.Source] = true
			}
		}
	}

	sources := make([]string, 0, len(sourceMap))
	for source := range sourceMap {
		sources = append(sources, source)
	}

	for _, source := range sources {
		// Count total clients from this source active in the period
		total, err := s.clientRepository.Count(ctx, bson.M{
			"source": source,
			"last_updated": bson.M{
				"$gte": startDate,
				"$lte": timestamp,
			},
		})
		if err != nil {
			return err
		}

		// Count completed clients from this source active in the period
		completed, err := s.clientRepository.Count(ctx, bson.M{
			"source":        source,
			"current_stage": lastStage,
			"last_updated": bson.M{
				"$gte": startDate,
				"$lte": timestamp,
			},
		})
		if err != nil {
			return err
		}

		var conversionRate float64
		if total > 0 {
			conversionRate = float64(completed) / float64(total)
		}

		logger.Info().
			Str("source", source).
			Int64("total", total).
			Int64("completed", completed).
			Float64("conversion_rate", conversionRate).
			Msg("Source conversion calculation results")

		m, err := s.createMetric("",
			metric.SourceConversion,
			conversionRate,
			interval,
			timestamp,
			map[string]string{"source": source},
		)
		if err != nil {
			return fmt.Errorf("failed to create source conversion metric: %w", err)
		}

		if _, err := s.MetricRepository.Add(ctx, m); err != nil {
			return fmt.Errorf("failed to store source conversion metric: %w", err)
		}

		s.PrometheusMetrics.SourceConversion.WithLabelValues(source).Set(conversionRate)

	}

	return nil
}

// calculateSourceConversion calculates the conversion rate for each source
func (s *Service) calculateChannelConversion(ctx context.Context, timestamp time.Time, interval string) error {
	logger := log.LoggerFromContext(ctx).With().Str("component", "service.track.metric.channel_conversion").Logger()

	stages, err := s.StageRepository.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list stages: %w", err)
	}

	if len(stages) < 2 {
		return nil
	}

	lastStage := stages[len(stages)-1].ID

	// Calculate time period based on interval
	var startDate time.Time
	switch interval {
	case "day":
		startDate = time.Date(timestamp.Year(), timestamp.Month(), timestamp.Day(), 0, 0, 0, 0, timestamp.Location())
	case "week":
		year, week := timestamp.ISOWeek()
		startDate = firstDayOfISOWeek(year, week, timestamp.Location())
	case "month":
		startDate = time.Date(timestamp.Year(), timestamp.Month(), 1, 0, 0, 0, 0, timestamp.Location())
	default:
		return fmt.Errorf("invalid interval: %s", interval)
	}

	logger.Info().
		Time("start_date", startDate).
		Time("end_date", timestamp).
		Str("interval", interval).
		Msg("Calculating channel conversion")

	// Get clients active within the time period
	clients, _, err := s.clientRepository.List(ctx, client.Filters{}, 0, 0)
	if err != nil {
		return fmt.Errorf("failed to list clients: %w", err)
	}

	// Collect unique channels from active clients in this period
	channelMap := make(map[string]bool)
	for _, c := range clients {
		if c.Channel != nil && *c.Channel != "" && c.LastUpdated != nil {
			if !c.LastUpdated.Before(startDate) && !c.LastUpdated.After(timestamp) {
				channelMap[*c.Channel] = true
			}
		}
	}

	channels := make([]string, 0, len(channelMap))
	for channel := range channelMap {
		channels = append(channels, channel)
	}

	for _, channel := range channels {
		// Count total clients from this channel active in the period
		total, err := s.clientRepository.Count(ctx, bson.M{
			"channel": channel,
			"last_updated": bson.M{
				"$gte": startDate,
				"$lte": timestamp,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to count clients with channel %s: %w", channel, err)
		}

		// Count completed clients from this channel active in the period
		completed, err := s.clientRepository.Count(ctx, bson.M{
			"channel":       channel,
			"current_stage": lastStage,
			"last_updated": bson.M{
				"$gte": startDate,
				"$lte": timestamp,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to count completed clients with channel %s: %w", channel, err)
		}

		var conversionRate float64
		if total > 0 {
			conversionRate = float64(completed) / float64(total)
		}

		logger.Info().
			Str("channel", channel).
			Int64("total", total).
			Int64("completed", completed).
			Float64("conversion_rate", conversionRate).
			Msg("Channel conversion calculation results")

		m, err := s.createMetric("",
			metric.ChannelConversion,
			conversionRate,
			interval,
			timestamp,
			map[string]string{"channel": channel},
		)
		if err != nil {
			return fmt.Errorf("failed to create channel conversion metric: %w", err)
		}

		if _, err := s.MetricRepository.Add(ctx, m); err != nil {
			return fmt.Errorf("failed to store channel conversion metric: %w", err)
		}
	}

	return nil
}

func (s *Service) calculateAppInstallRate(ctx context.Context, timestamp time.Time) error {
	// Общее количество клиентов с указанным статусом app
	total, err := s.clientRepository.Count(ctx, bson.M{"app": bson.M{"$exists": true}})
	if err != nil {
		return err
	}

	// Количество клиентов с установленным приложением
	installed, err := s.clientRepository.Count(ctx, bson.M{"app": "installed"})
	if err != nil {
		return err
	}

	var installRate float64
	if total > 0 {
		installRate = float64(installed) / float64(total)
	}

	m, err := s.createMetric("", metric.AppInstallRate, installRate, "", timestamp, nil)
	if err != nil {
		return err
	}

	_, err = s.MetricRepository.Add(ctx, m)
	if err != nil && !errors.Is(err, store.ErrorNotFound) {
		return err
	}
	s.PrometheusMetrics.AppInstallRate.Set(installRate)

	return nil
}

func (s *Service) calculateAutoPaymentRate(ctx context.Context, timestamp time.Time) error {
	// Получаем всех клиентов с договорами
	clients, _, err := s.clientRepository.List(ctx, client.Filters{}, 0, 0)
	if err != nil {
		return err
	}

	var totalContracts int
	var enabledContracts int

	for _, c := range clients {
		for _, contract := range c.Contracts {
			totalContracts++
			if contract.AutoPayment != nil && *contract.AutoPayment == "enabled" {
				enabledContracts++
			}
		}
	}

	var autopaymentRate float64
	if totalContracts > 0 {
		autopaymentRate = float64(enabledContracts) / float64(totalContracts)
	}

	m, err := s.createMetric("", metric.AutoPaymentRate, autopaymentRate, "", timestamp, nil)
	if err != nil {
		return err
	}

	_, err = s.MetricRepository.Add(ctx, m)
	if err != nil && !errors.Is(err, store.ErrorNotFound) {
		return err
	}

	s.PrometheusMetrics.AutoPaymentRate.Set(autopaymentRate)

	return err
}

func ptr(b bool) *bool {
	return &b
}

func (s *Service) createMetric(id string, metricType metric.Type, value float64, interval string, timestamp time.Time, metaData map[string]string) (metric.Entity, error) {

	if id == "" {
		id = primitive.NewObjectID().Hex()
	}

	return metric.Entity{
		ID:        id,
		Type:      &metricType,
		Value:     &value,
		Interval:  &interval,
		CreatedAt: &timestamp,
		Metadata:  metaData,
	}, nil
}

package track

import (
	"TrackMe/internal/domain/client"
	"TrackMe/internal/domain/metric"
	"TrackMe/pkg/store"
	"context"
	"errors"
	"fmt"
	"github.com/gofrs/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
	"time"
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

// AggregateWeeklyMetrics aggregates daily metrics into weekly metrics
func (s *Service) AggregateWeeklyMetrics(ctx context.Context, timestamp time.Time) error {
	logger := zap.L().Named("service.track.metric.weekly")

	year, week := timestamp.ISOWeek()
	startOfWeek := firstDayOfISOWeek(year, week, timestamp.Location())
	endOfWeek := startOfWeek.AddDate(0, 0, 7)

	logger.Info("Aggregating weekly metrics",
		zap.Time("startOfWeek", startOfWeek),
		zap.Time("endOfWeek", endOfWeek))

	dailyMetrics, err := s.MetricRepository.List(ctx, metric.Filters{
		Interval: "day",
	})
	if err != nil {
		return fmt.Errorf("failed to get daily metrics for weekly aggregation: %w", err)
	}

	weekMetrics := make([]metric.Entity, 0)
	for _, m := range dailyMetrics {
		if m.CreatedAt != nil && !m.CreatedAt.Before(startOfWeek) && m.CreatedAt.Before(endOfWeek) {
			weekMetrics = append(weekMetrics, m)
		}
	}

	groupedMetrics := make(map[string][]metric.Entity)
	for _, m := range weekMetrics {
		key := string(*m.Type)

		if m.Metadata != nil {
			for k, v := range m.Metadata {
				key += ":" + k + ":" + v
			}
		}

		groupedMetrics[key] = append(groupedMetrics[key], m)
	}

	for _, metrics := range groupedMetrics {
		var total float64
		var count int
		var metaData map[string]string
		var metricType metric.Type

		if len(metrics) > 0 {
			metricType = *metrics[0].Type
			metaData = metrics[0].Metadata
		}

		for _, m := range metrics {
			if m.Value != nil {
				total += *m.Value
				count++
			}
		}

		var weeklyValue float64
		if count > 0 {
			weeklyValue = total / float64(count)
		}

		weeklyMetric, err := s.createMetric(
			metricType,
			weeklyValue,
			"week",
			endOfWeek,
			metaData,
		)
		if err != nil {
			logger.Error("Failed to create weekly metric",
				zap.String("type", string(metricType)),
				zap.Error(err))
			continue
		}

		if _, err := s.MetricRepository.Add(ctx, weeklyMetric); err != nil {
			logger.Error("Failed to store weekly metric",
				zap.String("type", string(metricType)),
				zap.Error(err))
		}
	}

	return nil
}

// AggregateMonthlyMetrics aggregates daily metrics into monthly metrics
func (s *Service) AggregateMonthlyMetrics(ctx context.Context, timestamp time.Time) error {
	logger := zap.L().Named("service.track.metric.monthly")

	startOfMonth := time.Date(timestamp.Year(), timestamp.Month(), 1, 0, 0, 0, 0, timestamp.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	logger.Info("Aggregating monthly metrics",
		zap.Time("startOfMonth", startOfMonth),
		zap.Time("endOfMonth", endOfMonth))

	dailyMetrics, err := s.MetricRepository.List(ctx, metric.Filters{
		Interval: "day",
	})
	if err != nil {
		return fmt.Errorf("failed to get daily metrics for monthly aggregation: %w", err)
	}

	// Filter metrics from this month manually
	monthMetrics := make([]metric.Entity, 0)
	for _, m := range dailyMetrics {
		if m.CreatedAt != nil && !m.CreatedAt.Before(startOfMonth) && m.CreatedAt.Before(endOfMonth) {
			monthMetrics = append(monthMetrics, m)
		}
	}

	groupedMetrics := make(map[string][]metric.Entity)
	for _, m := range monthMetrics {
		key := string(*m.Type)

		if m.Metadata != nil {
			for k, v := range m.Metadata {
				key += ":" + k + ":" + v
			}
		}

		groupedMetrics[key] = append(groupedMetrics[key], m)
	}

	for _, metrics := range groupedMetrics {
		var total float64
		var count int
		var metaData map[string]string
		var metricType metric.Type

		if len(metrics) > 0 {
			metricType = *metrics[0].Type
			metaData = metrics[0].Metadata
		}

		for _, m := range metrics {
			if m.Value != nil {
				total += *m.Value
				count++
			}
		}

		var monthlyValue float64
		if count > 0 {
			monthlyValue = total / float64(count)
		}

		monthlyMetric, err := s.createMetric(
			metricType,
			monthlyValue,
			"month",
			endOfMonth,
			metaData,
		)
		if err != nil {
			logger.Error("Failed to create monthly metric",
				zap.String("type", string(metricType)),
				zap.Error(err))
			continue
		}

		if _, err := s.MetricRepository.Add(ctx, monthlyMetric); err != nil {
			logger.Error("Failed to store monthly metric",
				zap.String("type", string(metricType)),
				zap.Error(err))
		}
	}

	return nil
}

func firstDayOfISOWeek(year, week int, loc *time.Location) time.Time {
	jan1 := time.Date(year, 1, 1, 0, 0, 0, 0, loc)

	dayOfWeek := int(jan1.Weekday())
	if dayOfWeek == 0 {
		dayOfWeek = 7
	}

	daysToAdd := 1 - dayOfWeek
	firstMonday := jan1.AddDate(0, 0, daysToAdd)

	if dayOfWeek > 4 {
		firstMonday = firstMonday.AddDate(0, 0, 7)
	}

	return firstMonday.AddDate(0, 0, 7*(week-1))
}

func (s *Service) CalculateAllMetrics(ctx context.Context) error {
	logger := zap.L().Named("service.track.metric")
	now := time.Now()
	inactivePeriod := 30

	// Существующие метрики
	if err := s.calculateDAU(ctx, now); err != nil {
		logger.Error("failed to calculate DAU", zap.Error(err))
		return err
	}

	if err := s.calculateMAU(ctx, now); err != nil {
		logger.Error("failed to calculate MAU", zap.Error(err))
		return err
	}

	if err := s.calculateClientsPerStage(ctx, now); err != nil {
		logger.Error("failed to calculate clients per stage", zap.Error(err))
		return err
	}

	if err := s.calculateStageDuration(ctx, now); err != nil {
		logger.Error("failed to calculate stage duration", zap.Error(err))
		return err
	}

	if err := s.calculateDropout(ctx, now, inactivePeriod); err != nil {
		logger.Error("failed to calculate dropout", zap.Error(err))
		return err
	}

	if err := s.calculateConversion(ctx, now); err != nil {
		logger.Error("failed to calculate conversion", zap.Error(err))
		return err
	}

	if err := s.calculateTotalDuration(ctx, now); err != nil {
		logger.Error("failed to calculate total duration", zap.Error(err))
		return err
	}

	if err := s.calculateStatusUpdates(ctx, now); err != nil {
		logger.Error("failed to calculate status updates", zap.Error(err))
		return err
	}

	if err := s.calculateSourceConversion(ctx, now); err != nil {
		logger.Error("failed to calculate source conversion", zap.Error(err))
		return err
	}

	if err := s.calculateChannelConversion(ctx, now); err != nil {
		logger.Error("failed to calculate channel conversion", zap.Error(err))
		return err
	}

	if err := s.calculateAppInstallRate(ctx, now); err != nil {
		logger.Error("failed to calculate app install rate", zap.Error(err))
		return err
	}

	if err := s.calculateAutoPaymentRate(ctx, now); err != nil {
		logger.Error("failed to calculate autopayment rate", zap.Error(err))
		return err
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
	metric, err := s.createMetric(metric.DAU, float64(count), "day", timestamp, nil)
	if err != nil {
		return err
	}

	_, err = s.MetricRepository.Add(ctx, metric)

	if err != nil && !errors.Is(err, store.ErrorNotFound) {
		return err
	}

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
	metric, err := s.createMetric(metric.MAU, float64(count), "day", timestamp, nil)
	if err != nil {
		return err
	}
	_, err = s.MetricRepository.Add(ctx, metric)
	if err != nil && !errors.Is(err, store.ErrorNotFound) {
		return err
	}
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
		metric, err := s.createMetric(metric.ClientsPerStage, float64(count), "day", timestamp, map[string]string{"stage": stage.ID})
		if err != nil {
			return err
		}
		_, err = s.MetricRepository.Add(ctx, metric)

		if err != nil {
			return err
		}
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

		metric, err := s.createMetric(metric.StageDuration, avgDuration.Hours(), "day", timestamp, map[string]string{"stage": stageID})
		if err != nil {
			return err
		}

		if _, err := s.MetricRepository.Add(ctx, metric); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) calculateRollbackCount(ctx context.Context, timestamp time.Time) error {
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

			updated, _ := s.createMetric(metricType, m.Value+1, "day", timestamp, nil)

			if _, err = s.MetricRepository.Update(ctx, m.ID, updated); err != nil {
				return fmt.Errorf("failed to update rollback count: %w", err)
			}
			return nil
		}
	}

	if !found {
		newMetric, err := s.createMetric(metricType, 1.0, "day", timestamp, nil)
		if err != nil {
			return fmt.Errorf("failed to create rollback metric: %w", err)
		}

		if _, err := s.MetricRepository.Add(ctx, newMetric); err != nil {
			return fmt.Errorf("failed to store rollback metric: %w", err)
		}
	}

	return nil
}

func (s *Service) calculateDropout(ctx context.Context, timestamp time.Time, inactivePeriod int) error {
	cutoffDate := timestamp.AddDate(0, 0, -inactivePeriod)

	count, err := s.clientRepository.Count(ctx, bson.M{
		"last_updated": bson.M{"$lt": cutoffDate},
		"is_active":    true,
	})
	if err != nil {
		return err
	}

	m, err := s.createMetric(metric.Dropout, float64(count), "day", timestamp, nil)

	_, err = s.MetricRepository.Add(ctx, m)
	return err
}

func (s *Service) calculateConversion(ctx context.Context, timestamp time.Time) error {
	stages, err := s.StageRepository.List(ctx)
	if err != nil {
		return err
	}

	if len(stages) < 1 {
		return nil
	}

	lastStage := stages[len(stages)-1].ID

	lastStageCount, err := s.clientRepository.Count(ctx, bson.M{"current_stage": lastStage})
	if err != nil {
		return err
	}

	totalClientsCount, err := s.clientRepository.Count(ctx, bson.M{})
	if err != nil {
		return err
	}

	var conversionRate float64
	if totalClientsCount > 0 {
		conversionRate = float64(lastStageCount) / float64(totalClientsCount)
	}

	m, err := s.createMetric(metric.Conversion, conversionRate, "day", timestamp, nil)

	_, err = s.MetricRepository.Add(ctx, m)
	return err
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

	m, err := s.createMetric(metric.TotalDuration, avgDurationDays, "day", timestamp, nil)

	_, err = s.MetricRepository.Add(ctx, m)
	return err
}

func (s *Service) calculateStatusUpdates(ctx context.Context, timestamp time.Time) error {
	startOfDay := time.Date(timestamp.Year(), timestamp.Month(), timestamp.Day(), 0, 0, 0, 0, timestamp.Location())
	endOfDay := startOfDay.AddDate(0, 0, 1)

	count, err := s.clientRepository.Count(ctx, bson.M{
		"last_updated": bson.M{
			"$gte": startOfDay,
			"$lt":  endOfDay,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to count status updates: %w", err)
	}

	m, err := s.createMetric(metric.StatusUpdates, float64(count), "day", timestamp, nil)
	if err != nil {
		return fmt.Errorf("failed to create status updates metric: %w", err)
	}

	if _, err = s.MetricRepository.Add(ctx, m); err != nil {
		return fmt.Errorf("failed to store status updates metric: %w", err)
	}

	return nil
}

func (s *Service) calculateSourceConversion(ctx context.Context, timestamp time.Time) error {
	stages, err := s.StageRepository.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list stages: %w", err)
	}

	if len(stages) < 2 {
		return nil
	}

	lastStage := stages[len(stages)-1].ID

	clients, _, err := s.clientRepository.List(ctx, client.Filters{}, 0, 0)
	if err != nil {
		return fmt.Errorf("failed to list clients: %w", err)
	}

	sourceMap := make(map[string]bool)
	for _, c := range clients {
		if c.Source != nil && *c.Source != "" {
			sourceMap[*c.Source] = true
		}
	}

	sources := make([]string, 0, len(sourceMap))
	for source := range sourceMap {
		sources = append(sources, source)
	}

	for _, source := range sources {
		total, err := s.clientRepository.Count(ctx, bson.M{"source": source})
		if err != nil {
			return fmt.Errorf("failed to count clients with source %s: %w", source, err)
		}

		completed, err := s.clientRepository.Count(ctx, bson.M{
			"source":        source,
			"current_stage": lastStage,
		})
		if err != nil {
			return fmt.Errorf("failed to count completed clients with source %s: %w", source, err)
		}

		var conversionRate float64
		if total > 0 {
			conversionRate = float64(completed) / float64(total)
		}

		m, err := s.createMetric(
			metric.SourceConversion,
			conversionRate,
			"day",
			timestamp,
			map[string]string{"source": source},
		)
		if err != nil {
			return fmt.Errorf("failed to create source conversion metric: %w", err)
		}

		if _, err := s.MetricRepository.Add(ctx, m); err != nil {
			return fmt.Errorf("failed to store source conversion metric: %w", err)
		}
	}

	return nil
}

func (s *Service) calculateChannelConversion(ctx context.Context, timestamp time.Time) error {
	stages, err := s.StageRepository.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list stages: %w", err)
	}

	if len(stages) < 2 {
		return nil
	}

	lastStage := stages[len(stages)-1].ID

	clients, _, err := s.clientRepository.List(ctx, client.Filters{}, 0, 0)
	if err != nil {
		return fmt.Errorf("failed to list clients: %w", err)
	}

	channelMap := make(map[string]bool)
	for _, c := range clients {
		if c.Channel != nil && *c.Channel != "" {
			channelMap[*c.Channel] = true
		}
	}

	channels := make([]string, 0, len(channelMap))
	for channel := range channelMap {
		channels = append(channels, channel)
	}

	for _, channel := range channels {
		total, err := s.clientRepository.Count(ctx, bson.M{"channel": channel})
		if err != nil {
			return fmt.Errorf("failed to count clients with channel %s: %w", channel, err)
		}

		completed, err := s.clientRepository.Count(ctx, bson.M{
			"channel":       channel,
			"current_stage": lastStage,
		})
		if err != nil {
			return fmt.Errorf("failed to count completed clients with channel %s: %w", channel, err)
		}

		var conversionRate float64
		if total > 0 {
			conversionRate = float64(completed) / float64(total)
		}

		m, err := s.createMetric(
			metric.ChannelConversion,
			conversionRate,
			"day",
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

	m, err := s.createMetric(metric.AppInstallRate, installRate, "day", timestamp, nil)
	if err != nil {
		return err
	}

	_, err = s.MetricRepository.Add(ctx, m)
	return err
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

	m, err := s.createMetric(metric.AutoPaymentRate, autopaymentRate, "day", timestamp, nil)
	if err != nil {
		return err
	}

	_, err = s.MetricRepository.Add(ctx, m)
	return err
}

func ptr(b bool) *bool {
	return &b
}

func (s *Service) createMetric(metricType metric.Type, value float64, interval string, timestamp time.Time, metaData map[string]string) (metric.Entity, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return metric.Entity{}, err
	}

	return metric.Entity{
		ID:        id.String(),
		Type:      &metricType,
		Value:     &value,
		Interval:  &interval,
		CreatedAt: &timestamp,
		Metadata:  metaData,
	}, nil
}

// internal/worker/metric.go

package worker

import (
	"TrackMe/internal/service/track"
	"TrackMe/pkg/log"
	"context"
	"github.com/robfig/cron/v3"
	"sync"
	"time"
)

// MetricWorker handles scheduled metric calculations
type MetricWorker struct {
	trackService *track.Service
	cron         *cron.Cron
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewMetricWorker creates a new metric worker
func NewMetricWorker(trackService *track.Service) *MetricWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &MetricWorker{
		trackService: trackService,
		cron:         cron.New(cron.WithSeconds()),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start begins the background metric calculation process
func (w *MetricWorker) Start() {
	logger := log.LoggerFromContext(w.ctx).With().Str("component", "worker.metric").Logger()
	logger.Info().Msg("Starting metric worker")

	// Run daily calculations at midnight
	_, err := w.cron.AddFunc("0 0 0 * * *", func() {
		logger.Info().Msg("Running daily metric calculations")
		logger.Info().Msg("Running daily metric calculations")
		w.wg.Add(1)
		defer w.wg.Done()

		ctx, cancel := context.WithTimeout(w.ctx, 5*time.Minute)
		defer cancel()

		if err := w.trackService.CalculateAllMetrics(ctx, "day"); err != nil {
			logger.Error().Err(err).Msg("Failed to calculate daily metrics")
		}
	})
	if err != nil {
		logger.Error().Err(err).Msg("Failed to schedule daily metrics")

	}

	// Run weekly calculations on Sunday at midnight
	_, err = w.cron.AddFunc("0 0 0 * * 0", func() {
		logger.Info().Msg("Running weekly metric calculations")
		w.wg.Add(1)
		defer w.wg.Done()

		ctx, cancel := context.WithTimeout(w.ctx, 5*time.Minute)
		defer cancel()

		if err := w.trackService.CalculateAllMetrics(ctx, "week"); err != nil {
			logger.Error().Err(err).Msg("Failed to calculate weekly metrics")
		}
	})
	if err != nil {
		logger.Error().Err(err).Msg("Failed to schedule weekly metrics")
	}

	// Run monthly calculations on the 1st of each month at midnight
	_, err = w.cron.AddFunc("0 0 0 1 * *", func() {
		logger.Info().Msg("Running monthly metric calculations")
		w.wg.Add(1)
		defer w.wg.Done()

		ctx, cancel := context.WithTimeout(w.ctx, 5*time.Minute)
		defer cancel()

		if err := w.trackService.CalculateAllMetrics(ctx, "month"); err != nil {
			logger.Error().Err(err).Msg("Failed to calculate monthly metrics")
		}
	})
	if err != nil {
		logger.Error().Err(err).Msg("Failed to schedule monthly metrics")
	}
	// Start the cron scheduler
	w.cron.Start()
}

// Stop gracefully shuts down the metric worker
func (w *MetricWorker) Stop() {
	logger := log.LoggerFromContext(w.ctx).With().Str("component", "worker.metric").Logger()
	logger.Info().Msg("Stopping metric worker")
	// Stop the cron scheduler
	ctx := w.cron.Stop()

	// Signal all running jobs to finish
	w.cancel()

	// Wait for all jobs to complete with timeout
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info().Msg("All metric jobs completed successfully")
	case <-time.After(30 * time.Second):
		logger.Warn().Msg("Some metric jobs did not complete before timeout")
	case <-ctx.Done():
		logger.Info().Msg("Cron scheduler stopped")
	}
}

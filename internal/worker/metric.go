// internal/worker/metric.go

package worker

import (
	"TrackMe/internal/service/track"
	"context"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
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
	logger := zap.L().Named("worker.metric")
	logger.Info("Starting metric worker")

	// Run daily calculations at midnight
	_, err := w.cron.AddFunc("0 0 * * *", func() {
		logger.Info("Running daily metric calculations")
		w.wg.Add(1)
		defer w.wg.Done()

		ctx, cancel := context.WithTimeout(w.ctx, 5*time.Minute)
		defer cancel()

		if err := w.trackService.CalculateAllMetrics(ctx, "day"); err != nil {
			logger.Error("Failed to calculate daily metrics", zap.Error(err))
		}
	})
	if err != nil {
		logger.Error("Failed to schedule daily metrics", zap.Error(err))
	}

	// Run weekly calculations on Sunday at midnight
	_, err = w.cron.AddFunc("0 0 * * 0", func() {
		logger.Info("Running weekly metric calculations")
		w.wg.Add(1)
		defer w.wg.Done()

		ctx, cancel := context.WithTimeout(w.ctx, 5*time.Minute)
		defer cancel()

		if err := w.trackService.CalculateAllMetrics(ctx, "week"); err != nil {
			logger.Error("Failed to calculate weekly metrics", zap.Error(err))
		}
	})
	if err != nil {
		logger.Error("Failed to schedule weekly metrics", zap.Error(err))
	}

	// Run monthly calculations on the 1st of each month at midnight
	_, err = w.cron.AddFunc("0 0 1 * *", func() {
		logger.Info("Running monthly metric calculations")
		w.wg.Add(1)
		defer w.wg.Done()

		ctx, cancel := context.WithTimeout(w.ctx, 5*time.Minute)
		defer cancel()

		if err := w.trackService.CalculateAllMetrics(ctx, "month"); err != nil {
			logger.Error("Failed to calculate monthly metrics", zap.Error(err))
		}
	})
	if err != nil {
		logger.Error("Failed to schedule monthly metrics", zap.Error(err))
	}

	// Start the cron scheduler
	w.cron.Start()
}

// Stop gracefully shuts down the metric worker
func (w *MetricWorker) Stop() {
	logger := zap.L().Named("worker.metric")
	logger.Info("Stopping metric worker")

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
		logger.Info("All metric jobs completed successfully")
	case <-time.After(30 * time.Second):
		logger.Warn("Some metric jobs did not complete before timeout")
	case <-ctx.Done():
		logger.Info("Cron scheduler stopped")
	}
}

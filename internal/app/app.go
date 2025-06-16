package app

import (
	"TrackMe/internal/cache"
	"TrackMe/internal/config"
	"TrackMe/internal/domain/prometheus"
	"TrackMe/internal/handler"
	"TrackMe/internal/repository"
	"TrackMe/internal/service/track"
	"TrackMe/internal/worker"
	"TrackMe/pkg/log"
	"TrackMe/pkg/server"
	"TrackMe/pkg/store"
	"context"
	"flag"
	"fmt"
	_ "go.mongodb.org/mongo-driver/bson"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Run initializes whole application
func Run() {
	logger := log.LoggerFromContext(context.Background())

	configs, err := config.New()
	if err != nil {
		logger.Error().Err(err).Msg("ERR_INIT_CONFIGS")
		return
	}

	if err = store.Migrate(configs.MONGO.DSN); err != nil {
		logger.Error().Err(err).Msg("ERR_MIGRATE_DATABASE")
		return
	}

	promMetrics := prometheus.New()
	repositories, err := repository.New(
		repository.WithMongoStore(configs.MONGO.DSN, "trackme"), repository.WithMemoryStore())
	if err != nil {
		logger.Error().Err(err).Msg("ERR_INIT_REPOSITORIES")
		return
	}
	defer repositories.Close()

	caches, err := cache.New(
		cache.Dependencies{
			MetricRepository: repositories.Metric,
		},
		cache.WithRedisStore(configs.Redis.URL))
	if err != nil {
		logger.Error().Err(err).Msg("ERR_INIT_CACHES")
		return
	}
	defer caches.Close()

	trackService, err := track.New(
		track.WithClientRepository(repositories.Client), track.WithStageRepository(repositories.Stage), track.WithMetricRepository(repositories.Metric), track.WithPrometheusMetrics(promMetrics), track.WithMetricCache(caches.Metric))
	if err != nil {
		logger.Error().Err(err).Msg("ERR_INIT_LIBRARY_SERVICE")
		return
	}

	handlers, err := handler.New(
		handler.Dependencies{
			Configs:      configs,
			TrackService: trackService,
		},
		handler.WithHTTPHandler())
	if err != nil {
		logger.Error().Err(err).Msg("ERR_INIT_HANDLERS")
		return
	}

	servers, err := server.New(
		server.WithHTTPServer(handlers.HTTP, configs.APP.Port))
	if err != nil {
		logger.Error().Err(err).Msg("ERR_INIT_SERVERS")
		return
	}

	metricWorker := worker.NewMetricWorker(trackService)
	metricWorker.Start()

	// Run our server in a goroutine so that it doesn't block
	if err = servers.Run(logger); err != nil {
		logger.Error().Err(err).Msg("ERR_RUN_SERVERS")
		return
	}
	logger.Info().Str("url", "http://localhost:"+configs.APP.Port).Msg("http server started")

	// Graceful Shutdown
	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the httpServer gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.Parse()

	quit := make(chan os.Signal, 1) // Create channel to signify a signal being sent

	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught

	signal.Notify(quit, os.Interrupt, syscall.SIGTERM) // When an interrupt or termination signal is sent, notify the channel
	<-quit                                             // This blocks the main thread until an interrupt is received
	fmt.Println("gracefully shutting down...")

	// Create a deadline to wait for
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()

	// Stop the metric worker first
	metricWorker.Stop()

	// Doesn't block if no connections, but will otherwise wait until the timeout deadline
	if err = servers.Stop(ctx); err != nil {
		panic(err) // failure/timeout shutting down the httpServer gracefully
	}

	fmt.Println("running cleanup tasks...")
	// Your cleanup tasks go here

	fmt.Println("server was successful shutdown.")
}

//// runMigrations applies MongoDB migrations from the migrations directory
//func runMigrations(dsn string) error {
//	migrationPath := "file://migrations/mongo"
//	m, err := migrate.New(migrationPath, dsn)
//	if err != nil {
//		return fmt.Errorf("failed to create migration instance: %w", err)
//	}
//
//	// Check for dirty database state
//	version, dirty, err := m.Version()
//	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
//		return fmt.Errorf("failed to get migration version: %w", err)
//	}
//
//	// Force the version if the database is in a dirty state
//	if dirty {
//		if err = m.Force(int(version)); err != nil {
//			return fmt.Errorf("failed to force migration version: %w", err)
//		}
//	}
//
//	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
//		return fmt.Errorf("failed to apply migrations: %w", err)
//	}
//
//	return nil
//}

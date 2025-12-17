package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Business metrics for TrackMe application
var (
	// Client metrics
	ClientsCreatedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "trackme_clients_created_total",
			Help: "Total number of clients created",
		},
	)

	ClientsActiveTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "trackme_clients_active_total",
			Help: "Current number of active clients",
		},
	)

	// User metrics
	UsersCreatedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "trackme_users_created_total",
			Help: "Total number of users created",
		},
	)

	UsersActiveTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "trackme_users_active_total",
			Help: "Current number of active users",
		},
	)

	// Authentication metrics
	LoginAttemptsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trackme_login_attempts_total",
			Help: "Total number of login attempts",
		},
		[]string{"status"}, // success, failed
	)

	// Metric tracking metrics
	MetricsRecordedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "trackme_metrics_recorded_total",
			Help: "Total number of metrics recorded",
		},
	)

	MetricsProcessingDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "trackme_metrics_processing_duration_seconds",
			Help:    "Duration of metrics processing in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	// Cache metrics
	CacheHitsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trackme_cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"cache_name"},
	)

	CacheMissesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trackme_cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"cache_name"},
	)

	// Database metrics
	DatabaseQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trackme_database_queries_total",
			Help: "Total number of database queries",
		},
		[]string{"operation", "table"},
	)

	DatabaseQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "trackme_database_query_duration_seconds",
			Help:    "Duration of database queries in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		},
		[]string{"operation", "table"},
	)

	// Worker metrics
	WorkerJobsProcessedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trackme_worker_jobs_processed_total",
			Help: "Total number of worker jobs processed",
		},
		[]string{"worker", "status"}, // success, error
	)

	WorkerJobDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "trackme_worker_job_duration_seconds",
			Help:    "Duration of worker job execution in seconds",
			Buckets: []float64{.1, .5, 1, 5, 10, 30, 60, 120, 300},
		},
		[]string{"worker"},
	)
)

// Example usage functions

// RecordClientCreation increments the clients created counter
func RecordClientCreation() {
	ClientsCreatedTotal.Inc()
}

// UpdateActiveClients sets the current number of active clients
func UpdateActiveClients(count float64) {
	ClientsActiveTotal.Set(count)
}

// RecordLoginAttempt records a login attempt with status
func RecordLoginAttempt(success bool) {
	status := "failed"
	if success {
		status = "success"
	}
	LoginAttemptsTotal.WithLabelValues(status).Inc()
}

// RecordCacheAccess records cache hit or miss
func RecordCacheAccess(cacheName string, hit bool) {
	if hit {
		CacheHitsTotal.WithLabelValues(cacheName).Inc()
	} else {
		CacheMissesTotal.WithLabelValues(cacheName).Inc()
	}
}

// RecordDatabaseQuery records database query execution
func RecordDatabaseQuery(operation, table string, duration float64) {
	DatabaseQueriesTotal.WithLabelValues(operation, table).Inc()
	DatabaseQueryDuration.WithLabelValues(operation, table).Observe(duration)
}

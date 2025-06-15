package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Entity struct {
	ClientsPerStage  *prometheus.GaugeVec
	StageDuration    *prometheus.GaugeVec
	MAU              prometheus.Gauge
	DAU              prometheus.Gauge
	SourceConversion *prometheus.GaugeVec
	AppInstallRate   prometheus.Gauge
	AutoPaymentRate  prometheus.Gauge
	RollbackCount    prometheus.Counter
	Conversion       prometheus.Gauge
	TotalDuration    prometheus.Gauge
	Dropout          prometheus.Gauge
	StatusUpdates    prometheus.Gauge
}

// NewWithRegistry creates a new prometheus entity with a custom registry
func NewWithRegistry(registry prometheus.Registerer) Entity {
	factory := promauto.With(registry)

	return Entity{
		ClientsPerStage: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "trackme_clients_per_stage",
				Help: "Number of clients at each stage",
			},
			[]string{"stage"},
		),
		StageDuration: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "trackme_stage_duration_hours",
				Help: "Average duration spent at each stage in hours",
			},
			[]string{"stage"},
		),
		MAU: factory.NewGauge(
			prometheus.GaugeOpts{
				Name: "trackme_monthly_active_users",
				Help: "Monthly Active Users count",
			},
		),
		DAU: factory.NewGauge(
			prometheus.GaugeOpts{
				Name: "trackme_daily_active_users",
				Help: "Daily Active Users count",
			},
		),
		SourceConversion: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "trackme_source_conversion_ratio",
				Help: "Conversion ratio by source",
			},
			[]string{"source"},
		),
		AppInstallRate: factory.NewGauge(
			prometheus.GaugeOpts{
				Name: "trackme_app_install_rate",
				Help: "Percentage of clients with installed app",
			},
		),
		AutoPaymentRate: factory.NewGauge(
			prometheus.GaugeOpts{
				Name: "trackme_autopayment_rate",
				Help: "Percentage of contracts with autopayment enabled",
			},
		),
		RollbackCount: factory.NewCounter(
			prometheus.CounterOpts{
				Name: "trackme_rollback_count_total",
				Help: "Total number of stage rollbacks",
			},
		),
		Conversion: factory.NewGauge(
			prometheus.GaugeOpts{
				Name: "trackme_conversion_ratio",
				Help: "Overall conversion ratio",
			},
		),
		TotalDuration: factory.NewGauge(
			prometheus.GaugeOpts{
				Name: "trackme_total_process_duration_days",
				Help: "Average total process duration in days",
			},
		),
		Dropout: factory.NewGauge(
			prometheus.GaugeOpts{
				Name: "trackme_dropout_count",
				Help: "Number of clients considered dropped out",
			},
		),
		StatusUpdates: factory.NewGauge(
			prometheus.GaugeOpts{
				Name: "trackme_status_updates_count",
				Help: "Number of client status updates in a period",
			},
		),
	}
}

// New creates a new prometheus entity with the default registry (for production)
func New() Entity {
	return NewWithRegistry(prometheus.DefaultRegisterer)
}

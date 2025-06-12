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

func New() Entity {
	return Entity{
		ClientsPerStage: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "trackme_clients_per_stage",
				Help: "Number of clients at each stage",
			},
			[]string{"stage"},
		),
		StageDuration: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "trackme_stage_duration_hours",
				Help: "Average duration spent at each stage in hours",
			},
			[]string{"stage"},
		),
		MAU: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "trackme_monthly_active_users",
				Help: "Monthly Active Users count",
			},
		),
		DAU: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "trackme_daily_active_users",
				Help: "Daily Active Users count",
			},
		),
		SourceConversion: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "trackme_source_conversion_ratio",
				Help: "Conversion ratio by source",
			},
			[]string{"source"},
		),
		AppInstallRate: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "trackme_app_install_rate",
				Help: "Percentage of clients with installed app",
			},
		),
		AutoPaymentRate: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "trackme_autopayment_rate",
				Help: "Percentage of contracts with autopayment enabled",
			},
		),
		RollbackCount: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "trackme_rollback_count_total",
				Help: "Total number of stage rollbacks",
			},
		),
		Conversion: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "trackme_conversion_ratio",
				Help: "Overall conversion ratio",
			},
		),
		TotalDuration: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "trackme_total_process_duration_days",
				Help: "Average total process duration in days",
			},
		),
		Dropout: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "trackme_dropout_count",
				Help: "Number of clients considered dropped out",
			},
		),
		StatusUpdates: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "trackme_status_updates_count",
				Help: "Number of client status updates in a period",
			},
		),
	}
}

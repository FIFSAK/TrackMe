package http

import (
	"TrackMe/internal/domain/metric"
	"TrackMe/internal/service/track"
	"TrackMe/pkg/server/response"
	"github.com/go-chi/chi/v5"
	"net/http"
)

type MetricHandler struct {
	trackService *track.Service
}

func NewMetricHandler(s *track.Service) *MetricHandler {
	return &MetricHandler{trackService: s}
}

func (h *MetricHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.list)
	//r.Get("/prometheus", h.prometheus)

	return r
}

// @Summary Get metrics with filtering
// @Tags metrics
// @Accept json
// @Produce json
// @Param type query string false "Filter by metric type"
// @Param interval query string false "Filter by time interval (day, week, month)"
// @Success 200 {array} metric.Response
// @Failure 400 {object} response.Object
// @Failure 500 {object} response.Object
// @Router /metrics [get]
func (h *MetricHandler) list(w http.ResponseWriter, r *http.Request) {
	filters := metric.Filters{
		Type:     r.URL.Query().Get("type"),
		Interval: r.URL.Query().Get("interval"),
	}

	if interval := r.URL.Query().Get("interval"); interval != "" {
		filters.Interval = interval

	}

	if metricType := r.URL.Query().Get("type"); metricType != "" {
		filters.Type = metricType
	}

	metrics, err := h.trackService.ListMetrics(r.Context(), filters)
	if err != nil {
		response.InternalServerError(w, r, err)
		return
	}

	response.OK(w, r, metrics, nil)
}

//// @Summary Get metrics in Prometheus format
//// @Tags metrics
//// @Accept json
//// @Produce text/plain
//// @Success 200 {string} string "Prometheus metrics"
//// @Failure 500 {object} response.Object
//// @Router /metrics/prometheus [get]
//func (h *MetricHandler) prometheus(w http.ResponseWriter, r *http.Request) {
//	metrics, err := h.trackService.GetPrometheusMetrics(r.Context())
//	if err != nil {
//		response.InternalServerError(w, r, err)
//		return
//	}
//
//	w.Header().Set("Content-Type", "text/plain")
//	w.WriteHeader(http.StatusOK)
//	w.Write([]byte(metrics))
//}

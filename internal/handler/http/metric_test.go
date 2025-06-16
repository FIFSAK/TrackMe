package http

import (
	"TrackMe/internal/domain/metric"
	"TrackMe/pkg/store"
	"context"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type MockMetricService struct {
	mock.Mock
}

func (m *MockMetricService) ListMetrics(ctx context.Context, filters metric.Filters) ([]metric.Response, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).([]metric.Response), args.Error(1)
}

func (m *MockMetricService) CalculateAllMetrics(ctx context.Context, interval string) error {
	args := m.Called(ctx, interval)
	return args.Error(0)
}

func TestMetricHandler_List(t *testing.T) {
	now := time.Now()

	t.Run("successful listing without filters", func(t *testing.T) {

		mockService := new(MockMetricService)
		metrics := []metric.Response{
			{
				ID:        "metric1",
				Type:      "conversion",
				Value:     75.5,
				CreatedAt: now,
			},
			{
				ID:        "metric2",
				Type:      "app-install-rate",
				Value:     42.0,
				Interval:  "day",
				CreatedAt: now,
			},
		}

		filters := metric.Filters{}
		mockService.On("ListMetrics", mock.Anything, filters).Return(metrics, nil)

		handler := createTestMetricHandler(mockService)

		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()

		handler.list(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var response struct {
			Data []metric.Response `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, 2, len(response.Data))
		assert.Equal(t, "metric1", response.Data[0].ID)
		assert.Equal(t, "conversion", response.Data[0].Type)
		assert.Equal(t, 75.5, response.Data[0].Value)
	})

	t.Run("listing with type filter", func(t *testing.T) {

		mockService := new(MockMetricService)
		metrics := []metric.Response{
			{
				ID:        "metric2",
				Type:      "app-install-rate",
				Value:     42.0,
				Interval:  "day",
				CreatedAt: now,
			},
		}

		filters := metric.Filters{
			Type: "app-install-rate",
		}
		mockService.On("ListMetrics", mock.Anything, filters).Return(metrics, nil)

		handler := createTestMetricHandler(mockService)

		req := httptest.NewRequest("GET", "/metrics?type=app-install-rate", nil)
		w := httptest.NewRecorder()

		handler.list(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var response struct {
			Data []metric.Response `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, 1, len(response.Data))
		assert.Equal(t, "metric2", response.Data[0].ID)
		assert.Equal(t, "app-install-rate", response.Data[0].Type)
		assert.Equal(t, "day", response.Data[0].Interval)
	})

	t.Run("listing with interval filter", func(t *testing.T) {

		mockService := new(MockMetricService)
		metrics := []metric.Response{
			{
				ID:        "metric3",
				Type:      "rollback-count",
				Value:     5.0,
				Interval:  "week",
				CreatedAt: now,
			},
		}

		filters := metric.Filters{
			Interval: "week",
		}
		mockService.On("ListMetrics", mock.Anything, filters).Return(metrics, nil)

		handler := createTestMetricHandler(mockService)

		req := httptest.NewRequest("GET", "/metrics?interval=week", nil)
		w := httptest.NewRecorder()

		handler.list(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var response struct {
			Data []metric.Response `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, 1, len(response.Data))
		assert.Equal(t, "metric3", response.Data[0].ID)
		assert.Equal(t, "rollback-count", response.Data[0].Type)
		assert.Equal(t, "week", response.Data[0].Interval)
	})

	t.Run("listing with both filters", func(t *testing.T) {

		mockService := new(MockMetricService)
		metrics := []metric.Response{
			{
				ID:        "metric4",
				Type:      "status-updates",
				Value:     12.0,
				Interval:  "month",
				CreatedAt: now,
			},
		}

		filters := metric.Filters{
			Type:     "status-updates",
			Interval: "month",
		}
		mockService.On("ListMetrics", mock.Anything, filters).Return(metrics, nil)

		handler := createTestMetricHandler(mockService)

		req := httptest.NewRequest("GET", "/metrics?type=status-updates&interval=month", nil)
		w := httptest.NewRecorder()

		handler.list(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var response struct {
			Data []metric.Response `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, 1, len(response.Data))
		assert.Equal(t, "metric4", response.Data[0].ID)
		assert.Equal(t, "status-updates", response.Data[0].Type)
		assert.Equal(t, "month", response.Data[0].Interval)
	})

	t.Run("not found error", func(t *testing.T) {

		mockService := new(MockMetricService)

		filters := metric.Filters{
			Type: "non-existent-metric",
		}
		mockService.On("ListMetrics", mock.Anything, filters).Return([]metric.Response{}, store.ErrorNotFound)

		handler := createTestMetricHandler(mockService)

		req := httptest.NewRequest("GET", "/metrics?type=non-existent-metric", nil)
		w := httptest.NewRecorder()

		handler.list(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("internal server error", func(t *testing.T) {

		mockService := new(MockMetricService)

		filters := metric.Filters{}
		mockService.On("ListMetrics", mock.Anything, filters).Return([]metric.Response{}, errors.New("database error"))

		handler := createTestMetricHandler(mockService)

		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()

		handler.list(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMetricHandler_TriggerCalculateAllMetrics(t *testing.T) {
	t.Run("successful calculation", func(t *testing.T) {

		mockService := new(MockMetricService)
		mockService.On("CalculateAllMetrics", mock.Anything, "day").Return(nil)

		handler := createTestMetricHandler(mockService)

		req := httptest.NewRequest("GET", "/metrics/calculate?interval=day", nil)
		w := httptest.NewRecorder()

		handler.triggerCalculateAllMetrics(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var response struct {
			Data map[string]string `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "triggerred success", response.Data["message"])
	})

	t.Run("missing interval parameter", func(t *testing.T) {

		mockService := new(MockMetricService)

		handler := createTestMetricHandler(mockService)

		req := httptest.NewRequest("GET", "/metrics/calculate", nil)
		w := httptest.NewRecorder()

		handler.triggerCalculateAllMetrics(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("calculation error", func(t *testing.T) {

		mockService := new(MockMetricService)
		mockService.On("CalculateAllMetrics", mock.Anything, "day").Return(errors.New("calculation error"))

		handler := createTestMetricHandler(mockService)

		req := httptest.NewRequest("GET", "/metrics/calculate?interval=day", nil)
		w := httptest.NewRecorder()

		handler.triggerCalculateAllMetrics(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

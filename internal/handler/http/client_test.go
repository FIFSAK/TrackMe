package http

import (
	"TrackMe/internal/domain/client"
	"TrackMe/pkg/store"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type MockTrackService struct {
	mock.Mock
}

func (m *MockTrackService) ListClients(ctx context.Context, filters client.Filters, limit, offset int) ([]client.Response, int, error) {
	args := m.Called(ctx, filters, limit, offset)
	return args.Get(0).([]client.Response), args.Int(1), args.Error(2)
}

func (m *MockTrackService) UpdateClient(ctx context.Context, id string, req client.Request) (client.Response, error) {
	args := m.Called(ctx, id, req)
	return args.Get(0).(client.Response), args.Error(1)
}

func TestClientHandler_List(t *testing.T) {
	now := time.Now()
	regDate := now.Add(-30 * 24 * time.Hour)
	regDateStr := regDate.Format(time.RFC3339)

	t.Run("successful listing", func(t *testing.T) {

		mockService := new(MockTrackService)
		clients := []client.Response{
			{
				ID:               "client1",
				Name:             "John Doe",
				Email:            "john@example.com",
				CurrentStage:     "active",
				IsActive:         true,
				RegistrationDate: regDateStr,
			},
		}

		isActive := true
		filters := client.Filters{
			IsActive: &isActive,
		}
		mockService.On("ListClients", mock.Anything, filters, 50, 0).Return(clients, 1, nil)

		handler := NewClientHandler(mockService)

		req := httptest.NewRequest("GET", "/clients", nil)
		w := httptest.NewRecorder()

		handler.list(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var response struct {
			Data []client.Response      `json:"data"`
			Meta map[string]interface{} `json:"meta"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, 1, len(response.Data))
		assert.Equal(t, "client1", response.Data[0].ID)
		assert.Equal(t, float64(1), response.Meta["total"])
		assert.Equal(t, float64(50), response.Meta["limit"])
		assert.Equal(t, float64(0), response.Meta["offset"])
	})

	t.Run("with query parameters", func(t *testing.T) {

		mockService := new(MockTrackService)
		clients := []client.Response{
			{
				ID:               "client2",
				Name:             "Jane Smith",
				Email:            "jane@example.com",
				CurrentStage:     "onboarding",
				IsActive:         false,
				RegistrationDate: regDateStr,
			},
		}

		updatedDate, _ := time.Parse("2006-01-02", "2023-01-01")
		loginDate, _ := time.Parse("2006-01-02", "2023-02-01")

		isActive := false
		filters := client.Filters{
			Stage:          "onboarding",
			Source:         "website",
			Channel:        "direct",
			AppStatus:      "installed",
			IsActive:       &isActive,
			UpdatedAfter:   updatedDate,
			LastLoginAfter: loginDate,
		}
		mockService.On("ListClients", mock.Anything, filters, 10, 20).Return(clients, 1, nil)

		handler := NewClientHandler(mockService)

		req := httptest.NewRequest("GET", "/clients?stage=onboarding&source=website&channel=direct&app=installed&is_active=false&updated=2023-01-01&last_login=2023-02-01&limit=10&offset=20", nil)
		w := httptest.NewRecorder()

		handler.list(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("error from service", func(t *testing.T) {

		mockService := new(MockTrackService)
		serviceErr := errors.New("database error")

		isActive := true
		filters := client.Filters{
			IsActive: &isActive,
		}
		mockService.On("ListClients", mock.Anything, filters, 50, 0).Return([]client.Response{}, 0, serviceErr)

		handler := NewClientHandler(mockService)

		req := httptest.NewRequest("GET", "/clients", nil)
		w := httptest.NewRecorder()

		handler.list(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestClientHandler_Update(t *testing.T) {
	t.Run("successful update", func(t *testing.T) {

		mockService := new(MockTrackService)
		now := time.Now()
		regDateStr := now.Format(time.RFC3339)

		clientRequest := client.Request{
			Stage: "active",
		}

		clientResponse := client.Response{
			ID:               "client1",
			Name:             "John Doe",
			Email:            "john@example.com",
			CurrentStage:     "active",
			IsActive:         true,
			RegistrationDate: regDateStr,
		}

		mockService.On("UpdateClient", mock.Anything, "client1", clientRequest).Return(clientResponse, nil)

		handler := NewClientHandler(mockService)

		requestBody, _ := json.Marshal(clientRequest)
		req := httptest.NewRequest("PUT", "/clients/client1/stage", bytes.NewBuffer(requestBody))
		req.Header.Set("Content-Type", "application/json")

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "client1")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.update(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var response struct {
			Data client.Response `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "client1", response.Data.ID)
		assert.Equal(t, "active", response.Data.CurrentStage)
	})

	t.Run("not found error", func(t *testing.T) {

		mockService := new(MockTrackService)
		clientRequest := client.Request{
			Stage: "active",
		}

		mockService.On("UpdateClient", mock.Anything, "non-existent", clientRequest).
			Return(client.Response{}, store.ErrorNotFound)

		handler := NewClientHandler(mockService)

		requestBody, _ := json.Marshal(clientRequest)
		req := httptest.NewRequest("PUT", "/clients/non-existent/stage", bytes.NewBuffer(requestBody))
		req.Header.Set("Content-Type", "application/json")

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "non-existent")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.update(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("invalid stage transition", func(t *testing.T) {

		mockService := new(MockTrackService)
		clientRequest := client.Request{
			Stage: "invalid_stage",
		}

		mockService.On("UpdateClient", mock.Anything, "client1", clientRequest).
			Return(client.Response{}, errors.New("invalid stage transition: cannot move from current to invalid_stage"))

		handler := NewClientHandler(mockService)

		requestBody, _ := json.Marshal(clientRequest)
		req := httptest.NewRequest("PUT", "/clients/client1/stage", bytes.NewBuffer(requestBody))
		req.Header.Set("Content-Type", "application/json")

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "client1")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.update(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		input    string
		default_ bool
		want     bool
	}{
		{"", true, true},
		{"", false, false},
		{"true", true, true},
		{"true", false, true},
		{"false", true, false},
		{"false", false, false},
		{"1", true, true},
		{"0", true, false},
		{"invalid", true, true},
		{"invalid", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseBool(tt.input, tt.default_)
			assert.Equal(t, tt.want, *result)
		})
	}
}

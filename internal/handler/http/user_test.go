package http

import (
	"TrackMe/internal/domain/user"
	"TrackMe/pkg/store"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) ListUsers(ctx context.Context, limit, offset int) ([]user.Response, int, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]user.Response), args.Int(1), args.Error(2)
}

func (m *MockUserService) CreateUser(ctx context.Context, req user.Request) (user.Response, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(user.Response), args.Error(1)
}

func (m *MockUserService) GetUser(ctx context.Context, id string) (user.Response, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(user.Response), args.Error(1)
}

func (m *MockUserService) UpdateUser(ctx context.Context, id string, req user.Request) (user.Response, error) {
	args := m.Called(ctx, id, req)
	return args.Get(0).(user.Response), args.Error(1)
}

func (m *MockUserService) DeleteUser(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestUserHandler_List(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewUserHandler(mockService)

	now := time.Now()
	expectedUsers := []user.Response{
		{
			ID:        "1",
			Name:      "John Doe",
			Email:     "john@example.com",
			Role:      user.RoleAdmin,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	mockService.On("ListUsers", mock.Anything, 50, 0).Return(expectedUsers, 1, nil)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()

	handler.list(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestUserHandler_Create_Success(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewUserHandler(mockService)

	userReq := user.Request{
		Name:  "Jane Doe",
		Email: "jane@example.com",
		Role:  user.RoleManager,
	}

	now := time.Now()
	expectedResponse := user.Response{
		ID:        "123",
		Name:      userReq.Name,
		Email:     userReq.Email,
		Role:      userReq.Role,
		CreatedAt: now,
		UpdatedAt: now,
	}

	mockService.On("CreateUser", mock.Anything, userReq).Return(expectedResponse, nil)

	body, _ := json.Marshal(userReq)
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.create(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockService.AssertExpectations(t)
}

func TestUserHandler_Create_DuplicateEmail(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewUserHandler(mockService)

	userReq := user.Request{
		Name:  "Jane Doe",
		Email: "jane@example.com",
		Role:  user.RoleManager,
	}

	mockService.On("CreateUser", mock.Anything, userReq).
		Return(user.Response{}, errors.New("user with this email already exists"))

	body, _ := json.Marshal(userReq)
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.create(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	mockService.AssertExpectations(t)
}

func TestUserHandler_Create_InvalidRole(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewUserHandler(mockService)

	userReq := user.Request{
		Name:  "Jane Doe",
		Email: "jane@example.com",
		Role:  user.RoleManager,
	}

	mockService.On("CreateUser", mock.Anything, userReq).
		Return(user.Response{}, errors.New("invalid user role"))

	body, _ := json.Marshal(userReq)
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.create(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestUserHandler_Get_Success(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewUserHandler(mockService)

	now := time.Now()
	expectedUser := user.Response{
		ID:        "123",
		Name:      "John Doe",
		Email:     "john@example.com",
		Role:      user.RoleAdmin,
		CreatedAt: now,
		UpdatedAt: now,
	}

	mockService.On("GetUser", mock.Anything, "123").Return(expectedUser, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.get(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestUserHandler_Get_NotFound(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewUserHandler(mockService)

	mockService.On("GetUser", mock.Anything, "999").Return(user.Response{}, store.ErrorNotFound)

	req := httptest.NewRequest(http.MethodGet, "/users/999", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "999")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.get(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestUserHandler_Update_Success(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewUserHandler(mockService)

	userReq := user.Request{
		Name:  "Updated Name",
		Email: "updated@example.com",
		Role:  user.RoleAdmin,
	}

	now := time.Now()
	expectedResponse := user.Response{
		ID:        "123",
		Name:      userReq.Name,
		Email:     userReq.Email,
		Role:      userReq.Role,
		CreatedAt: now,
		UpdatedAt: now,
	}

	mockService.On("UpdateUser", mock.Anything, "123", userReq).Return(expectedResponse, nil)

	body, _ := json.Marshal(userReq)
	req := httptest.NewRequest(http.MethodPut, "/users/123", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.update(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestUserHandler_Update_NotFound(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewUserHandler(mockService)

	userReq := user.Request{
		Name:  "Updated Name",
		Email: "updated@example.com",
		Role:  user.RoleAdmin,
	}

	mockService.On("UpdateUser", mock.Anything, "999", userReq).Return(user.Response{}, store.ErrorNotFound)

	body, _ := json.Marshal(userReq)
	req := httptest.NewRequest(http.MethodPut, "/users/999", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "999")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.update(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestUserHandler_Delete_Success(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewUserHandler(mockService)

	mockService.On("DeleteUser", mock.Anything, "123").Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/users/123", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.delete(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockService.AssertExpectations(t)
}

func TestUserHandler_Delete_NotFound(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewUserHandler(mockService)

	mockService.On("DeleteUser", mock.Anything, "999").Return(store.ErrorNotFound)

	req := httptest.NewRequest(http.MethodDelete, "/users/999", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "999")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.delete(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

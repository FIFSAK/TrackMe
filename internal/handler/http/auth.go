package http

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"TrackMe/internal/domain/auth"
	"TrackMe/internal/domain/user"
	"TrackMe/pkg/jwt"
	"TrackMe/pkg/server/response"
)

type AuthService interface {
	CreateUser(ctx context.Context, req user.Request) (user.Response, error)
	Login(ctx context.Context, email, password string) (user.Entity, error)
}

type AuthHandler struct {
	authService  AuthService
	tokenManager *jwt.TokenManager
}

func NewAuthHandler(s AuthService, tm *jwt.TokenManager) *AuthHandler {
	return &AuthHandler{
		authService:  s,
		tokenManager: tm,
	}
}

func (h *AuthHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/register", h.register)
	r.Post("/login", h.login)

	return r
}

// @Summary Register a new user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body auth.RegisterRequest true "body param"
// @Success 201 {object} auth.TokenResponse
// @Failure 400 {object} response.Object
// @Failure 409 {object} response.Object
// @Failure 500 {object} response.Object
// @Router /auth/register [post]
func (h *AuthHandler) register(w http.ResponseWriter, r *http.Request) {
	var req auth.RegisterRequest
	if err := render.Bind(r, &req); err != nil {
		response.BadRequest(w, r, err, req)
		return
	}

	// Convert to user.Request and force default role to manager regardless of input
	userReq := user.Request{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
		Role:     user.RoleManager,
	}

	// Create user
	userResp, err := h.authService.CreateUser(r.Context(), userReq)
	if err != nil {
		if err.Error() == "user with this email already exists" {
			response.Conflict(w, r, err)
			return
		}
		response.InternalServerError(w, r, err)
		return
	}

	// Generate JWT token
	token, err := h.tokenManager.GenerateToken(userResp.ID, userResp.Email, userResp.Role, 24*time.Hour)
	if err != nil {
		response.InternalServerError(w, r, err)
		return
	}

	tokenResp := auth.TokenResponse{
		Token: token,
		User: auth.UserInfo{
			ID:    userResp.ID,
			Name:  userResp.Name,
			Email: userResp.Email,
			Role:  userResp.Role,
		},
	}

	response.Created(w, r, tokenResp)
}

// @Summary Login user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body auth.LoginRequest true "body param"
// @Success 200 {object} auth.TokenResponse
// @Failure 400 {object} response.Object
// @Failure 401 {object} response.Object
// @Failure 500 {object} response.Object
// @Router /auth/login [post]
func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	var req auth.LoginRequest
	if err := render.Bind(r, &req); err != nil {
		response.BadRequest(w, r, err, req)
		return
	}

	// Authenticate user
	userEntity, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if err.Error() == "invalid email or password" {
			response.Unauthorized(w, r, err)
			return
		}
		response.InternalServerError(w, r, err)
		return
	}

	// Generate JWT token
	token, err := h.tokenManager.GenerateToken(userEntity.ID, userEntity.Email, userEntity.Role, 24*time.Hour)
	if err != nil {
		response.InternalServerError(w, r, err)
		return
	}

	tokenResp := auth.TokenResponse{
		Token: token,
		User: auth.UserInfo{
			ID:    userEntity.ID,
			Name:  userEntity.Name,
			Email: userEntity.Email,
			Role:  userEntity.Role,
		},
	}

	response.OK(w, r, tokenResp, nil)
}

package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"TrackMe/internal/domain/user"
	"TrackMe/internal/service/track"
	"TrackMe/pkg/jwt"
	"TrackMe/pkg/server/middleware"
	"TrackMe/pkg/server/response"
	"TrackMe/pkg/store"
)

type UserHandler struct {
	userService  track.UserService
	tokenManager *jwt.TokenManager
}

func NewUserHandler(s track.UserService, tm *jwt.TokenManager) *UserHandler {
	return &UserHandler{
		userService:  s,
		tokenManager: tm,
	}
}

func (h *UserHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// All routes require authentication
	r.Use(middleware.AuthMiddleware(h.tokenManager))

	// List and Get require at least manager role (read-only access)
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAdminOrManager())
		r.Get("/", h.list)
		r.Get("/{id}", h.get)
	})

	// Create and Delete require super_user or admin
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireSuperUserOrAdmin())
		r.Post("/", h.create)

	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireSuperUser())
		r.Delete("/{id}", h.delete)
		r.Put("/{id}", h.update)
	})

	// Update has special logic in handler

	return r
}

// @Summary List users with pagination
// @Description Get a list of users with optional pagination
// @Tags users
// @Accept json
// @Produce json
// @Param limit query integer false "Pagination limit (default 50)"
// @Param offset query integer false "Pagination offset (default 0)"
// @Success 200 {array} user.Response
// @Failure 500 {object} response.Object
// @Router /users [get]
// @Security BearerAuth
func (h *UserHandler) list(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if lInt, err := strconv.Atoi(l); err == nil && lInt > 0 {
			limit = lInt
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if oInt, err := strconv.Atoi(o); err == nil && oInt >= 0 {
			offset = oInt
		}
	}

	res, total, err := h.userService.ListUsers(r.Context(), limit, offset)
	if err != nil {
		response.InternalServerError(w, r, err)
		return
	}

	response.OK(w, r, res, map[string]interface{}{
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// @Summary Create user
// @Tags users
// @Accept json
// @Produce json
// @Param request body user.Request true "body param"
// @Success 201 {object} user.Response
// @Failure 400 {object} response.Object
// @Failure 409 {object} response.Object
// @Failure 500 {object} response.Object
// @Router /users [post]
// @Security BearerAuth
func (h *UserHandler) create(w http.ResponseWriter, r *http.Request) {
	// Decode only the allowed fields from the client
	var payload struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.BadRequest(w, r, err, nil)
		return
	}

	// Always create users with manager role
	req := user.Request{
		Name:     payload.Name,
		Email:    payload.Email,
		Password: payload.Password,
		Role:     user.RoleAdmin,
	}

	// Reuse existing validation
	if err := req.Bind(r); err != nil {
		response.BadRequest(w, r, err, req)
		return
	}

	userResp, err := h.userService.CreateUser(r.Context(), req)
	if err != nil {
		if err.Error() == "user with this email already exists" {
			response.Conflict(w, r, err)
			return
		}
		if err.Error() == "invalid user role" {
			response.BadRequest(w, r, err, req.Role)
			return
		}
		response.InternalServerError(w, r, err)
		return
	}

	response.Created(w, r, userResp)
}

// @Summary Get user by ID
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} user.Response
// @Failure 404 {object} response.Object
// @Failure 500 {object} response.Object
// @Router /users/{id} [get]
// @Security BearerAuth
func (h *UserHandler) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	userResp, err := h.userService.GetUser(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrorNotFound) {
			response.NotFound(w, r, err)
			return
		}
		response.InternalServerError(w, r, err)
		return
	}

	response.OK(w, r, userResp, nil)
}

// @Summary Update user
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body user.Request true "body param"
// @Success 200 {object} user.Response
// @Failure 400 {object} response.Object
// @Failure 403 {object} response.Object
// @Failure 404 {object} response.Object
// @Failure 409 {object} response.Object
// @Failure 500 {object} response.Object
// @Router /users/{id} [put]
// @Security BearerAuth
func (h *UserHandler) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Get current user from context
	claims, err := middleware.GetUserFromContext(r.Context())
	if err != nil {
		response.Forbidden(w, r, err)
		return
	}

	var req user.Request
	if err := render.Bind(r, &req); err != nil {
		response.BadRequest(w, r, err, req)
		return
	}

	// Get target user to check current role
	targetUser, err := h.userService.GetUser(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrorNotFound) {
			response.NotFound(w, r, err)
			return
		}
		response.InternalServerError(w, r, err)
		return
	}

	// Permission checks:
	// 1. super_user can update anyone including role changes
	// 2. admin can update anyone but cannot change roles
	// 3. user can update themselves but cannot change their own role
	// 4. manager cannot update anyone

	if claims.Role == user.RoleManager {
		response.Forbidden(w, r, errors.New("managers cannot update users"))
		return
	}

	// Check if trying to change role
	if req.Role != targetUser.Role {
		// Only super_user can change roles
		if claims.Role != user.RoleSuperUser {
			response.Forbidden(w, r, errors.New("only super_user can change user roles"))
			return
		}
	}

	// Admin can update anyone (except role), users can only update themselves
	if claims.Role == user.RoleAdmin {
		// Admin can update anyone
	} else if claims.UserID != id && claims.Role != user.RoleSuperUser {
		// Regular users can only update themselves
		response.Forbidden(w, r, errors.New("you can only update your own profile"))
		return
	}

	userResp, err := h.userService.UpdateUser(r.Context(), id, req)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrorNotFound):
			response.NotFound(w, r, err)
			return
		case err.Error() == "user with this email already exists":
			response.Conflict(w, r, err)
			return
		case err.Error() == "invalid user role":
			response.BadRequest(w, r, err, req.Role)
			return
		default:
			response.InternalServerError(w, r, err)
			return
		}
	}

	response.OK(w, r, userResp, nil)
}

// @Summary Delete user
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 204 "No Content"
// @Failure 404 {object} response.Object
// @Failure 500 {object} response.Object
// @Router /users/{id} [delete]
// @Security BearerAuth
func (h *UserHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := h.userService.DeleteUser(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrorNotFound) {
			response.NotFound(w, r, err)
			return
		}
		response.InternalServerError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

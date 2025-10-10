package http

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"TrackMe/internal/domain/client"
	"TrackMe/pkg/server/response"
	"TrackMe/pkg/store"
)

type ClientTrackService interface {
	ListClients(ctx context.Context, filters client.Filters, limit, offset int) ([]client.Response, int, error)
	CreateClient(ctx context.Context, req client.Request) (client.Response, error)
	UpdateClient(ctx context.Context, id string, req client.Request) (client.Response, error)
	DeleteClient(ctx context.Context, id string) error
}
type ClientHandler struct {
	trackService ClientTrackService
}

func NewClientHandler(s ClientTrackService) *ClientHandler {
	return &ClientHandler{trackService: s}
}

func (h *ClientHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.list)
	r.Post("/", h.create)

	r.Route("/{id}", func(r chi.Router) {
		//r.Get("/", h.get)
		r.Put("/stage", h.update)
		r.Delete("/", h.delete)
	})

	return r
}

// @Summary    List clients with filtering and pagination
// @Description Get a list of clients with optional filtering and pagination
// @Tags        clients
// @Accept      json
// @Produce     json
// @Param       id query string false "Filter by client ID"
// @Param       stage query string false "Filter by client stage"
// @Param       source query string false "Filter by source"
// @Param       channel query string false "Filter by channel"
// @Param       app query string false "Filter by app status"
// @Param       is_active query boolean false "Filter by active status (default: true)"
// @Param       updated query string false "Filter by last updated after date (YYYY-MM-DD)"
// @Param       last_login query string false "Filter by last login date after (YYYY-MM-DD)"
// @Param       limit query integer false "Pagination limit (default 50)"
// @Param       offset query integer false "Pagination offset (default 0)"
// @Success     200 {array} client.Response
// @Failure     500 {object} response.Object
// @Router      /clients [get]
func (h *ClientHandler) list(w http.ResponseWriter, r *http.Request) {
	filters := client.Filters{
		ID:        r.URL.Query().Get("id"),
		Stage:     r.URL.Query().Get("stage"),
		Source:    r.URL.Query().Get("source"),
		Channel:   r.URL.Query().Get("channel"),
		AppStatus: r.URL.Query().Get("app"),
		IsActive:  parseBool(r.URL.Query().Get("is_active"), true),
	}

	if updated := r.URL.Query().Get("updated"); updated != "" {
		if t, err := time.Parse("2006-01-02", updated); err == nil {
			filters.UpdatedAfter = t
		}
	}

	if lastLogin := r.URL.Query().Get("last_login"); lastLogin != "" {
		if t, err := time.Parse("2006-01-02", lastLogin); err == nil {
			filters.LastLoginAfter = t
		}
	}

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

	res, total, err := h.trackService.ListClients(r.Context(), filters, limit, offset)
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

func parseBool(s string, defaultVal bool) *bool {
	if s == "" {
		return &defaultVal
	}
	b, err := strconv.ParseBool(s)
	if err != nil {
		return &defaultVal
	}
	return &b
}

// @Summary Create client
// @Tags clients
// @Accept json
// @Produce json
// @Param request body client.Request true "body param"
// @Success 201 {object} client.Response
// @Failure 400 {object} response.Object
// @Failure 409 {object} response.Object
// @Failure 500 {object} response.Object
// @Router /clients [post]
func (h *ClientHandler) create(w http.ResponseWriter, r *http.Request) {
	var req client.Request
	if err := render.Bind(r, &req); err != nil {
		response.BadRequest(w, r, err, req)
		return
	}
	if len(req.Contracts) > 0 {
		for _, contract := range req.Contracts {
			if err := contract.Bind(r); err != nil {
				response.BadRequest(w, r, err, req)
				return
			}
		}
	}

	clientResp, err := h.trackService.CreateClient(r.Context(), req)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			response.Conflict(w, r, err)
			return
		}
		if strings.Contains(err.Error(), "invalid initial stage") {
			response.BadRequest(w, r, err, req.Stage)
			return
		}
		response.InternalServerError(w, r, err)
		return
	}

	response.Created(w, r, clientResp)
}

// @Summary Update client
// @Tags clients
// @Accept json
// @Produce json
// @Param id path string true "Client ID"
// @Param request body client.Request true "body param"
// @Success 200 {object} client.Response
// @Success 201 {object} client.Response
// @Failure 400 {object} response.Object
// @Failure 404 {object} response.Object
// @Failure 500 {object} response.Object
// @Router /clients/{id}/stage [put]
func (h *ClientHandler) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req client.Request
	if err := render.Bind(r, &req); err != nil {
		response.BadRequest(w, r, err, req)
		return
	}
	if len(req.Contracts) > 0 {
		for _, contract := range req.Contracts {
			if err := contract.Bind(r); err != nil {
				response.BadRequest(w, r, err, req)
				return
			}
		}
	}

	clientResp, err := h.trackService.UpdateClient(r.Context(), id, req)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrorNotFound):
			response.NotFound(w, r, err)
		case strings.Contains(err.Error(), "invalid stage transition"):
			response.BadRequest(w, r, err, req.Stage)

		default:
			response.InternalServerError(w, r, err)
		}
		return
	}

	regDate, err := time.Parse(time.RFC3339, clientResp.RegistrationDate)
	if err == nil {
		// Truncate both times to hour precision for comparison
		lastUpdatedHour := clientResp.LastUpdated.Truncate(time.Hour)
		regDateHour := regDate.Truncate(time.Hour)

		if lastUpdatedHour.Equal(regDateHour) {
			response.Created(w, r, clientResp)
			return
		}
	}

	response.OK(w, r, clientResp, nil)
}

// @Summary Delete client
// @Tags clients
// @Accept json
// @Produce json
// @Param id path string true "Client ID"
// @Success 204 "No Content"
// @Failure 404 {object} response.Object
// @Failure 500 {object} response.Object
// @Router /clients/{id} [delete]
func (h *ClientHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := h.trackService.DeleteClient(r.Context(), id)
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

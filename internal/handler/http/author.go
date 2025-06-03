package http

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"TrackMe/internal/domain/client"
	"TrackMe/internal/service/library"
	"TrackMe/pkg/server/response"
	"TrackMe/pkg/store"
)

type AuthorHandler struct {
	libraryService *library.Service
}

func NewAuthorHandler(s *library.Service) *AuthorHandler {
	return &AuthorHandler{libraryService: s}
}

func (h *AuthorHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.list)
	r.Post("/", h.add)

	r.Route("/{id}", func(r chi.Router) {
		r.Get("/", h.get)
		r.Put("/", h.update)
		r.Delete("/", h.delete)
	})

	return r
}

// @Summary	list of authors from the repository
// @Tags		authors
// @Accept		json
// @Produce	json
// @Success	200			{array}		client.Response
// @Failure	500			{object}	response.Object
// @Router		/authors 	[get]
func (h *AuthorHandler) list(w http.ResponseWriter, r *http.Request) {
	res, err := h.libraryService.ListClients(r.Context())
	if err != nil {
		response.InternalServerError(w, r, err)
		return
	}

	response.OK(w, r, res)
}

// @Summary	add a new client to the repository
// @Tags		authors
// @Accept		json
// @Produce	json
// @Param		request	body		client.Request	true	"body param"
// @Success	200		{object}	client.Response
// @Failure	400		{object}	response.Object
// @Failure	500		{object}	response.Object
// @Router		/authors [post]
func (h *AuthorHandler) add(w http.ResponseWriter, r *http.Request) {
	req := client.Request{}
	if err := render.Bind(r, &req); err != nil {
		response.BadRequest(w, r, err, req)
		return
	}

	res, err := h.libraryService.AddClient(r.Context(), req)
	if err != nil {
		response.InternalServerError(w, r, err)
		return
	}

	response.OK(w, r, res)
}

// @Summary	get the client from the repository
// @Tags		authors
// @Accept		json
// @Produce	json
// @Param		id	path		int	true	"path param"
// @Success	200	{object}	client.Response
// @Failure	404	{object}	response.Object
// @Failure	500	{object}	response.Object
// @Router		/authors/{id} [get]
func (h *AuthorHandler) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	res, err := h.libraryService.GetClient(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrorNotFound):
			response.NotFound(w, r, err)
		default:
			response.InternalServerError(w, r, err)
		}
		return
	}

	response.OK(w, r, res)
}

// @Summary	update the client in the repository
// @Tags		authors
// @Accept		json
// @Produce	json
// @Param		id		path	int				true	"path param"
// @Param		request	body	client.Request	true	"body param"
// @Success	200
// @Failure	400	{object}	response.Object
// @Failure	404	{object}	response.Object
// @Failure	500	{object}	response.Object
// @Router		/authors/{id} [put]
func (h *AuthorHandler) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	req := client.Request{}
	if err := render.Bind(r, &req); err != nil {
		response.BadRequest(w, r, err, req)
		return
	}

	if err := h.libraryService.UpdateClient(r.Context(), id, req); err != nil {
		switch {
		case errors.Is(err, store.ErrorNotFound):
			response.NotFound(w, r, err)
		default:
			response.InternalServerError(w, r, err)
		}
		return
	}
}

// @Summary	delete the client from the repository
// @Tags		authors
// @Accept		json
// @Produce	json
// @Param		id	path	int	true	"path param"
// @Success	200
// @Failure	404	{object}	response.Object
// @Failure	500	{object}	response.Object
// @Router		/authors/{id} [delete]
func (h *AuthorHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.libraryService.DeleteClient(r.Context(), id); err != nil {
		switch {
		case errors.Is(err, store.ErrorNotFound):
			response.NotFound(w, r, err)
		default:
			response.InternalServerError(w, r, err)
		}
		return
	}
}

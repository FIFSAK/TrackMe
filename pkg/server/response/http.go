package response

import (
	"net/http"

	"github.com/go-chi/render"
)

type Object struct {
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Meta    any    `json:"meta,omitempty"`
}

func OK(w http.ResponseWriter, r *http.Request, data any, meta any) {
	render.Status(r, http.StatusOK)

	v := Object{
		Data: data,
		Meta: meta,
	}
	render.JSON(w, r, v)
}

func BadRequest(w http.ResponseWriter, r *http.Request, err error, data any) {
	render.Status(r, http.StatusBadRequest)

	v := Object{
		Data:    data,
		Message: err.Error(),
	}
	render.JSON(w, r, v)
}

func NotFound(w http.ResponseWriter, r *http.Request, err error) {
	render.Status(r, http.StatusNotFound)

	v := Object{
		Message: err.Error(),
	}
	render.JSON(w, r, v)
}

func InternalServerError(w http.ResponseWriter, r *http.Request, err error) {
	render.Status(r, http.StatusInternalServerError)

	v := Object{
		Message: err.Error(),
	}
	render.JSON(w, r, v)
}

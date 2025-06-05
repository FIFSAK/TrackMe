package app

import (
	"errors"
	"net/http"
)

type Request struct {
	Status string `json:"status"`
}

// Bind validates the app request payload.
func (s *Request) Bind(r *http.Request) error {
	if s.Status == "" {
		return errors.New("status: cannot be blank")
	}
	return nil
}

// Response represents the response payload for app operations.
type Response struct {
	Status    string `json:"status"`
	Highlight bool   `json:"highlight"`
}

// ParseFromEntity converts an app entity to a response payload.
func ParseFromEntity(data Entity) Response {
	highlight := false
	if data.Status == "" {
		return Response{
			Status:    "unknown",
			Highlight: false,
		}
	}
	if data.Status == "installed" {
		highlight = true
	}
	return Response{
		Status:    data.Status,
		Highlight: highlight,
	}
}

// ParseFromEntities converts a list of app entities to a list of response payloads.
func ParseFromEntities(data []Entity) []Response {
	res := make([]Response, len(data))
	for i, entity := range data {
		res[i] = ParseFromEntity(entity)
	}
	return res
}

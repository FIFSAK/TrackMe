package stage

import (
	"errors"
	"net/http"
)

// Request represents the request payload for stage operations.
type Request struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Order              int      `json:"order"`
	AllowedTransitions []string `json:"allowed_transitions"`
	LastUpdated        string   `json:"last_updated"`
}

// Bind validates the request payload.
func (req *Request) Bind(r *http.Request) error {
	if req.ID == "" {
		return errors.New("id: cannot be blank")
	}
	if req.Name == "" {
		return errors.New("name: cannot be blank")
	}
	if req.Order <= 0 {
		return errors.New("order: must be greater than zero")
	}
	if req.LastUpdated == "" {
		return errors.New("last_updated: cannot be blank")
	}
	return nil
}

// Response represents the response payload for stage operations.
type Response struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Order              int      `json:"order"`
	AllowedTransitions []string `json:"allowed_transitions"`
	LastUpdated        string   `json:"last_updated"`
}

// ParseFromEntity creates a new Response from a given Entity.
func ParseFromEntity(entity Entity) Response {
	return Response{
		ID:                 entity.ID,
		Name:               *entity.Name,
		Order:              *entity.Order,
		AllowedTransitions: entity.AllowedTransitions,
		LastUpdated:        *entity.LastUpdated,
	}
}

// ParseFromEntities creates a slice of Responses from a slice of Entities.
func ParseFromEntities(entities []Entity) []Response {
	responses := make([]Response, len(entities))
	for i, entity := range entities {
		responses[i] = ParseFromEntity(entity)
	}
	return responses
}

package client

import (
	"TrackMe/internal/domain/contract"
	"errors"
	"net/http"
)

type Request struct {
	ID               uint              `json:"id"`
	Name             string            `json:"name"`
	Email            string            `json:"email"`
	RegistrationDate string            `json:"registration_date"`
	CurrentStage     string            `json:"current_stage"`
	LastUpdated      string            `json:"last_updated"`
	IsActive         bool              `json:"is_active"`
	Source           string            `json:"source"`
	Channel          string            `json:"channel"`
	App              string            `json:"app"`
	LastLogin        string            `json:"last_login"`
	Contracts        []contract.Entity `json:"contracts"`
}

// Bind validates the request payload.
func (s *Request) Bind(r *http.Request) error {
	if s.Name == "" {
		return errors.New("name: cannot be blank")
	}
	if s.Email == "" {
		return errors.New("email: cannot be blank")
	}
	if s.RegistrationDate == "" {
		return errors.New("registrationDate: cannot be blank")
	}
	if s.Source == "" {
		return errors.New("source: cannot be blank")
	}
	if s.Channel == "" {
		return errors.New("channel: cannot be blank")
	}
	if s.LastUpdated == "" {
		return errors.New("lastUpdated: cannot be blank")
	}
	if s.LastLogin == "" {
		return errors.New("lastLogin: cannot be blank")
	}
	return nil
}

// Response represents the response payload for client operations.
type Response struct {
	ID               uint   `json:"id"`
	Name             string `json:"name"`
	Email            string `json:"email"`
	RegistrationDate string `json:"registrationDate"`
	Source           string `json:"source"`
	Channel          string `json:"channel"`
	IsActive         bool   `json:"isActive"`
}

// ParseFromEntity converts a client entity to a response payload.
func ParseFromEntity(data Entity) Response {
	return Response{
		ID:               data.ID,
		Name:             *data.Name,
		Email:            *data.Email,
		RegistrationDate: *data.RegistrationDate,
		Source:           *data.Source,
		Channel:          *data.Channel,
		IsActive:         *data.IsActive,
	}
}

// ParseFromEntities converts a list of client entities to a list of response payloads.
func ParseFromEntities(data []Entity) []Response {
	res := make([]Response, len(data))
	for i, entity := range data {
		res[i] = ParseFromEntity(entity)
	}
	return res
}

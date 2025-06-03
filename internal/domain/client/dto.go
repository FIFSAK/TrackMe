package client

import (
	"TrackMe/internal/domain/app"
	"TrackMe/internal/domain/contract"
	"TrackMe/internal/domain/lastLogin"
	"errors"
	"net/http"
	"time"
)

type Request struct {
	ID               string             `json:"id"`
	Name             string             `json:"name"`
	Email            string             `json:"email"`
	RegistrationDate time.Time          `json:"registration_date"`
	Stage            string             `json:"stage"`
	LastUpdated      time.Time          `json:"last_updated"`
	IsActive         bool               `json:"is_active"`
	Source           string             `json:"source"`
	Channel          string             `json:"channel"`
	App              app.Request        `json:"app"`
	LastLogin        lastLogin.Request  `json:"last_login"`
	Contracts        []contract.Request `json:"contracts"`
}

// Bind validates the request payload.
func (s *Request) Bind(r *http.Request) error {
	if s.ID == "" {
		return errors.New("id: cannot be blank")
	}
	if s.Name == "" {
		return errors.New("name: cannot be blank")
	}
	if s.Stage == "" {
		return errors.New("current_stage: cannot be blank")
	}
	if s.RegistrationDate == (time.Time{}) {
		return errors.New("registration_date: cannot be blank")
	}
	return nil
}

// Response represents the response payload for client operations.
type Response struct {
	ID               string              `json:"id"`
	Name             string              `json:"name"`
	Email            string              `json:"email"`
	CurrentStage     string              `json:"current_stage"`
	RegistrationDate time.Time           `json:"registration_date"`
	LastUpdated      time.Time           `json:"last_updated"`
	IsActive         bool                `json:"isActive"`
	Source           string              `json:"source"`
	Channel          string              `json:"channel"`
	App              app.Response        `json:"app"`
	LastLogin        lastLogin.Response  `json:"last_login"`
	Contracts        []contract.Response `json:"contracts"`
}

// ParseFromEntity converts a client entity to a response payload.
func ParseFromEntity(data Entity) Response {
	return Response{
		ID:               data.ID,
		Name:             *data.Name,
		Email:            *data.Email,
		CurrentStage:     *data.CurrentStage,
		RegistrationDate: *data.RegistrationDate,
		LastUpdated:      *data.LastUpdated,
		IsActive:         *data.IsActive,
		Source:           *data.Source,
		Channel:          *data.Channel,
		App:              app.ParseFromEntity(data.App),
		LastLogin:        lastLogin.ParseFromEntity(data.LastLogin),
		Contracts:        contract.ParseFromEntities(data.Contracts),
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

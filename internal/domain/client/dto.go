package client

import (
	"TrackMe/internal/domain/app"
	"TrackMe/internal/domain/contract"
	"TrackMe/internal/domain/lastLogin"
	"errors"
	"net/http"
	"time"
)

type Filters struct {
	ID             string
	Stage          string
	Source         string
	Channel        string
	AppStatus      string
	IsActive       *bool
	UpdatedAfter   time.Time
	LastLoginAfter time.Time
}
type Request struct {
	Name      string             `json:"name"`
	Email     string             `json:"email"`
	Stage     string             `json:"stage"`
	IsActive  *bool              `json:"is_active"`
	Source    string             `json:"source"`
	Channel   string             `json:"channel"`
	App       string             `json:"app"`
	LastLogin time.Time          `json:"last_login"`
	Contracts []contract.Request `json:"contracts"`
}

// Bind validates the request payload.
func (s *Request) Bind(r *http.Request) error {
	if s.Stage == "" {
		return errors.New("current_stage: cannot be blank")
	}
	return nil
}

// Response represents the response payload for client operations.
type Response struct {
	ID               string              `json:"id"`
	Name             string              `json:"name"`
	Email            string              `json:"email"`
	CurrentStage     string              `json:"current_stage"`
	RegistrationDate string              `json:"registration_date"`
	LastUpdated      time.Time           `json:"last_updated"`
	IsActive         bool                `json:"is_active"`
	Source           string              `json:"source"`
	Channel          string              `json:"channel"`
	App              app.Response        `json:"app"`
	LastLogin        lastLogin.Response  `json:"last_login"`
	Contracts        []contract.Response `json:"contracts"`
}

// ParseFromEntity converts a client entity to a response payload.
func ParseFromEntity(data Entity) Response {
	appEntity := app.Entity{Status: *data.App}
	parsedRegistrationDate := data.RegistrationDate.Format(time.RFC3339)
	if *data.App == "installed" {
		parsedRegistrationDate = data.RegistrationDate.UTC().Format("02.01.2006")
	}
	lastLoginEntity := lastLogin.Entity{
		Date: *data.LastLogin,
	}
	return Response{
		ID:               data.ID,
		Name:             *data.Name,
		Email:            *data.Email,
		CurrentStage:     *data.CurrentStage,
		RegistrationDate: parsedRegistrationDate,
		LastUpdated:      *data.LastUpdated,
		IsActive:         *data.IsActive,
		Source:           *data.Source,
		Channel:          *data.Channel,
		App:              app.ParseFromEntity(appEntity),
		LastLogin:        lastLogin.ParseFromEntity(lastLoginEntity),
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

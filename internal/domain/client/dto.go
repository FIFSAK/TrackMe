package client

import (
	"TrackMe/internal/domain/app"
	"TrackMe/internal/domain/contract"
	"TrackMe/internal/domain/lastLogin"
	"errors"
	"net/http"
	"regexp"
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
	if s.Email == "" {
		return errors.New("email: cannot be blank")
	}
	// Validate email format
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(s.Email) {
		return errors.New("email: invalid format")
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
// ParseFromEntity converts a client entity to a response payload.
func ParseFromEntity(data Entity) Response {
	// Initialize response with safe defaults
	resp := Response{
		ID: data.ID,
	}

	// Handle RegistrationDate safely
	if data.RegistrationDate != nil {
		parsedRegistrationDate := data.RegistrationDate.Format(time.RFC3339)

		// Check if App is installed to format differently
		if data.App != nil && *data.App == "installed" {
			parsedRegistrationDate = data.RegistrationDate.UTC().Format("02.01.2006")
		}
		resp.RegistrationDate = parsedRegistrationDate
	}

	// Handle App safely
	var appEntity app.Entity
	if data.App != nil {
		appEntity = app.Entity{Status: *data.App}
	}
	resp.App = app.ParseFromEntity(appEntity)

	// Handle LastLogin safely
	var lastLoginEntity lastLogin.Entity
	if data.LastLogin != nil {
		lastLoginEntity = lastLogin.Entity{
			Date: *data.LastLogin,
		}
	}
	resp.LastLogin = lastLogin.ParseFromEntity(lastLoginEntity)

	// Handle Contracts
	resp.Contracts = contract.ParseFromEntities(data.Contracts)

	// Safely dereference pointer fields with nil checks
	if data.Name != nil {
		resp.Name = *data.Name
	}

	if data.Email != nil {
		resp.Email = *data.Email
	}

	if data.CurrentStage != nil {
		resp.CurrentStage = *data.CurrentStage
	}

	// FIX: This was causing the panic - add nil check for LastUpdated
	if data.LastUpdated != nil {
		resp.LastUpdated = *data.LastUpdated
	} else {
		// Set default value if nil
		resp.LastUpdated = time.Now()
	}

	if data.IsActive != nil {
		resp.IsActive = *data.IsActive
	}

	if data.Source != nil {
		resp.Source = *data.Source
	}

	if data.Channel != nil {
		resp.Channel = *data.Channel
	}

	return resp
}

// ParseFromEntities converts a list of client entities to a list of response payloads.
func ParseFromEntities(data []Entity) []Response {
	res := make([]Response, len(data))
	for i, entity := range data {
		res[i] = ParseFromEntity(entity)
	}
	return res
}

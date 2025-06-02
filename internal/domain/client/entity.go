package client

import (
	"TrackMe/internal/domain/contract"
)

// Entity represents a client in the system.
type Entity struct {
	// ID is the unique identifier for the client.
	ID uint `db:"id" bson:"_id"`

	// Name is the full name of the client.
	Name *string `db:"name" bson:"name"`

	// Email is the email address of the client.
	Email *string `db:"email" bson:"email"`

	// RegisteredAt is the timestamp when the client registered in format DD.MM.YYYY.
	RegistrationDate *string `db:"registration_date" bson:"registration_date"`

	// Current stage of the registration process.
	CurrentStage *string `db:"current_stage" bson:"current_stage"`

	// Last update date of the client in format DD.MM.YYYY.
	LastUpdated *string `db:"last_updated" bson:"last_updated"`

	//Active indicates whether the client is active.
	IsActive *bool `db:"is_active" bson:"is_active"`

	// Source indicates the origin from which the client was acquired.
	Source *string `db:"source" bson:"source"`

	// Channel represents the communication channel used for client acquisition.
	Channel *string `db:"channel" bson:"channel"`

	//Client mobile application status: installed, not_installed
	App *string `db:"app" bson:"app"`

	// LastLogin is the timestamp of the client's last login in format DD.MM.YYYY.
	LastLogin *string `db:"last_login" bson:"last_login"`

	// Contracts is a list of contracts associated with the client.
	Contracts []contract.Entity `db:"contracts" bson:"contracts"`
}

// New creates a new Client instance.
func New(req Request) Entity {
	return Entity{
		ID:               req.ID,
		Name:             &req.Name,
		Email:            &req.Email,
		RegistrationDate: &req.RegistrationDate,
		CurrentStage:     &req.CurrentStage,

		LastUpdated: &req.LastUpdated,
		IsActive:    &req.IsActive,
		Source:      &req.Source,
		Channel:     &req.Channel,
		App:         &req.App,
		LastLogin:   &req.LastLogin,
		Contracts:   req.Contracts,
	}
}

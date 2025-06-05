package contract

import (
	autopayment "TrackMe/internal/domain/autopayment"
	"errors"
	"net/http"
	"time"
)

// Request represents the request payload for contract operations.
type Request struct {
	ID               string              `json:"id"`
	Name             string              `json:"name"`
	Number           string              `json:"number"`
	Status           string              `json:"status"`
	ConclusionDate   time.Time           `json:"conclusion_date"`
	ExpirationDate   time.Time           `json:"expiration_date"`
	Amount           float64             `json:"amount"`
	PaymentFrequency string              `json:"payment_frequency"`
	AutoPayment      autopayment.Request `json:"auto_payment"`
}

// Bind validates the request payload.
func (req *Request) Bind(r *http.Request) error {
	if req.Name == "" {
		return errors.New("name: cannot be blank")
	}
	if req.Number == "" {
		return errors.New("number: cannot be blank")
	}
	if req.Status == "" {
		return errors.New("status: cannot be blank")
	}
	if req.ConclusionDate == (time.Time{}) {
		return errors.New("conclusion_date: cannot be blank")
	}
	if req.ExpirationDate == (time.Time{}) {
		return errors.New("expiration_date: cannot be blank")
	}
	if req.PaymentFrequency == "" {
		return errors.New("payment_frequency: cannot be blank")
	}
	return nil
}

// Response represents the response payload for contract operations.
type Response struct {
	ID               string               `json:"id"`
	Name             string               `json:"name"`
	Number           string               `json:"number"`
	Status           string               `json:"status"`
	ConclusionDate   time.Time            `json:"conclusion_date"`
	ExpirationDate   time.Time            `json:"expiration_date"`
	Amount           float64              `json:"amount"`
	PaymentFrequency string               `json:"payment_frequency"`
	AutoPayment      autopayment.Response `json:"auto_payment"`
}

// ParseFromEntity creates a new Response from a given Entity.
func ParseFromEntity(entity Entity) Response {
	return Response{
		ID:               entity.ID,
		Name:             *entity.Name,
		Number:           *entity.Number,
		Status:           *entity.Status,
		ConclusionDate:   *entity.ConclusionDate,
		ExpirationDate:   *entity.ExpirationDate,
		Amount:           *entity.Amount,
		PaymentFrequency: *entity.PaymentFrequency,
		AutoPayment:      autopayment.ParseFromEntity(entity.AutoPayment),
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

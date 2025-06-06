package contract

import (
	"time"
)

// Entity represents a contract in the system.
type Entity struct {
	// ID is the unique identifier for the contract (UUID).
	ID string `db:"id" bson:"_id"`

	// Name is the name of the contract (e.g., "Subscription Service", "Insurance Product").
	Name *string `db:"name" bson:"name"`

	// Number is the unique number of the contract.
	Number *string `db:"number" bson:"number"`

	// Status is the status of the contract (e.g., active, pending, expired).
	Status *string `db:"status" bson:"status"`

	// ConclusionDate is the date of payment and contract signing (ISO 8601 format).
	ConclusionDate *time.Time `db:"conclusion_date" bson:"conclusion_date"`

	// ExpirationDate is the expiration date of the contract (ISO 8601 format).
	ExpirationDate *time.Time `db:"expiration_date" bson:"expiration_date"`

	// Amount is the payment amount for the contract (numeric, in currency).
	Amount *float64 `db:"amount" bson:"amount"`

	// PaymentFrequency is the payment frequency (e.g., monthly, quarterly, annually).
	PaymentFrequency *string `db:"payment_frequency" bson:"payment_frequency"`

	// AutoPayment indicates the status of auto-payment (enabled/disabled).
	AutoPayment *string `db:"autopayment" bson:"autopayment"`
}

// New creates a new Contract instance.
func New(req Request) Entity {
	return Entity{
		ID:               req.ID,
		Name:             &req.Name,
		Number:           &req.Number,
		Status:           &req.Status,
		ConclusionDate:   &req.ConclusionDate,
		ExpirationDate:   &req.ExpirationDate,
		Amount:           &req.Amount,
		PaymentFrequency: &req.PaymentFrequency,
		AutoPayment:      &req.AutoPayment,
	}
}

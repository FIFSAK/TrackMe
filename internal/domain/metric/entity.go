package metric

import "time"

type Type string

const (
	ClientsPerStage   Type = "clients-per-stage"
	StageDuration     Type = "stage-duration"
	RollbackCount     Type = "rollback-count"
	Dropout           Type = "dropout"
	Conversion        Type = "conversion"
	TotalDuration     Type = "total-duration"
	StatusUpdates     Type = "status-updates"
	MAU               Type = "mau"
	DAU               Type = "dau"
	SourceConversion  Type = "source-conversion"
	ChannelConversion Type = "channel-conversion"
	AppInstallRate    Type = "app-install-rate"
	AutoPaymentRate   Type = "autopayment-rate"
)

// Entity represents a metric in the system.
type Entity struct {
	// ID is the unique identifier for the metric (UUID).
	ID string `db:"id" bson:"_id"`

	// Type is the type of the metric (e.g., clients-per-stage, stage-duration, etc.).
	Type *Type `db:"type" bson:"type"`

	// Value is the value of the metric (e.g., number of clients or average time).
	Value *float64 `db:"value" bson:"value"`

	// TimeInterval is the time range for the metric (e.g., day, week, month).
	Interval *string `db:"interval,omitempty" bson:"interval,omitempty"`

	// CreatedAt is the creation date of the metric.
	CreatedAt *time.Time `db:"created_at" bson:"created_at"`

	Metadata map[string]string `db:"metadata,omitempty" bson:"metadata,omitempty"`
}

// New creates a new Metric instance.
func New(req Request) Entity {
	return Entity{
		Type:      (*Type)(&req.Type),
		Value:     &req.Value,
		Interval:  &req.Interval,
		CreatedAt: &req.CreatedAt,
	}
}

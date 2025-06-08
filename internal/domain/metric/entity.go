package metric

// Entity represents a metric in the system.
type Entity struct {
	// ID is the unique identifier for the metric (UUID).
	ID string `db:"id" bson:"_id"`

	// Type is the type of the metric (e.g., clients-per-stage, stage-duration, etc.).
	Type *string `db:"type" bson:"type"`

	// Value is the value of the metric (e.g., number of clients or average time).
	Value *float64 `db:"value" bson:"value"`

	// TimeInterval is the time range for the metric (e.g., day, week, month).
	Interval *string `db:"interval" bson:"interval"`

	// CreatedAt is the creation date of the metric.
	CreatedAt *string `db:"created_at" bson:"created_at"`
}

// New creates a new Metric instance.
func New(req Request) Entity {
	return Entity{
		Type:      &req.Type,
		Value:     &req.Value,
		Interval:  &req.Interval,
		CreatedAt: &req.CreatedAt,
	}
}

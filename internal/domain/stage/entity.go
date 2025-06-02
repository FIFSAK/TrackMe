package stage

// Entity represents a stage in the system.
type Entity struct {
	// ID is the unique identifier for the stage (e.g., "registration").
	ID string `db:"id" bson:"_id"`

	// Name is the name of the stage (e.g., "Registration").
	Name *string `db:"name" bson:"name"`

	// Order is the sequential number of the stage in the process (1–N).
	Order *int `db:"order" bson:"order"`

	// AllowedTransitions is the list of stages to which transitions are allowed (forward/backward).
	AllowedTransitions []string `db:"allowed_transitions" bson:"allowed_transitions"`

	// LastUpdated is the date when the client transitioned to this stage.
	LastUpdated *string `db:"last_updated" bson:"last_updated"`
}

// New creates a new Stage instance.
func New(req Request) Entity {
	return Entity{
		ID:                 req.ID,
		Name:               &req.Name,
		Order:              &req.Order,
		AllowedTransitions: req.AllowedTransitions,
		LastUpdated:        &req.LastUpdated,
	}
}

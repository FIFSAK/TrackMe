package app

// Entity represents a app in the system.
type Entity struct {
	Status *string `db:"status" bson:"status"`
}

// New creates a new app instance.
func New(req Request) Entity {
	return Entity{
		Status: &req.Status,
	}
}

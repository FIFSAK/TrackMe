package autopayment

// Entity represents an auto payment in the system.
type Entity struct {
	Status string `db:"status" bson:"status"`
}

// New creates a new auto payment instance.
func New(req Request) Entity {
	return Entity(req)
}

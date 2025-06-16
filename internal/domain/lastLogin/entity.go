package lastLogin

import "time"

// Entity represents a client in the system.
type Entity struct {
	Date time.Time `db:"date" bson:"date"`
}

// New creates a new Client instance.
func New(req Request) Entity {
	return Entity(req)
}

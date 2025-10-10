package user

import "time"

// Role constants
const (
	RoleSuperUser = "super_user"
	RoleAdmin     = "admin"
	RoleManager   = "manager"
)

// Entity represents a user in the system.
type Entity struct {
	// ID is the unique identifier for the user.
	ID string `db:"id" bson:"_id,omitempty"`

	// Name is the full name of the user.
	Name string `db:"name" bson:"name"`

	// Email is the email address of the user.
	Email string `db:"email" bson:"email"`

	// Password is the hashed password of the user.
	Password string `db:"password" bson:"password"`

	// Role is the user's role (super_user, admin, manager)
	Role string `db:"role" bson:"role"`

	// CreatedAt is the timestamp when the user was created.
	CreatedAt time.Time `db:"created_at" bson:"created_at"`

	// UpdatedAt is the timestamp when the user was last updated.
	UpdatedAt time.Time `db:"updated_at" bson:"updated_at"`
}

// New creates a new User instance.
func New(req Request) Entity {
	now := time.Now()
	return Entity{
		Name:      req.Name,
		Email:     req.Email,
		Password:  req.Password,
		Role:      req.Role,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// ValidRoles returns a list of valid user roles.
func ValidRoles() []string {
	return []string{RoleSuperUser, RoleAdmin, RoleManager}
}

// IsValidRole checks if the given role is valid.
func IsValidRole(role string) bool {
	for _, validRole := range ValidRoles() {
		if role == validRole {
			return true
		}
	}
	return false
}

package user

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"
)

type Request struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password,omitempty"`
	Role     string `json:"role"`
}

// Bind validates the request payload.
func (s *Request) Bind(r *http.Request) error {
	if s.Name == "" {
		return errors.New("name: cannot be blank")
	}
	if s.Email == "" {
		return errors.New("email: cannot be blank")
	}
	// Validate email format
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(s.Email) {
		return errors.New("email: invalid format")
	}
	if s.Role == "" {
		return errors.New("role: cannot be blank")
	}
	// Validate role
	if !IsValidRole(s.Role) {
		return fmt.Errorf("role: must be one of %v", ValidRoles())
	}
	return nil
}

// Response represents the response payload for user operations.
type Response struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ParseFromEntity converts a user entity to a response payload.
func ParseFromEntity(data Entity) Response {
	return Response{
		ID:        data.ID,
		Name:      data.Name,
		Email:     data.Email,
		Role:      data.Role,
		CreatedAt: data.CreatedAt,
		UpdatedAt: data.UpdatedAt,
	}
}

// ParseFromEntities converts a list of user entities to a list of response payloads.
func ParseFromEntities(data []Entity) []Response {
	res := make([]Response, len(data))
	for i, entity := range data {
		res[i] = ParseFromEntity(entity)
	}
	return res
}

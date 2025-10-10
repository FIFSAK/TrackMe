package auth

import (
	"errors"
	"net/http"
	"regexp"
)

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (r *RegisterRequest) Bind(req *http.Request) error {
	if r.Name == "" {
		return errors.New("name: cannot be blank")
	}
	if r.Email == "" {
		return errors.New("email: cannot be blank")
	}
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(r.Email) {
		return errors.New("email: invalid format")
	}
	if r.Password == "" {
		return errors.New("password: cannot be blank")
	}
	if len(r.Password) < 6 {
		return errors.New("password: must be at least 6 characters")
	}
	return nil
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (l *LoginRequest) Bind(req *http.Request) error {
	if l.Email == "" {
		return errors.New("email: cannot be blank")
	}
	if l.Password == "" {
		return errors.New("password: cannot be blank")
	}
	return nil
}

type TokenResponse struct {
	Token string   `json:"token"`
	User  UserInfo `json:"user"`
}

type UserInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

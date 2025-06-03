package lastLogin

import (
	"errors"
	"net/http"
	"time"
)

type Request struct {
	Date time.Time `json:"date"`
}

// Bind validates the request payload.
func (s *Request) Bind(r *http.Request) error {
	if s.Date == (time.Time{}) {
		return errors.New("date: cannot be blank")
	}
	return nil
}

// Response represents the response payload for client operations.
type Response struct {
	Date      time.Time `json:"date"`
	Highlight bool      `json:"highlight"`
}

// ParseFromEntity converts a client entity to a response payload.
func ParseFromEntity(data Entity) Response {
	highlight := false
	if time.Since(*data.Date) > 30*24*time.Hour {
		highlight = true
	}
	return Response{
		Date:      *data.Date,
		Highlight: highlight,
	}
}

// ParseFromEntities converts a list of client entities to a list of response payloads.
func ParseFromEntities(data []Entity) []Response {
	res := make([]Response, len(data))
	for i, entity := range data {
		res[i] = ParseFromEntity(entity)
	}
	return res
}

package metric

import (
	"errors"
	"net/http"
)

// Request represents the request payload for metric operations.
type Request struct {
	Type      string  `json:"type"`
	Value     float64 `json:"value"`
	Interval  string  `json:"interval"`
	CreatedAt string  `json:"created_at"`
}

// Bind validates the request payload.
func (req *Request) Bind(r *http.Request) error {
	if req.Type == "" {
		return errors.New("type: cannot be blank")
	}
	if req.Interval == "" {
		return errors.New("interval: cannot be blank")
	}
	if req.CreatedAt == "" {
		return errors.New("created_at: cannot be blank")
	}
	return nil
}

// Response represents the response payload for metric operations.
type Response struct {
	Type      string  `json:"type"`
	Value     float64 `json:"value"`
	Interval  string  `json:"interval"`
	CreatedAt string  `json:"created_at"`
}

// ParseFromEntity converts a metric entity to a response payload.
func ParseFromEntity(data Entity) Response {
	return Response{
		Type:      *data.Type,
		Value:     *data.Value,
		Interval:  *data.Interval,
		CreatedAt: *data.CreatedAt,
	}
}

// ParseFromEntities converts a list of metric entities to a list of response payloads.
func ParseFromEntities(data []Entity) []Response {
	res := make([]Response, len(data))
	for i, entity := range data {
		res[i] = ParseFromEntity(entity)
	}
	return res
}

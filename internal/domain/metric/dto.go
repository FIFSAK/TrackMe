package metric

import (
	"errors"
	"net/http"
	"time"
)

type Filters struct {
	Type     string
	Interval string
}

// Request represents the request payload for metric operations.
type Request struct {
	Type      string    `json:"type"`
	Value     float64   `json:"value"`
	Interval  string    `json:"interval"`
	CreatedAt time.Time `json:"created_at"`
}

// Bind validates the request payload.
func (req *Request) Bind(r *http.Request) error {
	if req.Type == "" {
		return errors.New("type: cannot be blank")
	}
	if req.Interval == "" {
		return errors.New("interval: cannot be blank")
	}
	if req.CreatedAt == (time.Time{}) {
		return errors.New("created_at: cannot be blank")
	}
	return nil
}

// Response represents the response payload for metric operations.
type Response struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Value     float64           `json:"value"`
	Interval  string            `json:"interval"`
	CreatedAt time.Time         `json:"created_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// ParseFromEntity converts a metric entity to a response payload.
func ParseFromEntity(data Entity) Response {
	return Response{
		ID:        data.ID,
		Type:      string(*data.Type),
		Value:     *data.Value,
		Interval:  *data.Interval,
		CreatedAt: *data.CreatedAt,
		Metadata:  data.Metadata,
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

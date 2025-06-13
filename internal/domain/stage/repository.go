package stage

import (
	"context"
)

// Repository defines the interface for member repository operations.
type Repository interface {
	// List retrieves all member entities.
	List(ctx context.Context) ([]Entity, error)

	// Get retrieves a member entity by its ID.
	Get(ctx context.Context, id string) (Entity, error)

	// UpdateStage updates the current stage of a member entity based on the provided option.
	UpdateStage(ctx context.Context, currentStage, option string) (string, error)
}

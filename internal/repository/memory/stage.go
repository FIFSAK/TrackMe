package memory

import (
	"context"
	"database/sql"
	"fmt"
	"gopkg.in/yaml.v2"
	"log"
	"os"
	"sync"

	"TrackMe/internal/domain/stage"
	"github.com/google/uuid"
)

// StageRepository handles CRUD operations for stages in an in-memory database using sync.Map
type StageRepository struct {
	db sync.Map // key: string (UUID), value: stage.Entity
}

// NewStageRepository creates a new StageRepository with stages loaded from yaml
func NewStageRepository() *StageRepository {
	repo := &StageRepository{
		db: sync.Map{},
	}

	// Load stages from YAML file
	yamlFile, err := os.ReadFile("stages.yaml")
	if err != nil {
		log.Printf("Failed to read stages.yaml: %v", err)
		return repo
	}

	// Parse YAML
	var config struct {
		Stages []struct {
			ID          string   `yaml:"id"`
			Name        string   `yaml:"name"`
			Order       int      `yaml:"order"`
			Transitions []string `yaml:"transitions"`
		} `yaml:"stages"`
	}

	if err := yaml.Unmarshal(yamlFile, &config); err != nil {
		log.Printf("Failed to parse stages.yaml: %v", err)
		return repo
	}

	// Store stages in sync.Map
	for _, s := range config.Stages {
		stageEntity := stage.Entity{
			ID:                 s.ID,
			Name:               &s.Name,
			Order:              &s.Order,
			AllowedTransitions: s.Transitions,
		}
		repo.db.Store(s.ID, stageEntity)
	}

	return repo
}

// List retrieves all stages from the in-memory database
func (r *StageRepository) List(ctx context.Context) ([]stage.Entity, error) {
	var stages []stage.Entity

	r.db.Range(func(key, value interface{}) bool {
		stageEntity, ok := value.(stage.Entity)
		if !ok {
			return true // continue iteration
		}
		stages = append(stages, stageEntity)
		return true // continue iteration
	})

	return stages, nil
}

// Add inserts a new stage into the in-memory database
func (r *StageRepository) Add(ctx context.Context, data stage.Entity) (string, error) {
	id := uuid.New().String()
	r.db.Store(id, data)
	return id, nil
}

// Get retrieves a stage by ID from the in-memory database
func (r *StageRepository) Get(ctx context.Context, id string) (stage.Entity, error) {
	value, ok := r.db.Load(id)
	if !ok {
		return stage.Entity{}, sql.ErrNoRows
	}

	stageEntity, ok := value.(stage.Entity)
	if !ok {
		return stageEntity, sql.ErrNoRows
	}

	return stageEntity, nil
}

// Update modifies an existing stage in the in-memory database
func (r *StageRepository) Update(ctx context.Context, id string, data stage.Entity) error {
	if _, ok := r.db.Load(id); !ok {
		return sql.ErrNoRows
	}

	r.db.Store(id, data)
	return nil
}

// Delete removes a stage by ID from the in-memory database
func (r *StageRepository) Delete(ctx context.Context, id string) error {
	if _, ok := r.db.Load(id); !ok {
		return sql.ErrNoRows
	}

	r.db.Delete(id)
	return nil
}

// UpdateStage returns the next or previous stage ID based on the given option
func (r *StageRepository) UpdateStage(ctx context.Context, currentStageID, direction string) (string, error) {

	currentStage, ok := r.db.Load(currentStageID)
	if !ok {
		return "", fmt.Errorf("current stage not found: %s", currentStageID)
	}
	currentStageEntity, ok := currentStage.(stage.Entity)
	if !ok {
		return "", fmt.Errorf("current stage not found or invalid type: %s", currentStageID)
	}
	if direction != "next" && direction != "prev" {
		newStage, ok := r.db.Load(direction)
		if !ok {
			return "", fmt.Errorf("invalid direction: %s", direction)
		}
		return newStage.(stage.Entity).ID, nil

	}
	if direction == "next" {
		nextStage, err := r.Get(ctx, currentStageEntity.AllowedTransitions[1])
		if err != nil {
			return "", fmt.Errorf("failed to get next stage: %w", err)
		}
		return nextStage.ID, nil

	}
	prevStage, err := r.Get(ctx, currentStageEntity.AllowedTransitions[0])
	if err != nil {
		return "", fmt.Errorf("failed to get previous stage: %w", err)
	}

	return prevStage.ID, nil

}

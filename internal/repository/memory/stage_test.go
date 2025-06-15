package memory

import (
	"context"
	"database/sql"
	"github.com/stretchr/testify/suite"
	"os"
	"testing"
)

type StageRepositorySuite struct {
	suite.Suite
	repository *StageRepository
	mockYaml   string
}

func (suite *StageRepositorySuite) SetupSuite() {

	suite.mockYaml = `stages:
  - id: "stage1"
    name: "Registration"
    order: 1
    transitions: ["stage1", "stage2"]
  - id: "stage2"
    name: "Onboarding"
    order: 2
    transitions: ["stage1", "stage3"]
  - id: "stage3"
    name: "Active"
    order: 3
    transitions: ["stage2", "stage4"]
  - id: "stage4"
    name: "Completed"
    order: 4
    transitions: ["stage3", "stage4"]`

	err := os.WriteFile("stages.yaml", []byte(suite.mockYaml), 0644)
	suite.NoError(err)

	suite.repository = NewStageRepository()
}

func (suite *StageRepositorySuite) TearDownSuite() {

	os.Remove("stages.yaml")
}

func (suite *StageRepositorySuite) TestList() {
	suite.Run("list all stages", func() {
		stages, err := suite.repository.List(context.Background())

		suite.NoError(err)
		suite.Equal(4, len(stages))

		hasRegistration := false
		hasOnboarding := false
		hasActive := false
		hasCompleted := false

		for _, s := range stages {
			switch *s.Name {
			case "Registration":
				hasRegistration = true
				suite.Equal(1, *s.Order)
			case "Onboarding":
				hasOnboarding = true
				suite.Equal(2, *s.Order)
			case "Active":
				hasActive = true
				suite.Equal(3, *s.Order)
			case "Completed":
				hasCompleted = true
				suite.Equal(4, *s.Order)
			}
		}

		suite.True(hasRegistration, "Should have Registration stage")
		suite.True(hasOnboarding, "Should have Onboarding stage")
		suite.True(hasActive, "Should have Active stage")
		suite.True(hasCompleted, "Should have Completed stage")
	})
}

func (suite *StageRepositorySuite) TestGet() {
	suite.Run("get existing stage", func() {
		stageEntity, err := suite.repository.Get(context.Background(), "stage2")

		suite.NoError(err)
		suite.Equal("stage2", stageEntity.ID)
		suite.Equal("Onboarding", *stageEntity.Name)
		suite.Equal(2, *stageEntity.Order)
		suite.Contains(stageEntity.AllowedTransitions, "stage1")
		suite.Contains(stageEntity.AllowedTransitions, "stage3")
	})

	suite.Run("get non-existent stage", func() {
		_, err := suite.repository.Get(context.Background(), "non-existent")

		suite.Error(err)
		suite.Equal(sql.ErrNoRows, err)
	})
}

func (suite *StageRepositorySuite) TestUpdateStage() {
	suite.Run("move to next stage", func() {
		nextID, err := suite.repository.UpdateStage(context.Background(), "stage1", "next")

		suite.NoError(err)
		suite.Equal("stage2", nextID)
	})

	suite.Run("move to previous stage", func() {
		prevID, err := suite.repository.UpdateStage(context.Background(), "stage2", "prev")

		suite.NoError(err)
		suite.Equal("stage1", prevID)
	})

	suite.Run("move to specific stage", func() {
		specificID, err := suite.repository.UpdateStage(context.Background(), "stage1", "stage3")

		suite.NoError(err)
		suite.Equal("stage3", specificID)
	})

	suite.Run("with invalid current stage", func() {
		_, err := suite.repository.UpdateStage(context.Background(), "non-existent", "next")

		suite.Error(err)
		suite.Contains(err.Error(), "current stage not found")
	})

	suite.Run("with invalid direction", func() {
		_, err := suite.repository.UpdateStage(context.Background(), "stage1", "invalid-direction")

		suite.Error(err)
		suite.Contains(err.Error(), "invalid direction")
	})
}

func TestStageRepositorySuite(t *testing.T) {
	suite.Run(t, new(StageRepositorySuite))
}

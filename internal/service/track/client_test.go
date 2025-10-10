package track

import (
	"TrackMe/internal/domain/client"
	"TrackMe/internal/domain/contract"
	"TrackMe/internal/domain/metric"
	"TrackMe/internal/domain/prometheus"
	"TrackMe/internal/domain/stage"
	"TrackMe/pkg/store"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
)

type MockClientRepository struct {
	mock.Mock
}

func (m *MockClientRepository) Get(ctx context.Context, id string) (client.Entity, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(client.Entity), args.Error(1)
}

func (m *MockClientRepository) GetByEmail(ctx context.Context, email string) (client.Entity, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(client.Entity), args.Error(1)
}

func (m *MockClientRepository) List(ctx context.Context, filters client.Filters, limit, offset int) ([]client.Entity, int, error) {
	args := m.Called(ctx, filters, limit, offset)
	return args.Get(0).([]client.Entity), args.Int(1), args.Error(2)
}

func (m *MockClientRepository) Create(ctx context.Context, data client.Entity) (client.Entity, error) {
	args := m.Called(ctx, data)
	return args.Get(0).(client.Entity), args.Error(1)
}

func (m *MockClientRepository) Update(ctx context.Context, id string, entity client.Entity) (client.Entity, error) {
	args := m.Called(ctx, id, entity)
	return args.Get(0).(client.Entity), args.Error(1)
}

func (m *MockClientRepository) Count(ctx context.Context, filter bson.M) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockClientRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type MockStageRepository struct {
	mock.Mock
}

func (m *MockStageRepository) Get(ctx context.Context, id string) (stage.Entity, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(stage.Entity), args.Error(1)
}

func (m *MockStageRepository) List(ctx context.Context) ([]stage.Entity, error) {
	args := m.Called(ctx)
	return args.Get(0).([]stage.Entity), args.Error(1)
}

func (m *MockStageRepository) UpdateStage(ctx context.Context, currentStageID, direction string) (string, error) {
	args := m.Called(ctx, currentStageID, direction)
	return args.String(0), args.Error(1)
}

type MockMetricRepository struct {
	mock.Mock
}

func (m *MockMetricRepository) List(ctx context.Context, filters metric.Filters) ([]metric.Entity, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).([]metric.Entity), args.Error(1)
}

func (m *MockMetricRepository) Add(ctx context.Context, data metric.Entity) (string, error) {
	args := m.Called(ctx, data)
	return args.String(0), args.Error(1)
}

func (m *MockMetricRepository) Update(ctx context.Context, id string, data metric.Entity) (metric.Entity, error) {
	args := m.Called(ctx, id, data)
	return args.Get(0).(metric.Entity), args.Error(1)
}

type ClientServiceTestSuite struct {
	suite.Suite
	service              *Service
	clientRepositoryMock *MockClientRepository
	stageRepositoryMock  *MockStageRepository
	metricRepositoryMock *MockMetricRepository
	metricCacheMock      *MockClientMetricCache
	PrometheusMetrics    prometheus.Entity
}

func (suite *ClientServiceTestSuite) SetupSuite() {
	suite.PrometheusMetrics = NewForTesting()
}

func (suite *ClientServiceTestSuite) SetupTest() {
	suite.clientRepositoryMock = new(MockClientRepository)
	suite.stageRepositoryMock = new(MockStageRepository)
	suite.metricRepositoryMock = new(MockMetricRepository)
	suite.metricCacheMock = new(MockClientMetricCache)

	suite.service = &Service{
		clientRepository:  suite.clientRepositoryMock,
		StageRepository:   suite.stageRepositoryMock,
		MetricRepository:  suite.metricRepositoryMock,
		PrometheusMetrics: suite.PrometheusMetrics,
		MetricCache:       suite.metricCacheMock,
	}
}

func (suite *ClientServiceTestSuite) TestListClients() {
	ctx := context.Background()
	filters := client.Filters{
		Stage: "registration",
	}
	limit := 10
	offset := 0

	name1 := "John Doe"
	email1 := "john@example.com"
	stage1 := "registration"
	source1 := "website"
	channel1 := "organic"
	app1 := "installed"
	isActive1 := true
	now := time.Now()
	lastUpdated1 := now.Add(-24 * time.Hour)
	lastLogin1 := now.Add(-2 * time.Hour)

	name2 := "Jane Smith"
	email2 := "jane@example.com"
	stage2 := "registration"
	source2 := "referral"
	channel2 := "partner"
	app2 := "not_installed"
	isActive2 := false
	lastUpdated2 := now.Add(-12 * time.Hour)
	lastLogin2 := now.Add(-1 * time.Hour)

	entities := []client.Entity{
		{
			ID:               "client1",
			Name:             &name1,
			Email:            &email1,
			CurrentStage:     &stage1,
			RegistrationDate: &now,
			LastUpdated:      &lastUpdated1,
			IsActive:         &isActive1,
			Source:           &source1,
			Channel:          &channel1,
			App:              &app1,
			LastLogin:        &lastLogin1,
			Contracts:        []contract.Entity{},
		},
		{
			ID:               "client2",
			Name:             &name2,
			Email:            &email2,
			CurrentStage:     &stage2,
			RegistrationDate: &now,
			LastUpdated:      &lastUpdated2,
			IsActive:         &isActive2,
			Source:           &source2,
			Channel:          &channel2,
			App:              &app2,
			LastLogin:        &lastLogin2,
			Contracts:        []contract.Entity{},
		},
	}

	suite.clientRepositoryMock.On("List", ctx, filters, limit, offset).Return(entities, 2, nil)

	responses, total, err := suite.service.ListClients(ctx, filters, limit, offset)

	suite.NoError(err)
	suite.Equal(2, total)
	suite.Len(responses, 2)
	suite.Equal("John Doe", responses[0].Name)
	suite.Equal("jane@example.com", responses[1].Email)
	suite.Equal("registration", responses[0].CurrentStage)
	suite.Equal("client2", responses[1].ID)
}

func (suite *ClientServiceTestSuite) TestUpdateClient() {
	ctx := context.Background()
	clientID := "client123"

	regDate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	currentStage := "registration"
	isActive := false
	source := "old-source"
	channel := "old-channel"
	app := "not_installed"
	lastLogin := time.Date(2022, 12, 31, 0, 0, 0, 0, time.UTC)
	now := time.Now()
	existingClient := client.Entity{
		ID:               clientID,
		CurrentStage:     &currentStage,
		RegistrationDate: &regDate,
		IsActive:         &isActive,
		Source:           &source,
		Channel:          &channel,
		LastUpdated:      &now,
		App:              &app,
		LastLogin:        &lastLogin,
	}

	name := "Updated Name"
	email := "updated@example.com"
	newIsActive := true
	newSource := "website"
	newChannel := "direct"
	newApp := "installed"
	newLastLogin := time.Now()

	req := client.Request{
		Name:      name,
		Email:     email,
		Stage:     "next",
		IsActive:  &newIsActive,
		Source:    newSource,
		Channel:   newChannel,
		App:       newApp,
		LastLogin: newLastLogin,
	}

	newStage := "onboarding"
	suite.stageRepositoryMock.On("UpdateStage", ctx, currentStage, req.Stage).Return(newStage, nil)

	suite.clientRepositoryMock.On("Get", ctx, clientID).Return(existingClient, nil)

	suite.clientRepositoryMock.On(
		"Update",
		ctx,
		clientID,
		mock.AnythingOfType("client.Entity"),
	).Run(func(args mock.Arguments) {

		entity := args.Get(2).(client.Entity)
		suite.Equal(clientID, entity.ID)
		suite.Equal(name, *entity.Name)
		suite.Equal(email, *entity.Email)
		suite.Equal(newStage, *entity.CurrentStage)
		suite.Equal(newIsActive, *entity.IsActive)
		suite.Equal(newSource, *entity.Source)
		suite.Equal(newChannel, *entity.Channel)
		suite.Equal(newApp, *entity.App)
		suite.NotNil(entity.LastLogin)

		suite.NotNil(entity.LastUpdated)
	}).Return(client.Entity{
		ID:               clientID,
		Name:             &name,
		Email:            &email,
		CurrentStage:     &newStage,
		RegistrationDate: &regDate,
		IsActive:         &newIsActive,
		Source:           &newSource,
		Channel:          &newChannel,
		App:              &newApp,
		LastLogin:        &newLastLogin,
		LastUpdated:      &now,
		Contracts:        []contract.Entity{},
	}, nil)

	response, err := suite.service.UpdateClient(ctx, clientID, req)

	suite.NoError(err)
	suite.Equal(clientID, response.ID)
	suite.Equal(name, response.Name)
	suite.Equal(email, response.Email)
	suite.Equal(newStage, response.CurrentStage)
	suite.Equal(newSource, response.Source)
	suite.Equal(newChannel, response.Channel)
	suite.Equal(newIsActive, response.IsActive)
	suite.Equal("installed", response.App.Status)

	suite.Equal(0, len(response.Contracts))
}

func (suite *ClientServiceTestSuite) TestUpdateClientRollback() {
	ctx := context.Background()
	clientID := "client123"

	currentStage := "onboarding"
	name := "Test Client"
	email := "test@example.com"
	isActive := true
	now := time.Now()
	source := "website"
	channel := "direct"
	app := "installed"

	existingClient := client.Entity{
		ID:               clientID,
		Name:             &name,
		Email:            &email,
		CurrentStage:     &currentStage,
		IsActive:         &isActive,
		LastUpdated:      &now,
		Source:           &source,
		Channel:          &channel,
		App:              &app,
		LastLogin:        &now,
		RegistrationDate: &now,
		Contracts:        []contract.Entity{},
	}

	req := client.Request{
		Stage: "prev",
	}

	prevStage := "registration"
	suite.stageRepositoryMock.On("UpdateStage", ctx, currentStage, req.Stage).Return(prevStage, nil)

	suite.metricCacheMock.On("List", ctx, metric.Filters{
		Type:     "rollback-count",
		Interval: "day",
	}).Return([]metric.Entity{}, nil)

	suite.metricRepositoryMock.On("List", ctx, metric.Filters{
		Type:     "rollback-count",
		Interval: "day",
	}).Return([]metric.Entity{}, nil)

	suite.metricRepositoryMock.On("Add", ctx, mock.AnythingOfType("metric.Entity")).Return("metric123", nil)

	suite.clientRepositoryMock.On("Get", ctx, clientID).Return(existingClient, nil)

	suite.clientRepositoryMock.On(
		"Update",
		ctx,
		clientID,
		mock.MatchedBy(func(entity client.Entity) bool {
			return *entity.CurrentStage == prevStage
		}),
	).Return(client.Entity{
		ID:               clientID,
		Name:             &name,
		Email:            &email,
		CurrentStage:     &prevStage,
		IsActive:         &isActive,
		LastUpdated:      &now,
		Source:           &source,
		Channel:          &channel,
		App:              &app,
		LastLogin:        &now,
		RegistrationDate: &now,
	}, nil)

	response, err := suite.service.UpdateClient(ctx, clientID, req)

	suite.NoError(err)
	suite.Equal(prevStage, response.CurrentStage)
}

func (suite *ClientServiceTestSuite) TestListClientsWithStageFilter() {
	ctx := context.Background()
	filters := client.Filters{
		Stage: "onboarding",
	}
	limit := 10
	offset := 0

	name1 := "John Doe"
	email1 := "john@example.com"
	stage1 := "onboarding"
	isActive1 := true
	now := time.Now()
	source1 := "website"
	channel1 := "direct"
	app1 := "installed"

	entities := []client.Entity{
		{
			ID:               "client1",
			Name:             &name1,
			Email:            &email1,
			CurrentStage:     &stage1,
			IsActive:         &isActive1,
			RegistrationDate: &now,
			LastUpdated:      &now,
			Source:           &source1,
			Channel:          &channel1,
			App:              &app1,
			LastLogin:        &now,
			Contracts:        []contract.Entity{},
		},
	}

	suite.clientRepositoryMock.On("List", ctx, filters, limit, offset).Return(entities, 1, nil)

	responses, total, err := suite.service.ListClients(ctx, filters, limit, offset)

	suite.NoError(err)
	suite.Equal(1, total)
	suite.Len(responses, 1)
	suite.Equal("onboarding", responses[0].CurrentStage)
}

func (suite *ClientServiceTestSuite) TestListClientsWithSourceFilter() {
	ctx := context.Background()
	filters := client.Filters{
		Source: "referral",
	}
	limit := 10
	offset := 0

	name1 := "Jane Smith"
	email1 := "jane@example.com"
	stage1 := "registration"
	source1 := "referral"
	channel1 := "partner"
	app1 := "installed"
	isActive1 := true
	now := time.Now()

	entities := []client.Entity{
		{
			ID:               "client2",
			Name:             &name1,
			Email:            &email1,
			CurrentStage:     &stage1,
			Source:           &source1,
			Channel:          &channel1,
			App:              &app1,
			IsActive:         &isActive1,
			RegistrationDate: &now,
			LastUpdated:      &now,
			LastLogin:        &now,
			Contracts:        []contract.Entity{},
		},
	}

	suite.clientRepositoryMock.On("List", ctx, filters, limit, offset).Return(entities, 1, nil)

	responses, total, err := suite.service.ListClients(ctx, filters, limit, offset)

	suite.NoError(err)
	suite.Equal(1, total)
	suite.Len(responses, 1)
	suite.Equal("referral", responses[0].Source)
}

func (suite *ClientServiceTestSuite) TestListClientsWithPagination() {
	ctx := context.Background()
	filters := client.Filters{}
	now := time.Now()

	name := "Test User"
	email := "test@example.com"
	curStage := "registration"
	isActive := true
	source := "website"
	channel := "direct"
	app := "installed"

	suite.Run("first page", func() {
		limit := 2
		offset := 0

		entities := []client.Entity{
			{
				ID:               "client1",
				Name:             &name,
				Email:            &email,
				CurrentStage:     &curStage,
				IsActive:         &isActive,
				RegistrationDate: &now,
				LastUpdated:      &now,
				Source:           &source,
				Channel:          &channel,
				App:              &app,
				LastLogin:        &now,
				Contracts:        []contract.Entity{},
			},
			{
				ID:               "client2",
				Name:             &name,
				Email:            &email,
				CurrentStage:     &curStage,
				IsActive:         &isActive,
				RegistrationDate: &now,
				LastUpdated:      &now,
				Source:           &source,
				Channel:          &channel,
				App:              &app,
				LastLogin:        &now,
				Contracts:        []contract.Entity{},
			},
		}

		suite.clientRepositoryMock.On("List", ctx, filters, limit, offset).Return(entities, 5, nil)

		responses, total, err := suite.service.ListClients(ctx, filters, limit, offset)

		suite.NoError(err)
		suite.Equal(5, total)
		suite.Len(responses, 2)
		suite.Equal("client1", responses[0].ID)
		suite.Equal("client2", responses[1].ID)
	})

	suite.Run("second page", func() {
		limit := 2
		offset := 2

		entities := []client.Entity{
			{
				ID:               "client3",
				Name:             &name,
				Email:            &email,
				CurrentStage:     &curStage,
				IsActive:         &isActive,
				RegistrationDate: &now,
				LastUpdated:      &now,
				Source:           &source,
				Channel:          &channel,
				App:              &app,
				LastLogin:        &now,
				Contracts:        []contract.Entity{},
			},
			{
				ID:               "client4",
				Name:             &name,
				Email:            &email,
				CurrentStage:     &curStage,
				IsActive:         &isActive,
				RegistrationDate: &now,
				LastUpdated:      &now,
				Source:           &source,
				Channel:          &channel,
				App:              &app,
				LastLogin:        &now,
				Contracts:        []contract.Entity{},
			},
		}

		suite.clientRepositoryMock.On("List", ctx, filters, limit, offset).Return(entities, 5, nil)

		responses, total, err := suite.service.ListClients(ctx, filters, limit, offset)

		suite.NoError(err)
		suite.Equal(5, total)
		suite.Len(responses, 2)
		suite.Equal("client3", responses[0].ID)
		suite.Equal("client4", responses[1].ID)
	})

	suite.Run("last page", func() {
		limit := 2
		offset := 4

		entities := []client.Entity{
			{
				ID:               "client5",
				Name:             &name,
				Email:            &email,
				CurrentStage:     &curStage,
				IsActive:         &isActive,
				RegistrationDate: &now,
				LastUpdated:      &now,
				Source:           &source,
				Channel:          &channel,
				App:              &app,
				LastLogin:        &now,
				Contracts:        []contract.Entity{},
			},
		}

		suite.clientRepositoryMock.On("List", ctx, filters, limit, offset).Return(entities, 5, nil)

		responses, total, err := suite.service.ListClients(ctx, filters, limit, offset)

		suite.NoError(err)
		suite.Equal(5, total)
		suite.Len(responses, 1)
		suite.Equal("client5", responses[0].ID)
	})
}

func TestServiceConfiguration(t *testing.T) {

	clientRepo := new(MockClientRepository)
	stageRepo := new(MockStageRepository)
	metricRepo := new(MockMetricRepository)
	metricCache := new(MockClientMetricCache)
	prometheusMetrics := prometheus.New()

	t.Run("new service with no config", func(t *testing.T) {
		service, err := New()
		assert.NoError(t, err)
		assert.NotNil(t, service)
	})

	t.Run("with client repository", func(t *testing.T) {
		service, err := New(WithClientRepository(clientRepo))
		assert.NoError(t, err)
		assert.Equal(t, clientRepo, service.clientRepository)
	})

	t.Run("with stage repository", func(t *testing.T) {
		service, err := New(WithStageRepository(stageRepo))
		assert.NoError(t, err)
		assert.Equal(t, stageRepo, service.StageRepository)
	})

	t.Run("with metric repository", func(t *testing.T) {
		service, err := New(WithMetricRepository(metricRepo))
		assert.NoError(t, err)
		assert.Equal(t, metricRepo, service.MetricRepository)
	})

	t.Run("with prometheus metrics", func(t *testing.T) {
		service, err := New(WithPrometheusMetrics(prometheusMetrics))
		assert.NoError(t, err)
		assert.Equal(t, prometheusMetrics, service.PrometheusMetrics)
	})

	t.Run("with metric cache", func(t *testing.T) {
		service, err := New(WithMetricCache(metricCache))
		assert.NoError(t, err)
		assert.Equal(t, metricCache, service.MetricCache)
	})

	t.Run("with all configurations", func(t *testing.T) {
		service, err := New(
			WithClientRepository(clientRepo),
			WithStageRepository(stageRepo),
			WithMetricRepository(metricRepo),
			WithPrometheusMetrics(prometheusMetrics),
			WithMetricCache(metricCache),
		)

		assert.NoError(t, err)
		assert.Equal(t, clientRepo, service.clientRepository)
		assert.Equal(t, stageRepo, service.StageRepository)
		assert.Equal(t, metricRepo, service.MetricRepository)
		assert.Equal(t, prometheusMetrics, service.PrometheusMetrics)
		assert.Equal(t, metricCache, service.MetricCache)
	})

	t.Run("with configuration error", func(t *testing.T) {
		expectedErr := errors.New("configuration error")
		errorConfig := func(s *Service) error {
			return expectedErr
		}

		service, err := New(errorConfig)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, Service{}, *service)
	})
}

func (suite *ClientServiceTestSuite) TestUpdateClientWithStageUpdateError() {
	ctx := context.Background()
	clientID := "client123"
	currentStage := "registration"

	name := "Test Client"
	email := "test@example.com"
	isActive := true
	existingClient := client.Entity{
		ID:           clientID,
		Name:         &name,
		Email:        &email,
		CurrentStage: &currentStage,
		IsActive:     &isActive,
	}

	req := client.Request{
		Stage: "next",
	}

	suite.clientRepositoryMock.On("Get", ctx, clientID).Return(existingClient, nil)

	stageError := errors.New("invalid stage transition")
	suite.stageRepositoryMock.On("UpdateStage", ctx, currentStage, req.Stage).Return("", stageError)

	_, err := suite.service.UpdateClient(ctx, clientID, req)

	suite.Error(err)
	suite.Contains(err.Error(), "invalid stage transition")
	suite.clientRepositoryMock.AssertExpectations(suite.T())
	suite.stageRepositoryMock.AssertExpectations(suite.T())
}

func (suite *ClientServiceTestSuite) TestUpdateClientWithGetClientError() {
	ctx := context.Background()
	clientID := "client123"

	req := client.Request{
		Stage: "next",
	}

	getError := errors.New("database error")
	suite.clientRepositoryMock.On("Get", ctx, clientID).Return(client.Entity{}, getError)

	_, err := suite.service.UpdateClient(ctx, clientID, req)

	suite.Error(err)
	suite.Equal(getError, err)
	suite.clientRepositoryMock.AssertExpectations(suite.T())
}

func (suite *ClientServiceTestSuite) TestListClientsWithNonExistentID() {
	ctx := context.Background()
	nonExistentID := "non-existent-id"

	filters := client.Filters{
		ID: nonExistentID,
	}
	limit := 10
	offset := 0

	suite.clientRepositoryMock.On("List", ctx, filters, limit, offset).Return(
		[]client.Entity{}, 0, store.ErrorNotFound)

	responses, total, err := suite.service.ListClients(ctx, filters, limit, offset)

	suite.Error(err)
	suite.Equal(store.ErrorNotFound, err)
	suite.Equal(0, total)
	suite.Empty(responses)
	suite.clientRepositoryMock.AssertExpectations(suite.T())
}

func (suite *ClientServiceTestSuite) TestUpdateClientWithEmptyName() {
	ctx := context.Background()
	clientID := "client123"
	currentStage := "registration"

	name := "Test Client"
	email := "test@example.com"
	isActive := true
	now := time.Now()
	source := "website"
	channel := "direct"
	app := "installed"

	existingClient := client.Entity{
		ID:               clientID,
		Name:             &name,
		Email:            &email,
		CurrentStage:     &currentStage,
		IsActive:         &isActive,
		LastUpdated:      &now,
		RegistrationDate: &now,
		Source:           &source,
		Channel:          &channel,
		App:              &app,
		LastLogin:        &now,
		Contracts:        []contract.Entity{},
	}

	req := client.Request{
		Stage:    "next",
		Name:     "",
		Email:    "updated@example.com",
		IsActive: &isActive,
		Source:   "website",
		Channel:  "direct",
	}

	suite.clientRepositoryMock.On("Get", ctx, clientID).Return(existingClient, nil)

	suite.stageRepositoryMock.On("UpdateStage", ctx, currentStage, req.Stage).Return("active", nil)

	suite.clientRepositoryMock.On(
		"Update",
		ctx,
		clientID,
		mock.MatchedBy(func(e client.Entity) bool {
			return e.ID == clientID && *e.Name == "Guest_"+clientID
		}),
	).Return(client.Entity{
		ID:               clientID,
		Name:             stringPtr("Guest_" + clientID),
		CurrentStage:     stringPtr("active"),
		IsActive:         boolPtr(true),
		Email:            &email,
		LastUpdated:      &now,
		RegistrationDate: &now,
		Source:           &source,
		Channel:          &channel,
		App:              &app,
		LastLogin:        &now,
	}, nil)

	response, err := suite.service.UpdateClient(ctx, clientID, req)

	suite.NoError(err)
	suite.Equal("Guest_"+clientID, response.Name)
}

func (suite *ClientServiceTestSuite) TestUpdateClientWithUpdateError() {
	ctx := context.Background()
	clientID := "client123"
	currentStage := "registration"

	name := "Test Client"
	email := "test@example.com"
	existingClient := client.Entity{
		ID:           clientID,
		Name:         &name,
		Email:        &email,
		CurrentStage: &currentStage,
	}

	req := client.Request{
		Stage: "next",
		Name:  "Updated Name",
	}

	suite.clientRepositoryMock.On("Get", ctx, clientID).Return(existingClient, nil)

	suite.stageRepositoryMock.On("UpdateStage", ctx, currentStage, req.Stage).Return("active", nil)

	updateError := errors.New("database error")
	suite.clientRepositoryMock.On(
		"Update",
		ctx,
		clientID,
		mock.Anything,
	).Return(client.Entity{}, updateError)

	_, err := suite.service.UpdateClient(ctx, clientID, req)

	suite.Error(err)
	suite.Equal(updateError, err)
}

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

type MockClientMetricCache struct {
	mock.Mock
}

func (m *MockClientMetricCache) Set(ctx context.Context, id string, entity metric.Entity) error {
	args := m.Called(ctx, id, entity)
	return args.Error(0)
}

func (m *MockClientMetricCache) List(ctx context.Context, filters metric.Filters) ([]metric.Entity, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).([]metric.Entity), args.Error(1)
}

func (m *MockClientMetricCache) StoreList(ctx context.Context, filters metric.Filters, entities []metric.Entity) error {
	args := m.Called(ctx, filters, entities)
	return args.Error(0)
}

func (m *MockClientMetricCache) Get(ctx context.Context, id string) (metric.Entity, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(metric.Entity), args.Error(1)
}

func (m *MockClientMetricCache) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockClientMetricCache) InvalidateListCache(ctx context.Context, filters metric.Filters) error {
	return nil
}

func TestClientService(t *testing.T) {
	suite.Run(t, new(ClientServiceTestSuite))
}

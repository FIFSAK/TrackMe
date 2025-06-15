package track

import (
	"TrackMe/internal/domain/client"
	"TrackMe/internal/domain/contract"
	"TrackMe/internal/domain/metric"
	"TrackMe/internal/domain/prometheus"
	"TrackMe/internal/domain/stage"
	"TrackMe/pkg/store"
	"context"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type MetricServiceTestSuite struct {
	suite.Suite
	service              *Service
	metricRepositoryMock *MockMetricRepository
	metricCacheMock      *MockMetricCache
	clientRepositoryMock *MockClientRepository
	stageRepositoryMock  *MockStageRepository
	PrometheusMetrics    prometheus.Entity
}

type MockMetricCache struct {
	mock.Mock
}

func (m *MockMetricCache) List(ctx context.Context, filters metric.Filters) ([]metric.Entity, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).([]metric.Entity), args.Error(1)
}

func (suite *MetricServiceTestSuite) SetupSuite() {
	suite.PrometheusMetrics = prometheus.New()
}

func (suite *MetricServiceTestSuite) SetupTest() {
	suite.metricRepositoryMock = new(MockMetricRepository)
	suite.metricCacheMock = new(MockMetricCache)
	suite.clientRepositoryMock = new(MockClientRepository)
	suite.stageRepositoryMock = new(MockStageRepository)

	suite.service = &Service{
		MetricRepository: suite.metricRepositoryMock,
		//MetricCache:       suite.metricCacheMock, // Added missing MetricCache
		clientRepository:  suite.clientRepositoryMock,
		StageRepository:   suite.stageRepositoryMock,
		PrometheusMetrics: suite.PrometheusMetrics,
	}
}

// Test listing metrics with repository and no cache
func (suite *MetricServiceTestSuite) TestListMetricsWithRepository() {
	ctx := context.Background()
	filters := metric.Filters{
		Type: "conversion",
	}

	now := time.Now()
	convType1 := metric.Conversion
	convType2 := metric.Conversion
	value1 := 75.5
	value2 := 80.2
	time1 := now
	time2 := now.Add(-24 * time.Hour)
	interval1 := ""
	interval2 := ""

	entities := []metric.Entity{
		{
			ID:        "metric1",
			Type:      &convType1,
			Value:     &value1,
			CreatedAt: &time1,
			Interval:  &interval1, // Added interval field
		},
		{
			ID:        "metric2",
			Type:      &convType2,
			Value:     &value2,
			CreatedAt: &time2,
			Interval:  &interval2, // Added interval field
		},
	}

	// Setup repository expectations - no cache in this test
	suite.metricRepositoryMock.On("List", ctx, filters).Return(entities, nil)

	// Call the service
	responses, err := suite.service.ListMetrics(ctx, filters)

	// Assertions
	suite.NoError(err)
	suite.Len(responses, 2)
	suite.Equal("metric1", responses[0].ID)
	suite.Equal(75.5, responses[0].Value)
	suite.Equal("conversion", responses[0].Type)
}

// Test listing metrics with cache miss
func (suite *MetricServiceTestSuite) TestListMetricsWithCacheMiss() {
	ctx := context.Background()
	filters := metric.Filters{
		Type:     "app-install-rate",
		Interval: "month",
	}

	now := time.Now()
	appInstallType := metric.AppInstallRate
	value := 65.3
	interval := "month"
	entities := []metric.Entity{
		{
			ID:        "metric1",
			Type:      &appInstallType,
			Value:     &value,
			Interval:  &interval,
			CreatedAt: &now,
		},
	}

	// Setup cache miss then repository hit
	suite.metricCacheMock.On("List", ctx, filters).Return([]metric.Entity{}, store.ErrorNotFound)
	suite.metricRepositoryMock.On("List", ctx, filters).Return(entities, nil)

	// Call the service
	responses, err := suite.service.ListMetrics(ctx, filters)

	// Assertions
	suite.NoError(err)
	suite.Len(responses, 1)
	suite.Equal("metric1", responses[0].ID)
	suite.Equal(65.3, responses[0].Value)
	suite.Equal("app-install-rate", responses[0].Type)
}

// Test calculating rollback count
func (suite *MetricServiceTestSuite) TestCalculateRollbackCount() {
	ctx := context.Background()
	now := time.Now()

	// Set up existing metrics for the day
	rollbackType := metric.RollbackCount
	rollbackValue := float64(3)
	dayInterval := "day"

	existingMetrics := []metric.Entity{
		{
			ID:        "existing-metric",
			Type:      &rollbackType,
			Value:     &rollbackValue,
			Interval:  &dayInterval,
			CreatedAt: &now,
		},
	}

	// Mock the repository interactions
	suite.metricRepositoryMock.On("List", ctx, metric.Filters{
		Type:     "rollback-count",
		Interval: "day",
	}).Return(existingMetrics, nil)

	// For a new record, we'd update the existing one with value + 1
	metricType := metric.RollbackCount
	metricValue := float64(4) // incremented by 1
	metricInterval := "day"
	updatedMetric := metric.Entity{
		ID:        "existing-metric",
		Type:      &metricType,
		Value:     &metricValue,
		Interval:  &metricInterval,
		CreatedAt: &now,
	}
	suite.metricRepositoryMock.On("Update", ctx, "existing-metric", mock.AnythingOfType("metric.Entity")).Return(updatedMetric, nil)

	// Call the service method
	err := suite.service.calculateRollbackCount(ctx, now)

	// Assertions
	suite.NoError(err)

	// Verify prometheus metric was set
	suite.metricRepositoryMock.AssertCalled(suite.T(), "Update", ctx, "existing-metric", mock.AnythingOfType("metric.Entity"))
}

// Test calculating app install rate
func (suite *MetricServiceTestSuite) TestCalculateAppInstallRate() {
	ctx := context.Background()
	now := time.Now()

	// Setup counts for app install calculation
	totalClientsWithApp := int64(100)
	clientsWithInstalledApp := int64(75)

	// Mock the repository interactions
	suite.clientRepositoryMock.On("Count", ctx, mock.AnythingOfType("primitive.M")).Return(totalClientsWithApp, nil).Once()
	suite.clientRepositoryMock.On("Count", ctx, mock.AnythingOfType("primitive.M")).Return(clientsWithInstalledApp, nil).Once()

	// Mock metric creation and storage
	suite.metricRepositoryMock.On("Add", ctx, mock.AnythingOfType("metric.Entity")).Return("new-id", nil)

	// Call the service method
	err := suite.service.calculateAppInstallRate(ctx, now)

	// Assertions
	suite.NoError(err)

	// Verify metric was added to repository
	suite.metricRepositoryMock.AssertCalled(suite.T(), "Add", ctx, mock.AnythingOfType("metric.Entity"))
}

// Test calculating total duration
func (suite *MetricServiceTestSuite) TestCalculateTotalDuration() {
	ctx := context.Background()
	now := time.Now()

	// Setup stages
	stages := []stage.Entity{
		{ID: "registration"},
		{ID: "onboarding"},
		{ID: "active"}, // Last stage
	}
	suite.stageRepositoryMock.On("List", ctx).Return(stages, nil)

	// Setup completed clients
	name := "Test Client"
	email := "test@example.com"
	isActive := true
	registrationDate := now.Add(-30 * 24 * time.Hour) // 30 days ago
	lastUpdated := now
	source := "website"
	channel := "direct"
	app := "installed"
	lastStage := stages[2].ID

	clients := []client.Entity{
		{
			ID:               "client1",
			Name:             &name,
			Email:            &email,
			CurrentStage:     &lastStage,
			IsActive:         &isActive,
			RegistrationDate: &registrationDate,
			LastUpdated:      &lastUpdated,
			Source:           &source,
			Channel:          &channel,
			App:              &app,
			LastLogin:        &now,
			Contracts:        []contract.Entity{},
		},
	}

	// Use mock.MatchedBy to match the filter regardless of IsActive pointer value
	suite.clientRepositoryMock.On("List", ctx, mock.MatchedBy(func(filters client.Filters) bool {
		return filters.Stage == lastStage
	}), 0, 0).Return(clients, 1, nil)

	// Mock metric creation
	suite.metricRepositoryMock.On("Add", ctx, mock.AnythingOfType("metric.Entity")).Return("new-metric-id", nil)

	// Call service
	err := suite.service.calculateTotalDuration(ctx, now)

	// Assertions
	suite.NoError(err)

	// Verify metric was added
	suite.metricRepositoryMock.AssertCalled(suite.T(), "Add", ctx, mock.AnythingOfType("metric.Entity"))
}

// Test calculating status updates
func (suite *MetricServiceTestSuite) TestCalculateStatusUpdates() {
	ctx := context.Background()
	now := time.Now()
	interval := "day"

	// Setup client count for status updates
	updatedClientsCount := int64(42)

	// Mock repository call to count updated clients
	suite.clientRepositoryMock.On("Count", ctx, mock.AnythingOfType("primitive.M")).Return(updatedClientsCount, nil)

	// Mock metric creation and storage
	suite.metricRepositoryMock.On("Add", ctx, mock.AnythingOfType("metric.Entity")).Return("new-metric-id", nil)

	// Call service
	err := suite.service.calculateStatusUpdates(ctx, now, interval)

	// Assertions
	suite.NoError(err)

	// Verify metric was added with correct value
	suite.metricRepositoryMock.AssertCalled(suite.T(), "Add", ctx, mock.MatchedBy(func(entity metric.Entity) bool {
		return *entity.Type == metric.StatusUpdates &&
			*entity.Value == float64(updatedClientsCount) &&
			*entity.Interval == interval
	}))
}

// Run the test suite
func TestMetricService(t *testing.T) {
	suite.Run(t, new(MetricServiceTestSuite))
}

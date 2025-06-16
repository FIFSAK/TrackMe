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
	prome "github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"testing"
	"time"
)

func NewForTesting() prometheus.Entity {
	testRegistry := prome.NewRegistry()
	return prometheus.NewWithRegistry(testRegistry)
}

type MetricServiceTestSuite struct {
	suite.Suite
	service              *Service
	metricRepositoryMock *MockMetricRepository
	metricCacheMock      *MockMetricCache
	clientRepositoryMock *MockClientRepository
	stageRepositoryMock  *MockStageRepository
	prometheusMetrics    prometheus.Entity
}

type MockMetricCache struct {
	mock.Mock
}

func (m *MockMetricCache) Set(ctx context.Context, id string, entity metric.Entity) error {
	args := m.Called(ctx, id, entity)
	return args.Error(0)
}

func (m *MockMetricCache) List(ctx context.Context, filters metric.Filters) ([]metric.Entity, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).([]metric.Entity), args.Error(1)
}

func (m *MockMetricCache) StoreList(ctx context.Context, filters metric.Filters, entities []metric.Entity) error {
	args := m.Called(ctx, filters, entities)
	return args.Error(0)
}

func (m *MockMetricCache) Get(ctx context.Context, id string) (metric.Entity, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(metric.Entity), args.Error(1)
}

func (m *MockMetricCache) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMetricCache) InvalidateListCache(ctx context.Context, filters metric.Filters) error {
	args := m.Called(ctx, filters)
	return args.Error(0)
}

func (suite *MetricServiceTestSuite) SetupSuite() {
}

func (suite *MetricServiceTestSuite) SetupTest() {
	suite.metricRepositoryMock = new(MockMetricRepository)
	suite.metricCacheMock = new(MockMetricCache)
	suite.clientRepositoryMock = new(MockClientRepository)
	suite.stageRepositoryMock = new(MockStageRepository)
	suite.prometheusMetrics = NewForTesting()

	suite.service = &Service{
		MetricRepository:  suite.metricRepositoryMock,
		clientRepository:  suite.clientRepositoryMock,
		StageRepository:   suite.stageRepositoryMock,
		PrometheusMetrics: suite.prometheusMetrics,
	}
}

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
			Interval:  &interval1,
		},
		{
			ID:        "metric2",
			Type:      &convType2,
			Value:     &value2,
			CreatedAt: &time2,
			Interval:  &interval2,
		},
	}

	suite.metricRepositoryMock.On("List", ctx, filters).Return(entities, nil)

	responses, err := suite.service.ListMetrics(ctx, filters)

	suite.NoError(err)
	suite.Len(responses, 2)
	suite.Equal("metric1", responses[0].ID)
	suite.Equal(75.5, responses[0].Value)
	suite.Equal("conversion", responses[0].Type)
}

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

	suite.metricCacheMock.On("List", ctx, filters).Return([]metric.Entity{}, store.ErrorNotFound)
	suite.metricRepositoryMock.On("List", ctx, filters).Return(entities, nil)

	responses, err := suite.service.ListMetrics(ctx, filters)

	suite.NoError(err)
	suite.Len(responses, 1)
	suite.Equal("metric1", responses[0].ID)
	suite.Equal(65.3, responses[0].Value)
	suite.Equal("app-install-rate", responses[0].Type)
}

func (suite *MetricServiceTestSuite) TestCalculateRollbackCount() {
	ctx := context.Background()
	now := time.Now()

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

	suite.metricRepositoryMock.On("List", ctx, metric.Filters{
		Type:     "rollback-count",
		Interval: "day",
	}).Return(existingMetrics, nil)

	metricType := metric.RollbackCount
	metricValue := float64(4)
	metricInterval := "day"
	updatedMetric := metric.Entity{
		ID:        "existing-metric",
		Type:      &metricType,
		Value:     &metricValue,
		Interval:  &metricInterval,
		CreatedAt: &now,
	}
	suite.metricRepositoryMock.On("Update", ctx, "existing-metric", mock.AnythingOfType("metric.Entity")).Return(updatedMetric, nil)

	err := suite.service.calculateRollbackCount(ctx, now)

	suite.NoError(err)

	suite.metricRepositoryMock.AssertCalled(suite.T(), "Update", ctx, "existing-metric", mock.AnythingOfType("metric.Entity"))
}

func (suite *MetricServiceTestSuite) TestCalculateAppInstallRate() {
	ctx := context.Background()
	now := time.Now()

	totalClientsWithApp := int64(100)
	clientsWithInstalledApp := int64(75)

	suite.clientRepositoryMock.On("Count", ctx, mock.AnythingOfType("primitive.M")).Return(totalClientsWithApp, nil).Once()
	suite.clientRepositoryMock.On("Count", ctx, mock.AnythingOfType("primitive.M")).Return(clientsWithInstalledApp, nil).Once()

	suite.metricRepositoryMock.On("Add", ctx, mock.AnythingOfType("metric.Entity")).Return("new-id", nil)

	err := suite.service.calculateAppInstallRate(ctx, now)

	suite.NoError(err)

	suite.metricRepositoryMock.AssertCalled(suite.T(), "Add", ctx, mock.AnythingOfType("metric.Entity"))
}

func (suite *MetricServiceTestSuite) TestCalculateTotalDuration() {
	ctx := context.Background()
	now := time.Now()

	stages := []stage.Entity{
		{ID: "registration"},
		{ID: "onboarding"},
		{ID: "active"},
	}
	suite.stageRepositoryMock.On("List", ctx).Return(stages, nil)

	name := "Test Client"
	email := "test@example.com"
	isActive := true
	registrationDate := now.Add(-30 * 24 * time.Hour)
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

	suite.clientRepositoryMock.On("List", ctx, mock.MatchedBy(func(filters client.Filters) bool {
		return filters.Stage == lastStage
	}), 0, 0).Return(clients, 1, nil)

	suite.metricRepositoryMock.On("Add", ctx, mock.AnythingOfType("metric.Entity")).Return("new-metric-id", nil)

	err := suite.service.calculateTotalDuration(ctx, now)

	suite.NoError(err)

	suite.metricRepositoryMock.AssertCalled(suite.T(), "Add", ctx, mock.AnythingOfType("metric.Entity"))
}

func (suite *MetricServiceTestSuite) TestCalculateStatusUpdates() {
	ctx := context.Background()
	now := time.Now()
	interval := "day"

	updatedClientsCount := int64(42)

	suite.clientRepositoryMock.On("Count", ctx, mock.AnythingOfType("primitive.M")).Return(updatedClientsCount, nil)

	suite.metricRepositoryMock.On("Add", ctx, mock.AnythingOfType("metric.Entity")).Return("new-metric-id", nil)

	err := suite.service.calculateStatusUpdates(ctx, now, interval)

	suite.NoError(err)

	suite.metricRepositoryMock.AssertCalled(suite.T(), "Add", ctx, mock.MatchedBy(func(entity metric.Entity) bool {
		return *entity.Type == metric.StatusUpdates &&
			*entity.Value == float64(updatedClientsCount) &&
			*entity.Interval == interval
	}))
}

func (suite *MetricServiceTestSuite) TestCalculateClientsPerStage() {
	ctx := context.Background()
	now := time.Now()

	stages := []stage.Entity{
		{ID: "registration"},
		{ID: "onboarding"},
		{ID: "active"},
	}
	suite.stageRepositoryMock.On("List", ctx).Return(stages, nil)

	suite.clientRepositoryMock.On("Count", ctx, bson.M{"current_stage": "registration"}).Return(int64(50), nil)
	suite.clientRepositoryMock.On("Count", ctx, bson.M{"current_stage": "onboarding"}).Return(int64(30), nil)
	suite.clientRepositoryMock.On("Count", ctx, bson.M{"current_stage": "active"}).Return(int64(20), nil)

	suite.metricRepositoryMock.On("Add", ctx, mock.MatchedBy(func(m metric.Entity) bool {
		return *m.Type == metric.ClientsPerStage
	})).Return("metric-id-1", nil).Times(3)

	err := suite.service.calculateClientsPerStage(ctx, now)

	suite.NoError(err)
	suite.metricRepositoryMock.AssertNumberOfCalls(suite.T(), "Add", 3)
}

func (suite *MetricServiceTestSuite) TestCalculateStageDuration() {
	ctx := context.Background()
	now := time.Now()

	stages := []stage.Entity{
		{ID: "registration"},
		{ID: "onboarding"},
	}
	suite.stageRepositoryMock.On("List", ctx).Return(stages, nil)

	regStage := "registration"
	onbStage := "onboarding"
	isActive := true
	regDate := now.Add(-10 * 24 * time.Hour)
	lastUpdate1 := now.Add(-5 * 24 * time.Hour)
	lastUpdate2 := now.Add(-2 * 24 * time.Hour)

	clients := []client.Entity{
		{
			ID:               "client1",
			CurrentStage:     &regStage,
			IsActive:         &isActive,
			RegistrationDate: &regDate,
			LastUpdated:      &lastUpdate1,
		},
		{
			ID:               "client2",
			CurrentStage:     &onbStage,
			IsActive:         &isActive,
			RegistrationDate: &regDate,
			LastUpdated:      &lastUpdate2,
		},
	}

	suite.clientRepositoryMock.On("List", ctx, client.Filters{IsActive: &isActive}, 0, 0).Return(clients, 2, nil)

	suite.metricRepositoryMock.On("Add", ctx, mock.MatchedBy(func(m metric.Entity) bool {
		return *m.Type == metric.StageDuration
	})).Return("metric-id", nil).Times(2)

	err := suite.service.calculateStageDuration(ctx, now)

	suite.NoError(err)
	suite.metricRepositoryMock.AssertNumberOfCalls(suite.T(), "Add", 2)
}

func (suite *MetricServiceTestSuite) TestCalculateDAU() {
	ctx := context.Background()
	now := time.Now()

	dauCount := int64(150)

	suite.clientRepositoryMock.On("Count", ctx, mock.AnythingOfType("primitive.M")).Return(dauCount, nil)

	suite.metricRepositoryMock.On("Add", ctx, mock.MatchedBy(func(m metric.Entity) bool {
		return *m.Type == metric.DAU && *m.Value == float64(dauCount) && *m.Interval == "day"
	})).Return("dau-metric-id", nil)

	err := suite.service.calculateDAU(ctx, now)

	suite.NoError(err)
	suite.metricRepositoryMock.AssertCalled(suite.T(), "Add", ctx, mock.MatchedBy(func(m metric.Entity) bool {
		return *m.Type == metric.DAU && *m.Value == float64(dauCount)
	}))
}

func (suite *MetricServiceTestSuite) TestCalculateMAU() {
	ctx := context.Background()
	now := time.Now()

	mauCount := int64(2500)

	suite.clientRepositoryMock.On("Count", ctx, mock.AnythingOfType("primitive.M")).Return(mauCount, nil)

	suite.metricRepositoryMock.On("Add", ctx, mock.MatchedBy(func(m metric.Entity) bool {
		return *m.Type == metric.MAU && *m.Value == float64(mauCount) && *m.Interval == "month"
	})).Return("mau-metric-id", nil)

	err := suite.service.calculateMAU(ctx, now)

	suite.NoError(err)
	suite.metricRepositoryMock.AssertCalled(suite.T(), "Add", ctx, mock.MatchedBy(func(m metric.Entity) bool {
		return *m.Type == metric.MAU && *m.Value == float64(mauCount)
	}))
}

func (suite *MetricServiceTestSuite) TestCalculateDropout() {
	ctx := context.Background()
	now := time.Now()
	interval := "week"

	dropoutCount := int64(15)

	suite.clientRepositoryMock.On("Count", ctx, mock.AnythingOfType("primitive.M")).Return(dropoutCount, nil)

	suite.metricRepositoryMock.On("Add", ctx, mock.MatchedBy(func(m metric.Entity) bool {
		return *m.Type == metric.Dropout && *m.Value == float64(dropoutCount) && *m.Interval == interval
	})).Return("dropout-metric-id", nil)

	err := suite.service.calculateDropout(ctx, now, interval)

	suite.NoError(err)
	suite.metricRepositoryMock.AssertCalled(suite.T(), "Add", ctx, mock.MatchedBy(func(m metric.Entity) bool {
		return *m.Type == metric.Dropout && *m.Value == float64(dropoutCount)
	}))
}

func (suite *MetricServiceTestSuite) TestCalculateConversion() {
	ctx := context.Background()
	now := time.Now()
	interval := "week"

	stages := []stage.Entity{
		{ID: "registration"},
		{ID: "onboarding"},
		{ID: "active"},
	}
	suite.stageRepositoryMock.On("List", ctx).Return(stages, nil)

	lastStageRecentCount := int64(25)
	totalClientsCount := int64(100)

	suite.clientRepositoryMock.On("Count", ctx, mock.MatchedBy(func(filter bson.M) bool {
		_, hasCurrentStage := filter["current_stage"]
		return hasCurrentStage
	})).Return(lastStageRecentCount, nil).Once()

	suite.clientRepositoryMock.On("Count", ctx, mock.MatchedBy(func(filter bson.M) bool {
		_, hasLastUpdated := filter["last_updated"]
		_, hasCurrentStage := filter["current_stage"]
		return hasLastUpdated && !hasCurrentStage
	})).Return(totalClientsCount, nil).Once()

	expectedConversionRate := float64(lastStageRecentCount) / float64(totalClientsCount)

	suite.metricRepositoryMock.On("Add", ctx, mock.MatchedBy(func(m metric.Entity) bool {
		return *m.Type == metric.Conversion &&
			*m.Value == expectedConversionRate &&
			*m.Interval == interval
	})).Return("conversion-metric-id", nil)

	err := suite.service.calculateConversion(ctx, now, interval)

	suite.NoError(err)
	suite.metricRepositoryMock.AssertCalled(suite.T(), "Add", ctx, mock.MatchedBy(func(m metric.Entity) bool {
		return *m.Type == metric.Conversion && *m.Value == expectedConversionRate
	}))
}

func (suite *MetricServiceTestSuite) TestCalculateAutoPaymentRate() {
	ctx := context.Background()
	now := time.Now()

	autoPayEnabled := "enabled"
	autoPayDisabled := "disabled"
	name := "Test Client"
	email := "test@example.com"
	isActive := true

	clients := []client.Entity{
		{
			ID:       "client1",
			Name:     &name,
			Email:    &email,
			IsActive: &isActive,
			Contracts: []contract.Entity{
				{ID: "contract1", AutoPayment: &autoPayEnabled},
				{ID: "contract2", AutoPayment: &autoPayDisabled},
			},
		},
		{
			ID:       "client2",
			Name:     &name,
			Email:    &email,
			IsActive: &isActive,
			Contracts: []contract.Entity{
				{ID: "contract3", AutoPayment: &autoPayEnabled},
			},
		},
	}

	expectedRate := float64(2) / float64(3)

	suite.clientRepositoryMock.On("List", ctx, client.Filters{}, 0, 0).Return(clients, 2, nil)

	suite.metricRepositoryMock.On("Add", ctx, mock.MatchedBy(func(m metric.Entity) bool {
		return *m.Type == metric.AutoPaymentRate && *m.Value == expectedRate
	})).Return("autopay-metric-id", nil)

	err := suite.service.calculateAutoPaymentRate(ctx, now)

	suite.NoError(err)
	suite.metricRepositoryMock.AssertCalled(suite.T(), "Add", ctx, mock.MatchedBy(func(m metric.Entity) bool {
		return *m.Type == metric.AutoPaymentRate && *m.Value == expectedRate
	}))
}

func (suite *MetricServiceTestSuite) TestCalculateSourceConversion() {
	ctx := context.Background()
	now := time.Now()
	interval := "month"

	stages := []stage.Entity{
		{ID: "registration"},
		{ID: "active"},
	}
	suite.stageRepositoryMock.On("List", ctx).Return(stages, nil)

	source1 := "website"
	source2 := "mobile"
	lastUpdate := now.Add(-5 * 24 * time.Hour)
	stage1 := "registration"
	stage2 := "active"

	clients := []client.Entity{
		{ID: "client1", Source: &source1, LastUpdated: &lastUpdate, CurrentStage: &stage1},
		{ID: "client2", Source: &source1, LastUpdated: &lastUpdate, CurrentStage: &stage2},
		{ID: "client3", Source: &source2, LastUpdated: &lastUpdate, CurrentStage: &stage1},
	}

	suite.clientRepositoryMock.On("List", ctx, client.Filters{}, 0, 0).Return(clients, 3, nil)

	suite.clientRepositoryMock.On("Count", ctx, mock.MatchedBy(func(filter bson.M) bool {
		source, hasSource := filter["source"]
		return hasSource && source == source1
	})).Return(int64(2), nil).Once()

	suite.clientRepositoryMock.On("Count", ctx, mock.MatchedBy(func(filter bson.M) bool {
		source, hasSource := filter["source"]
		stage, hasStage := filter["current_stage"]
		return hasSource && hasStage && source == source1 && stage == stage2
	})).Return(int64(1), nil).Once()

	suite.clientRepositoryMock.On("Count", ctx, mock.MatchedBy(func(filter bson.M) bool {
		source, hasSource := filter["source"]
		return hasSource && source == source2
	})).Return(int64(1), nil).Once()

	suite.clientRepositoryMock.On("Count", ctx, mock.MatchedBy(func(filter bson.M) bool {
		source, hasSource := filter["source"]
		stage, hasStage := filter["current_stage"]
		return hasSource && hasStage && source == source2 && stage == stage2
	})).Return(int64(0), nil).Once()

	expectedWebsiteRate := float64(1) / float64(2)
	expectedMobileRate := float64(0) / float64(1)

	suite.metricRepositoryMock.On("Add", ctx, mock.MatchedBy(func(m metric.Entity) bool {
		return *m.Type == metric.SourceConversion && *m.Interval == interval &&
			((*m.Value == expectedWebsiteRate && m.Metadata["source"] == source1) ||
				(*m.Value == expectedMobileRate && m.Metadata["source"] == source2))
	})).Return("source-conversion-id", nil).Times(2)

	err := suite.service.calculateSourceConversion(ctx, now, interval)

	suite.NoError(err)
	suite.metricRepositoryMock.AssertNumberOfCalls(suite.T(), "Add", 2)
}
func (suite *MetricServiceTestSuite) TestCalculateChannelConversion() {
	ctx := context.Background()
	now := time.Now()
	interval := "day"

	stages := []stage.Entity{
		{ID: "registration"},
		{ID: "active"},
	}
	suite.stageRepositoryMock.On("List", ctx).Return(stages, nil)

	channel1 := "direct"
	channel2 := "referral"
	lastUpdate := now.Add(-1 * time.Minute)
	stage1 := "registration"
	stage2 := "active"

	clients := []client.Entity{
		{ID: "client1", Channel: &channel1, LastUpdated: &lastUpdate, CurrentStage: &stage1},
		{ID: "client2", Channel: &channel1, LastUpdated: &lastUpdate, CurrentStage: &stage2},
		{ID: "client3", Channel: &channel2, LastUpdated: &lastUpdate, CurrentStage: &stage1},
		{ID: "client4", Channel: &channel2, LastUpdated: &lastUpdate, CurrentStage: &stage2},
	}

	suite.clientRepositoryMock.On("List", ctx, client.Filters{}, 0, 0).Return(clients, 4, nil)

	expectedDirectRate := float64(1) / float64(2)
	expectedReferralRate := float64(1) / float64(2)

	suite.clientRepositoryMock.On("Count", ctx, mock.MatchedBy(func(filter bson.M) bool {
		channel, hasChannel := filter["channel"]
		_, hasLastUpdated := filter["last_updated"]
		return hasChannel && hasLastUpdated && channel == channel1
	})).Return(int64(2), nil).Once()

	suite.clientRepositoryMock.On("Count", ctx, mock.MatchedBy(func(filter bson.M) bool {
		channel, hasChannel := filter["channel"]
		stage, hasStage := filter["current_stage"]
		_, hasLastUpdated := filter["last_updated"]
		return hasChannel && hasStage && hasLastUpdated && channel == channel1 && stage == stage2
	})).Return(int64(1), nil).Once()

	suite.clientRepositoryMock.On("Count", ctx, mock.MatchedBy(func(filter bson.M) bool {
		channel, hasChannel := filter["channel"]
		_, hasLastUpdated := filter["last_updated"]
		return hasChannel && hasLastUpdated && channel == channel2
	})).Return(int64(2), nil).Once()

	suite.clientRepositoryMock.On("Count", ctx, mock.MatchedBy(func(filter bson.M) bool {
		channel, hasChannel := filter["channel"]
		stage, hasStage := filter["current_stage"]
		_, hasLastUpdated := filter["last_updated"]
		return hasChannel && hasStage && hasLastUpdated && channel == channel2 && stage == stage2
	})).Return(int64(1), nil).Once()

	suite.metricRepositoryMock.On("Add", ctx, mock.MatchedBy(func(m metric.Entity) bool {
		return *m.Type == metric.ChannelConversion && *m.Interval == interval &&
			((*m.Value == expectedDirectRate && m.Metadata["channel"] == channel1) ||
				(*m.Value == expectedReferralRate && m.Metadata["channel"] == channel2))
	})).Return("channel-conversion-id", nil).Times(2)

	err := suite.service.calculateChannelConversion(ctx, now, interval)

	suite.NoError(err)
	suite.metricRepositoryMock.AssertNumberOfCalls(suite.T(), "Add", 2)
}

func (suite *MetricServiceTestSuite) TestCalculateAllMetrics() {
	ctx := context.Background()
	interval := "day"

	suite.clientRepositoryMock.On("Count", mock.Anything, mock.Anything).Return(int64(10), nil)
	suite.clientRepositoryMock.On("List", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]client.Entity{}, 0, nil)

	suite.stageRepositoryMock.On("List", mock.Anything).Return([]stage.Entity{{ID: "stage1"}, {ID: "stage2"}}, nil)

	suite.metricRepositoryMock.On("Add", mock.Anything, mock.Anything).Return("metric-id", nil)
	suite.metricRepositoryMock.On("List", mock.Anything, mock.Anything).Return([]metric.Entity{}, nil)
	suite.metricRepositoryMock.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(metric.Entity{}, nil)

	err := suite.service.CalculateAllMetrics(ctx, interval)

	suite.NoError(err)
}

func (suite *MetricServiceTestSuite) TestAggregateRollbackCount() {
	ctx := context.Background()

	suite.Run("Weekly aggregation with no existing aggregate", func() {

		metricRepoMock := new(MockMetricRepository)
		suite.service.MetricRepository = metricRepoMock

		timestamp := time.Date(2023, 5, 10, 12, 0, 0, 0, time.UTC)

		rollbackType := metric.RollbackCount
		dayInterval := "day"
		weekInterval := "week"

		value1 := float64(2)
		date1 := time.Date(2023, 5, 8, 0, 0, 0, 0, time.UTC)

		value2 := float64(3)
		date2 := time.Date(2023, 5, 9, 0, 0, 0, 0, time.UTC)

		dailyMetrics := []metric.Entity{
			{
				ID:        "daily-1",
				Type:      &rollbackType,
				Value:     &value1,
				Interval:  &dayInterval,
				CreatedAt: &date1,
			},
			{
				ID:        "daily-2",
				Type:      &rollbackType,
				Value:     &value2,
				Interval:  &dayInterval,
				CreatedAt: &date2,
			},
		}

		metricRepoMock.On("List", ctx, metric.Filters{
			Type:     string(metric.RollbackCount),
			Interval: "day",
		}).Return(dailyMetrics, nil)

		metricRepoMock.On("List", ctx, metric.Filters{
			Type:     string(metric.RollbackCount),
			Interval: "week",
		}).Return([]metric.Entity{}, nil)

		expectedTotal := value1 + value2

		metricRepoMock.On("Add", ctx, mock.MatchedBy(func(m metric.Entity) bool {
			return *m.Type == metric.RollbackCount &&
				*m.Value == expectedTotal &&
				*m.Interval == weekInterval
		})).Return("new-aggregate-id", nil)

		err := suite.service.aggregateRollBackCount(ctx, timestamp, "week")

		suite.NoError(err)
		metricRepoMock.AssertExpectations(suite.T())
	})

	suite.Run("Monthly aggregation with existing aggregate", func() {
		metricRepoMock := new(MockMetricRepository)
		suite.service.MetricRepository = metricRepoMock

		timestamp := time.Date(2023, 5, 15, 12, 0, 0, 0, time.UTC)

		rollbackType := metric.RollbackCount
		dayInterval := "day"
		monthInterval := "month"

		value1 := float64(5)
		date1 := time.Date(2023, 5, 1, 0, 0, 0, 0, time.UTC)

		value2 := float64(3)
		date2 := time.Date(2023, 5, 5, 0, 0, 0, 0, time.UTC)

		dailyMetrics := []metric.Entity{
			{
				ID:        "daily-1",
				Type:      &rollbackType,
				Value:     &value1,
				Interval:  &dayInterval,
				CreatedAt: &date1,
			},
			{
				ID:        "daily-2",
				Type:      &rollbackType,
				Value:     &value2,
				Interval:  &dayInterval,
				CreatedAt: &date2,
			},
		}

		existingValue := float64(0)
		existingDate := time.Date(2023, 5, 1, 0, 0, 0, 0, time.UTC)

		existingMetric := metric.Entity{
			ID:        "existing-monthly",
			Type:      &rollbackType,
			Value:     &existingValue,
			Interval:  &monthInterval,
			CreatedAt: &existingDate,
		}

		metricRepoMock.On("List", ctx, metric.Filters{
			Type:     string(metric.RollbackCount),
			Interval: "day",
		}).Return(dailyMetrics, nil)

		metricRepoMock.On("List", ctx, metric.Filters{
			Type:     string(metric.RollbackCount),
			Interval: "month",
		}).Return([]metric.Entity{existingMetric}, nil)

		expectedTotal := value1 + value2

		metricRepoMock.On("Update", ctx, "existing-monthly", mock.MatchedBy(func(m metric.Entity) bool {
			return *m.Type == metric.RollbackCount &&
				*m.Value == expectedTotal &&
				*m.Interval == monthInterval
		})).Return(metric.Entity{}, nil)

		err := suite.service.aggregateRollBackCount(ctx, timestamp, "month")

		suite.NoError(err)
		metricRepoMock.AssertExpectations(suite.T())
	})

	suite.Run("Invalid interval", func() {
		err := suite.service.aggregateRollBackCount(ctx, time.Now(), "day")

		suite.Error(err)
		suite.Contains(err.Error(), "invalid interval")
	})
}

func (suite *MetricServiceTestSuite) TestFirstDayOfISOWeek() {
	testCases := []struct {
		name        string
		year        int
		week        int
		expectedDay time.Time
	}{
		{
			name:        "First week of 2023",
			year:        2023,
			week:        1,
			expectedDay: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name:        "Week 19 of 2023",
			year:        2023,
			week:        19,
			expectedDay: time.Date(2023, 5, 8, 0, 0, 0, 0, time.UTC),
		},
		{
			name:        "Last week of 2023",
			year:        2023,
			week:        52,
			expectedDay: time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result := firstDayOfISOWeek(tc.year, tc.week, time.UTC)
			suite.Equal(tc.expectedDay, result)
		})
	}
}

func (suite *MetricServiceTestSuite) TestListMetricsWithRepositoryError() {
	ctx := context.Background()
	filters := metric.Filters{
		Type:     "conversion",
		Interval: "day",
	}

	// Mock repository to return error
	repoError := errors.New("database connection error")
	suite.metricRepositoryMock.On("List", ctx, filters).Return([]metric.Entity{}, repoError)

	// If cache is used, set up cache miss first
	suite.metricCacheMock.On("List", ctx, filters).Return([]metric.Entity{}, store.ErrorNotFound)

	// Call the service method
	responses, err := suite.service.ListMetrics(ctx, filters)

	// Verify expectations
	suite.Error(err)
	suite.Equal(repoError, err)
	suite.Empty(responses)
	suite.metricRepositoryMock.AssertExpectations(suite.T())
}

func (suite *MetricServiceTestSuite) TestListMetricsWithCacheHit() {
	ctx := context.Background()
	filters := metric.Filters{
		Type:     "clients-per-stage",
		Interval: "day",
	}

	// Prepare cache hit data
	now := time.Now()
	metricType := metric.ClientsPerStage
	value := 42.0
	interval := "day"
	entities := []metric.Entity{
		{
			ID:        "cached-metric",
			Type:      &metricType,
			Value:     &value,
			Interval:  &interval,
			CreatedAt: &now,
			Metadata:  map[string]string{"stage": "registration"},
		},
	}

	// Ensure the service's MetricCache field is set to the mock
	// This line may be redundant if already set in SetupTest
	suite.service.MetricCache = suite.metricCacheMock

	// Mock cache to return data (cache hit)
	suite.metricCacheMock.On("List", ctx, filters).Return(entities, nil)

	// Call the service method
	responses, err := suite.service.ListMetrics(ctx, filters)

	// Verify expectations
	suite.NoError(err)
	suite.Len(responses, 1)
	suite.Equal("cached-metric", responses[0].ID)
	suite.Equal(42.0, responses[0].Value)
	suite.Equal("clients-per-stage", responses[0].Type)

	// Repository should not be called on cache hit
	suite.metricRepositoryMock.AssertNotCalled(suite.T(), "List", ctx, filters)
}

func (suite *MetricServiceTestSuite) TestCalculateClientsPerStageWithError() {
	ctx := context.Background()
	now := time.Now()

	// Mock stage repository to return error
	stageError := errors.New("failed to retrieve stages")
	// Return empty slice instead of nil to avoid panic
	suite.stageRepositoryMock.On("List", ctx).Return([]stage.Entity{}, stageError)

	// Call the function
	err := suite.service.calculateClientsPerStage(ctx, now)

	// Verify expectations
	suite.Error(err)
	suite.Equal(stageError, err)

	// Client repository should not be called if stage retrieval fails
	suite.clientRepositoryMock.AssertNotCalled(suite.T(), "Count")
	suite.metricRepositoryMock.AssertNotCalled(suite.T(), "Add")
}

func (suite *MetricServiceTestSuite) TestCalculateDAUWithCountError() {
	ctx := context.Background()
	now := time.Now()

	// Mock repository to return error on Count
	countError := errors.New("count operation failed")
	suite.clientRepositoryMock.On("Count", ctx, mock.AnythingOfType("primitive.M")).Return(int64(0), countError)

	// Call the function
	err := suite.service.calculateDAU(ctx, now)

	// Verify expectations
	suite.Error(err)
	suite.Equal(countError, err)

	// Metric repository should not be called if count fails
	suite.metricRepositoryMock.AssertNotCalled(suite.T(), "Add")
}

func (suite *MetricServiceTestSuite) TestCalculateAllMetricsWithStageDurationError() {
	ctx := context.Background()
	interval := "day"

	// Setup success for calculateClientsPerStage
	suite.stageRepositoryMock.On("List", mock.Anything).Return([]stage.Entity{{ID: "stage1"}}, nil).Once()
	suite.clientRepositoryMock.On("Count", mock.Anything, mock.AnythingOfType("primitive.M")).Return(int64(10), nil).Once()
	suite.metricRepositoryMock.On("Add", mock.Anything, mock.AnythingOfType("metric.Entity")).Return("metric-id", nil).Once()

	// Setup error for calculateStageDuration
	stageDurationError := errors.New("failed to calculate stage duration")
	isActive := true
	// Return empty slice instead of nil to avoid panic when iterating
	suite.clientRepositoryMock.On("List", ctx, client.Filters{IsActive: &isActive}, 0, 0).Return([]client.Entity{}, 0, stageDurationError)

	// Call the function
	err := suite.service.CalculateAllMetrics(ctx, interval)

	// Verify expectations
	suite.Error(err)
	suite.Contains(err.Error(), "failed to calculate stage duration")
}

func (suite *MetricServiceTestSuite) TestCreateMetric() {
	// Create test data
	id := "test-id"
	metricType := metric.DAU
	value := 123.45
	interval := "day"
	timestamp := time.Now()
	metadata := map[string]string{"key": "value"}

	// Call createMetric directly
	entity, err := suite.service.createMetric(id, metricType, value, interval, timestamp, metadata)

	// Verify expectations
	suite.NoError(err)
	suite.Equal(id, entity.ID)
	suite.Equal(metricType, *entity.Type)
	suite.Equal(value, *entity.Value)
	suite.Equal(interval, *entity.Interval)
	suite.Equal(timestamp, *entity.CreatedAt)
	suite.Equal("value", entity.Metadata["key"])
}

func (suite *MetricServiceTestSuite) TestCreateMetricWithEmptyID() {
	// Call createMetric with empty ID
	entity, err := suite.service.createMetric("", metric.MAU, 100.0, "month", time.Now(), nil)

	// Verify expectations
	suite.NoError(err)
	suite.NotEmpty(entity.ID) // ID should be generated
	suite.Equal(metric.MAU, *entity.Type)
	suite.Equal(100.0, *entity.Value)

	// Verify metadata is initialized even when nil is passed
	suite.NotNil(entity.Metadata)
}

// Test cache invalidation in CalculateAllMetrics
func (suite *MetricServiceTestSuite) TestCalculateAllMetricsWithCacheInvalidation() {
	ctx := context.Background()
	interval := "day"

	// Setup mocks for successful calculations
	suite.clientRepositoryMock.On("Count", mock.Anything, mock.Anything).Return(int64(10), nil)
	suite.clientRepositoryMock.On("List", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]client.Entity{}, 0, nil)
	suite.stageRepositoryMock.On("List", mock.Anything).Return([]stage.Entity{{ID: "stage1"}, {ID: "stage2"}}, nil)
	suite.metricRepositoryMock.On("Add", mock.Anything, mock.Anything).Return("metric-id", nil)
	suite.metricRepositoryMock.On("List", mock.Anything, mock.Anything).Return([]metric.Entity{}, nil)
	suite.metricRepositoryMock.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(metric.Entity{}, nil)

	// Set up cache mock with a catch-all expectation
	cacheMock := new(MockMetricCache)
	suite.service.MetricCache = cacheMock

	// Simply expect 12 cache invalidations when interval is "day"
	cacheMock.On("InvalidateListCache", mock.Anything, mock.Anything).Return(nil).Times(12)

	err := suite.service.CalculateAllMetrics(ctx, interval)

	suite.NoError(err)
	cacheMock.AssertExpectations(suite.T())
}

// Test CalculateAllMetrics with 'month' interval
func (suite *MetricServiceTestSuite) TestCalculateAllMetricsWithMonthInterval() {
	ctx := context.Background()
	interval := "month"

	// Setup mocks for successful calculations
	suite.clientRepositoryMock.On("Count", mock.Anything, mock.Anything).Return(int64(10), nil)
	suite.clientRepositoryMock.On("List", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]client.Entity{}, 0, nil)
	suite.stageRepositoryMock.On("List", mock.Anything).Return([]stage.Entity{{ID: "stage1"}, {ID: "stage2"}}, nil)
	suite.metricRepositoryMock.On("Add", mock.Anything, mock.Anything).Return("metric-id", nil)
	suite.metricRepositoryMock.On("List", mock.Anything, mock.Anything).Return([]metric.Entity{}, nil)
	suite.metricRepositoryMock.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(metric.Entity{}, nil)

	err := suite.service.CalculateAllMetrics(ctx, interval)

	suite.NoError(err)
	// Verify MAU is calculated instead of DAU with month interval
	suite.clientRepositoryMock.AssertCalled(suite.T(), "Count", mock.Anything, mock.MatchedBy(func(filter bson.M) bool {
		_, hasLastLogin := filter["last_login"]
		return hasLastLogin // Match MAU calculation query
	}))
}

// Test error handling in calculateMAU
func (suite *MetricServiceTestSuite) TestCalculateMAUWithError() {
	ctx := context.Background()
	now := time.Now()

	// Mock repository to return error on Count
	countError := errors.New("MAU count failed")
	suite.clientRepositoryMock.On("Count", ctx, mock.AnythingOfType("primitive.M")).Return(int64(0), countError)

	err := suite.service.calculateMAU(ctx, now)

	suite.Error(err)
	suite.Equal(countError, err)
	suite.metricRepositoryMock.AssertNotCalled(suite.T(), "Add")
}

// Test error handling in calculateAppInstallRate
func (suite *MetricServiceTestSuite) TestCalculateAppInstallRateWithError() {
	ctx := context.Background()
	now := time.Now()

	// First count succeeds, second fails
	suite.clientRepositoryMock.On("Count", ctx, mock.MatchedBy(func(filter bson.M) bool {
		_, hasAppExists := filter["app"]
		return hasAppExists
	})).Return(int64(100), nil).Once()

	countError := errors.New("app install count failed")
	suite.clientRepositoryMock.On("Count", ctx, mock.MatchedBy(func(filter bson.M) bool {
		app, hasApp := filter["app"]
		return hasApp && app == "installed"
	})).Return(int64(0), countError)

	err := suite.service.calculateAppInstallRate(ctx, now)

	suite.Error(err)
	suite.Equal(countError, err)
	suite.metricRepositoryMock.AssertNotCalled(suite.T(), "Add")
}

// Test error handling in calculateTotalDuration
func (suite *MetricServiceTestSuite) TestCalculateTotalDurationWithMetricRepositoryError() {
	ctx := context.Background()
	now := time.Now()

	// Setup stages successfully
	stages := []stage.Entity{
		{ID: "registration"},
		{ID: "active"},
	}
	suite.stageRepositoryMock.On("List", ctx).Return(stages, nil)

	// Setup client list successfully
	lastStage := stages[1].ID
	isActive := true
	registrationDate := now.Add(-30 * 24 * time.Hour)
	lastUpdated := now

	clients := []client.Entity{
		{
			ID:               "client1",
			CurrentStage:     &lastStage,
			IsActive:         &isActive,
			RegistrationDate: &registrationDate,
			LastUpdated:      &lastUpdated,
		},
	}

	suite.clientRepositoryMock.On("List", ctx, mock.MatchedBy(func(filters client.Filters) bool {
		return filters.Stage == lastStage
	}), 0, 0).Return(clients, 1, nil)

	// But repository add fails
	repoError := errors.New("metric repository add failed")
	suite.metricRepositoryMock.On("Add", ctx, mock.AnythingOfType("metric.Entity")).Return("", repoError)

	err := suite.service.calculateTotalDuration(ctx, now)

	suite.Error(err)
	suite.Contains(err.Error(), "repository add failed")
}

// Test edge case for conversion calculation with no clients
func (suite *MetricServiceTestSuite) TestCalculateConversionWithNoClients() {
	ctx := context.Background()
	now := time.Now()
	interval := "week"

	stages := []stage.Entity{
		{ID: "registration"},
		{ID: "active"},
	}
	suite.stageRepositoryMock.On("List", ctx).Return(stages, nil)

	// Zero clients in both queries
	suite.clientRepositoryMock.On("Count", ctx, mock.MatchedBy(func(filter bson.M) bool {
		_, hasCurrentStage := filter["current_stage"]
		return hasCurrentStage
	})).Return(int64(0), nil).Once()

	suite.clientRepositoryMock.On("Count", ctx, mock.MatchedBy(func(filter bson.M) bool {
		_, hasLastUpdated := filter["last_updated"]
		_, hasCurrentStage := filter["current_stage"]
		return hasLastUpdated && !hasCurrentStage
	})).Return(int64(0), nil).Once()

	// Should create metric with 0 conversion rate
	suite.metricRepositoryMock.On("Add", ctx, mock.MatchedBy(func(m metric.Entity) bool {
		return *m.Type == metric.Conversion &&
			*m.Value == 0.0 &&
			*m.Interval == interval
	})).Return("conversion-metric-id", nil)

	err := suite.service.calculateConversion(ctx, now, interval)

	suite.NoError(err)
	suite.metricRepositoryMock.AssertCalled(suite.T(), "Add", ctx, mock.MatchedBy(func(m metric.Entity) bool {
		return *m.Type == metric.Conversion && *m.Value == 0.0
	}))
}

// Test cache invalidation error handling
func (suite *MetricServiceTestSuite) TestCalculateRollbackCountWithCacheInvalidationError() {
	ctx := context.Background()
	now := time.Now()

	// Set up cache mock to return cache miss for List
	cacheMock := new(MockMetricCache)
	suite.service.MetricCache = cacheMock
	cacheMock.On("List", ctx, metric.Filters{
		Type:     "rollback-count",
		Interval: "day",
	}).Return([]metric.Entity{}, errors.New("cache miss"))

	// No existing metrics today from repository
	suite.metricRepositoryMock.On("List", ctx, metric.Filters{
		Type:     "rollback-count",
		Interval: "day",
	}).Return([]metric.Entity{}, nil)

	// Add metric succeeds
	suite.metricRepositoryMock.On("Add", ctx, mock.AnythingOfType("metric.Entity")).Return("new-metric-id", nil)

	// But cache invalidation fails
	cacheError := errors.New("cache invalidation failed")
	cacheMock.On("InvalidateListCache", ctx, metric.Filters{
		Type:     string(metric.RollbackCount),
		Interval: "day",
	}).Return(cacheError)

	// The function should still succeed even if cache invalidation fails
	err := suite.service.calculateRollbackCount(ctx, now)

	suite.NoError(err)
	cacheMock.AssertExpectations(suite.T())
}

func (suite *MetricServiceTestSuite) TestCalculateRollbackCountComprehensive() {
	ctx := context.Background()
	now := time.Now()

	suite.Run("No existing metric for today", func() {
		// Reset mocks
		metricRepoMock := new(MockMetricRepository)
		suite.service.MetricRepository = metricRepoMock

		// Setup mock to return no metrics
		metricRepoMock.On("List", ctx, metric.Filters{
			Type:     string(metric.RollbackCount),
			Interval: "day",
		}).Return([]metric.Entity{}, nil)

		// Expect adding a new metric with value 1.0
		metricRepoMock.On("Add", ctx, mock.MatchedBy(func(m metric.Entity) bool {
			return *m.Type == metric.RollbackCount && *m.Value == 1.0 && *m.Interval == "day"
		})).Return("new-rollback-id", nil)

		// Set up cache mock
		cacheMock := new(MockMetricCache)
		suite.service.MetricCache = cacheMock
		cacheMock.On("List", ctx, mock.Anything).Return([]metric.Entity{}, errors.New("cache miss"))
		cacheMock.On("InvalidateListCache", ctx, mock.Anything).Return(nil)

		err := suite.service.calculateRollbackCount(ctx, now)

		suite.NoError(err)
		metricRepoMock.AssertCalled(suite.T(), "Add", ctx, mock.MatchedBy(func(m metric.Entity) bool {
			return *m.Type == metric.RollbackCount && *m.Value == 1.0
		}))
		cacheMock.AssertExpectations(suite.T())
	})

	suite.Run("Existing metric for today", func() {
		// Reset mocks
		metricRepoMock := new(MockMetricRepository)
		suite.service.MetricRepository = metricRepoMock

		// Create existing metric for today
		metricType := metric.RollbackCount
		existingValue := float64(5)
		interval := "day"
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		existingMetric := metric.Entity{
			ID:        "existing-rollback-id",
			Type:      &metricType,
			Value:     &existingValue,
			Interval:  &interval,
			CreatedAt: &today,
			Metadata:  make(map[string]string),
		}

		// Setup mock to return existing metric
		metricRepoMock.On("List", ctx, metric.Filters{
			Type:     string(metric.RollbackCount),
			Interval: "day",
		}).Return([]metric.Entity{existingMetric}, nil)

		// Expect update with incremented value
		expectedNewValue := existingValue + 1
		metricRepoMock.On("Update", ctx, "existing-rollback-id", mock.MatchedBy(func(m metric.Entity) bool {
			return *m.Type == metric.RollbackCount && *m.Value == expectedNewValue
		})).Return(metric.Entity{}, nil)

		// Set up cache mock
		cacheMock := new(MockMetricCache)
		suite.service.MetricCache = cacheMock
		cacheMock.On("List", ctx, mock.Anything).Return([]metric.Entity{existingMetric}, nil)
		cacheMock.On("InvalidateListCache", ctx, mock.Anything).Return(nil)

		err := suite.service.calculateRollbackCount(ctx, now)

		suite.NoError(err)
		metricRepoMock.AssertCalled(suite.T(), "Update", ctx, "existing-rollback-id", mock.MatchedBy(func(m metric.Entity) bool {
			return *m.Type == metric.RollbackCount && *m.Value == expectedNewValue
		}))
	})

	suite.Run("Repository error when listing metrics", func() {
		// Reset mocks
		metricRepoMock := new(MockMetricRepository)
		suite.service.MetricRepository = metricRepoMock

		// Setup mock to return error
		repoError := errors.New("database connection error")
		metricRepoMock.On("List", ctx, mock.Anything).Return([]metric.Entity{}, repoError)

		// Cache miss
		cacheMock := new(MockMetricCache)
		suite.service.MetricCache = cacheMock
		cacheMock.On("List", ctx, mock.Anything).Return([]metric.Entity{}, errors.New("cache miss"))

		err := suite.service.calculateRollbackCount(ctx, now)

		suite.Error(err)
		suite.Contains(err.Error(), "database connection error")
		metricRepoMock.AssertNotCalled(suite.T(), "Add")
		metricRepoMock.AssertNotCalled(suite.T(), "Update")
	})

	suite.Run("Repository error when updating metric", func() {
		// Reset mocks
		metricRepoMock := new(MockMetricRepository)
		suite.service.MetricRepository = metricRepoMock

		// Create existing metric for today
		metricType := metric.RollbackCount
		existingValue := float64(5)
		interval := "day"
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		existingMetric := metric.Entity{
			ID:        "existing-rollback-id",
			Type:      &metricType,
			Value:     &existingValue,
			Interval:  &interval,
			CreatedAt: &today,
			Metadata:  make(map[string]string),
		}

		// Setup mock to return existing metric
		metricRepoMock.On("List", ctx, metric.Filters{
			Type:     string(metric.RollbackCount),
			Interval: "day",
		}).Return([]metric.Entity{existingMetric}, nil)

		// Setup update to fail
		updateError := errors.New("update failed")
		metricRepoMock.On("Update", ctx, "existing-rollback-id", mock.Anything).Return(metric.Entity{}, updateError)

		cacheMock := new(MockMetricCache)
		suite.service.MetricCache = cacheMock
		cacheMock.On("List", ctx, mock.Anything).Return([]metric.Entity{existingMetric}, nil)

		err := suite.service.calculateRollbackCount(ctx, now)

		suite.Error(err)
		suite.Contains(err.Error(), "update failed")
	})

	suite.Run("Repository error when adding new metric", func() {
		// Reset mocks
		metricRepoMock := new(MockMetricRepository)
		suite.service.MetricRepository = metricRepoMock

		// No existing metrics
		metricRepoMock.On("List", ctx, mock.Anything).Return([]metric.Entity{}, nil)

		// Add operation fails
		addError := errors.New("add failed")
		metricRepoMock.On("Add", ctx, mock.Anything).Return("", addError)

		cacheMock := new(MockMetricCache)
		suite.service.MetricCache = cacheMock
		cacheMock.On("List", ctx, mock.Anything).Return([]metric.Entity{}, errors.New("cache miss"))

		err := suite.service.calculateRollbackCount(ctx, now)

		suite.Error(err)
		suite.Contains(err.Error(), "add failed")
	})
}

func (suite *MetricServiceTestSuite) TestCalculateStatusUpdatesComprehensive() {
	ctx := context.Background()
	now := time.Now()

	suite.Run("Daily interval with successful count", func() {
		// Reset mocks
		clientRepoMock := new(MockClientRepository)
		suite.service.clientRepository = clientRepoMock
		metricRepoMock := new(MockMetricRepository)
		suite.service.MetricRepository = metricRepoMock

		// Setup mock to return client count
		clientCount := int64(42)
		clientRepoMock.On("Count", ctx, mock.MatchedBy(func(filter bson.M) bool {
			// Verify filter contains expected date range for daily interval
			updateFilter, ok := filter["last_updated"].(bson.M)
			if !ok {
				return false
			}
			return updateFilter["$gte"] != nil && updateFilter["$lte"] != nil
		})).Return(clientCount, nil)

		// Expect metric storage
		metricRepoMock.On("Add", ctx, mock.MatchedBy(func(m metric.Entity) bool {
			return *m.Type == metric.StatusUpdates &&
				*m.Value == float64(clientCount) &&
				*m.Interval == "day"
		})).Return("metric-id", nil)

		err := suite.service.calculateStatusUpdates(ctx, now, "day")

		suite.NoError(err)
		clientRepoMock.AssertExpectations(suite.T())
		metricRepoMock.AssertExpectations(suite.T())
	})

	suite.Run("Weekly interval with successful count", func() {
		// Reset mocks
		clientRepoMock := new(MockClientRepository)
		suite.service.clientRepository = clientRepoMock
		metricRepoMock := new(MockMetricRepository)
		suite.service.MetricRepository = metricRepoMock

		clientCount := int64(75)
		clientRepoMock.On("Count", ctx, mock.MatchedBy(func(filter bson.M) bool {
			// Verify filter contains date range for weekly interval
			updateFilter, ok := filter["last_updated"].(bson.M)
			if !ok {
				return false
			}
			return updateFilter["$gte"] != nil && updateFilter["$lte"] != nil
		})).Return(clientCount, nil)

		metricRepoMock.On("Add", ctx, mock.MatchedBy(func(m metric.Entity) bool {
			return *m.Type == metric.StatusUpdates &&
				*m.Value == float64(clientCount) &&
				*m.Interval == "week"
		})).Return("metric-id", nil)

		err := suite.service.calculateStatusUpdates(ctx, now, "week")

		suite.NoError(err)
		clientRepoMock.AssertExpectations(suite.T())
		metricRepoMock.AssertExpectations(suite.T())
	})

	suite.Run("Monthly interval with successful count", func() {
		// Reset mocks
		clientRepoMock := new(MockClientRepository)
		suite.service.clientRepository = clientRepoMock
		metricRepoMock := new(MockMetricRepository)
		suite.service.MetricRepository = metricRepoMock

		clientCount := int64(150)
		clientRepoMock.On("Count", ctx, mock.MatchedBy(func(filter bson.M) bool {
			// Verify filter contains date range for monthly interval
			updateFilter, ok := filter["last_updated"].(bson.M)
			if !ok {
				return false
			}
			return updateFilter["$gte"] != nil && updateFilter["$lte"] != nil
		})).Return(clientCount, nil)

		metricRepoMock.On("Add", ctx, mock.MatchedBy(func(m metric.Entity) bool {
			return *m.Type == metric.StatusUpdates &&
				*m.Value == float64(clientCount) &&
				*m.Interval == "month"
		})).Return("metric-id", nil)

		err := suite.service.calculateStatusUpdates(ctx, now, "month")

		suite.NoError(err)
		clientRepoMock.AssertExpectations(suite.T())
		metricRepoMock.AssertExpectations(suite.T())
	})

	suite.Run("Invalid interval", func() {
		// Reset mocks
		clientRepoMock := new(MockClientRepository)
		suite.service.clientRepository = clientRepoMock
		metricRepoMock := new(MockMetricRepository)
		suite.service.MetricRepository = metricRepoMock

		// Try with invalid interval
		invalidInterval := "quarter"
		err := suite.service.calculateStatusUpdates(ctx, now, invalidInterval)

		suite.Error(err)
		suite.Contains(err.Error(), "invalid interval")
		clientRepoMock.AssertNotCalled(suite.T(), "Count")
		metricRepoMock.AssertNotCalled(suite.T(), "Add")
	})

	suite.Run("Error when counting clients", func() {
		// Reset mocks
		clientRepoMock := new(MockClientRepository)
		suite.service.clientRepository = clientRepoMock
		metricRepoMock := new(MockMetricRepository)
		suite.service.MetricRepository = metricRepoMock

		// Setup repository to return error on count
		countError := errors.New("database connection error")
		clientRepoMock.On("Count", ctx, mock.Anything).Return(int64(0), countError)

		err := suite.service.calculateStatusUpdates(ctx, now, "day")

		suite.Error(err)
		suite.Contains(err.Error(), "failed to count status updates")
		suite.Contains(err.Error(), "database connection error")
		metricRepoMock.AssertNotCalled(suite.T(), "Add")
	})

	suite.Run("Error when storing metric", func() {
		// Reset mocks
		clientRepoMock := new(MockClientRepository)
		suite.service.clientRepository = clientRepoMock
		metricRepoMock := new(MockMetricRepository)
		suite.service.MetricRepository = metricRepoMock

		// Setup successful count
		clientCount := int64(10)
		clientRepoMock.On("Count", ctx, mock.Anything).Return(clientCount, nil)

		// Setup error on metric storage
		storeError := errors.New("metric storage failed")
		metricRepoMock.On("Add", ctx, mock.Anything).Return("", storeError)

		err := suite.service.calculateStatusUpdates(ctx, now, "day")

		suite.Error(err)
		suite.Contains(err.Error(), "failed to store status updates metric")
		clientRepoMock.AssertExpectations(suite.T())
	})

	suite.Run("Zero clients found", func() {
		// Reset mocks
		clientRepoMock := new(MockClientRepository)
		suite.service.clientRepository = clientRepoMock
		metricRepoMock := new(MockMetricRepository)
		suite.service.MetricRepository = metricRepoMock

		// Setup count to return zero
		clientRepoMock.On("Count", ctx, mock.Anything).Return(int64(0), nil)

		// Should still create metric with zero value
		metricRepoMock.On("Add", ctx, mock.MatchedBy(func(m metric.Entity) bool {
			return *m.Type == metric.StatusUpdates &&
				*m.Value == 0.0 &&
				*m.Interval == "day"
		})).Return("metric-id", nil)

		err := suite.service.calculateStatusUpdates(ctx, now, "day")

		suite.NoError(err)
		clientRepoMock.AssertExpectations(suite.T())
		metricRepoMock.AssertExpectations(suite.T())
	})
}

func TestMetricService(t *testing.T) {
	suite.Run(t, new(MetricServiceTestSuite))
}

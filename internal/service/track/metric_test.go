package track

import (
	"TrackMe/internal/domain/client"
	"TrackMe/internal/domain/contract"
	"TrackMe/internal/domain/metric"
	"TrackMe/internal/domain/prometheus"
	"TrackMe/internal/domain/stage"
	"TrackMe/pkg/store"
	"context"
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
	suite.PrometheusMetrics = NewForTesting()
}

func (suite *MetricServiceTestSuite) SetupTest() {
	suite.metricRepositoryMock = new(MockMetricRepository)
	suite.metricCacheMock = new(MockMetricCache)
	suite.clientRepositoryMock = new(MockClientRepository)
	suite.stageRepositoryMock = new(MockStageRepository)

	suite.service = &Service{
		MetricRepository:  suite.metricRepositoryMock,
		clientRepository:  suite.clientRepositoryMock,
		StageRepository:   suite.stageRepositoryMock,
		PrometheusMetrics: suite.PrometheusMetrics,
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
		return *m.Type == metric.MAU && *m.Value == float64(mauCount) && *m.Interval == "day"
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

func TestMetricService(t *testing.T) {
	suite.Run(t, new(MetricServiceTestSuite))
}

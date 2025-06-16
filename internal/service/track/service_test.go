package track

import (
	"TrackMe/internal/domain/prometheus"
	"errors"
	"github.com/stretchr/testify/suite"
	"testing"
)

type ServiceTestSuite struct {
	suite.Suite
	clientRepositoryMock *MockClientRepository
	stageRepositoryMock  *MockStageRepository
	metricRepositoryMock *MockMetricRepository
	metricCacheMock      *MockMetricCache
	prometheusMetrics    prometheus.Entity
}

func (suite *ServiceTestSuite) SetupTest() {
	suite.clientRepositoryMock = new(MockClientRepository)
	suite.stageRepositoryMock = new(MockStageRepository)
	suite.metricRepositoryMock = new(MockMetricRepository)
	suite.metricCacheMock = new(MockMetricCache)
	suite.prometheusMetrics = NewForTesting()
}

func (suite *ServiceTestSuite) TestNewServiceNilConfigs() {
	service, err := New(nil)
	suite.NoError(err)
	suite.NotNil(service)
}

func (suite *ServiceTestSuite) TestServiceWithMultipleConfigurations() {

	configCalls := []string{}

	config1 := func(s *Service) error {
		configCalls = append(configCalls, "config1")
		return nil
	}

	config2 := func(s *Service) error {
		configCalls = append(configCalls, "config2")
		return nil
	}

	service, err := New(config1, config2)

	suite.NoError(err)
	suite.NotNil(service)
	suite.Equal([]string{"config1", "config2"}, configCalls)
}

func (suite *ServiceTestSuite) TestServiceWithEarlyConfigError() {

	configCalls := []string{}

	config1 := func(s *Service) error {
		configCalls = append(configCalls, "config1")
		return errors.New("config1 error")
	}

	config2 := func(s *Service) error {
		configCalls = append(configCalls, "config2")
		return nil
	}

	service, err := New(config1, config2)

	suite.Error(err)
	suite.Equal("config1 error", err.Error())
	suite.Equal([]string{"config1"}, configCalls)
	suite.Equal(Service{}, *service)
}

func (suite *ServiceTestSuite) TestServiceComponentAccess() {

	service, err := New(
		WithClientRepository(suite.clientRepositoryMock),
		WithStageRepository(suite.stageRepositoryMock),
		WithMetricRepository(suite.metricRepositoryMock),
		WithPrometheusMetrics(suite.prometheusMetrics),
		WithMetricCache(suite.metricCacheMock),
	)

	suite.NoError(err)

	suite.Equal(suite.clientRepositoryMock, service.clientRepository)
	suite.Equal(suite.stageRepositoryMock, service.StageRepository)
	suite.Equal(suite.metricRepositoryMock, service.MetricRepository)
	suite.Equal(suite.prometheusMetrics, service.PrometheusMetrics)
	suite.Equal(suite.metricCacheMock, service.MetricCache)
}

func TestService(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}

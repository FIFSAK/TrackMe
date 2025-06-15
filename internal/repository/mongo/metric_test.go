package mongo

import (
	"TrackMe/internal/domain/metric"
	"context"
	"fmt"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"testing"
	"time"
)

type MetricRepositorySuite struct {
	suite.Suite
	repository   metric.Repository
	testDatabase *TestDatabase
}

func (suite *MetricRepositorySuite) SetupSuite() {
	suite.testDatabase = SetupTestDatabaseWithName("metric_test_db")
	suite.repository = NewMetricRepository(suite.testDatabase.DbInstance)
}

func (suite *MetricRepositorySuite) TearDownSuite() {
	suite.testDatabase.container.Terminate(context.Background())
}

func (suite *MetricRepositorySuite) SetupTest() {
	// Clean up before each test
	_, err := suite.testDatabase.DbInstance.Collection("metrics").DeleteMany(
		context.Background(), bson.M{})
	suite.NoError(err)
}

func (suite *MetricRepositorySuite) TestAdd() {
	suite.Run("add new metric", func() {
		// Create a test metric
		id := primitive.NewObjectID().Hex()
		now := time.Now()
		metricType := metric.DAU
		value := float64(150)
		interval := "daily"

		metricData := metric.Entity{
			ID:        id,
			Type:      &metricType,
			Value:     &value,
			Interval:  &interval,
			CreatedAt: &now,
		}

		// After Add
		insertedID, err := suite.repository.Add(context.Background(), metricData)
		suite.NoError(err)
		suite.Equal(id, insertedID)

		objID, err := primitive.ObjectIDFromHex(id)
		suite.NoError(err)

		// Debug: Count documents in collection
		count, err := suite.testDatabase.DbInstance.Collection("metrics").CountDocuments(context.Background(), bson.M{})
		suite.NoError(err)
		suite.Equal(int64(1), count, "Should have exactly one document after insert")

		// Verify it was added by checking the database directly
		// Verify it was added by checking the database directly
		var result bson.M
		err = suite.testDatabase.DbInstance.Collection("metrics").FindOne(
			context.Background(),
			bson.M{"_id": objID}).Decode(&result)

		if err != nil {
			// Debug: Print all documents if the specific one isn't found
			cursor, _ := suite.testDatabase.DbInstance.Collection("metrics").Find(context.Background(), bson.M{})
			var all []bson.M
			_ = cursor.All(context.Background(), &all)
			fmt.Printf("All documents in collection: %+v\n", all)
		}
		suite.NoError(err)

		suite.NoError(err)
		suite.Equal(string(metricType), result["type"])
		suite.Equal(value, result["value"])
		suite.Equal(interval, result["interval"])
	})

	suite.Run("with invalid id", func() {
		metricType := metric.DAU
		value := float64(50)
		interval := "hourly"
		now := time.Now()

		metricData := metric.Entity{
			ID:        "invalid-id",
			Type:      &metricType,
			Value:     &value,
			Interval:  &interval,
			CreatedAt: &now,
		}

		_, err := suite.repository.Add(context.Background(), metricData)

		suite.Error(err)
	})
}

func (suite *MetricRepositorySuite) TestUpdate() {
	suite.Run("update existing metric", func() {
		// First add a metric
		id := primitive.NewObjectID().Hex()
		now := time.Now()
		metricType := metric.MAU
		value := float64(200)
		interval := "daily"

		originalMetric := metric.Entity{
			ID:        id,
			Type:      &metricType,
			Value:     &value,
			Interval:  &interval,
			CreatedAt: &now,
		}

		_, err := suite.repository.Add(context.Background(), originalMetric)
		suite.NoError(err)

		// Now update the metric
		updatedValue := float64(300)
		updatedMetric := metric.Entity{
			Type:      &metricType,
			Value:     &updatedValue,
			Interval:  &interval,
			CreatedAt: &now,
		}

		result, err := suite.repository.Update(context.Background(), id, updatedMetric)

		suite.NoError(err)
		suite.Equal(id, result.ID)
		suite.Equal(*updatedMetric.Type, *result.Type)
		suite.Equal(*updatedMetric.Value, *result.Value)
		suite.Equal(*updatedMetric.Interval, *result.Interval)
	})

	suite.Run("insert new metric if not exists", func() {
		// Create a new metric via update (upsert)
		id := primitive.NewObjectID().Hex()
		now := time.Now()
		metricType := metric.Conversion
		value := float64(25)
		interval := "weekly"

		newMetric := metric.Entity{
			Type:      &metricType,
			Value:     &value,
			Interval:  &interval,
			CreatedAt: &now,
		}

		result, err := suite.repository.Update(context.Background(), id, newMetric)

		suite.NoError(err)
		suite.Equal(id, result.ID)
		suite.Equal(*newMetric.Type, *result.Type)
		suite.Equal(*newMetric.Value, *result.Value)
		suite.Equal(*newMetric.Interval, *result.Interval)
	})

	suite.Run("with invalid id", func() {
		metricType := metric.Dropout
		value := float64(75)
		interval := "daily"
		now := time.Now()

		metricData := metric.Entity{
			Type:      &metricType,
			Value:     &value,
			Interval:  &interval,
			CreatedAt: &now,
		}

		_, err := suite.repository.Update(context.Background(), "invalid-id", metricData)
		suite.Error(err)
	})
}

func (suite *MetricRepositorySuite) TestList() {
	// Create test metrics
	suite.createTestMetrics()

	suite.Run("list all metrics", func() {
		metrics, err := suite.repository.List(context.Background(), metric.Filters{})

		suite.NoError(err)
		suite.Equal(3, len(metrics))
	})

	suite.Run("list with type filter", func() {
		metrics, err := suite.repository.List(context.Background(), metric.Filters{
			Type: string(metric.DAU),
		})

		suite.NoError(err)
		suite.Equal(2, len(metrics))
		suite.Equal(metric.DAU, *metrics[0].Type)
		suite.Equal(metric.DAU, *metrics[1].Type)
	})

	suite.Run("list with interval filter", func() {
		metrics, err := suite.repository.List(context.Background(), metric.Filters{
			Interval: "weekly",
		})

		suite.NoError(err)
		suite.Equal(1, len(metrics))
		suite.Equal("weekly", *metrics[0].Interval)
	})

	suite.Run("list with both filters", func() {
		metrics, err := suite.repository.List(context.Background(), metric.Filters{
			Type:     string(metric.DAU),
			Interval: "daily",
		})

		suite.NoError(err)
		suite.Equal(1, len(metrics))
		suite.Equal(metric.DAU, *metrics[0].Type)
		suite.Equal("daily", *metrics[0].Interval)
	})

	suite.Run("list with no matching results", func() {
		metrics, err := suite.repository.List(context.Background(), metric.Filters{
			Type: "unknown-type",
		})

		suite.NoError(err)
		suite.Equal(0, len(metrics))
	})
}

// Helper method to create test metrics
func (suite *MetricRepositorySuite) createTestMetrics() {
	now := time.Now()

	// Metric 1: daily DAU
	id1 := primitive.NewObjectID().Hex()
	metricType1 := metric.DAU
	value1 := float64(100)
	interval1 := "daily"

	metric1 := metric.Entity{
		ID:        id1,
		Type:      &metricType1,
		Value:     &value1,
		Interval:  &interval1,
		CreatedAt: &now,
	}

	// Metric 2: weekly DAU
	id2 := primitive.NewObjectID().Hex()
	metricType2 := metric.DAU
	value2 := float64(500)
	interval2 := "weekly"

	metric2 := metric.Entity{
		ID:        id2,
		Type:      &metricType2,
		Value:     &value2,
		Interval:  &interval2,
		CreatedAt: &now,
	}

	// Metric 3: daily Conversion
	id3 := primitive.NewObjectID().Hex()
	metricType3 := metric.Conversion
	value3 := float64(25)
	interval3 := "daily"

	metric3 := metric.Entity{
		ID:        id3,
		Type:      &metricType3,
		Value:     &value3,
		Interval:  &interval3,
		CreatedAt: &now,
	}

	_, err1 := suite.repository.Add(context.Background(), metric1)
	_, err2 := suite.repository.Add(context.Background(), metric2)
	_, err3 := suite.repository.Add(context.Background(), metric3)

	suite.NoError(err1)
	suite.NoError(err2)
	suite.NoError(err3)
}

func TestMetricRepositorySuite(t *testing.T) {
	suite.Run(t, new(MetricRepositorySuite))
}

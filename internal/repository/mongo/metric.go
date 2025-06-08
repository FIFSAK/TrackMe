package mongo

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"TrackMe/internal/domain/metric"
	"TrackMe/pkg/store"
)

// MetricRepository handles CRUD operations for metric in a MongoDB database.
type MetricRepository struct {
	db *mongo.Collection
}

// NewMetricRepository creates a new MetricRepository.
func NewMetricRepository(db *mongo.Database) *MetricRepository {
	return &MetricRepository{db: db.Collection("metrics")}
}

// List retrieves all metric from the database.
func (r *MetricRepository) List(ctx context.Context, filters metric.Filters) ([]metric.Entity, error) {
	filter := bson.M{}

	if filters.Type != "" {
		filter["type"] = filters.Type
	}

	if filters.Interval != "" {
		filter["interval"] = filters.Interval
	}

	fmt.Printf("MongoDB filter: %+v\n", filter)

	opts := options.Find()

	cur, err := r.db.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var metrics []metric.Entity
	if err = cur.All(ctx, &metrics); err != nil {
		return nil, err
	}

	// Debug the results
	fmt.Printf("Found %d metrics in MongoDB\n", len(metrics))

	return metrics, nil
}

// Add inserts a new metric into the database.
func (r *MetricRepository) Add(ctx context.Context, data metric.Entity) (string, error) {
	res, err := r.db.InsertOne(ctx, data)
	if err != nil {
		return "", err
	}
	return res.InsertedID.(primitive.ObjectID).Hex(), nil
}

// Get retrieves a metric by ID from the database.
func (r *MetricRepository) Get(ctx context.Context, id string) (metric.Entity, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return metric.Entity{}, err
	}
	var metric metric.Entity
	err = r.db.FindOne(ctx, bson.M{"_id": objID}).Decode(&metric)
	if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
		return metric, store.ErrorNotFound
	}
	return metric, err
}

// Update modifies an existing metric in the database.
func (r *MetricRepository) Update(ctx context.Context, id string, data metric.Entity) (metric.Entity, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return metric.Entity{}, err
	}

	update := bson.M{
		"$set":         data,
		"$setOnInsert": bson.M{"registration_date": time.Now()},
	}

	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var updated metric.Entity
	err = r.db.FindOneAndUpdate(
		ctx,
		bson.M{"_id": objID},
		update,
		opts,
	).Decode(&updated)

	if err != nil {
		return metric.Entity{}, err
	}

	return updated, nil
}

// Delete removes a metric by ID from the database.
func (r *MetricRepository) Delete(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	res, err := r.db.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return store.ErrorNotFound
	}
	return nil
}

package mongo

import (
	"TrackMe/internal/domain/metric"
	"TrackMe/pkg/log"
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
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

	opts := options.Find()

	cur, err := r.db.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer func(cur *mongo.Cursor, ctx context.Context) {
		err = cur.Close(ctx)
		if err != nil {
			logger := log.LoggerFromContext(ctx)
			logger.Error().Err(err).Msg("failed to close cursor")
		}
	}(cur, ctx)

	var metrics []metric.Entity
	if err = cur.All(ctx, &metrics); err != nil {
		return nil, err
	}

	return metrics, nil
}

// Add inserts a new metric into the database.
func (r *MetricRepository) Add(ctx context.Context, data metric.Entity) (string, error) {
	objId, err := primitive.ObjectIDFromHex(data.ID)
	if err != nil {
		return "", err
	}
	res, err := r.db.InsertOne(ctx, bson.M{"_id": objId, "type": data.Type, "value": data.Value, "interval": data.Interval, "created_at": data.CreatedAt, "updated_at": time.Now(), "metadata": data.Metadata})
	if err != nil {
		return "", err
	}
	return res.InsertedID.(primitive.ObjectID).Hex(), nil
}

// Update modifies an existing metric in the database.
func (r *MetricRepository) Update(ctx context.Context, id string, data metric.Entity) (metric.Entity, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return metric.Entity{}, err
	}

	// Manually build update document excluding ID
	update := bson.M{
		"$set": bson.M{
			"type":       data.Type,
			"value":      data.Value,
			"interval":   data.Interval,
			"created_at": data.CreatedAt,
			"updated_at": time.Now(),
			"metadata":   data.Metadata,
			// Add other fields but NOT id/_id
		},
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

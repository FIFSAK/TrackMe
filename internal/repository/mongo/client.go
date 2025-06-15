package mongo

import (
	"TrackMe/internal/domain/client"
	"TrackMe/pkg/store"
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

// ClientRepository handles CRUD operations for client in a MongoDB database.
type ClientRepository struct {
	db *mongo.Collection
}

// NewClientRepository creates a new ClientRepository.
func NewClientRepository(db *mongo.Database) *ClientRepository {
	return &ClientRepository{db: db.Collection("clients")}
}

// List retrieves all client from the database.
func (r *ClientRepository) List(ctx context.Context, filters client.Filters, limit, offset int) ([]client.Entity, int, error) {
	filter := bson.M{}

	if filters.ID != "" {
		objID, err := primitive.ObjectIDFromHex(filters.ID)
		if err != nil {
			return nil, 0, err
		}
		filter["_id"] = objID
	}

	if filters.Stage != "" {
		filter["current_stage"] = filters.Stage
	}

	if filters.Source != "" {
		filter["source"] = filters.Source
	}

	if filters.Channel != "" {
		filter["channel"] = filters.Channel
	}

	if filters.AppStatus != "" {
		filter["app"] = filters.AppStatus
	}

	if filters.IsActive != nil {
		filter["is_active"] = filters.IsActive
	}

	if !filters.UpdatedAfter.IsZero() {
		filter["last_updated"] = bson.M{"$gte": filters.UpdatedAfter}
	}

	if !filters.LastLoginAfter.IsZero() {
		filter["last_login"] = bson.M{"$gte": filters.LastLoginAfter}
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.M{"last_updated": -1})

	total, err := r.db.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 10
	}

	cur, err := r.db.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)

	var clients []client.Entity
	if err = cur.All(ctx, &clients); err != nil {
		return nil, 0, err
	}

	return clients, int(total), nil
}

// Get retrieves a client by ID from the database.
func (r *ClientRepository) Get(ctx context.Context, id string) (client.Entity, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return client.Entity{}, err
	}
	var client client.Entity
	err = r.db.FindOne(ctx, bson.M{"_id": objID}).Decode(&client)
	if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
		return client, store.ErrorNotFound
	}
	return client, err
}

// Update modifies an existing client in the database.
func (r *ClientRepository) Update(ctx context.Context, id string, data client.Entity) (client.Entity, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return client.Entity{}, err
	}

	update := bson.M{
		"$set": bson.M{
			"name":          data.Name,
			"email":         data.Email,
			"current_stage": data.CurrentStage,
			"last_updated":  data.LastUpdated,
			"is_active":     data.IsActive,
			"source":        data.Source,
			"channel":       data.Channel,
			"app":           data.App,
			"last_login":    data.LastLogin,
			"contracts":     data.Contracts,
		},
		"$setOnInsert": bson.M{"registration_date": time.Now()},
	}

	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var updated client.Entity
	err = r.db.FindOneAndUpdate(
		ctx,
		bson.M{"_id": objID},
		update,
		opts,
	).Decode(&updated)

	if err != nil {
		return client.Entity{}, err
	}

	return updated, nil
}

func (r *ClientRepository) Count(ctx context.Context, filter bson.M) (int64, error) {
	return r.db.CountDocuments(ctx, filter)
}

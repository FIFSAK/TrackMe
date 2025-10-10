package mongo

import (
	"TrackMe/internal/domain/client"
	"TrackMe/pkg/log"
	"TrackMe/pkg/store"
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

	if limit <= 0 {
		limit = 10
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.M{"last_updated": -1})

	total, err := r.db.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	cur, err := r.db.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer func(cur *mongo.Cursor, ctx context.Context) {
		err = cur.Close(ctx)
		if err != nil {
			logger := log.LoggerFromContext(ctx)
			logger.Error().Err(err).Msg("failed to close cursor")
		}
	}(cur, ctx)

	var clients []client.Entity
	if err = cur.All(ctx, &clients); err != nil {
		return nil, 0, err
	}

	return clients, int(total), nil
}

// Create inserts a new client into the database.
func (r *ClientRepository) Create(ctx context.Context, data client.Entity) (client.Entity, error) {
	result, err := r.db.InsertOne(ctx, data)
	if err != nil {
		return client.Entity{}, err
	}

	data.ID = result.InsertedID.(primitive.ObjectID).Hex()
	return data, nil
}

// Get retrieves a client by ID from the database.
func (r *ClientRepository) Get(ctx context.Context, id string) (client.Entity, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return client.Entity{}, err
	}
	var clientEntity client.Entity
	err = r.db.FindOne(ctx, bson.M{"_id": objID}).Decode(&clientEntity)
	if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
		return clientEntity, store.ErrorNotFound
	}
	return clientEntity, err
}

// GetByEmail retrieves a client by email from the database.
func (r *ClientRepository) GetByEmail(ctx context.Context, email string) (client.Entity, error) {
	var clientEntity client.Entity
	err := r.db.FindOne(ctx, bson.M{"email": email}).Decode(&clientEntity)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return clientEntity, store.ErrorNotFound
		}
		return clientEntity, err
	}
	return clientEntity, nil
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
	}

	opts := options.FindOneAndUpdate().
		SetReturnDocument(options.After)

	var updated client.Entity
	err = r.db.FindOneAndUpdate(
		ctx,
		bson.M{"_id": objID},
		update,
		opts,
	).Decode(&updated)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return client.Entity{}, store.ErrorNotFound
		}
		return client.Entity{}, err
	}

	return updated, nil
}

func (r *ClientRepository) Count(ctx context.Context, filter bson.M) (int64, error) {
	return r.db.CountDocuments(ctx, filter)
}

// Delete removes a client from the database.
func (r *ClientRepository) Delete(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	result, err := r.db.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return store.ErrorNotFound
	}

	return nil
}

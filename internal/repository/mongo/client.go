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

	"TrackMe/internal/domain/client"
	"TrackMe/pkg/store"
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

	// Add debugging to see what filter is being applied
	fmt.Printf("MongoDB filter: %+v\n", filter)

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

	// Debug the results
	fmt.Printf("Found %d clients in MongoDB\n", len(clients))

	return clients, int(total), nil
}

// Add inserts a new client into the database.
func (r *ClientRepository) Add(ctx context.Context, data client.Entity) (string, error) {
	res, err := r.db.InsertOne(ctx, data)
	if err != nil {
		return "", err
	}
	return res.InsertedID.(primitive.ObjectID).Hex(), nil
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
		"$set":         prepareUpdateFields(data),
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

func prepareUpdateFields(data client.Entity) bson.M {
	isActiveDefault := true
	if data.IsActive != nil {
		isActiveDefault = *data.IsActive
	}
	fields := bson.M{
		"name":          data.Name,
		"email":         data.Email,
		"current_stage": data.CurrentStage,
		"last_updated":  time.Now(),
		"is_active":     isActiveDefault,
		"source":        data.Source,
		"channel":       data.Channel,
		"app":           data.App,
		"last_login":    data.LastLogin,
	}

	if len(data.Contracts) > 0 {
		fields["contracts"] = data.Contracts
	}

	return fields
}

// Delete removes a client by ID from the database.
func (r *ClientRepository) Delete(ctx context.Context, id string) error {
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

// prepareArgs prepares the update arguments for the MongoDB query.
func (r *ClientRepository) prepareArgs(data client.Entity) bson.M {
	args := bson.M{}
	if data.Name != nil {
		args["name"] = data.Name
	}
	if data.Email != nil {
		args["email"] = data.Email
	}
	if data.CurrentStage != nil {
		args["current_stage"] = data.CurrentStage
	}
	if data.RegistrationDate != nil {
		args["registration_date"] = data.RegistrationDate
	}
	if data.LastUpdated != nil {
		args["last_updated"] = data.LastUpdated
	}
	if data.IsActive != nil {
		args["is_active"] = data.IsActive
	}
	if data.Source != nil {
		args["source"] = data.Source
	}
	if data.Channel != nil {
		args["channel"] = data.Channel
	}
	return args
}

func (r *ClientRepository) Count(ctx context.Context, filter bson.M) (int64, error) {
	return r.db.CountDocuments(ctx, filter)
}

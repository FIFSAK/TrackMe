package mongo

import (
	"context"
	"errors"

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
func (r *ClientRepository) List(ctx context.Context) ([]client.Entity, error) {
	cur, err := r.db.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	var clients []client.Entity
	if err = cur.All(ctx, &clients); err != nil {
		return nil, err
	}
	return clients, nil
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
func (r *ClientRepository) Update(ctx context.Context, id string, data client.Entity) error {
	if id == "" {
		data.ID = id
		_, err := r.Add(ctx, data)
		if err != nil {
			return err
		}
		return nil
	}
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	args := r.prepareArgs(data)
	if len(args) == 0 {
		return nil
	}
	res, err := r.db.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": args})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return store.ErrorNotFound
	}
	return nil
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

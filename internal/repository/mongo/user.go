package mongo

import (
	"TrackMe/internal/domain/user"
	"TrackMe/pkg/log"
	"TrackMe/pkg/store"
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UserRepository handles CRUD operations for users in a MongoDB database.
type UserRepository struct {
	db *mongo.Collection
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{db: db.Collection("users")}
}

// List retrieves all users from the database.
func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]user.Entity, int, error) {
	if limit <= 0 {
		limit = 50
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.M{"created_at": -1})

	total, err := r.db.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, err
	}

	cur, err := r.db.Find(ctx, bson.M{}, opts)
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

	var users []user.Entity
	if err = cur.All(ctx, &users); err != nil {
		return nil, 0, err
	}

	return users, int(total), nil
}

// Create inserts a new user into the database.
func (r *UserRepository) Create(ctx context.Context, data user.Entity) (user.Entity, error) {
	result, err := r.db.InsertOne(ctx, data)
	if err != nil {
		return user.Entity{}, err
	}

	data.ID = result.InsertedID.(primitive.ObjectID).Hex()
	return data, nil
}

// Get retrieves a user by ID from the database.
func (r *UserRepository) Get(ctx context.Context, id string) (user.Entity, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return user.Entity{}, err
	}

	var userEntity user.Entity
	err = r.db.FindOne(ctx, bson.M{"_id": objID}).Decode(&userEntity)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return userEntity, store.ErrorNotFound
		}
		return userEntity, err
	}

	return userEntity, nil
}

// GetByEmail retrieves a user by email from the database.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (user.Entity, error) {
	var userEntity user.Entity
	err := r.db.FindOne(ctx, bson.M{"email": email}).Decode(&userEntity)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return userEntity, store.ErrorNotFound
		}
		return userEntity, err
	}
	return userEntity, nil
}

// Update modifies an existing user in the database.
func (r *UserRepository) Update(ctx context.Context, id string, data user.Entity) (user.Entity, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return user.Entity{}, err
	}

	data.UpdatedAt = time.Now()
	update := bson.M{
		"$set": bson.M{
			"name":       data.Name,
			"email":      data.Email,
			"role":       data.Role,
			"updated_at": data.UpdatedAt,
		},
	}

	opts := options.FindOneAndUpdate().
		SetReturnDocument(options.After)

	var updated user.Entity
	err = r.db.FindOneAndUpdate(
		ctx,
		bson.M{"_id": objID},
		update,
		opts,
	).Decode(&updated)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return user.Entity{}, store.ErrorNotFound
		}
		return user.Entity{}, err
	}

	return updated, nil
}

// Delete removes a user from the database.
func (r *UserRepository) Delete(ctx context.Context, id string) error {
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

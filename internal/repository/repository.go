package repository

import (
	"TrackMe/internal/domain/client"
	"TrackMe/internal/domain/metric"
	"TrackMe/internal/domain/stage"
	"TrackMe/internal/domain/user"
	clickhouse "TrackMe/internal/repository/click_house"
	"TrackMe/internal/repository/memory"
	"TrackMe/internal/repository/mongo"
	"TrackMe/internal/repository/postgres"
	"TrackMe/pkg/store"
	"context"
)

// Configuration is an alias for a function that will take in a pointer to a Repository and modify it
type Configuration func(r *Repository) error

// Repository is an implementation of the Repository
type Repository struct {
	clickhouse store.ClickHouse
	mongo      store.Mongo
	postgres   store.PostgreSQL
	Stage      stage.Repository
	Client     client.Repository
	User       user.Repository
	Metric     metric.Repository
}

// New takes a variable amount of Configuration functions and returns a new Repository
// Each Configuration will be called in the order they are passed in
func New(configs ...Configuration) (s *Repository, err error) {
	// Create the repository
	s = &Repository{}

	// Apply all Configurations passed in
	for _, cfg := range configs {
		// Pass the repository into the configuration function
		if err = cfg(s); err != nil {
			return
		}
	}

	return
}

// Close closes the repository and prevents new queries from starting.
// Close then waits for all queries that have started processing on the server to finish.
func (r *Repository) Close() {
	if r.postgres.Client != nil {
		r.postgres.Client.Close() // no :=, just call it
	}

	if r.mongo.Client != nil {
		err := r.mongo.Client.Disconnect(context.Background())
		if err != nil {
			return
		}
	}
}

// WithMemoryStore applies a memory store to the Repository
func WithMemoryStore() Configuration {
	return func(s *Repository) (err error) {
		// Create the memory store, if we needed parameters, such as connection strings they could be inputted here
		s.Stage = memory.NewStageRepository()

		return
	}
}

// WithMongoStore applies a mongo store to the Repository
func WithMongoStore(uri, name string) Configuration {
	return func(s *Repository) (err error) {
		// Create the mongo store, if we needed parameters, such as connection strings they could be inputted here
		s.mongo, err = store.NewMongo(uri)
		if err != nil {
			return
		}
		database := s.mongo.Client.Database(name)

		s.Client = mongo.NewClientRepository(database)

		s.User = mongo.NewUserRepository(database)

		// s.Metric = mongo.NewMetricRepository(database)

		return
	}
}

// WithClickHouseStore sets ClickHouse repositories
func WithClickHouseStore(dsn string) Configuration {
	return func(s *Repository) (err error) {
		s.clickhouse, err = store.NewClickHouse()
		if err != nil {
			return err
		}

		// pass the inner Conn
		// s.Client = clickhouse.NewClientRepository(s.clickhouse.Conn)
		// s.User = clickhouse.NewUserRepository(s.clickhouse.Conn)
		s.Metric = clickhouse.NewMetricRepository(s.clickhouse.Conn)

		return nil
	}
}

func WithPostgresStore(dsn string) Configuration {
	return func(s *Repository) error {
		db, err := store.NewPostgres(dsn) // returns PostgreSQL VALUE, not pointer
		if err != nil {
			return err
		}

		s.postgres = db

		s.Client = postgres.NewClientRepository(s.postgres.Client)
		s.User = postgres.NewUserRepository(s.postgres.Client)

		return nil
	}
}

package track

import (
	"TrackMe/internal/domain/client"
	"TrackMe/internal/domain/stage"
)

// Configuration is an alias for a function that will take in a pointer to a Service and modify it
type Configuration func(s *Service) error

// Service is an implementation of the Service
type Service struct {
	clientRepository client.Repository
	StageRepository  stage.Repository
	clientCache      client.Cache
}

// New takes a variable amount of Configuration functions and returns a new Service
// Each Configuration will be called in the order they are passed in
func New(configs ...Configuration) (s *Service, err error) {
	// Add the service
	s = &Service{}

	// Apply all Configurations passed in
	for _, cfg := range configs {
		// Pass the service into the configuration function
		if err = cfg(s); err != nil {
			return
		}
	}
	return
}

// WithClientRepository applies a given client repository to the Service
func WithClientRepository(authorRepository client.Repository) Configuration {
	// return a function that matches the Configuration alias,
	// You need to return this so that the parent function can take in all the needed parameters
	return func(s *Service) error {
		s.clientRepository = authorRepository
		return nil
	}
}

// WithStageRepository applies a given stage repository to the Service
func WithStageRepository(stageRepository stage.Repository) Configuration {
	// return a function that matches the Configuration alias,
	// You need to return this so that the parent function can take in all the needed parameters
	return func(s *Service) error {
		s.StageRepository = stageRepository
		return nil
	}
}

// WithClientCache applies a given client cache to the Service
func WithClientCache(clientCache client.Cache) Configuration {
	// return a function that matches the Configuration alias,
	// You need to return this so that the parent function can take in all the needed parameters
	return func(s *Service) error {
		s.clientCache = clientCache
		return nil
	}
}

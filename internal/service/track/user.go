package track

import (
	"TrackMe/internal/domain/user"
	"TrackMe/pkg/log"
	"TrackMe/pkg/store"
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	CreateUser(ctx context.Context, req user.Request) (user.Response, error)
	Login(ctx context.Context, email, password string) (user.Entity, error)
}

type UserService interface {
	ListUsers(ctx context.Context, limit, offset int) ([]user.Response, int, error)
	CreateUser(ctx context.Context, req user.Request) (user.Response, error)
	GetUser(ctx context.Context, id string) (user.Response, error)
	UpdateUser(ctx context.Context, id string, req user.Request) (user.Response, error)
	DeleteUser(ctx context.Context, id string) error
}

// ListUsers retrieves all users from the repository.
func (s *Service) ListUsers(ctx context.Context, limit, offset int) ([]user.Response, int, error) {
	logger := log.LoggerFromContext(ctx).With().
		Int("limit", limit).
		Int("offset", offset).
		Str("component", "service.user").
		Logger()

	entities, total, err := s.userRepository.List(ctx, limit, offset)
	if err != nil {
		logger.Error().Err(err).Msg("failed to list users")
		return nil, 0, err
	}

	responses := user.ParseFromEntities(entities)

	return responses, total, nil
}

// CreateUser creates a new user in the repository.
func (s *Service) CreateUser(ctx context.Context, req user.Request) (user.Response, error) {
	logger := log.LoggerFromContext(ctx).With().
		Interface("request", req).
		Str("component", "service.user.create").
		Logger()

	// Check if user with this email already exists
	existingUser, err := s.userRepository.GetByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, store.ErrorNotFound) {
		logger.Error().Err(err).Msg("failed to check existing user by email")
		return user.Response{}, err
	}
	if existingUser.ID != "" {
		logger.Warn().Str("email", req.Email).Msg("user with this email already exists")
		return user.Response{}, errors.New("user with this email already exists")
	}

	// Validate role
	if !user.IsValidRole(req.Role) {
		logger.Warn().Str("role", req.Role).Msg("invalid user role")
		return user.Response{}, errors.New("invalid user role")
	}

	// Validate password
	if req.Password == "" {
		return user.Response{}, errors.New("password is required")
	}
	if len(req.Password) < 6 {
		return user.Response{}, errors.New("password must be at least 6 characters")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error().Err(err).Msg("failed to hash password")
		return user.Response{}, err
	}

	newUser := user.New(req)
	newUser.Password = string(hashedPassword)

	result, err := s.userRepository.Create(ctx, newUser)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create user")
		return user.Response{}, err
	}

	logger.Info().
		Str("user_id", result.ID).
		Str("email", result.Email).
		Str("role", result.Role).
		Msg("user created successfully")

	return user.ParseFromEntity(result), nil
}

// GetUser retrieves a user by ID from the repository.
func (s *Service) GetUser(ctx context.Context, id string) (user.Response, error) {
	logger := log.LoggerFromContext(ctx).With().
		Str("user_id", id).
		Str("component", "service.user.get").
		Logger()

	entity, err := s.userRepository.Get(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrorNotFound) {
			logger.Warn().Msg("user not found")
			return user.Response{}, store.ErrorNotFound
		}
		logger.Error().Err(err).Msg("failed to get user")
		return user.Response{}, err
	}

	return user.ParseFromEntity(entity), nil
}

// UpdateUser updates an existing user in the repository.
func (s *Service) UpdateUser(ctx context.Context, id string, req user.Request) (user.Response, error) {
	logger := log.LoggerFromContext(ctx).With().
		Str("user_id", id).
		Interface("request", req).
		Str("component", "service.user.update").
		Logger()

	// Check if user exists
	existingUser, err := s.userRepository.Get(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrorNotFound) {
			logger.Warn().Msg("user not found")
			return user.Response{}, store.ErrorNotFound
		}
		logger.Error().Err(err).Msg("failed to get user")
		return user.Response{}, err
	}

	// Check if email is being changed to an existing email
	if req.Email != existingUser.Email {
		existingByEmail, err := s.userRepository.GetByEmail(ctx, req.Email)
		if err != nil && !errors.Is(err, store.ErrorNotFound) {
			logger.Error().Err(err).Msg("failed to check existing user by email")
			return user.Response{}, err
		}
		if existingByEmail.ID != "" && existingByEmail.ID != id {
			logger.Warn().Str("email", req.Email).Msg("user with this email already exists")
			return user.Response{}, errors.New("user with this email already exists")
		}
	}

	// Validate role
	if !user.IsValidRole(req.Role) {
		logger.Warn().Str("role", req.Role).Msg("invalid user role")
		return user.Response{}, errors.New("invalid user role")
	}

	updateUser := user.New(req)
	updateUser.ID = existingUser.ID
	updateUser.CreatedAt = existingUser.CreatedAt

	// If password is provided, hash it; otherwise keep the old one
	if req.Password != "" {
		if len(req.Password) < 6 {
			return user.Response{}, errors.New("password must be at least 6 characters")
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			logger.Error().Err(err).Msg("failed to hash password")
			return user.Response{}, err
		}
		updateUser.Password = string(hashedPassword)
	} else {
		updateUser.Password = existingUser.Password
	}

	result, err := s.userRepository.Update(ctx, id, updateUser)
	if err != nil {
		if errors.Is(err, store.ErrorNotFound) {
			logger.Warn().Msg("user not found")
			return user.Response{}, store.ErrorNotFound
		}
		logger.Error().Err(err).Msg("failed to update user")
		return user.Response{}, err
	}

	logger.Info().
		Str("user_id", result.ID).
		Str("email", result.Email).
		Str("role", result.Role).
		Msg("user updated successfully")

	return user.ParseFromEntity(result), nil
}

// DeleteUser removes a user from the repository.
func (s *Service) DeleteUser(ctx context.Context, id string) error {
	logger := log.LoggerFromContext(ctx).With().
		Str("user_id", id).
		Str("component", "service.user.delete").
		Logger()

	err := s.userRepository.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrorNotFound) {
			logger.Warn().Msg("user not found")
			return store.ErrorNotFound
		}
		logger.Error().Err(err).Msg("failed to delete user")
		return err
	}

	logger.Info().Msg("user deleted successfully")
	return nil
}

// Login authenticates a user and returns user info
func (s *Service) Login(ctx context.Context, email, password string) (user.Entity, error) {
	logger := log.LoggerFromContext(ctx).With().
		Str("email", email).
		Str("component", "service.user.login").
		Logger()

	// Get user by email
	userEntity, err := s.userRepository.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, store.ErrorNotFound) {
			logger.Warn().Msg("user not found")
			return user.Entity{}, errors.New("invalid email or password")
		}
		logger.Error().Err(err).Msg("failed to get user by email")
		return user.Entity{}, err
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(userEntity.Password), []byte(password))
	if err != nil {
		logger.Warn().Msg("invalid password")
		return user.Entity{}, errors.New("invalid email or password")
	}

	logger.Info().Str("user_id", userEntity.ID).Msg("user logged in successfully")
	return userEntity, nil
}

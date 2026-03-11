package service

import (
	"context"
	"errors"
	"fmt"

	"strings"

	"github.com/devaloi/restgo/internal/auth"
	"github.com/devaloi/restgo/internal/domain"
	"github.com/devaloi/restgo/internal/middleware"
	"github.com/devaloi/restgo/internal/repository"
)

// UserService handles user business logic.
type UserService struct {
	repo repository.UserRepository
	jwt  *auth.JWTService
}

// NewUserService creates a UserService.
func NewUserService(repo repository.UserRepository, jwt *auth.JWTService) *UserService {
	return &UserService{repo: repo, jwt: jwt}
}

// Register creates a new user and returns the user with a JWT.
func (s *UserService) Register(ctx context.Context, req domain.CreateUserRequest) (*domain.User, string, error) {
	if err := validateCreateUser(req); err != nil {
		return nil, "", err
	}

	exists, err := s.repo.Exists(ctx, req.Email)
	if err != nil {
		return nil, "", fmt.Errorf("checking email: %w", err)
	}
	if exists {
		return nil, "", domain.ErrConflict
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, "", fmt.Errorf("hashing password: %w", err)
	}

	user := &domain.User{
		ID:           middleware.NewID(),
		Email:        req.Email,
		PasswordHash: hash,
		Name:         req.Name,
	}

	err = s.repo.Create(ctx, user)
	if err != nil {
		return nil, "", fmt.Errorf("creating user: %w", err)
	}

	token, err := s.jwt.Generate(user.ID, user.Email)
	if err != nil {
		return nil, "", fmt.Errorf("generating token: %w", err)
	}

	return user, token, nil
}

// Login authenticates a user and returns the user with a JWT.
func (s *UserService) Login(ctx context.Context, req domain.LoginRequest) (*domain.User, string, error) {
	var loginErrs []domain.ValidationError
	if req.Email == "" {
		loginErrs = append(loginErrs, domain.ValidationError{Field: "email", Message: "email is required"})
	}
	if req.Password == "" {
		loginErrs = append(loginErrs, domain.ValidationError{Field: "password", Message: "password is required"})
	}
	if len(loginErrs) > 0 {
		return nil, "", &domain.ValidationErrors{Errors: loginErrs}
	}

	user, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, "", domain.ErrUnauthorized
		}
		return nil, "", fmt.Errorf("finding user: %w", err)
	}

	err = auth.ComparePassword(user.PasswordHash, req.Password)
	if err != nil {
		return nil, "", domain.ErrUnauthorized
	}

	token, err := s.jwt.Generate(user.ID, user.Email)
	if err != nil {
		return nil, "", fmt.Errorf("generating token: %w", err)
	}

	return user, token, nil
}

// GetProfile returns a user by ID.
func (s *UserService) GetProfile(ctx context.Context, userID string) (*domain.User, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("fetching user profile: %w", err)
	}
	return user, nil
}

func validateCreateUser(req domain.CreateUserRequest) error {
	var errs []domain.ValidationError
	if req.Email == "" {
		errs = append(errs, domain.ValidationError{Field: "email", Message: "email is required"})
	} else if !strings.Contains(req.Email, "@") {
		errs = append(errs, domain.ValidationError{Field: "email", Message: "invalid email format"})
	}
	if req.Password == "" {
		errs = append(errs, domain.ValidationError{Field: "password", Message: "password is required"})
	} else if len(req.Password) < 8 {
		errs = append(errs, domain.ValidationError{Field: "password", Message: "password must be at least 8 characters"})
	}
	if req.Name == "" {
		errs = append(errs, domain.ValidationError{Field: "name", Message: "name is required"})
	}
	if len(errs) > 0 {
		return &domain.ValidationErrors{Errors: errs}
	}
	return nil
}

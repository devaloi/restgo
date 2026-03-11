package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/devaloi/restgo/internal/auth"
	"github.com/devaloi/restgo/internal/domain"
	"github.com/devaloi/restgo/internal/repository"
)

const testJWTSecret = "test-secret-key-for-unit-tests"

func newUserTestService() (*UserService, *repository.MockUserRepository) {
	repo := repository.NewMockUserRepository()
	jwt := auth.New(testJWTSecret, 1*time.Hour)
	svc := NewUserService(repo, jwt)
	return svc, repo
}

func TestUserService_Register_Success(t *testing.T) {
	svc, _ := newUserTestService()

	user, token, err := svc.Register(context.Background(), domain.CreateUserRequest{
		Email:    "alice@example.com",
		Password: "securepass123",
		Name:     "Alice",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user == nil {
		t.Fatal("expected user, got nil")
	}
	if user.ID == "" {
		t.Error("expected user ID to be generated")
	}
	if user.Email != "alice@example.com" {
		t.Errorf("expected email 'alice@example.com', got %q", user.Email)
	}
	if user.Name != "Alice" {
		t.Errorf("expected name 'Alice', got %q", user.Name)
	}
	if token == "" {
		t.Error("expected JWT token, got empty string")
	}

	// Verify token is valid
	jwt := auth.New(testJWTSecret, 1*time.Hour)
	claims, err := jwt.Validate(token)
	if err != nil {
		t.Fatalf("token should be valid: %v", err)
	}
	if claims.UserID != user.ID {
		t.Errorf("token user ID mismatch: got %q, want %q", claims.UserID, user.ID)
	}
	if claims.Email != user.Email {
		t.Errorf("token email mismatch: got %q, want %q", claims.Email, user.Email)
	}
}

func TestUserService_Register_ValidationErrors(t *testing.T) {
	svc, _ := newUserTestService()

	tests := []struct {
		name      string
		req       domain.CreateUserRequest
		wantField string
	}{
		{
			name:      "empty email",
			req:       domain.CreateUserRequest{Email: "", Password: "securepass", Name: "Alice"},
			wantField: "email",
		},
		{
			name:      "invalid email format",
			req:       domain.CreateUserRequest{Email: "notanemail", Password: "securepass", Name: "Alice"},
			wantField: "email",
		},
		{
			name:      "empty password",
			req:       domain.CreateUserRequest{Email: "alice@example.com", Password: "", Name: "Alice"},
			wantField: "password",
		},
		{
			name:      "short password",
			req:       domain.CreateUserRequest{Email: "alice@example.com", Password: "short", Name: "Alice"},
			wantField: "password",
		},
		{
			name:      "empty name",
			req:       domain.CreateUserRequest{Email: "alice@example.com", Password: "securepass", Name: ""},
			wantField: "name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := svc.Register(context.Background(), tt.req)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}

			var verr *domain.ValidationErrors
			if !errors.As(err, &verr) {
				t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
			}

			found := false
			for _, e := range verr.Errors {
				if e.Field == tt.wantField {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected validation error on field %q, got %v", tt.wantField, verr.Errors)
			}
		})
	}
}

func TestUserService_Register_AllFieldsEmpty(t *testing.T) {
	svc, _ := newUserTestService()

	_, _, err := svc.Register(context.Background(), domain.CreateUserRequest{})
	if err == nil {
		t.Fatal("expected error for empty request")
	}

	var verr *domain.ValidationErrors
	if !errors.As(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}
	if len(verr.Errors) < 3 {
		t.Errorf("expected at least 3 validation errors, got %d", len(verr.Errors))
	}
}

func TestUserService_Register_DuplicateEmail(t *testing.T) {
	svc, _ := newUserTestService()

	req := domain.CreateUserRequest{
		Email:    "alice@example.com",
		Password: "securepass123",
		Name:     "Alice",
	}

	_, _, err := svc.Register(context.Background(), req)
	if err != nil {
		t.Fatalf("first register: %v", err)
	}

	_, _, err = svc.Register(context.Background(), req)
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict for duplicate email, got %v", err)
	}
}

func TestUserService_Register_PasswordHashed(t *testing.T) {
	svc, repo := newUserTestService()

	_, _, err := svc.Register(context.Background(), domain.CreateUserRequest{
		Email:    "alice@example.com",
		Password: "securepass123",
		Name:     "Alice",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Fetch from repo to verify password is hashed
	user, err := repo.GetByEmail(context.Background(), "alice@example.com")
	if err != nil {
		t.Fatalf("fetch user: %v", err)
	}
	if user.PasswordHash == "securepass123" {
		t.Error("password should be hashed, not stored in plaintext")
	}
	if !strings.HasPrefix(user.PasswordHash, "$2a$") && !strings.HasPrefix(user.PasswordHash, "$2b$") {
		t.Errorf("expected bcrypt hash prefix, got %q", user.PasswordHash[:10])
	}
}

func TestUserService_Login_Success(t *testing.T) {
	svc, _ := newUserTestService()

	// Register first
	_, _, err := svc.Register(context.Background(), domain.CreateUserRequest{
		Email:    "alice@example.com",
		Password: "securepass123",
		Name:     "Alice",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// Login
	user, token, err := svc.Login(context.Background(), domain.LoginRequest{
		Email:    "alice@example.com",
		Password: "securepass123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user == nil {
		t.Fatal("expected user, got nil")
	}
	if user.Email != "alice@example.com" {
		t.Errorf("expected email 'alice@example.com', got %q", user.Email)
	}
	if token == "" {
		t.Error("expected JWT token, got empty string")
	}
}

func TestUserService_Login_ValidationErrors(t *testing.T) {
	svc, _ := newUserTestService()

	tests := []struct {
		name      string
		req       domain.LoginRequest
		wantField string
	}{
		{
			name:      "empty email",
			req:       domain.LoginRequest{Email: "", Password: "pass"},
			wantField: "email",
		},
		{
			name:      "empty password",
			req:       domain.LoginRequest{Email: "alice@example.com", Password: ""},
			wantField: "password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := svc.Login(context.Background(), tt.req)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}

			var verr *domain.ValidationErrors
			if !errors.As(err, &verr) {
				t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
			}

			found := false
			for _, e := range verr.Errors {
				if e.Field == tt.wantField {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected validation error on field %q, got %v", tt.wantField, verr.Errors)
			}
		})
	}
}

func TestUserService_Login_WrongPassword(t *testing.T) {
	svc, _ := newUserTestService()

	_, _, err := svc.Register(context.Background(), domain.CreateUserRequest{
		Email:    "alice@example.com",
		Password: "securepass123",
		Name:     "Alice",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	_, _, err = svc.Login(context.Background(), domain.LoginRequest{
		Email:    "alice@example.com",
		Password: "wrongpassword",
	})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for wrong password, got %v", err)
	}
}

func TestUserService_Login_UserNotFound(t *testing.T) {
	svc, _ := newUserTestService()

	_, _, err := svc.Login(context.Background(), domain.LoginRequest{
		Email:    "nobody@example.com",
		Password: "anypassword",
	})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for unknown user, got %v", err)
	}
}

func TestUserService_GetProfile_Success(t *testing.T) {
	svc, _ := newUserTestService()

	registered, _, err := svc.Register(context.Background(), domain.CreateUserRequest{
		Email:    "alice@example.com",
		Password: "securepass123",
		Name:     "Alice",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	user, err := svc.GetProfile(context.Background(), registered.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Email != "alice@example.com" {
		t.Errorf("expected email 'alice@example.com', got %q", user.Email)
	}
}

func TestUserService_GetProfile_NotFound(t *testing.T) {
	svc, _ := newUserTestService()

	_, err := svc.GetProfile(context.Background(), "nonexistent-id")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

package auth

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateAndValidate(t *testing.T) {
	svc := New("test-secret", time.Hour)

	token, err := svc.Generate("user-123", "test@example.com")
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	claims, err := svc.Validate(token)
	if err != nil {
		t.Fatalf("Validate() error: %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "user-123")
	}
	if claims.Email != "test@example.com" {
		t.Errorf("Email = %q, want %q", claims.Email, "test@example.com")
	}
	if claims.IssuedAt.IsZero() {
		t.Error("IssuedAt should not be zero")
	}
	if claims.ExpiresAt.Before(claims.IssuedAt) {
		t.Error("ExpiresAt should be after IssuedAt")
	}
}

func TestExpiredToken(t *testing.T) {
	svc := New("test-secret", -time.Hour) // already expired

	token, err := svc.Generate("user-123", "test@example.com")
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	_, err = svc.Validate(token)
	if err != ErrExpiredToken {
		t.Errorf("Validate() error = %v, want ErrExpiredToken", err)
	}
}

func TestTamperedPayload(t *testing.T) {
	svc := New("test-secret", time.Hour)

	token, err := svc.Generate("user-123", "test@example.com")
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	// Tamper with the payload (middle part)
	parts := strings.Split(token, ".")
	parts[1] = parts[1] + "xx"
	tampered := strings.Join(parts, ".")

	_, err = svc.Validate(tampered)
	if err != ErrInvalidToken {
		t.Errorf("Validate() error = %v, want ErrInvalidToken", err)
	}
}

func TestTamperedSignature(t *testing.T) {
	svc := New("test-secret", time.Hour)

	token, err := svc.Generate("user-123", "test@example.com")
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	// Tamper with the signature (last part)
	parts := strings.Split(token, ".")
	parts[2] = "invalidsignature"
	tampered := strings.Join(parts, ".")

	_, err = svc.Validate(tampered)
	if err != ErrInvalidToken {
		t.Errorf("Validate() error = %v, want ErrInvalidToken", err)
	}
}

func TestMalformedToken(t *testing.T) {
	svc := New("test-secret", time.Hour)

	tests := []string{
		"",
		"one",
		"one.two",
		"one.two.three.four",
	}

	for _, tok := range tests {
		_, err := svc.Validate(tok)
		if err != ErrInvalidToken {
			t.Errorf("Validate(%q) error = %v, want ErrInvalidToken", tok, err)
		}
	}
}

func TestHashAndComparePassword(t *testing.T) {
	hash, err := HashPassword("mypassword")
	if err != nil {
		t.Fatalf("HashPassword() error: %v", err)
	}

	if err := ComparePassword(hash, "mypassword"); err != nil {
		t.Errorf("ComparePassword() with correct password: %v", err)
	}
}

func TestWrongPassword(t *testing.T) {
	hash, err := HashPassword("mypassword")
	if err != nil {
		t.Fatalf("HashPassword() error: %v", err)
	}

	if err := ComparePassword(hash, "wrongpassword"); err == nil {
		t.Error("ComparePassword() with wrong password should return error")
	}
}

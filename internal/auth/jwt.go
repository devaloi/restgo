package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

// Claims holds the decoded JWT payload.
type Claims struct {
	UserID    string
	Email     string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// JWTService handles JWT token generation and validation.
type JWTService struct {
	secret []byte
	expiry time.Duration
}

// New creates a JWTService with the given secret and token expiry duration.
func New(secret string, expiry time.Duration) *JWTService {
	return &JWTService{
		secret: []byte(secret),
		expiry: expiry,
	}
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type jwtPayload struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Iat   int64  `json:"iat"`
	Exp   int64  `json:"exp"`
}

// Generate creates a signed JWT for the given user.
func (s *JWTService) Generate(userID, email string) (string, error) {
	header := jwtHeader{Alg: "HS256", Typ: "JWT"}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("marshaling header: %w", err)
	}

	now := time.Now().UTC()
	payload := jwtPayload{
		Sub:   userID,
		Email: email,
		Iat:   now.Unix(),
		Exp:   now.Add(s.expiry).Unix(),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshaling payload: %w", err)
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	sigInput := headerB64 + "." + payloadB64
	sig := sign([]byte(sigInput), s.secret)

	return sigInput + "." + sig, nil
}

// Validate parses and verifies a JWT token string.
func (s *JWTService) Validate(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	// Verify signature
	sigInput := parts[0] + "." + parts[1]
	expectedSig := sign([]byte(sigInput), s.secret)
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return nil, ErrInvalidToken
	}

	// Decode payload
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}

	var payload jwtPayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return nil, ErrInvalidToken
	}

	claims := &Claims{
		UserID:    payload.Sub,
		Email:     payload.Email,
		IssuedAt:  time.Unix(payload.Iat, 0).UTC(),
		ExpiresAt: time.Unix(payload.Exp, 0).UTC(),
	}

	if time.Now().UTC().After(claims.ExpiresAt) {
		return nil, ErrExpiredToken
	}

	return claims, nil
}

func sign(data, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(data)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// HashPassword returns a bcrypt hash of the password.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hashing password: %w", err)
	}
	return string(hash), nil
}

// ComparePassword checks a bcrypt hash against a plaintext password.
func ComparePassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

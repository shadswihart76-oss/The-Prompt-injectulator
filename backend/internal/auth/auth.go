// Package auth provides JWT-based authentication and privileged user authorization.
// Privileged (admin) users are identified by server-side configuration only;
// client-supplied claims are never trusted for privilege escalation.
package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Role constants.
const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

// Claims is the JWT payload we embed.
type Claims struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

// IsAdmin returns true when the role is admin.
func (c *Claims) IsAdmin() bool { return c.Role == RoleAdmin }

// TokenService handles JWT signing and parsing with an injected secret.
// All callers must obtain a TokenService from Config.TokenService() to ensure
// the secret has been validated at configuration load time.
type TokenService struct {
	secret []byte
}

// IssueToken creates a signed JWT for the given email and role.
func (ts *TokenService) IssueToken(email, role string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		Email: email,
		Role:  role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(ts.secret)
}

// ParseToken validates and parses a JWT string.
func (ts *TokenService) ParseToken(tokenStr string) (*Claims, error) {
	tok, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return ts.secret, nil
	})
	if err != nil {
		return nil, err
	}
	c, ok := tok.Claims.(*Claims)
	if !ok || !tok.Valid {
		return nil, errors.New("invalid token claims")
	}
	return c, nil
}

// FromRequest extracts and validates the JWT from an HTTP request.
func (ts *TokenService) FromRequest(r *http.Request) (*Claims, error) {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return nil, errors.New("missing or malformed Authorization header")
	}
	return ts.ParseToken(strings.TrimPrefix(h, "Bearer "))
}

// HashPassword bcrypt-hashes a plaintext password.
func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

// CheckPassword returns nil iff plaintext matches hash.
func CheckPassword(hash, plaintext string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintext))
}

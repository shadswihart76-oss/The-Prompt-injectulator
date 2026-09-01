// Package auth – in-memory user store (suitable for dev/test; swap for a real DB in production).
package auth

import (
	"errors"
	"strings"
	"sync"
)

// User represents an authenticated account.
type User struct {
	Email        string
	PasswordHash string
	Role         string
}

// Store is an in-memory user registry.
type Store struct {
	mu    sync.RWMutex
	users map[string]*User // keyed by lowercase email
}

// NewStore returns an empty Store.
func NewStore() *Store {
	return &Store{users: make(map[string]*User)}
}

// Register adds a new user. Returns an error if the email already exists.
func (s *Store) Register(email, passwordHash, role string) error {
	key := strings.ToLower(email)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.users[key]; exists {
		return errors.New("email already registered")
	}
	s.users[key] = &User{Email: key, PasswordHash: passwordHash, Role: role}
	return nil
}

// FindByEmail looks up a user by email (case-insensitive).
func (s *Store) FindByEmail(email string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[strings.ToLower(email)]
	return u, ok
}

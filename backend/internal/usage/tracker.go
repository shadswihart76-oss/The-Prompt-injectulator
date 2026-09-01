// Package usage tracks per-user token consumption and enforces spending limits.
package usage

import (
	"errors"
	"fmt"
	"sync"
)

// ErrLimitReached is returned when a user has exhausted their token budget.
var ErrLimitReached = errors.New("usage limit reached: request would exceed your token budget")

// DefaultUserLimit is the default per-user token budget.
// Set USER_TOKEN_LIMIT in the environment to override.
const DefaultUserLimit = 10_000

// Tracker maintains per-user usage counts.
type Tracker struct {
	mu     sync.Mutex
	counts map[string]int
	limit  int
}

// NewTracker creates a Tracker with the given per-user token limit.
func NewTracker(limit int) *Tracker {
	if limit <= 0 {
		limit = DefaultUserLimit
	}
	return &Tracker{
		counts: make(map[string]int),
		limit:  limit,
	}
}

// Check returns an error if adding delta tokens would exceed the user's budget.
// Call before consuming tokens.
func (t *Tracker) Check(email string, delta int) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	current := t.counts[email]
	if current+delta >= t.limit {
		return fmt.Errorf("%w (used %d / %d, requested %d)", ErrLimitReached, current, t.limit, delta)
	}
	return nil
}

// Record adds delta tokens to the user's running total.
// Call after a successful LLM response.
func (t *Tracker) Record(email string, delta int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.counts[email] += delta
}

// Status returns the current usage and configured limit for a user.
func (t *Tracker) Status(email string) (used, limit int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.counts[email], t.limit
}

// Reset clears the usage counter for a user (admin action).
func (t *Tracker) Reset(email string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.counts, email)
}

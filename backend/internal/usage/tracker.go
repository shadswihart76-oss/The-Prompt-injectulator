// Package usage tracks per-user token consumption and enforces spending limits.
//
// NOTE: This implementation uses an in-memory store suitable for single-instance
// development and testing only. It is not suitable for production multi-instance
// deployment; replace with a distributed cache or database for production use.
package usage

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// ErrLimitReached is returned when a user has exhausted their token budget.
var ErrLimitReached = errors.New("usage limit reached: request would exceed your token budget")

// DefaultUserLimit is the default per-user token budget.
// Set USER_TOKEN_LIMIT in the environment to override.
const DefaultUserLimit = 10_000

// MaxTokensUpperBound is the maximum accepted value for max_tokens in a request.
// Requests specifying a higher value will be rejected.
const MaxTokensUpperBound = 32_768

// Tracker maintains per-user usage counts with atomic reserve/commit semantics
// to prevent concurrent requests from collectively exceeding the budget.
type Tracker struct {
	mu          sync.Mutex
	committed   map[string]int // finalized token usage
	reserved    map[string]int // tokens reserved but not yet committed
	limit       int
}

// NewTracker creates a Tracker with the given per-user token limit.
func NewTracker(limit int) *Tracker {
	if limit <= 0 {
		limit = DefaultUserLimit
	}
	return &Tracker{
		committed: make(map[string]int),
		reserved:  make(map[string]int),
		limit:     limit,
	}
}

// normalizeEmail returns a consistently lower-cased email key.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// Reserve atomically checks and reserves delta tokens for the user.
// Returns ErrLimitReached if committed + reserved + delta would exceed the limit.
// A successful Reserve must be followed by either Commit or Release.
func (t *Tracker) Reserve(email string, delta int) error {
	key := normalizeEmail(email)
	t.mu.Lock()
	defer t.mu.Unlock()
	total := t.committed[key] + t.reserved[key]
	if total+delta > t.limit {
		return fmt.Errorf("%w (used %d / %d, requested %d)", ErrLimitReached, total, t.limit, delta)
	}
	t.reserved[key] += delta
	return nil
}

// Commit moves delta tokens from the reservation to committed usage.
// actual may differ from the reserved delta (e.g. actual response tokens).
// The reservation for this call (delta) is fully released; actual is added to committed.
func (t *Tracker) Commit(email string, reserved, actual int) {
	key := normalizeEmail(email)
	t.mu.Lock()
	defer t.mu.Unlock()
	t.reserved[key] -= reserved
	if t.reserved[key] < 0 {
		t.reserved[key] = 0
	}
	t.committed[key] += actual
}

// Release cancels a reservation without charging committed usage (e.g. on provider error).
func (t *Tracker) Release(email string, reserved int) {
	key := normalizeEmail(email)
	t.mu.Lock()
	defer t.mu.Unlock()
	t.reserved[key] -= reserved
	if t.reserved[key] < 0 {
		t.reserved[key] = 0
	}
}

// Check returns an error if adding delta tokens would exceed the user's budget.
// Unlike Reserve it does not hold the quota; use Reserve/Commit for atomic enforcement.
func (t *Tracker) Check(email string, delta int) error {
	key := normalizeEmail(email)
	t.mu.Lock()
	defer t.mu.Unlock()
	total := t.committed[key] + t.reserved[key]
	if total+delta > t.limit {
		return fmt.Errorf("%w (used %d / %d, requested %d)", ErrLimitReached, total, t.limit, delta)
	}
	return nil
}

// Record adds delta tokens to the user's committed running total.
// Prefer Reserve/Commit for new callers; Record is kept for backwards compatibility.
func (t *Tracker) Record(email string, delta int) {
	key := normalizeEmail(email)
	t.mu.Lock()
	defer t.mu.Unlock()
	t.committed[key] += delta
}

// Status returns the current usage (committed + reserved) and configured limit for a user.
func (t *Tracker) Status(email string) (used, limit int) {
	key := normalizeEmail(email)
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.committed[key] + t.reserved[key], t.limit
}

// Reset clears the usage counter for a user (admin action).
func (t *Tracker) Reset(email string) {
	key := normalizeEmail(email)
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.committed, key)
	delete(t.reserved, key)
}

// ValidateMaxTokens returns an error if maxTokens is outside acceptable bounds.
// A value of 0 is allowed (callers should substitute a server default).
func ValidateMaxTokens(maxTokens int) error {
	if maxTokens < 0 {
		return fmt.Errorf("max_tokens must be non-negative (got %d)", maxTokens)
	}
	if maxTokens > MaxTokensUpperBound {
		return fmt.Errorf("max_tokens exceeds the server maximum of %d (got %d)", MaxTokensUpperBound, maxTokens)
	}
	return nil
}

// EstimateTokens returns a safe upper-bound estimate for a prompt + max_tokens request.
// Uses integer arithmetic that cannot overflow for inputs within validated bounds.
func EstimateTokens(promptLen, maxTokens int) int {
	if maxTokens <= 0 {
		maxTokens = 256 // conservative server default
	}
	// ~1 token per 4 chars, plus the response cap, plus overhead.
	est := promptLen/4 + maxTokens + 64
	if est <= 0 {
		return 64
	}
	return est
}

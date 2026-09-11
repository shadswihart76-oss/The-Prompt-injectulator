package usage_test

import (
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/shadswihart76-oss/the-prompt-injectulator/internal/usage"
)

// TestTrackerCheckExactLimit verifies that a request exactly at the limit is permitted.
func TestTrackerCheckExactLimit(t *testing.T) {
	tr := usage.NewTracker(100)
	tr.Record("alice@example.com", 50)

	// 50 used + 50 requested = 100 == limit → should be permitted.
	if err := tr.Check("alice@example.com", 50); err != nil {
		t.Errorf("exact-limit request should pass: %v", err)
	}
}

// TestTrackerCheckOverLimit verifies that a request exceeding the limit is denied.
func TestTrackerCheckOverLimit(t *testing.T) {
	tr := usage.NewTracker(100)
	tr.Record("alice@example.com", 50)

	// 50 used + 51 requested = 101 > limit → must be denied.
	if err := tr.Check("alice@example.com", 51); err == nil {
		t.Error("over-limit request should return error")
	}
}

// TestTrackerReserveCommit validates the atomic reserve/commit flow.
func TestTrackerReserveCommit(t *testing.T) {
	tr := usage.NewTracker(100)

	if err := tr.Reserve("bob@example.com", 60); err != nil {
		t.Fatalf("Reserve should succeed: %v", err)
	}

	// While 60 are reserved, a further 41 should be rejected (60+41=101 > 100).
	if err := tr.Reserve("bob@example.com", 41); err == nil {
		t.Error("second reserve exceeding limit should fail")
	}

	// Commit with actual = 55 tokens; reservation of 60 is released.
	tr.Commit("bob@example.com", 60, 55)

	used, limit := tr.Status("bob@example.com")
	if used != 55 || limit != 100 {
		t.Errorf("after commit: want used=55 limit=100, got used=%d limit=%d", used, limit)
	}
}

// TestTrackerReserveRelease validates that releasing a failed reservation frees quota.
func TestTrackerReserveRelease(t *testing.T) {
	tr := usage.NewTracker(100)

	if err := tr.Reserve("carol@example.com", 80); err != nil {
		t.Fatalf("Reserve should succeed: %v", err)
	}

	// Release without committing (provider call failed).
	tr.Release("carol@example.com", 80)

	// Full budget should be available again.
	if err := tr.Reserve("carol@example.com", 100); err != nil {
		t.Errorf("after release, full budget should be available: %v", err)
	}
}

// TestTrackerReserveConcurrencyBudget ensures concurrent requests cannot collectively
// exceed the total budget.
func TestTrackerReserveConcurrencyBudget(t *testing.T) {
	const limit = 100
	const perRequest = 10
	const goroutines = 20 // 20 × 10 = 200 > 100; only 10 should succeed

	tr := usage.NewTracker(limit)
	var successes atomic.Int64

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := tr.Reserve("shared@example.com", perRequest); err == nil {
				successes.Add(1)
				// Immediately commit so we don't leave reservations dangling.
				tr.Commit("shared@example.com", perRequest, perRequest)
			}
		}()
	}
	wg.Wait()

	got := int(successes.Load())
	if got > limit/perRequest {
		t.Errorf("too many requests succeeded: %d (max %d)", got, limit/perRequest)
	}
}

// TestTrackerReset verifies that reset clears committed and reserved usage.
func TestTrackerReset(t *testing.T) {
	tr := usage.NewTracker(100)
	tr.Record("bob@example.com", 90)
	tr.Reset("bob@example.com")
	if err := tr.Check("bob@example.com", 99); err != nil {
		t.Errorf("after reset, nearly full limit should be available: %v", err)
	}
}

// TestTrackerStatus verifies Status includes committed usage.
func TestTrackerStatus(t *testing.T) {
	tr := usage.NewTracker(200)
	tr.Record("carol@example.com", 75)
	used, limit := tr.Status("carol@example.com")
	if used != 75 || limit != 200 {
		t.Errorf("status: want used=75 limit=200, got used=%d limit=%d", used, limit)
	}
}

// TestErrLimitReachedMessage verifies the error message content.
func TestErrLimitReachedMessage(t *testing.T) {
	tr := usage.NewTracker(10)
	tr.Record("dave@example.com", 10)
	err := tr.Check("dave@example.com", 1)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "usage limit reached") {
		t.Errorf("error message should mention limit: %s", err.Error())
	}
}

// TestValidateMaxTokens verifies bounds checking on max_tokens.
func TestValidateMaxTokens(t *testing.T) {
	if err := usage.ValidateMaxTokens(0); err != nil {
		t.Errorf("0 should be valid (means use default): %v", err)
	}
	if err := usage.ValidateMaxTokens(1024); err != nil {
		t.Errorf("1024 should be valid: %v", err)
	}
	if err := usage.ValidateMaxTokens(-1); err == nil {
		t.Error("negative max_tokens should be rejected")
	}
	if err := usage.ValidateMaxTokens(usage.MaxTokensUpperBound + 1); err == nil {
		t.Error("value exceeding upper bound should be rejected")
	}
	if err := usage.ValidateMaxTokens(usage.MaxTokensUpperBound); err != nil {
		t.Errorf("exact upper bound should be valid: %v", err)
	}
}

// TestEstimateTokens verifies safe estimation.
func TestEstimateTokens(t *testing.T) {
	est := usage.EstimateTokens(400, 256) // 100 + 256 + 64 = 420
	if est <= 0 {
		t.Error("estimate should be positive")
	}
	// Zero max_tokens uses default (256).
	est0 := usage.EstimateTokens(0, 0)
	if est0 <= 0 {
		t.Error("estimate with zero inputs should be positive")
	}
}

// TestEmailNormalization verifies that different casings of the same email
// share the same quota bucket.
func TestEmailNormalization(t *testing.T) {
	tr := usage.NewTracker(100)
	tr.Record("User@Example.COM", 40)
	tr.Record("user@example.com", 40)
	used, _ := tr.Status("USER@EXAMPLE.COM")
	if used != 80 {
		t.Errorf("email normalization: want used=80, got %d", used)
	}
}

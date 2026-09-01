package usage_test

import (
	"strings"
	"testing"

	"github.com/shadswihart76-oss/the-prompt-injectulator/internal/usage"
)

func TestTrackerCheck(t *testing.T) {
	tr := usage.NewTracker(100)

	if err := tr.Check("alice@example.com", 50); err != nil {
		t.Fatalf("first check should pass: %v", err)
	}
	tr.Record("alice@example.com", 50)

	if err := tr.Check("alice@example.com", 51); err == nil {
		t.Error("check that would exceed limit should return error")
	}
	if err := tr.Check("alice@example.com", 50); err == nil {
		t.Error("check at exact limit should return error (50+50=100 == limit)")
	}
	if err := tr.Check("alice@example.com", 49); err != nil {
		t.Errorf("check within limit should pass: %v", err)
	}
}

func TestTrackerReset(t *testing.T) {
	tr := usage.NewTracker(100)
	tr.Record("bob@example.com", 90)
	tr.Reset("bob@example.com")
	if err := tr.Check("bob@example.com", 99); err != nil {
		t.Errorf("after reset, nearly full limit should be available: %v", err)
	}
}

func TestTrackerStatus(t *testing.T) {
	tr := usage.NewTracker(200)
	tr.Record("carol@example.com", 75)
	used, limit := tr.Status("carol@example.com")
	if used != 75 || limit != 200 {
		t.Errorf("status: want used=75 limit=200, got used=%d limit=%d", used, limit)
	}
}

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

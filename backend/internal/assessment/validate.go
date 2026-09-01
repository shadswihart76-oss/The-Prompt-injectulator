package assessment

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// Validate checks that r has the required fields and valid enum values.
func Validate(r Request) error {
	trimmed := strings.TrimSpace(r.Objective)
	if trimmed == "" {
		return errors.New("objective is required")
	}
	if utf8.RuneCountInString(trimmed) < 10 {
		return errors.New("objective must be at least 10 characters")
	}

	validSurfaces := map[string]bool{"chat": true, "rag": true, "agent": true}
	if !validSurfaces[r.Surface] {
		return errors.New("surface must be one of: chat, rag, agent")
	}

	validRisks := map[string]bool{
		"instruction-override":  true,
		"data-exposure":         true,
		"unsafe-tool-actions":   true,
		"cross-tenant-isolation": true,
	}
	if !validRisks[r.RiskFocus] {
		return errors.New("riskFocus must be one of: instruction-override, data-exposure, unsafe-tool-actions, cross-tenant-isolation")
	}

	return nil
}

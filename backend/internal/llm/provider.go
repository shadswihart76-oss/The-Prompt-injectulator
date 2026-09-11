// Package llm defines the provider interface and concrete implementations.
package llm

import (
	"context"
	"fmt"
	"strings"
)

// Provider names recognised by the API.
const (
	ProviderMock    = "mock"
	ProviderOpenAI  = "openai"
	// Add further real providers as required.
)

// Request is the normalised input to any LLM provider.
type Request struct {
	Prompt string `json:"prompt"`
	// MaxTokens caps the response length.  0 = provider default.
	MaxTokens int `json:"max_tokens,omitempty"`
}

// Response is the normalised output from any LLM provider.
type Response struct {
	Provider string `json:"provider"`
	Text     string `json:"text"`
	// TokensUsed is the number of tokens consumed (real or simulated).
	TokensUsed int `json:"tokens_used"`
}

// Provider is the interface implemented by all LLM backends.
type Provider interface {
	Name() string
	Complete(ctx context.Context, req Request) (*Response, error)
}

// -----------------------------------------------------------------------
// Mock provider – deterministic, zero-cost, admin-only.
// -----------------------------------------------------------------------

// MockProvider generates synthetic responses without calling any external API.
type MockProvider struct{}

// Name implements Provider.
func (m *MockProvider) Name() string { return ProviderMock }

// Complete implements Provider with deterministic responses based on prompt content.
func (m *MockProvider) Complete(_ context.Context, req Request) (*Response, error) {
	lower := strings.ToLower(req.Prompt)
	var text string
	switch {
	case strings.Contains(lower, "inject"):
		text = "[MOCK] Detected injection-related prompt. Synthetic analysis: the prompt attempts to override system instructions via role-switching. Suggested defence: strict input validation and output filtering."
	case strings.Contains(lower, "jailbreak"):
		text = "[MOCK] Jailbreak pattern recognised. Synthetic analysis: the prompt uses hypothetical framing to bypass content policy. Suggested defence: context-invariant safety classifiers."
	case strings.Contains(lower, "ignore"):
		text = "[MOCK] 'Ignore previous instructions' pattern detected. Synthetic defence: treat all user turns as untrusted; do not allow them to alter system-level directives."
	default:
		text = fmt.Sprintf("[MOCK] Processed prompt (%d chars). No known attack patterns detected in this synthetic run.", len(req.Prompt))
	}
	return &Response{
		Provider:   ProviderMock,
		Text:       text,
		TokensUsed: len(strings.Fields(req.Prompt)) + 10, // deterministic estimate
	}, nil
}

// -----------------------------------------------------------------------
// OpenAI provider stub – real calls not implemented here.
// Replace with a real HTTP client as needed.
// -----------------------------------------------------------------------

// OpenAIProvider is a stub for a real OpenAI backend.
type OpenAIProvider struct {
	APIKey string
}

// Name implements Provider.
func (o *OpenAIProvider) Name() string { return ProviderOpenAI }

// Complete implements Provider (stub – returns an error until wired up).
func (o *OpenAIProvider) Complete(_ context.Context, req Request) (*Response, error) {
	if o.APIKey == "" {
		return nil, fmt.Errorf("openai: OPENAI_API_KEY not configured")
	}
	// TODO: implement real OpenAI HTTP call.
	return nil, fmt.Errorf("openai: real provider not yet implemented; use mock provider for testing")
}

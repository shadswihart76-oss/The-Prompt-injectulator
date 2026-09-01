package assessment

import (
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		req     Request
		wantErr bool
	}{
		{
			name:    "valid chat instruction-override",
			req:     Request{Objective: "Test if the chat assistant can be overridden", Surface: "chat", RiskFocus: "instruction-override"},
			wantErr: false,
		},
		{
			name:    "valid rag data-exposure",
			req:     Request{Objective: "Check for data leakage in RAG pipeline", Surface: "rag", RiskFocus: "data-exposure"},
			wantErr: false,
		},
		{
			name:    "empty objective",
			req:     Request{Objective: "", Surface: "chat", RiskFocus: "instruction-override"},
			wantErr: true,
		},
		{
			name:    "objective too short",
			req:     Request{Objective: "test", Surface: "chat", RiskFocus: "instruction-override"},
			wantErr: true,
		},
		{
			name:    "invalid surface",
			req:     Request{Objective: "Test if the chat assistant can be overridden", Surface: "unknown", RiskFocus: "instruction-override"},
			wantErr: true,
		},
		{
			name:    "invalid risk focus",
			req:     Request{Objective: "Test if the chat assistant can be overridden", Surface: "chat", RiskFocus: "invalid-risk"},
			wantErr: true,
		},
		{
			name:    "whitespace-only objective",
			req:     Request{Objective: "   ", Surface: "chat", RiskFocus: "instruction-override"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeterministicGenerator(t *testing.T) {
	g := NewDeterministicGenerator()

	t.Run("returns consistent ID for same input", func(t *testing.T) {
		r := Request{Objective: "Test if the chat assistant can be overridden", Surface: "chat", RiskFocus: "instruction-override"}
		a1, _ := g.Generate(r)
		a2, _ := g.Generate(r)
		if a1.ID != a2.ID {
			t.Errorf("expected same ID, got %q and %q", a1.ID, a2.ID)
		}
	})

	t.Run("instruction-override chat returns direct scenarios", func(t *testing.T) {
		r := Request{Objective: "Test if the chat assistant can be overridden", Surface: "chat", RiskFocus: "instruction-override"}
		a, err := g.Generate(r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(a.Scenarios) == 0 {
			t.Fatal("expected scenarios, got none")
		}
		for _, s := range a.Scenarios {
			if s.Type != "direct" && s.Type != "indirect" {
				t.Errorf("unexpected scenario type %q", s.Type)
			}
		}
	})

	t.Run("rag surface includes indirect scenarios", func(t *testing.T) {
		r := Request{Objective: "Test if the rag pipeline treats documents as instructions", Surface: "rag", RiskFocus: "instruction-override"}
		a, _ := g.Generate(r)
		hasIndirect := false
		for _, s := range a.Scenarios {
			if s.Type == "indirect" {
				hasIndirect = true
				break
			}
		}
		if !hasIndirect {
			t.Error("expected at least one indirect scenario for rag surface")
		}
	})

	t.Run("findings are populated", func(t *testing.T) {
		r := Request{Objective: "Test if the chat assistant can be overridden", Surface: "chat", RiskFocus: "instruction-override"}
		a, _ := g.Generate(r)
		if len(a.Findings) == 0 {
			t.Error("expected findings, got none")
		}
	})

	t.Run("authorization notice is present", func(t *testing.T) {
		r := Request{Objective: "Test if the chat assistant can be overridden", Surface: "chat", RiskFocus: "instruction-override"}
		a, _ := g.Generate(r)
		if a.AuthorizationNotice == "" {
			t.Error("expected authorization notice")
		}
	})

	t.Run("normalizes objective whitespace", func(t *testing.T) {
		r := Request{Objective: "  test   the  system  ", Surface: "chat", RiskFocus: "data-exposure"}
		a, _ := g.Generate(r)
		if a.NormalizedObjective != "Test the system" {
			t.Errorf("expected 'Test the system', got %q", a.NormalizedObjective)
		}
	})

	t.Run("default environment when empty", func(t *testing.T) {
		r := Request{Objective: "Test the environment default behavior", Surface: "agent", RiskFocus: "unsafe-tool-actions"}
		a, _ := g.Generate(r)
		if a.Environment != "unspecified" {
			t.Errorf("expected 'unspecified', got %q", a.Environment)
		}
	})
}

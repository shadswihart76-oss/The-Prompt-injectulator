package assessment

// Surface represents the target system surface type.
type Surface string

const (
	SurfaceChat  Surface = "chat"
	SurfaceRAG   Surface = "rag"
	SurfaceAgent Surface = "agent"
)

// RiskFocus represents the category of risk being assessed.
type RiskFocus string

const (
	RiskInstructionOverride  RiskFocus = "instruction-override"
	RiskDataExposure         RiskFocus = "data-exposure"
	RiskUnsafeToolActions    RiskFocus = "unsafe-tool-actions"
	RiskContextIsolation     RiskFocus = "cross-tenant-isolation"
)

// Request is the incoming payload for POST /api/assessments.
type Request struct {
	Objective   string `json:"objective"`
	Surface     string `json:"surface"`
	RiskFocus   string `json:"riskFocus"`
	Environment string `json:"environment"`
}

// Scenario describes one test scenario.
type Scenario struct {
	ID               string `json:"id"`
	Type             string `json:"type"`     // "direct" or "indirect"
	Intent           string `json:"intent"`
	Fixture          string `json:"fixture"`
	ExpectedBehavior string `json:"expectedBehavior"`
	Evidence         string `json:"evidence"`
	Severity         string `json:"severity"` // "critical", "high", "medium", "low"
}

// Finding describes a vulnerability class.
type Finding struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	OWASP       string `json:"owasp"`
}

// Assessment is the full response returned by the generator.
type Assessment struct {
	ID                  string     `json:"id"`
	NormalizedObjective string     `json:"normalizedObjective"`
	ThreatModelSummary  string     `json:"threatModelSummary"`
	Surface             string     `json:"surface"`
	RiskFocus           string     `json:"riskFocus"`
	Environment         string     `json:"environment"`
	Scenarios           []Scenario `json:"scenarios"`
	Findings            []Finding  `json:"findings"`
	AuthorizationNotice string     `json:"authorizationNotice"`
}

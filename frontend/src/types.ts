export type Surface = 'chat' | 'rag' | 'agent';
export type RiskFocus =
  | 'instruction-override'
  | 'data-exposure'
  | 'unsafe-tool-actions'
  | 'cross-tenant-isolation';

export interface AssessmentRequest {
  objective: string;
  surface: Surface;
  riskFocus: RiskFocus;
  environment?: string;
}

export interface Scenario {
  id: string;
  type: 'direct' | 'indirect';
  intent: string;
  fixture: string;
  expectedBehavior: string;
  evidence: string;
  severity: 'critical' | 'high' | 'medium' | 'low';
}

export interface Finding {
  name: string;
  description: string;
  owasp: string;
}

export interface Assessment {
  id: string;
  normalizedObjective: string;
  threatModelSummary: string;
  surface: string;
  riskFocus: string;
  environment: string;
  scenarios: Scenario[];
  findings: Finding[];
  authorizationNotice: string;
}

export interface ApiError {
  error: string;
}

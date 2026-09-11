import { useState } from 'react';
import type { Assessment, AssessmentRequest, RiskFocus, Surface } from './types';
import { generateAssessment } from './api';
import './App.css';

const EXAMPLE_OBJECTIVES = [
  'Test whether a chat assistant follows instructions embedded in imported documents.',
  'Verify the RAG pipeline treats retrieved content as data, not commands.',
  'Check if the agent requires confirmation before executing high-impact tool calls.',
  'Ensure cross-tenant context isolation in a shared knowledge assistant.',
];

const SURFACE_LABELS: Record<Surface, string> = {
  chat: 'Chat Assistant',
  rag: 'RAG / Document Ingestion',
  agent: 'Tool-Using Agent',
};

const RISK_LABELS: Record<RiskFocus, string> = {
  'instruction-override': 'Instruction Override',
  'data-exposure': 'Data Exposure',
  'unsafe-tool-actions': 'Unsafe Tool Actions',
  'cross-tenant-isolation': 'Cross-Tenant Isolation',
};

const SEVERITY_COLOR: Record<string, string> = {
  critical: '#ef4444',
  high: '#f97316',
  medium: '#eab308',
  low: '#22c55e',
};

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);
  const handleCopy = async () => {
    await navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };
  return (
    <button className="copy-btn" onClick={handleCopy} title="Copy fixture to clipboard">
      {copied ? '✓ Copied' : '⎘ Copy'}
    </button>
  );
}

function ScenarioCard({ scenario, index }: { scenario: Assessment['scenarios'][0]; index: number }) {
  const [open, setOpen] = useState(false);
  return (
    <div className="scenario-card">
      <div className="scenario-header" onClick={() => setOpen(!open)} role="button" tabIndex={0}
        onKeyDown={e => e.key === 'Enter' && setOpen(!open)}>
        <span className="scenario-index">#{index + 1}</span>
        <span className={`badge badge-type badge-${scenario.type}`}>{scenario.type}</span>
        <span className="scenario-intent">{scenario.intent}</span>
        <span className="badge badge-severity" style={{ borderColor: SEVERITY_COLOR[scenario.severity], color: SEVERITY_COLOR[scenario.severity] }}>
          {scenario.severity}
        </span>
        <span className="chevron">{open ? '▲' : '▼'}</span>
      </div>
      {open && (
        <div className="scenario-body">
          <div className="fixture-block">
            <div className="fixture-label">Test Fixture / Injection Artifact</div>
            <pre className="fixture-pre">{scenario.fixture}</pre>
            <CopyButton text={scenario.fixture} />
          </div>
          <div className="detail-grid">
            <div>
              <div className="detail-label">Expected Secure Behavior</div>
              <div className="detail-value">{scenario.expectedBehavior}</div>
            </div>
            <div>
              <div className="detail-label">Evidence to Capture</div>
              <div className="detail-value">{scenario.evidence}</div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function AssessmentView({ assessment, onReset }: { assessment: Assessment; onReset: () => void }) {
  return (
    <div className="assessment-view">
      <div className="notice-banner">
        <span className="notice-icon">⚠</span>
        <span>{assessment.authorizationNotice}</span>
      </div>

      <div className="result-section">
        <div className="result-label">Normalized Objective</div>
        <div className="result-objective">{assessment.normalizedObjective}</div>
      </div>

      <div className="result-section">
        <div className="result-label">Threat Model Summary</div>
        <div className="result-threat">{assessment.threatModelSummary}</div>
      </div>

      <div className="result-section">
        <div className="result-label">Test Scenarios ({assessment.scenarios.length})</div>
        <div className="scenarios-list">
          {assessment.scenarios.map((s, i) => (
            <ScenarioCard key={s.id} scenario={s} index={i} />
          ))}
        </div>
      </div>

      <div className="result-section">
        <div className="result-label">Vulnerability / Finding Map</div>
        <div className="findings-grid">
          {assessment.findings.map((f, i) => (
            <div key={i} className="finding-card">
              <div className="finding-name">{f.name}</div>
              <div className="finding-desc">{f.description}</div>
              <div className="finding-owasp">{f.owasp}</div>
            </div>
          ))}
        </div>
      </div>

      <div className="result-meta">
        Assessment ID: <code>{assessment.id}</code> &nbsp;·&nbsp;
        Surface: <strong>{SURFACE_LABELS[assessment.surface as Surface] ?? assessment.surface}</strong> &nbsp;·&nbsp;
        Environment: <strong>{assessment.environment}</strong>
      </div>

      <button className="btn btn-secondary reset-btn" onClick={onReset}>
        ＋ New Assessment
      </button>
    </div>
  );
}

function Composer({
  onResult,
}: {
  onResult: (a: Assessment) => void;
}) {
  const [objective, setObjective] = useState('');
  const [surface, setSurface] = useState<Surface>('chat');
  const [riskFocus, setRiskFocus] = useState<RiskFocus>('instruction-override');
  const [environment, setEnvironment] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const req: AssessmentRequest = { objective, surface, riskFocus, environment };
      const result = await generateAssessment(req);
      onResult(result);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Unexpected error');
    } finally {
      setLoading(false);
    }
  };

  return (
    <form className="composer" onSubmit={handleSubmit}>
      <div className="composer-field">
        <label htmlFor="objective">Test Objective</label>
        <textarea
          id="objective"
          className="composer-textarea"
          rows={3}
          placeholder="Describe what you want to test, e.g. 'Verify the RAG pipeline treats retrieved content as data only.'"
          value={objective}
          onChange={e => setObjective(e.target.value)}
          required
          minLength={10}
        />
        <div className="examples-row">
          {EXAMPLE_OBJECTIVES.map((ex, i) => (
            <button key={i} type="button" className="example-chip" onClick={() => setObjective(ex)}>
              {ex.length > 60 ? ex.slice(0, 57) + '…' : ex}
            </button>
          ))}
        </div>
      </div>

      <div className="composer-row">
        <div className="composer-field">
          <label htmlFor="surface">Target Surface</label>
          <select id="surface" value={surface} onChange={e => setSurface(e.target.value as Surface)}>
            {(Object.entries(SURFACE_LABELS) as [Surface, string][]).map(([v, l]) => (
              <option key={v} value={v}>{l}</option>
            ))}
          </select>
        </div>
        <div className="composer-field">
          <label htmlFor="riskFocus">Risk Focus</label>
          <select id="riskFocus" value={riskFocus} onChange={e => setRiskFocus(e.target.value as RiskFocus)}>
            {(Object.entries(RISK_LABELS) as [RiskFocus, string][]).map(([v, l]) => (
              <option key={v} value={v}>{l}</option>
            ))}
          </select>
        </div>
        <div className="composer-field">
          <label htmlFor="environment">Environment <span className="optional">(optional)</span></label>
          <input
            id="environment"
            type="text"
            placeholder="e.g. staging"
            value={environment}
            onChange={e => setEnvironment(e.target.value)}
          />
        </div>
      </div>

      {error && <div className="error-msg">⚠ {error}</div>}

      <button className="btn btn-primary" type="submit" disabled={loading}>
        {loading ? (
          <><span className="spinner" aria-hidden="true" /> Generating…</>
        ) : (
          '→ Generate Assessment'
        )}
      </button>
    </form>
  );
}

type RecentEntry = { id: string; objective: string; surface: string };

export default function App() {
  const [assessment, setAssessment] = useState<Assessment | null>(null);
  const [recent, setRecent] = useState<RecentEntry[]>([]);

  const handleResult = (a: Assessment) => {
    setAssessment(a);
    setRecent(prev => [{ id: a.id, objective: a.normalizedObjective, surface: a.surface }, ...prev.slice(0, 4)]);
  };

  const handleReset = () => setAssessment(null);

  return (
    <div className="layout">
      {/* Left nav rail */}
      <aside className="sidebar">
        <div className="sidebar-brand">
          <div className="brand-icon">🔐</div>
          <div>
            <div className="brand-name">PromptBench</div>
            <div className="brand-sub">LLM Security Workbench</div>
          </div>
        </div>

        <button className="sidebar-new-btn" onClick={handleReset}>
          <span>＋</span> New Assessment
        </button>

        {recent.length > 0 && (
          <div className="sidebar-section">
            <div className="sidebar-section-label">Recent</div>
            {recent.map(r => (
              <div key={r.id} className="sidebar-item">
                <div className="sidebar-item-surface">{SURFACE_LABELS[r.surface as Surface] ?? r.surface}</div>
                <div className="sidebar-item-objective">{r.objective.length > 50 ? r.objective.slice(0, 47) + '…' : r.objective}</div>
              </div>
            ))}
          </div>
        )}

        <div className="sidebar-footer">
          <a href="https://owasp.org/www-project-top-10-for-large-language-model-applications/" target="_blank" rel="noopener noreferrer" className="sidebar-link">
            OWASP LLM Top 10 ↗
          </a>
        </div>
      </aside>

      {/* Main workspace */}
      <main className="workspace">
        <div className="workspace-inner">
          {assessment ? (
            <AssessmentView assessment={assessment} onReset={handleReset} />
          ) : (
            <>
              <div className="intro">
                <div className="intro-icon">🔐</div>
                <h1 className="intro-title">LLM Security Prompt Workbench</h1>
                <p className="intro-sub">
                  Transform a weak testing objective into a structured, reproducible test plan for authorized
                  LLM security assessments. Covers direct and indirect prompt-injection scenarios and maps them to
                  OWASP LLM vulnerability classes.
                </p>
                <p className="intro-notice">
                  For systems you own or have <strong>explicit written permission</strong> to test.
                </p>
              </div>
              <Composer onResult={handleResult} />
            </>
          )}
        </div>
      </main>
    </div>
  );
}

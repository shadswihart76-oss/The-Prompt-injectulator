import { useState } from 'react';
import type { Assessment, Surface } from './types';
import { SURFACE_LABELS } from './Composer';

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


export default AssessmentView;

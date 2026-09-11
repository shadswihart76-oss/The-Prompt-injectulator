import { useState } from 'react';
import type { Assessment, AssessmentRequest, RiskFocus, Surface } from './types';
import { generateAssessment } from './api';

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

export default function Composer({
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

export { SURFACE_LABELS };

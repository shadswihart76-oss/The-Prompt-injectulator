import { useEffect, useState } from 'react'
import { api, type AuthResponse, type AdminStatus } from './api/api'
import Composer from './Composer'
import AssessmentView from './AssessmentView'
import type { Assessment } from './types'
import type { CompleteResponse } from './api/api'
import './App.css'

type View = 'workbench' | 'admin'

function AdminPanel({
  auth,
  onBack,
}: {
  auth: AuthResponse
  onBack: () => void
}) {
  const [adminStatus, setAdminStatus] = useState<AdminStatus | null>(null)
  const [resetEmail, setResetEmail] = useState('')
  const [statusMsg, setStatusMsg] = useState('')
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api
      .adminStatus(auth.token)
      .then(setAdminStatus)
      .catch(e => setError(e instanceof Error ? e.message : String(e)))
  }, [auth.token])

  const handleResetUsage = async () => {
    setStatusMsg('')
    setError(null)
    try {
      const r = await api.resetUsage(resetEmail, auth.token)
      setStatusMsg(r.status)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  return (
    <div className="admin-view">
      <div className="admin-header">
        <h2 className="admin-title">Admin Panel</h2>
        <button className="btn btn-secondary" onClick={onBack}>
          ← Back to workbench
        </button>
      </div>

      {error && <div className="error-msg">⚠ {error}</div>}

      {adminStatus && (
        <>
          {adminStatus.dev_mode && (
            <div className="notice-banner">
              <span className="notice-icon">⚠</span>
              <span>DEV MODE is active — for local testing only. Do not expose to the internet.</span>
            </div>
          )}
          <div className="detail-grid">
            <div>
              <div className="detail-label">Account</div>
              <div className="detail-value">{adminStatus.email}</div>
            </div>
            <div>
              <div className="detail-label">Role</div>
              <div className="detail-value">{adminStatus.role}</div>
            </div>
            <div>
              <div className="detail-label">Mock provider</div>
              <div className="detail-value">
                {adminStatus.mock_enabled ? '✅ enabled' : '❌ disabled'}
              </div>
            </div>
            <div>
              <div className="detail-label">Cloud provider</div>
              <div className="detail-value">
                {(adminStatus as { cloud_enabled?: boolean }).cloud_enabled
                  ? '✅ enabled'
                  : '❌ disabled (CLOUD_API_ENABLED=false)'}
              </div>
            </div>
            <div>
              <div className="detail-label">Token usage</div>
              <div className="detail-value">
                {adminStatus.usage.used} / {adminStatus.usage.limit} tokens
              </div>
            </div>
          </div>

          <div className="result-section">
            <div className="result-label">Reset user budget</div>
            {statusMsg && <div className="success-msg">✓ {statusMsg}</div>}
            <div className="admin-reset-row">
              <input
                type="text"
                className="admin-reset-input"
                placeholder="user@example.com"
                value={resetEmail}
                onChange={e => setResetEmail(e.target.value)}
              />
              <button
                className="btn btn-secondary"
                onClick={handleResetUsage}
                disabled={!resetEmail}
              >
                Reset
              </button>
            </div>
          </div>
        </>
      )}
    </div>
  )
}

export default function App() {
  const [auth, setAuth] = useState<AuthResponse | null>(null)
  const [view, setView] = useState<View>('workbench')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [loginError, setLoginError] = useState<string | null>(null)
  const [loginLoading, setLoginLoading] = useState(false)
  const [assessment, setAssessment] = useState<Assessment | null>(null)
  const [lastComplete, setLastComplete] = useState<CompleteResponse | null>(null)

  const isAdmin = auth?.role === 'admin'

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoginError(null)
    setLoginLoading(true)
    try {
      const res = await api.login(email, password)
      setAuth(res)
    } catch (err: unknown) {
      setLoginError(err instanceof Error ? err.message : String(err))
    } finally {
      setLoginLoading(false)
    }
  }

  const handleLogout = () => {
    setAuth(null)
    setView('workbench')
    setAssessment(null)
    setLastComplete(null)
    setEmail('')
    setPassword('')
  }

  if (!auth) {
    return (
      <div className="login-layout">
        <form className="login-card" onSubmit={handleLogin}>
          <div className="login-brand">
            <div className="brand-icon">🔐</div>
            <div>
              <div className="brand-name">Prompt Injectulator</div>
              <div className="brand-sub">LLM Security Workbench</div>
            </div>
          </div>
          <p className="login-sub">
            Sign in to generate structured prompt-injection test plans for systems you
            own or are authorized to assess.
          </p>
          {loginError && <div className="error-msg">⚠ {loginError}</div>}
          <div className="composer-field">
            <label htmlFor="email">Email</label>
            <input
              id="email"
              type="email"
              className="composer-input"
              value={email}
              onChange={e => setEmail(e.target.value)}
              required
              autoFocus
            />
          </div>
          <div className="composer-field">
            <label htmlFor="password">Password</label>
            <input
              id="password"
              type="password"
              className="composer-input"
              value={password}
              onChange={e => setPassword(e.target.value)}
              required
            />
          </div>
          <button className="btn btn-primary login-btn" type="submit" disabled={loginLoading}>
            {loginLoading ? (
              <>
                <span className="spinner" aria-hidden="true" /> Signing in…
              </>
            ) : (
              '→ Sign in'
            )}
          </button>
          <p className="login-notice">
            Authorized use only. Test artifacts are for systems you own or have
            explicit written permission to assess.
          </p>
        </form>
      </div>
    )
  }

  if (view === 'admin') {
    return (
      <div className="layout">
        <aside className="sidebar">
          <div className="sidebar-brand">
            <div className="brand-icon">🔐</div>
            <div>
              <div className="brand-name">Prompt Injectulator</div>
              <div className="brand-sub">LLM Security Workbench</div>
            </div>
          </div>
          <button className="sidebar-new-btn" onClick={() => setView('workbench')}>
            <span>←</span> Back to Workbench
          </button>
          <div className="sidebar-section">
            <div className="sidebar-section-label">Session</div>
            <div className="sidebar-item">
              <div className="sidebar-item-surface">{auth.email}</div>
              <div className="sidebar-item-objective">role: {auth.role}</div>
            </div>
          </div>
          <div className="sidebar-footer">
            <button className="sidebar-link" onClick={handleLogout}>
              Sign out
            </button>
          </div>
        </aside>
        <main className="workspace">
          <div className="workspace-inner">
            <AdminPanel auth={auth} onBack={() => setView('workbench')} />
          </div>
        </main>
      </div>
    )
  }

  return (
    <div className="layout">
      {/* Left nav rail */}
      <aside className="sidebar">
        <div className="sidebar-brand">
          <div className="brand-icon">🔐</div>
          <div>
            <div className="brand-name">Prompt Injectulator</div>
            <div className="brand-sub">LLM Security Workbench</div>
          </div>
        </div>

        <button
          className="sidebar-new-btn"
          onClick={() => {
            setAssessment(null)
            setLastComplete(null)
          }}
        >
          <span>＋</span> New Assessment
        </button>

        {isAdmin && (
          <button className="sidebar-new-btn sidebar-admin-btn" onClick={() => setView('admin')}>
            <span>⚙</span> Admin Panel
          </button>
        )}

        <div className="sidebar-section">
          <div className="sidebar-section-label">Session</div>
          <div className="sidebar-item">
            <div className="sidebar-item-surface">{auth.email}</div>
            <div className="sidebar-item-objective">role: {auth.role}</div>
          </div>
          {lastComplete && (
            <div className="sidebar-item">
              <div className="sidebar-item-surface">
                Last run: {lastComplete.provider} · {lastComplete.tokens_used} tokens
              </div>
              <div className="sidebar-item-objective">
                {lastComplete.usage.used} / {lastComplete.usage.limit} tokens used
              </div>
            </div>
          )}
        </div>

        <div className="sidebar-footer">
          <a
            href="https://owasp.org/www-project-top-10-for-large-language-model-applications/"
            target="_blank"
            rel="noopener noreferrer"
            className="sidebar-link"
          >
            OWASP LLM Top 10 ↗
          </a>
          <button className="sidebar-link" onClick={handleLogout}>
            Sign out
          </button>
        </div>
      </aside>

      {/* Main workspace */}
      <main className="workspace">
        <div className="workspace-inner">
          {assessment ? (
            <AssessmentView assessment={assessment} onReset={() => setAssessment(null)} />
          ) : (
            <>
              <div className="intro">
                <div className="intro-icon">🔐</div>
                <h1 className="intro-title">LLM Security Prompt Workbench</h1>
                <p className="intro-sub">
                  Transform a weak testing objective into a structured, reproducible test
                  plan for authorized LLM security assessments. Covers direct and indirect
                  prompt-injection scenarios and maps them to OWASP LLM vulnerability
                  classes.
                </p>
                <p className="intro-notice">
                  For systems you own or have <strong>explicit written permission</strong>{' '}
                  to test.
                </p>
              </div>
              <Composer onResult={setAssessment} />
            </>
          )}
        </div>
      </main>
    </div>
  )
}
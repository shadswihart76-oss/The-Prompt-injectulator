import { useState } from 'react'
import { api, type AuthResponse, type CompleteResponse, type AdminStatus } from './api/api'

type View = 'login' | 'app' | 'admin'

function App() {
  const [view, setView] = useState<View>('login')
  const [auth, setAuth] = useState<AuthResponse | null>(null)
  const [error, setError] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [prompt, setPrompt] = useState('')
  const [provider, setProvider] = useState('mock')
  const [result, setResult] = useState<CompleteResponse | null>(null)
  const [adminStatus, setAdminStatus] = useState<AdminStatus | null>(null)
  const [resetEmail, setResetEmail] = useState('')
  const [statusMsg, setStatusMsg] = useState('')

  const isAdmin = auth?.role === 'admin'

  const handleLogin = async () => {
    setError('')
    try {
      const res = await api.login(email, password)
      setAuth(res)
      setView('app')
      if (res.role === 'admin') setProvider('mock')
      else setProvider('openai')
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  const handleSubmit = async () => {
    if (!auth) return
    setError('')
    setResult(null)
    try {
      const res = await api.complete(provider, prompt, auth.token)
      setResult(res)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  const loadAdminStatus = async () => {
    if (!auth) return
    try {
      const s = await api.adminStatus(auth.token)
      setAdminStatus(s)
      setView('admin')
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  const handleResetUsage = async () => {
    if (!auth) return
    setStatusMsg('')
    try {
      const r = await api.resetUsage(resetEmail, auth.token)
      setStatusMsg(r.status)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  if (view === 'login') {
    return (
      <div style={{ maxWidth: 400, margin: '80px auto', fontFamily: 'sans-serif' }}>
        <h1>Prompt Injectulator</h1>
        <p style={{ color: '#666' }}>Security prompt testing tool</p>
        {error && <p style={{ color: 'red' }}>{error}</p>}
        <div>
          <label>Email<br />
            <input value={email} onChange={e => setEmail(e.target.value)} type="email" style={{ width: '100%', marginBottom: 8 }} />
          </label>
        </div>
        <div>
          <label>Password<br />
            <input value={password} onChange={e => setPassword(e.target.value)} type="password" style={{ width: '100%', marginBottom: 8 }} />
          </label>
        </div>
        <button onClick={handleLogin} style={{ marginRight: 8 }}>Login</button>
      </div>
    )
  }

  if (view === 'admin' && adminStatus) {
    return (
      <div style={{ maxWidth: 600, margin: '40px auto', fontFamily: 'sans-serif' }}>
        <h2>Admin Panel</h2>
        {adminStatus.dev_mode && (
          <p style={{ background: '#fff3cd', padding: 8, border: '1px solid #ffc107' }}>
            ⚠️ DEV MODE is active – for local testing only.
          </p>
        )}
        <ul>
          <li>Email: {adminStatus.email}</li>
          <li>Role: {adminStatus.role}</li>
          <li>Mock provider: {adminStatus.mock_enabled ? '✅ enabled' : '❌ disabled'}</li>
          <li>Usage: {adminStatus.usage.used} / {adminStatus.usage.limit} tokens</li>
        </ul>
        <h3>Reset user budget</h3>
        {statusMsg && <p style={{ color: 'green' }}>{statusMsg}</p>}
        {error && <p style={{ color: 'red' }}>{error}</p>}
        <input placeholder="user@example.com" value={resetEmail} onChange={e => setResetEmail(e.target.value)} style={{ marginRight: 8 }} />
        <button onClick={handleResetUsage}>Reset</button>
        <br /><br />
        <button onClick={() => setView('app')}>← Back to app</button>
      </div>
    )
  }

  return (
    <div style={{ maxWidth: 700, margin: '40px auto', fontFamily: 'sans-serif' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h1>Prompt Injectulator</h1>
        <div>
          <span style={{ marginRight: 12, fontSize: 14, color: '#666' }}>{auth?.email} ({auth?.role})</span>
          {isAdmin && <button onClick={loadAdminStatus} style={{ marginRight: 8 }}>⚙️ Admin</button>}
          <button onClick={() => { setAuth(null); setView('login') }}>Logout</button>
        </div>
      </div>

      {error && <p style={{ color: 'red', background: '#fee', padding: 8 }}>{error}</p>}

      <div style={{ marginBottom: 12 }}>
        <label>Provider: </label>
        <select value={provider} onChange={e => setProvider(e.target.value)}>
          {isAdmin && <option value="mock">Mock (no cost)</option>}
          <option value="openai">OpenAI (paid)</option>
        </select>
        {provider !== 'mock' && (
          <span style={{ marginLeft: 12, color: '#c60', fontWeight: 'bold' }}>
            ⚠️ Real paid API active
          </span>
        )}
        {provider === 'mock' && (
          <span style={{ marginLeft: 12, color: '#090' }}>✅ Mock mode – no cost</span>
        )}
      </div>

      <textarea
        value={prompt}
        onChange={e => setPrompt(e.target.value)}
        rows={6}
        style={{ width: '100%', fontFamily: 'monospace' }}
        placeholder="Enter your security test prompt here..."
      />
      <br />
      <button onClick={handleSubmit} disabled={!prompt}>Submit</button>

      {result && (
        <div style={{ marginTop: 20, background: '#f5f5f5', padding: 12 }}>
          <strong>Provider:</strong> {result.provider} &nbsp;
          <strong>Tokens:</strong> {result.tokens_used} &nbsp;
          <span style={{ fontSize: 12, color: '#666' }}>Mode: {result.usage.mode}</span>
          <pre style={{ whiteSpace: 'pre-wrap', marginTop: 8 }}>{result.text}</pre>
          <small>Budget: {result.usage.used} / {result.usage.limit} tokens used</small>
        </div>
      )}
    </div>
  )
}

export default App

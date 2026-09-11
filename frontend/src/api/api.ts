// api.ts – typed wrappers around the backend REST API.

const BASE = '/api'

export interface AuthResponse {
  token: string
  role: string
  email: string
}

export interface UsageInfo {
  used: number
  limit: number
  remaining: number
}

export interface CompleteResponse {
  provider: string
  text: string
  tokens_used: number
  usage: {
    used: number
    limit: number
    mode: string
  }
}

export interface AdminStatus {
  dev_mode: boolean
  email: string
  role: string
  mock_enabled: boolean
  usage: { used: number; limit: number }
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  token?: string
): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (token) headers['Authorization'] = 'Bearer ' + token
  const res = await fetch(BASE + path, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
  const data = await res.json()
  if (!res.ok) {
    throw new Error((data as { error?: string }).error ?? 'Request failed (' + res.status + ')')
  }
  return data as T
}

export const api = {
  login: (email: string, password: string) =>
    request<AuthResponse>('POST', '/auth/login', { email, password }),

  register: (email: string, password: string) =>
    request<AuthResponse>('POST', '/auth/register', { email, password }),

  complete: (provider: string, prompt: string, token: string, maxTokens?: number) =>
    request<CompleteResponse>('POST', '/llm/complete', { provider, prompt, max_tokens: maxTokens ?? 256 }, token),

  usage: (token: string) =>
    request<UsageInfo>('GET', '/usage', undefined, token),

  adminStatus: (token: string) =>
    request<AdminStatus>('GET', '/admin/status', undefined, token),

  resetUsage: (email: string, token: string) =>
    request<{ status: string }>('POST', '/admin/reset-usage', { email }, token),
}

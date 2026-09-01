# Prompt Injectulator

A defensive security-testing tool for exploring and documenting LLM prompt-injection techniques. It helps security researchers and developers understand, reproduce, and remediate prompt-injection vulnerabilities in their own systems.

> **Intended use:** Security research, testing, and education. This tool is designed to help defenders—do not use it to attack systems you do not own or have explicit permission to test.

---

## Quick Start (local / mock mode – no API key required)

```bash
# 1. Clone and enter the repo
git clone https://github.com/shadswihart76-oss/The-Prompt-injectulator.git
cd The-Prompt-injectulator

# 2. Configure dev environment
cp .env.example .env
# Edit .env and fill in the required values:
#   JWT_SECRET   – generate with: openssl rand -base64 48
#   DEV_TEST_EMAIL    – e.g. you@localhost
#   DEV_TEST_PASSWORD – choose a unique local password (min 8 chars)
#   DEV_MODE=true

# 3. Build the frontend
cd frontend && npm install && npm run build && cd ..

# 4. Run the backend (serves the frontend from backend/static/)
cd backend
export $(grep -v '^#' ../.env | xargs)
go run ./cmd/server

# 5. Open http://localhost:8080 in your browser
#    Login with the credentials you set in DEV_TEST_EMAIL / DEV_TEST_PASSWORD
```

The mock provider runs entirely locally with no external API calls or costs.

---

## Architecture

```
.
├── backend/                    # Go HTTP API server
│   ├── cmd/server/main.go      # Entry point & routing
│   └── internal/
│       ├── auth/               # JWT auth, config, user store
│       ├── llm/                # Provider interface, mock & OpenAI providers
│       ├── usage/              # Per-user token budget tracker
│       └── handler/            # HTTP request handlers
├── frontend/                   # React + TypeScript (Vite)
│   └── src/
│       ├── App.tsx             # Main UI (login, prompt, admin panel)
│       └── api/api.ts          # Typed API client
├── .env.example                # Environment variable template
└── README.md
```

---

## Privileged User / Admin Mode

### How it works

- Admin status is determined **entirely server-side** from `ADMIN_EMAILS` in the environment.
- Clients cannot self-elevate by modifying JWTs, localStorage, or request parameters.
- The JWT role claim is set at login from the server config; **every** protected endpoint re-reads `IsAdmin()` from server config on each request — the JWT role claim is never the sole authorization source.
- In **dev mode** (`DEV_MODE=true`), a bootstrap admin account (`DEV_TEST_EMAIL`) is created so you can test without a real API key. Both `DEV_TEST_EMAIL` and `DEV_TEST_PASSWORD` must be explicitly supplied; no defaults are provided.

### Admin capabilities

| Feature | Admin | Regular User |
|---|---|---|
| Mock provider (zero cost) | ✅ | ❌ (403 Forbidden) |
| OpenAI / real provider | ✅ | ✅ (subject to budget) |
| Admin status panel | ✅ | ❌ (403 Forbidden) |
| Reset any user's budget | ✅ | ❌ (403 Forbidden) |

---

## API Reference

All requests that require authentication must include:
```
Authorization: ******
```

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/auth/register` | None | Register a new user |
| POST | `/api/auth/login` | None | Login and receive a JWT |
| POST | `/api/llm/complete` | User | Submit a prompt to an LLM provider |
| GET  | `/api/usage` | User | View your token budget status |
| GET  | `/api/admin/status` | Admin | Admin dashboard |
| POST | `/api/admin/reset-usage` | Admin | Reset a user's token budget |

### POST `/api/llm/complete`

```json
{
  "provider": "mock",     // "mock" (admin only) or "openai"
  "prompt": "...",
  "max_tokens": 256
}
```

Response includes a `usage.mode` field that is either `"mock (no cost)"` or `"real (paid API active)"`.

---

## Usage / Budget Controls

- Each non-admin user has a configurable token budget (default: `10,000`; set `USER_TOKEN_LIMIT` in `.env`).
- Requests that would exceed the budget are rejected with **HTTP 402 Payment Required** and a clear error message.
- Admin users' token usage on the mock provider is not counted (zero cost).
- Admins can reset any user's budget via `POST /api/admin/reset-usage`.

---

## Environment Variables

See `.env.example` for all variables. Key ones:

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET` | *(required, all modes)* | HMAC-SHA256 signing key (min 32 chars) |
| `DEV_MODE` | `false` | Enable dev/test mode. **Never true in production.** |
| `DEV_TEST_EMAIL` | *(required when DEV_MODE=true)* | Dev-only bootstrap admin email |
| `DEV_TEST_PASSWORD` | *(required when DEV_MODE=true)* | Dev-only bootstrap admin password (min 8 chars, no common placeholders) |
| `ADMIN_EMAILS` | *(required in production)* | Comma-separated admin email allowlist |
| `CLOUD_API_ENABLED` | `false` | Enable paid cloud provider (e.g. OpenAI). Must be `true` to allow cloud requests. |
| `OPENAI_API_KEY` | *(optional)* | OpenAI API key (only used when CLOUD_API_ENABLED=true) |
| `USER_TOKEN_LIMIT` | `10000` | Per-user token budget |
| `PORT` | `8080` | HTTP listen port |

---

## Running Tests

```bash
# Backend
cd backend && go test ./...

# Frontend build check
cd frontend && npm run build
```

---

## Threat Model & Security Notes

### Development Mock Mode vs Production API Mode

| | Dev Mode (`DEV_MODE=true`) | Production (`DEV_MODE=false`) |
|---|---|---|
| Bootstrap admin account | ✅ created from DEV_TEST_EMAIL/PASSWORD | ❌ not created |
| JWT_SECRET required | **Yes – required in all modes** | **Yes – required in all modes** |
| ADMIN_EMAILS required | No (dev account auto-added) | **Yes – server refuses to start without it** |
| DEV_TEST_EMAIL/PASSWORD | Required (no defaults; rejects common placeholders) | Must not be set |
| Cloud API calls | Only when CLOUD_API_ENABLED=true | Only when CLOUD_API_ENABLED=true |
| Suitable for internet exposure | **No** | Yes (with proper secrets) |

### Key Design Decisions

1. **Server-side role enforcement.** Every protected endpoint re-derives admin status from `s.Cfg.IsAdmin(claims.Email)` using the server-side config on each request. A valid JWT containing `role=admin` for an email absent from `ADMIN_EMAILS` is denied. A stale `role=user` token for an allowlisted email is granted admin access. The JWT role claim is never the sole authorization source.

2. **No hard-coded production credentials, no fallback defaults.** `JWT_SECRET` is required in all modes (min 32 chars). `DEV_TEST_EMAIL` and `DEV_TEST_PASSWORD` must be explicitly supplied when `DEV_MODE=true`; no defaults are provided. Common placeholder passwords (such as `changeme`) are rejected at startup. In production, dev-mode variables must not be set.

3. **Dev mode is explicit and visible.** `DEV_MODE=true` must be set intentionally. The server logs a prominent warning and the dev account email on startup (password is never logged).

4. **Cloud provider disabled by default.** All cloud (paid) provider requests are rejected unless `CLOUD_API_ENABLED=true` is explicitly set. The response includes a machine-readable `"code": "cloud_disabled"` field. No automatic fallback between providers occurs.

5. **Atomic quota enforcement.** The tracker uses a reserve/commit/release model: quota is reserved atomically before a paid call, committed to actual usage on success, and released without charge on provider failure. Exact-limit requests are permitted; over-limit requests are denied.

6. **In-memory user store.** The current implementation uses an in-memory store (suitable for single-instance development and testing). For production multi-instance deployment, replace `auth.Store` and `usage.Tracker` with a persistent, distributed store.

7. **Password hashing.** Passwords are hashed with bcrypt (`golang.org/x/crypto/bcrypt`). Plaintext passwords are never stored or logged.

8. **Security headers.** All API responses include `X-Content-Type-Options: nosniff`, `X-Frame-Options`, and a conservative `Content-Security-Policy`. Auth/login endpoints additionally set `Cache-Control: no-store`.

9. **Request hardening.** Request bodies are limited to 1 MiB before JSON decoding. Trailing garbage after valid JSON is rejected. Authentication failure responses return generic messages without exposing JWT parsing details.

### Out of Scope / Future Work

- Persistent database (currently in-memory only)
- OAuth / SSO login
- Rate limiting per IP
- Audit logging of admin actions
- HTTPS / TLS termination (use a reverse proxy in production)

---

## License

See [LICENSE](LICENSE).

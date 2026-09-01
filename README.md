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
# .env already has DEV_MODE=true and default test credentials – edit if desired

# 3. Build the frontend
cd frontend && npm install && npm run build && cd ..

# 4. Run the backend (serves the frontend from backend/static/)
cd backend
export $(grep -v '^#' ../.env | xargs)
go run ./cmd/server

# 5. Open http://localhost:8080 in your browser
#    Login with: admin@localhost / changeme  (or whatever you set in .env)
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
- The JWT role claim is set at login from the server config; the mock-provider check re-reads the claim on every request.
- In **dev mode** (`DEV_MODE=true`), a bootstrap admin account (`DEV_TEST_EMAIL`) is created automatically so you can test without a real API key.

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
| `DEV_MODE` | `false` | Enable dev/test mode. **Never true in production.** |
| `DEV_TEST_EMAIL` | `admin@localhost` | Dev-only bootstrap admin email |
| `DEV_TEST_PASSWORD` | `changeme` | Dev-only bootstrap admin password |
| `JWT_SECRET` | *(required in prod)* | HMAC-SHA256 signing key |
| `ADMIN_EMAILS` | *(required in prod)* | Comma-separated admin email allowlist |
| `OPENAI_API_KEY` | *(optional)* | OpenAI API key for real completions |
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
| Bootstrap admin account | ✅ auto-created from env vars | ❌ not created |
| JWT_SECRET required | No (insecure default used) | **Yes – server refuses to start without it** |
| ADMIN_EMAILS required | No (dev account auto-added) | **Yes – server refuses to start without it** |
| External API calls | None (mock only) | Only when provider=openai and key is set |
| Suitable for internet exposure | **No** | Yes (with proper secrets) |

### Key Design Decisions

1. **Server-side role enforcement.** The `mock` provider check happens inside the HTTP handler after JWT validation. No client-side gate exists alone; authorization is always re-validated per-request.

2. **No hard-coded production credentials.** Production mode fails closed: if `JWT_SECRET` or `ADMIN_EMAILS` are absent, the process exits at startup rather than running in an insecure state.

3. **Dev mode is explicit and visible.** `DEV_MODE=true` must be set intentionally. The server logs a prominent warning and exposes the dev account email on startup.

4. **In-memory user store.** The current implementation uses an in-memory store (suitable for testing). For production, replace `auth.Store` with a persistent database and use environment-injected credentials for the DB connection.

5. **Password hashing.** Passwords are hashed with bcrypt (`golang.org/x/crypto/bcrypt`). Plaintext passwords are never stored or logged.

6. **Token budget guardrails.** API usage is tracked per user in memory. Limits prevent runaway spend if a user account is compromised. Admins can reset budgets.

### Out of Scope / Future Work

- Persistent database (currently in-memory only)
- OAuth / SSO login
- Rate limiting per IP
- Audit logging of admin actions
- HTTPS / TLS termination (use a reverse proxy in production)

---

## License

See [LICENSE](LICENSE).

# LLM Security Prompt Workbench

> **Authorized use only.** This tool is intended exclusively for evaluating LLM applications that you own or have explicit written permission to test.

A polished, ChatGPT/Claude-style workspace that transforms a weak security-testing objective into a structured, reproducible test plan covering direct and indirect prompt-injection scenarios, mapped to OWASP LLM Top-10 vulnerability classes.

---

## Architecture

```
llm-security-prompt-optimizer/
├── backend/          # Go (net/http) REST API
│   ├── main.go
│   └── internal/assessment/
│       ├── types.go       # request/response types
│       ├── validate.go    # input validation
│       ├── generator.go   # deterministic assessment generator (implements Generator interface)
│       └── assessment_test.go
└── frontend/         # React + TypeScript + Vite
    ├── src/
    │   ├── App.tsx    # full UI (sidebar, composer, result view)
    │   ├── App.css    # dark design system tokens + components
    │   ├── api.ts     # typed fetch wrapper
    │   └── types.ts   # shared TypeScript types
    └── vite.config.ts # proxies /api → localhost:8080
```

### Backend API

| Endpoint | Method | Description |
|---|---|---|
| `/api/health` | GET | Health check |
| `/api/assessments` | POST | Generate a structured assessment |

**POST /api/assessments** – request body:
```json
{
  "objective": "Test whether a knowledge assistant follows instructions embedded in imported content.",
  "surface": "rag",
  "riskFocus": "instruction-override",
  "environment": "staging"
}
```

Valid values:
- `surface`: `chat` | `rag` | `agent`
- `riskFocus`: `instruction-override` | `data-exposure` | `unsafe-tool-actions` | `cross-tenant-isolation`

The generator is implemented behind a `Generator` interface so a future LLM-backed implementation can be swapped in without changing the HTTP layer.

---

## Setup & Running

### Prerequisites

- Go ≥ 1.21
- Node.js ≥ 18 / npm ≥ 9

### Backend

```bash
cd backend
go run .
# Listening on :8080
```

Set `PORT` environment variable to override.

### Frontend (development)

```bash
cd frontend
npm install
npm run dev
# Opens http://localhost:5173 — /api proxied to :8080
```

### Production build

```bash
cd frontend
npm run build
# Output in frontend/dist — serve via any static host
```

### Running tests

```bash
# Go
cd backend
go test ./...

# TypeScript / build check
cd frontend
npm run build
```

---

## UX Overview

- **Dark, high-contrast interface** inspired by ChatGPT/Claude.
- **Left navigation rail** with product branding, "New Assessment" CTA, and compact recent-assessment history.
- **Prompt Composer** with objective textarea, example chips, target surface selector, risk-focus selector, and optional environment label.
- **Conversation-style result flow**: normalized objective → threat-model summary → expandable scenario cards → vulnerability/finding map.
- Each scenario card shows: type badge (direct/indirect), severity badge, intent, copyable test fixture, expected behavior, and evidence to capture.
- **Authorization/safety notice** prominently displayed on every result.
- Responsive: sidebar hides on narrow viewports.

---

## LLM Provider / Model Integration

The current `DeterministicGenerator` produces rule-based assessments without any external API calls, so the MVP works on a clean clone without credentials.

A future LLM-backed generator can be implemented by satisfying the `Generator` interface in `backend/internal/assessment/generator.go`:

```go
type Generator interface {
    Generate(r Request) (Assessment, error)
}
```

**Important:** A GitHub Copilot subscription is **not** a generic application backend inference API. Copilot is a developer productivity tool scoped to code completion and chat inside supported IDEs and GitHub.com. If you want to integrate a hosted language model, you need a separate provider arrangement — for example:

- [OpenAI API](https://platform.openai.com/)
- [Anthropic API](https://www.anthropic.com/api)
- [Azure OpenAI Service](https://azure.microsoft.com/en-us/products/ai-services/openai-service)
- [GitHub Models](https://github.com/marketplace/models) (preview, subject to terms)
- Any other provider whose terms of service permit the intended use

Evaluate each provider's terms of service for your deployment scenario.

---

## Security & Responsible Use

All test fixtures use **harmless canary tokens** (`CANARY_*`) and **fictional document/tool names**. They do not contain:
- Jailbreak recipes
- Hidden-prompt extraction techniques  
- Credential-stealing prompts
- Instructions intended to defeat live third-party safeguards

This tool generates authorized test plans — not attack tools. Always obtain explicit written permission before testing any system you do not own.

---

## License

MIT

# LocalID React Example

Minimal Vite + React + TypeScript demo for the LocalID auth loop.

## Prerequisites

- Node.js 20+
- pnpm 10+
- LocalID Agent running with `config.example.json` (mock provider, port `17443`)
- Mock backend running on port `8000`

## Setup

```bash
cp .env.example .env
pnpm install
```

## Environment

| Variable | Default | Description |
|----------|---------|-------------|
| `VITE_AGENT_URL` | `http://127.0.0.1:17443` | LocalID Agent base URL |
| `VITE_BACKEND_URL` | `http://localhost:8000` | Backend issuing challenges and verifying proofs |

## Development

```bash
pnpm run dev
```

Open `http://localhost:5173`. The app checks agent health/status on load and runs the full flow when you click **Authenticate with LocalID**:

1. `POST /localid/challenge` on the backend
2. `POST /sign-challenge` on the agent (browser `Origin` + body `origin` must match)
3. `POST /localid/verify` on the backend

## Build

```bash
pnpm run build
pnpm run preview
```

## Troubleshooting

- **Agent unreachable** — start the agent: `go run ./cmd/localid-agent --config config.example.json`
- **403 from agent** — ensure you use `http://localhost:5173` (not another port); origin must match `security.allowed_origins` in config
- **Verify failed** — restart the mock backend; challenges expire after 60 seconds and are one-time use

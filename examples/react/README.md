# React example

Minimal **React 19** + **Vite 8** + **TypeScript 6** demo for the LocalID auth loop.

## Prerequisites

- Node.js 20+ and pnpm 10+ (from repo root)
- LocalID Agent running with `http://localhost:5173` in `allowed_origins`
- A backend on port `8000` (see [examples/README.md](../README.md))

## Setup

```bash
pnpm install
cp examples/react/.env.example examples/react/.env
```

## Run

```bash
# Terminal 1 — agent
cd services/agent && go run ./cmd/localid-agent --config config.example.json

# Terminal 2 — backend (Go mock, FastAPI, or Laravel)
cd services/agent && go run ./cmd/mock-backend

# Terminal 3 — this app
pnpm run dev:react
```

Open [http://localhost:5173](http://localhost:5173) → **Authenticate with LocalID** → expect **Mock Dev User**.

## Environment

| Variable | Default | Description |
|----------|---------|-------------|
| `VITE_AGENT_URL` | `http://127.0.0.1:17443` | LocalID Agent base URL |
| `VITE_BACKEND_URL` | `http://localhost:8000` | Backend API URL |

## Build

```bash
pnpm run build:react
```

## Shared client

Uses [`@rqc-icu/localid-client`](../../packages/localid-client) — same types generated from `proto/localid/v1/protocol.proto`.

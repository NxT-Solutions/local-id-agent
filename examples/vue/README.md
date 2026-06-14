# Vue example

Minimal [Vue 3](https://vuejs.org/) + Vite + TypeScript demo for the LocalID auth loop.

## Prerequisites

- Node.js 20+ and pnpm 10+ (from repo root)
- LocalID Agent running with `http://localhost:5174` in `allowed_origins`
- A backend on port `8000` (Go mock, FastAPI, or Laravel example)

## Setup

```bash
# from repo root
pnpm install
cp examples/vue/.env.example examples/vue/.env
```

Ensure `services/agent/config.example.json` includes `http://localhost:5174` under `allowed_origins`.

## Run

```bash
pnpm run dev:vue
```

Open [http://localhost:5174](http://localhost:5174) and click **Authenticate with LocalID**.

## Backends

Pick one API on port `8000`:

| Backend | Command |
|---------|---------|
| Go mock | `cd services/agent && go run ./cmd/mock-backend` |
| FastAPI | `cd examples/fastapi && uvicorn app.main:app --reload --port 8000` |
| Laravel | `cd examples/laravel && php artisan serve --port=8000` |

## Build

```bash
pnpm run build:vue
```

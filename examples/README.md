# Examples

Integration examples for LocalID Agent — frontends and backend APIs that implement the `/localid/*` contract.

## Frontends

| Example | Stack | Port | Dev command |
|---------|-------|------|-------------|
| [react/](react/) | React 19 + Vite 8 + TypeScript 6 | 5173 | `pnpm run dev:react` |
| [vue/](vue/) | Vue 3.5 + Vite 8 + TypeScript 6 | 5174 | `pnpm run dev:vue` |

Both use the shared [`@rqc-icu/localid-client`](../../packages/localid-client) package.

### Typed API clients (Orval)

The monorepo ships OpenAPI specs in [`openapi/`](../../openapi/) and generates typed fetch clients with [Orval](https://orval.dev/):

```bash
pnpm generate:api   # from repo root
```

Import generated functions from `@rqc-icu/localid-client`:

```typescript
import { agentOpenAPI, backendOpenAPI } from "@rqc-icu/localid-client";

const { data: health } = await agentOpenAPI.getHealth();
```

With the agent running in dev mode (`server.dev_mode: true`), Orval can also use `http://127.0.0.1:17443/openapi.json` as its `input` URL.

## Backends (pick one on port 8000)

| Example | Stack | Run |
|---------|-------|-----|
| Go mock | Go (in agent module) | `cd services/agent && go run ./cmd/mock-backend` |
| [fastapi/](fastapi/) | Python FastAPI | `uvicorn app.main:app --reload --port 8000` |
| [laravel/](laravel/) | PHP Laravel 12 | `php artisan serve --port=8000` |

## Quick test (React + FastAPI)

```bash
# Terminal 1 — agent
cd services/agent && go run ./cmd/localid-agent --config config.example.json

# Terminal 2 — FastAPI
cd examples/fastapi && python3 -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt && uvicorn app.main:app --reload --port 8000

# Terminal 3 — React
pnpm run dev:react
```

Set `VITE_BACKEND_URL=http://localhost:8000` in `examples/react/.env` or `examples/vue/.env`.

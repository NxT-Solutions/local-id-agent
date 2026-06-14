# LocalID Agent — Reference

## Agent HTTP API

Base URL: `http://127.0.0.1:17443` (default)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Liveness `{ ok, name, version }` |
| GET | `/status` | Provider `{ provider, ready, cardPresent }` |
| POST | `/sign-challenge` | Sign canonical payload; requires `Content-Type: application/json` + `Origin` header |
| GET | `/openapi.json` | OpenAPI JSON (**dev_mode only**) |
| GET | `/openapi.yaml` | OpenAPI YAML (**dev_mode only**) |

### POST /sign-challenge

**Request body:**
```json
{
  "challenge": "<base64url from backend>",
  "backend": "http://localhost:8000",
  "purpose": "login",
  "origin": "http://localhost:5173"
}
```

**Response 200:**
```json
{
  "provider": "mock",
  "algorithm": "RS256",
  "challenge": "...",
  "signature": "...",
  "certificate": "...",
  "signedAt": "2026-06-14T12:00:00Z"
}
```

**Errors:** 400 bad request, 403 forbidden (origin/backend/purpose), 413 too large, 415 wrong content-type.

**Handler chain:** CORS → logging → max body (64KB) → JSON content-type (POST only) → dev_mode gate (openapi routes).

## Backend HTTP API (customer implements)

Base URL: customer's API (e.g. `http://localhost:8000`)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/localid/challenge` | Issue one-time challenge `{ challenge }` |
| POST | `/localid/verify` | Verify proof + issue session `{ success, user }` |

### POST /localid/verify request

```json
{
  "challenge": "...",
  "backend": "http://localhost:8000",
  "origin": "http://localhost:5173",
  "purpose": "login",
  "provider": "mock",
  "algorithm": "RS256",
  "signature": "...",
  "certificate": "...",
  "signedAt": "2026-06-14T12:00:00Z"
}
```

Backends must:
1. Store challenges server-side with ~60s TTL, single use.
2. Rebuild canonical JSON with `timestamp` = `signedAt`.
3. Verify RS256 signature against certificate public key.
4. Map certificate identity to application user.

**Example implementations:**
- Go: `services/agent/cmd/mock-backend/main.go`
- Python: `examples/fastapi/`
- PHP: `examples/laravel/`

## Canonical signed payload

Alphabetical JSON keys, compact encoding:

```json
{"backend":"http://localhost:8000","challenge":"...","origin":"http://localhost:5173","purpose":"login","timestamp":"2026-06-14T12:00:00Z"}
```

Go implementation: `services/agent/internal/security/canonical.go`

## Agent config schema

```json
{
  "server": {
    "host": "127.0.0.1",
    "port": 17443,
    "allow_remote_bind": false,
    "dev_mode": true
  },
  "security": {
    "allowed_origins": ["http://localhost:5173", "http://localhost:5174", "tauri://localhost"],
    "allowed_backends": ["http://localhost:8000"],
    "challenge_max_age_seconds": 60,
    "production": false
  },
  "providers": {
    "default": "mock",
    "mock": { "enabled": true },
    "pkcs11": { "enabled": false, "module_path": "auto", "token_label": "", "certificate_label": "", "pin_prompt": "terminal" },
    "belgian_eid": { "enabled": false, "pkcs11_module_path": "auto" }
  },
  "logging": { "level": "info" }
}
```

- `dev_mode: true` → serves `/openapi.json` and `/openapi.yaml`; disable in production.
- Desktop first-run copies `apps/desktop/src-tauri/config.desktop.json` to OS app data as `config.json`.

## Go package layout (`services/agent/`)

```
cmd/
  localid-agent/     Main agent entry
  mock-backend/      Reference backend on :8000
internal/
  api/               HTTP server, routes, handlers, middleware
  config/            Config types + JSON loader
  logging/           slog setup
  openapi/           Embedded OpenAPI spec + serve helpers
  protocol/          HTTP response helpers, type aliases to pb/
  protocol/pb/       Buf-generated protobuf Go code
  providers/         Provider interface + mock/pkcs11/belgian_eid
  security/          Origin/backend validation, canonical JSON, nonces
```

**Provider interface:**
```go
type Provider interface {
    Name() string
    Status(ctx context.Context) (*protocol.Status, error)
    SignChallenge(ctx context.Context, req protocol.SignChallengeRequest) (*protocol.SignChallengeResponse, error)
}
```

Factory: `providers.New(cfg.Providers)` — `default` selects active provider.

## TypeScript client (`packages/localid-client/`)

```
src/
  agent.ts, backend.ts, config.ts   Hand-written fetch helpers (used by examples)
  types.ts                          Re-exports proto shapes + backend-only types
  generated/localid/v1/             Buf-generated TS (not compiled directly)
  openapi/                          Orval-generated agent + backend clients
    agent.ts, backend.ts            Generated — do not edit
    mutators.ts                     Custom fetch with base URL + Origin header
orval.config.ts
```

**Exports from `index.ts`:**
- Hand-written: `signChallenge`, `fetchChallenge`, `verifyProof`, `configureLocalIDClient`, types
- Orval namespaces: `agentOpenAPI`, `backendOpenAPI`

**Build:** `tsc` compiles `src/` except `src/generated/` (proto TS used only for type inference in `types.ts`).

## Turborepo tasks (`turbo.json`)

| Task | Depends on | Outputs |
|------|------------|---------|
| `//#generate` | — | `services/agent/internal/protocol/pb/**`, `packages/localid-client/src/generated/**` |
| `//#generate:api` | — | `packages/localid-client/src/openapi/**`, embedded openapi spec |
| `build` | `^build`, `//#generate` | `dist/**` |
| `test` | `//#generate` | — |

Root scripts in `package.json`; workspace packages in `pnpm-workspace.yaml` (`packages/*`, `apps/*`, `examples/*`).

## Buf configuration

- `buf.yaml` — module root at `proto/`
- `buf.gen.yaml` — Go to `services/agent/internal/protocol/pb`, ES to `packages/localid-client/src/generated`
- Go package prefix: `github.com/rqc-icu/localid-agent/services/agent/internal/protocol/pb`

## OpenAPI sync

`scripts/sync-openapi.sh` copies `openapi/agent.openapi.yaml` → `services/agent/internal/openapi/spec/agent.openapi.yaml` (go:embed for runtime `/openapi.json`).

Edit source at `openapi/`; never edit the embedded copy directly.

## Desktop app structure

| Path | Purpose |
|------|---------|
| `apps/desktop/src/` | React UI (Dashboard, Settings, Setup, Diagnostics, Demo) |
| `apps/desktop/src-tauri/src/lib.rs` | Sidecar lifecycle, config read/write, tray |
| `apps/desktop/src-tauri/tauri.conf.json` | Tauri 2 config, `externalBin` sidecar |
| `apps/desktop/src-tauri/binaries/` | Platform-specific Go agent binary |

**Routes:** `/` Dashboard, `/settings` Settings, `/setup` Setup guide, `/about` About, `/demo` Auth demo.

**Sidecar build:** `scripts/build-agent-sidecar.sh` → `localid-agent-<rustc-host-triple>`.

## Environment variables (examples)

| Variable | Default | Used by |
|----------|---------|---------|
| `VITE_AGENT_URL` | `http://127.0.0.1:17443` | React/Vue/desktop demos |
| `VITE_BACKEND_URL` | `http://localhost:8000` | React/Vue/desktop demos |

## Testing

```bash
cd services/agent && go test ./...
pnpm --filter @rqc-icu/localid-client test   # tsc --noEmit
pnpm run build                                  # full monorepo build
```

Key test files:
- `internal/api/handlers_test.go` — HTTP integration (sign-challenge, openapi dev_mode)
- `internal/security/*_test.go` — validation
- `internal/providers/mock/mock_test.go` — deterministic signing
- `internal/openapi/openapi_test.go` — embedded spec JSON

## Protobuf messages (`proto/localid/v1/protocol.proto`)

- `SignChallengeRequest` — challenge, backend, purpose, origin
- `SignChallengeResponse` — provider, algorithm, challenge, signature, certificate, signedAt
- `Status` — provider, ready, cardPresent
- `HealthResponse` — ok, name, version

Backend challenge/verify types are **not** in protobuf — they live in OpenAPI + hand-written `types.ts`.

---
name: localid-agent
description: >-
  Develop and extend the LocalID Agent monorepo — Go localhost signing agent,
  Tauri desktop app, protobuf/OpenAPI contracts, and TypeScript client. Use when
  working in local-id-agent, on sign-challenge, allowed_origins, canonical JSON
  signing, sidecar, mock backend, @rqc-icu/localid-client, Buf, Orval, or
  examples/react|vue|fastapi|laravel.
---

# LocalID Agent

Localhost identity bridge: browser frontends get cryptographic proofs from a local smartcard/eID/PKCS#11 device via a Go agent on `127.0.0.1:17443`. **The agent never issues sessions or tokens** — backends verify RS256 signatures and own auth.

## Monorepo map

| Path | Role |
|------|------|
| `proto/localid/v1/` | Protobuf source of truth (agent HTTP message shapes) |
| `openapi/` | OpenAPI specs (agent + backend HTTP; Orval input) |
| `services/agent/` | Go agent (`cmd/localid-agent`), mock backend (`cmd/mock-backend`) |
| `packages/localid-client/` | `@rqc-icu/localid-client` — hand-written + generated clients |
| `apps/desktop/` | Tauri 2 + React + shadcn; bundles Go agent as sidecar |
| `examples/react`, `examples/vue` | Browser demos (`:5173`, `:5174`) |
| `examples/fastapi`, `examples/laravel` | Backend demos (`:8000`) |
| `docs/` | Architecture, config, desktop guides |

**Go module:** `github.com/rqc-icu/localid-agent/services/agent`

## Invariants (do not break)

1. **Agent scope:** Sign challenges only. No login tokens, sessions, or user DB in the agent.
2. **Allowlists:** `security.allowed_origins` and `security.allowed_backends` use **exact string match** (scheme + host + port, no wildcards, no trailing slash).
3. **Sign payload:** Agent signs **canonical JSON**, not the raw challenge. Keys in order: `backend`, `challenge`, `origin`, `purpose`, `timestamp` (compact, no extra whitespace). `timestamp` = `signedAt` from the response (RFC3339 UTC).
4. **Algorithm:** RS256 (RSA PKCS#1 v1.5 + SHA-256). `signature` is base64url; `certificate` is standard base64 DER.
5. **Purpose:** Only `"login"` is accepted today on `/sign-challenge`.
6. **CORS:** Browser calls agent cross-origin; `Origin` header must match body `origin` and allowlist.
7. **Secrets:** Never log PINs, private keys, or full signatures.

## Auth flow

```
Frontend → POST /localid/challenge (backend) → challenge
Frontend → POST /sign-challenge (agent) → proof
Frontend → POST /localid/verify (backend) → session
```

## Where to change what

| Task | Location |
|------|----------|
| Agent HTTP handlers / routes | `services/agent/internal/api/` |
| Origin/backend validation | `services/agent/internal/security/` |
| Provider logic | `services/agent/internal/providers/{mock,pkcs11,belgian_eid}/` |
| Agent config schema | `services/agent/internal/config/` + `config.example.json` |
| Desktop config template | `apps/desktop/src-tauri/config.desktop.json` |
| Agent message types (source) | `proto/localid/v1/protocol.proto` → `pnpm generate` |
| HTTP OpenAPI contracts | `openapi/*.yaml` → `pnpm generate:api` |
| TS hand-written client | `packages/localid-client/src/{agent,backend,config,types}.ts` |
| TS Orval output | `packages/localid-client/src/openapi/` (generated) |
| Embedded agent OpenAPI | `services/agent/internal/openapi/spec/` (synced by `scripts/sync-openapi.sh`) |
| Desktop UI | `apps/desktop/src/` |
| Tauri / sidecar | `apps/desktop/src-tauri/` |

## Code generation (two pipelines)

**Protobuf (agent message shapes):**
```bash
pnpm generate   # buf → Go pb/ + TS generated/
```
- Go: `services/agent/internal/protocol/pb/`
- TS: `packages/localid-client/src/generated/` (excluded from tsc; re-exported via `types.ts`)

**OpenAPI (HTTP clients for frontends):**
```bash
pnpm generate:api   # sync-openapi.sh + Orval in localid-client
```
- Specs: `openapi/agent.openapi.yaml`, `openapi/backend.openapi.yaml`
- Dev endpoint: `GET /openapi.json` when `server.dev_mode: true`
- Orval config: `packages/localid-client/orval.config.ts`
- Mutators add `Origin` header from JSON body for `/sign-challenge`

After changing `.proto` or `openapi/`, regenerate and commit outputs.

## Common commands

```bash
# Agent
cd services/agent && go run ./cmd/localid-agent --config config.example.json
cd services/agent && go run ./cmd/mock-backend
cd services/agent && go test ./...

# Full stack demo
pnpm run dev:react          # :5173
pnpm run dev:vue            # :5174
pnpm run build:sidecar      # after Go agent changes (desktop)
pnpm --filter desktop tauri dev

# Monorepo
pnpm install
pnpm generate
pnpm generate:api
pnpm run build
pnpm run test:go
```

**Ports:** agent `17443`, backend `8000`, react `5173`, vue `5174`, desktop UI `1420`.

## Client usage (pick one)

**Hand-written (examples use this):**
```typescript
import { fetchChallenge, signChallenge, verifyProof, getBackendUrl } from "@rqc-icu/localid-client";
```

**Orval-generated:**
```typescript
import { agentOpenAPI, backendOpenAPI } from "@rqc-icu/localid-client";
```

Env vars in examples: `VITE_AGENT_URL`, `VITE_BACKEND_URL`.

## Tauri / sidecar gotchas

- Sidecar binary: `apps/desktop/src-tauri/binaries/localid-agent-<target-triple>` — rebuild with `pnpm run build:sidecar` after Go changes.
- `externalBin` in `tauri.conf.json` references `binaries/localid-agent`.
- Tauri `Command.spawn()` returns `(Receiver, CommandChild)` — destructure both.
- `pnpm run dev:desktop` = Vite UI only (no Rust/tray/sidecar). Full app needs Rust + `tauri dev`.
- Desktop allowed origins include `tauri://localhost` and `http://localhost:1420`.

## Troubleshooting quick ref

| Symptom | Likely fix |
|---------|------------|
| 403 on `/sign-challenge` | Origin/backend not in allowlist or mismatch between header and body |
| Agent unreachable | Start agent; check `127.0.0.1:17443` |
| Verify fails | Challenge expired (60s) or already used — restart mock backend |
| Sidecar won't start | `pnpm run build:sidecar` |
| TS types stale | `pnpm generate` or `pnpm generate:api` |

## Development checklist

When adding agent API fields:
- [ ] Update `proto/localid/v1/protocol.proto`
- [ ] Update `openapi/agent.openapi.yaml`
- [ ] Run `pnpm generate` and `pnpm generate:api`
- [ ] Update Go handlers/providers if behavior changes
- [ ] Update `packages/localid-client` hand-written client if needed
- [ ] Add/update tests in `services/agent/internal/api/`
- [ ] Update `config.example.json` / desktop template if config changes
- [ ] Rebuild sidecar if desktop bundles agent

When adding backend contract fields:
- [ ] Update `openapi/backend.openapi.yaml`
- [ ] Update mock-backend + fastapi + laravel examples together
- [ ] Run `pnpm generate:api`

## Code style for this repo

- **Minimize scope** — focused diffs; match existing patterns in each package.
- **pnpm** (not npm) for JS; **Turborepo** orchestrates `build`/`test`.
- **Go:** chi router, structured logging, testify for tests.
- **Desktop:** React 19, shadcn/ui, path alias `@/`.
- Do not add markdown docs unless the user asks.

## Additional resources

- Full API contracts, config schema, package layout: [reference.md](reference.md)
- Step-by-step workflows (new endpoint, new example, desktop): [workflows.md](workflows.md)
- Human docs: `README.md`, `docs/architecture.md`, `docs/agent-config.md`, `docs/desktop.md`

# LocalID Agent — Workflows

## Run full auth loop locally

```bash
# Terminal 1 — agent
cd services/agent && go run ./cmd/localid-agent --config config.example.json

# Terminal 2 — backend (pick one)
cd services/agent && go run ./cmd/mock-backend
# OR: cd examples/fastapi && uvicorn app.main:app --reload --port 8000
# OR: cd examples/laravel && php artisan serve --port=8000

# Terminal 3 — frontend
pnpm run dev:react   # or dev:vue
```

Ensure `config.example.json` lists the frontend origin and backend URL in allowlists.

## Add a field to agent API responses

1. Edit `proto/localid/v1/protocol.proto`
2. Run `pnpm generate`
3. Update Go type aliases in `services/agent/internal/protocol/types.go` if needed
4. Update provider(s) returning the field (`internal/providers/mock/`, etc.)
5. Update `openapi/agent.openapi.yaml`
6. Run `pnpm generate:api`
7. Update `packages/localid-client/src/types.ts` if hand-written types need the field
8. Add tests in `internal/api/handlers_test.go`
9. `go test ./...` and `pnpm --filter @rqc-icu/localid-client build`

## Add a new agent HTTP endpoint

1. Add route in `services/agent/internal/api/routes.go`
2. Add handler in `handlers.go`
3. Add middleware if needed in `middleware.go`
4. Document in `openapi/agent.openapi.yaml`
5. Run `pnpm generate:api`
6. Add handler tests
7. Update README / Setup page only if user requests docs

## Change OpenAPI / Orval clients

1. Edit `openapi/agent.openapi.yaml` or `openapi/backend.openapi.yaml`
2. Run `pnpm generate:api` (syncs embed + regenerates Orval)
3. Adjust `packages/localid-client/src/openapi/mutators.ts` only if fetch behavior must change
4. Re-export from `src/index.ts` if adding new namespaces
5. `pnpm --filter @rqc-icu/localid-client build`

To point external Orval at live spec (agent running with `dev_mode: true`):
```
input: http://127.0.0.1:17443/openapi.json
```

## Desktop development

**Full native app:**
```bash
pnpm run build:sidecar
pnpm --filter desktop tauri dev
```

**UI only (no sidecar):**
```bash
pnpm run dev:desktop
cd services/agent && go run ./cmd/localid-agent --config config.example.json
```

After any Go agent code change affecting the bundled binary:
```bash
pnpm run build:sidecar
```

Auth demo page (`/demo`) also needs mock backend on `:8000`.

## Add a new backend example

All backends must implement the same contract:
- `POST /localid/challenge` → `{ challenge }`
- `POST /localid/verify` → verify canonical JSON + RS256, return `{ success, user }`

1. Copy patterns from `cmd/mock-backend`, `examples/fastapi`, or `examples/laravel`
2. Update `openapi/backend.openapi.yaml` if contract changes (keep all examples in sync)
3. Add row to `examples/README.md` when user asks for docs
4. Ensure `config.example.json` `allowed_backends` includes the example URL

## Add a new frontend example

1. Create `examples/<name>/` with Vite + workspace dep on `@rqc-icu/localid-client`
2. Add to `pnpm-workspace.yaml` (already covers `examples/*`)
3. Add root script `dev:<name>` in `package.json` if needed
4. Add frontend origin to `config.example.json` and `config.desktop.json`
5. Use `window.location.origin` for the `origin` field in sign requests

## Debug 403 on sign-challenge

Check in order:
1. Agent running on expected host/port
2. `Origin` header matches body `origin` (browser sets this automatically for cross-origin fetch)
3. Both values appear **exactly** in `security.allowed_origins`
4. `backend` field appears **exactly** in `security.allowed_backends`
5. `purpose` is `"login"`

## Verify canonical signing in mock provider

Mock provider uses a deterministic RSA key — useful for tests. See `internal/providers/mock/mock.go`. Backend verification must use the same canonical payload rules as `internal/security/canonical.go`.

## Pre-merge verification

```bash
pnpm generate
pnpm generate:api
pnpm run test:go
pnpm --filter @rqc-icu/localid-client build
pnpm run build
```

If desktop agent changed: `pnpm run build:sidecar` and smoke-test `tauri dev`.

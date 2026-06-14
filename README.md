# LocalID Agent

Localhost identity bridge for web applications — sign backend challenges with a local smartcard, eID, or PKCS#11 device without exposing private keys to the browser.

The agent does **not** issue login tokens. Your backend verifies the signature and owns the session.

## Repository layout

```
services/agent/              Go agent + mock backend
packages/localid-client/       Shared TypeScript client (generated from proto)
apps/desktop/                Tauri 2 desktop app (React + shadcn)
examples/react/              React 19 browser demo
examples/vue/                Vue 3 browser demo
examples/fastapi/            Python FastAPI backend
examples/laravel/            PHP Laravel backend
proto/                       Protobuf API contract (Go + TS codegen)
openapi/                     OpenAPI specs (agent + backend; Orval input)
docs/                        Detailed guides
```

---

## Prerequisites

Install what you need for the parts you want to run:

| Tool | Required for | Install |
|------|----------------|---------|
| **Go 1.25+** | Agent, mock backend, sidecar build | [go.dev/dl](https://go.dev/dl/) |
| **pnpm 10+** & **Node.js 20+** | React/Vue examples, desktop UI | `npm i -g pnpm` |
| **Python 3.12+** | FastAPI example | [python.org](https://www.python.org/) |
| **PHP 8.2+** & **Composer** | Laravel example | [getcomposer.org](https://getcomposer.org/) |
| **Buf CLI** | Regenerating types from `proto/` | `brew install bufbuild/buf/buf` |
| **Rust** ([rustup](https://rustup.rs/)) | Full desktop app (`tauri dev`) | `curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs \| sh` |
| **Xcode CLT** (macOS) | Tauri native build | `xcode-select --install` |

> **Desktop app:** `pnpm run dev:desktop` runs **Vite only** (browser UI). The native window, system tray, and bundled agent require Rust and `pnpm --filter desktop tauri dev`.

---

## First-time setup

From the repository root:

```bash
pnpm install
pnpm generate    # optional: regenerate Go + TS from proto/ (needs Buf)
pnpm generate:api # optional: regenerate Orval clients from openapi/
```

---

## Docker demo stack

Run the production-style demo stack (frontend + backend + agent containers):

```bash
pnpm run docker:demo
```

For host-agent and smartcard-specific flows (Belgium eID / PKCS#11), see `docker/README.md`.

---

## Belgian eID demo modes (no guesswork)

Run from repo root and choose one mode:

| Goal | Command |
|------|---------|
| Native app + eID card (recommended on macOS) | `pnpm demo:native-eid` |
| Browser + eID container | `pnpm demo:docker-eid` |

### Mode 1: `pnpm demo:native-eid` (recommended on macOS)

- Stops Docker **frontend** and **agent** containers (`frontend`, `agent`, `agent-eid`, `agent-pkcs11`); keeps **backend** on `:8000`.
- Builds sidecar (unless `--skip-sidecar-build` is passed to the script directly).
- Launches the **Tauri desktop app** with `LOCALID_PKCS11_PIN` exported — **not** the browser at `http://localhost:5173`.
- First run copies `config.desktop.json` with `belgian_eid` as the default provider; if you used the app before, open **Settings → Belgian eID → Save**, then use the **Demo** tab.

Example:

```bash
LOCALID_PKCS11_PIN=1234 pnpm demo:native-eid
```

If a browser tab at `:5173` still shows **Provider: mock (ready)**, you are on the Docker/browser demo from an earlier `docker:demo` or `demo:docker-eid` run — close that tab and use the Tauri window instead, or run `pnpm demo:stop` first.

### Mode 2: `pnpm demo:docker-eid`

- Runs backend + frontend + `agent-eid` in Docker.
- Opens browser flow at `http://localhost:5173`.
- Requires `LOCALID_PKCS11_PIN`.
- Do not run Tauri at the same time.

Example:

```bash
LOCALID_PKCS11_PIN=1234 pnpm demo:docker-eid
```

Stop demo containers:

```bash
pnpm demo:stop
```

---

## How to run

### 1. Agent only (CLI)

Starts the localhost signing service on `127.0.0.1:17443`.

```bash
cd services/agent
go run ./cmd/localid-agent --config config.example.json
```

Smoke test:

```bash
curl http://127.0.0.1:17443/health
curl http://127.0.0.1:17443/status
curl http://127.0.0.1:17443/openapi.json   # when server.dev_mode is true
```

With `server.dev_mode: true` (default in `config.example.json`), the agent also serves its OpenAPI document at `GET /openapi.json` and `GET /openapi.yaml`. Disable in production.

More: [docs/agent-config.md](docs/agent-config.md)

---

### 2. Browser demos (React or Vue)

Full auth loop: backend challenge → agent signature → backend verify. **Three terminals:**

**Terminal 1 — agent**

```bash
cd services/agent
go run ./cmd/localid-agent --config config.example.json
```

**Terminal 2 — backend** (`:8000`, pick one)

```bash
# Go mock
cd services/agent && go run ./cmd/mock-backend

# FastAPI
cd examples/fastapi && uvicorn app.main:app --reload --port 8000

# Laravel
cd examples/laravel && php artisan serve --port=8000
```

**Terminal 3 — frontend**

```bash
pnpm run dev:react   # http://localhost:5173
# or
pnpm run dev:vue     # http://localhost:5174
```

Click **Authenticate with LocalID** → expect **Mock Dev User**.

See [examples/README.md](examples/README.md) for setup details per stack.

---

### 3. Desktop app (Tauri)

Native cross-platform app. Bundles the Go agent as a sidecar, system tray, settings UI.

**One-time:** build the Go sidecar binary for your OS:

```bash
pnpm run build:sidecar
```

Output: `apps/desktop/src-tauri/binaries/localid-agent-<target-triple>`  
Re-run after changing agent Go code.

**Full desktop dev** (window + tray + sidecar):

```bash
pnpm --filter desktop tauri dev
```

- UI: `http://localhost:1420`
- First run: copies desktop config to OS app data dir
- Routes: Dashboard `/`, Settings `/settings`, About `/about`, Auth demo `/demo`

**Auth demo page** also needs the mock backend:

```bash
cd services/agent && go run ./cmd/mock-backend
```

**UI-only dev** (no Rust, no tray, no sidecar — browser at `:1420`):

```bash
pnpm run dev:desktop
# same as: pnpm --filter desktop dev
```

`pnpm run dev:desktop` does not start the sidecar. You must run the agent separately for UI-only mode:

```bash
cd services/agent && go run ./cmd/localid-agent --config config.example.json
```

**Production build:**

```bash
pnpm run build:sidecar
pnpm --filter desktop tauri build
```

Bundles: `apps/desktop/src-tauri/target/release/bundle/`

More: [docs/desktop.md](docs/desktop.md)

---

## Command reference

Run from the **repository root** unless noted.

| Command | What it does |
|---------|----------------|
| `pnpm install` | Install all JS workspace dependencies |
| `pnpm generate` | `buf generate` — Go + TS types from `proto/` |
| `pnpm generate:api` | Sync OpenAPI spec + Orval clients in `localid-client` |
| `pnpm run test` | Go tests + TypeScript checks (all packages) |
| `pnpm run test:go` | `go test ./...` in `services/agent` |
| `pnpm run test:coverage` | Go + TS coverage (`test:coverage:go` / `:ts`) |
| `pnpm run build` | Build client, React example, desktop frontend |
| `pnpm run build:sidecar` | Compile Go agent for desktop bundle |
| `pnpm demo:native-eid` | Full native Belgian eID demo orchestration |
| `pnpm demo:docker-eid` | Full Docker Belgian eID demo orchestration |
| `pnpm demo:stop` | Stop demo containers and remove orphans |
| `pnpm run build:react` | Production build of React demo |
| `pnpm run build:vue` | Production build of Vue demo |
| `pnpm run build:desktop` | Typecheck + Vite build of desktop UI |
| `pnpm run dev:react` | React demo dev server (`:5173`) |
| `pnpm run dev:vue` | Vue demo dev server (`:5174`) |
| `pnpm run dev:desktop` | Desktop **UI only** via Vite (`:1420`) |
| `pnpm --filter desktop tauri dev` | **Full** desktop app (needs Rust) |
| `pnpm --filter desktop tauri build` | Desktop installer/bundle (needs Rust) |

### Go (from `services/agent/`)

| Command | What it does |
|---------|----------------|
| `go run ./cmd/localid-agent --config config.example.json` | Start agent |
| `go run ./cmd/mock-backend` | Start mock backend on `:8000` |
| `go test ./...` | Run agent tests |
| `go build -o localid-agent ./cmd/localid-agent` | Build agent binary |

---

## Troubleshooting

| Problem | Fix |
|---------|-----|
| `cargo metadata: No such file or directory` | Install Rust: [rustup.rs](https://rustup.rs/), then `source "$HOME/.cargo/env"` |
| Agent unreachable in browser/desktop | Start agent: `cd services/agent && go run ./cmd/localid-agent --config config.example.json` |
| Browser at `:1420` shows `NetworkError when attempting to fetch resource` | `pnpm run dev:desktop` is UI-only and does not start sidecar; use `pnpm demo:native-eid` (or run agent separately) |
| Browser at `:5173` shows **Provider: mock (ready)** after `demo:native-eid` | Stale Docker frontend from `docker:demo` / `demo:docker-eid`; close the tab, run `pnpm demo:stop`, then use the **Tauri desktop window** from `demo:native-eid`. In the app: **Settings → Belgian eID → Save** if provider is still mock |
| 403 from `/sign-challenge` | Origin must match `security.allowed_origins` exactly (e.g. `http://localhost:5173` or `tauri://localhost`) |
| Verify failed in demo | Restart mock backend; challenges expire after 60s and are one-time use |
| Desktop sidecar won't start | Run `pnpm run build:sidecar` after agent code changes |
| Types out of sync | Run `pnpm generate` after editing `proto/` |
| OpenAPI / Orval out of sync | Run `pnpm generate:api` after editing `openapi/` |

---

## OpenAPI & typed clients (Orval)

HTTP contracts live in [`openapi/`](openapi/):

| Spec | Describes |
|------|-----------|
| [`agent.openapi.yaml`](openapi/agent.openapi.yaml) | Agent: `/health`, `/status`, `/sign-challenge` |
| [`backend.openapi.yaml`](openapi/backend.openapi.yaml) | Your backend: `/localid/challenge`, `/localid/verify` |

**Dev mode:** set `server.dev_mode: true` in agent config so the running agent serves the spec at `http://127.0.0.1:17443/openapi.json`. Your frontend tooling can point Orval at that live URL or at the checked-in YAML files.

Regenerate the shared TypeScript client:

```bash
pnpm generate:api
```

This syncs the spec into the Go binary (`scripts/sync-openapi.sh`) and runs [Orval](https://orval.dev/) in `@rqc-icu/localid-client`:

```typescript
import { agentOpenAPI, backendOpenAPI } from "@rqc-icu/localid-client";

const health = await agentOpenAPI.getHealth();
const { challenge } = (await backendOpenAPI.createChallenge()).data;
```

For your own app, add Orval with `input` pointing at `openapi/agent.openapi.yaml` or the live `openapi.json` endpoint. See `packages/localid-client/orval.config.ts` for a working config.

The hand-written helpers in `@rqc-icu/localid-client` (`signChallenge`, `fetchChallenge`, etc.) remain the simplest path for the examples; the Orval output is for teams that want generated types and operation functions.

---

## Documentation

| Topic | Guide |
|-------|-------|
| Architecture & API | [docs/architecture.md](docs/architecture.md) |
| Agent config & smoke tests | [docs/agent-config.md](docs/agent-config.md) |
| React browser demo | [docs/react-example.md](docs/react-example.md) |
| Vue browser demo | [examples/vue/README.md](examples/vue/README.md) |
| FastAPI backend | [examples/fastapi/README.md](examples/fastapi/README.md) |
| Laravel backend | [examples/laravel/README.md](examples/laravel/README.md) |
| All examples | [examples/README.md](examples/README.md) |
| Tauri desktop app | [docs/desktop.md](docs/desktop.md) |

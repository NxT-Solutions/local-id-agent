# Desktop app

Tauri 2 desktop shell for the LocalID Agent. The app bundles the Go agent as a sidecar, manages its config in the OS app data directory, and exposes a small React + shadcn UI.

## Prerequisites

- Node.js 20+ and pnpm 10+
- Go 1.25+ (to build the sidecar)
- **Rust toolchain** ([rustup](https://rustup.rs/)) for `tauri dev` / `tauri build`

Platform dependencies for Tauri are documented in the [Tauri prerequisites guide](https://v2.tauri.app/start/prerequisites/).

## Setup

From the repository root:

```bash
pnpm install
pnpm run build:sidecar
```

The sidecar script writes a platform-specific binary to `apps/desktop/src-tauri/binaries/localid-agent-<rust-target-triple>` (for example `localid-agent-aarch64-apple-darwin`).

## Development

### Full desktop app (recommended)

Requires Rust ([rustup](https://rustup.rs/)). From the repository root:

```bash
pnpm run build:sidecar          # once, or after agent code changes
pnpm --filter desktop tauri dev
```

This starts Vite on `http://localhost:1420`, launches the native window, starts the Go agent sidecar, and enables the system tray. On first run, the app copies `config.desktop.json` into the OS app data directory as `config.json` (includes `tauri://localhost` and `http://localhost:1420` in `allowed_origins`).

### UI only (no Rust)

Runs the React UI in the browser only — no Tauri shell, tray, or bundled agent:

```bash
pnpm run dev:desktop
# same as: pnpm --filter desktop dev
```

Run the agent separately if you need live `/health` and `/status`:

```bash
cd services/agent && go run ./cmd/localid-agent --config config.example.json
```

### Auth demo (`/demo`)

Also start the mock backend:

```bash
cd services/agent && go run ./cmd/mock-backend
```

## Build

```bash
pnpm run build:sidecar
pnpm --filter desktop tauri build
```

Installers/bundles are emitted under `apps/desktop/src-tauri/target/release/bundle/`.

## UI

| Route | Purpose |
|-------|---------|
| `/` | Dashboard — health/status polling, restart agent |
| `/settings` | Edit config fields, save, restart agent |
| `/about` | Version, config path, copy diagnostics JSON |
| `/demo` | Auth flow (requires mock backend on `:8000`) |

## System tray

Closing the window hides it to the tray. Tray menu: **Open**, **Restart Agent**, **Quit**.

## Environment

Optional Vite env vars (`.env` in `apps/desktop`):

- `VITE_AGENT_URL` — default `http://127.0.0.1:17443`
- `VITE_BACKEND_URL` — default `http://localhost:8000` (demo page)

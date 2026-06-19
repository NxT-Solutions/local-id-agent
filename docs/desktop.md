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
| `/` | Dashboard — health/status polling, restart agent (all users) |
| `/about` | Diagnostics — limited view for users; full view when admin unlocked |
| `/settings` | Edit config fields, save, restart agent (admin only) |
| `/setup` | Integration guide (admin only) |
| `/demo` | Auth flow (admin only; requires mock backend on `:8000`) |

### Admin passcode

On first launch (no `admin-lock.json` in the app data directory), the app shows a one-time admin passcode setup wizard. After that:

- **User mode (default):** Dashboard health/status, refresh, restart agent, limited diagnostics.
- **Admin mode (after unlock):** Settings, Setup, Demo, full diagnostics (including config path), config read/write.

Sensitive Tauri commands are gated in Rust — UI route hiding is not sufficient on its own. Idle admin sessions auto-lock after 15 minutes.

| App data file | Purpose |
|---------------|---------|
| `config.json` | Go agent configuration |
| `admin-lock.json` | Argon2id hash of the admin passcode (never plaintext) |

Default app data directory:

- **macOS:** `~/Library/Application Support/icu.rqc.localid-agent/`
- **Windows:** `%APPDATA%\icu.rqc.localid-agent\`
- **Linux:** `~/.local/share/icu.rqc.localid-agent/` (XDG)

## Enterprise deployment (Intune)

Recommended rollout for managed Windows fleets:

1. **Install** the desktop app via Intune Win32 or MSI from `pnpm --filter desktop tauri build` output.
2. **Immediately after install:** IT admin launches the app once, completes first-run passcode setup, configures Settings (origins, backends, provider), then locks admin before user handoff.
3. **Optional pre-placement:** drop an organization `config.json` into `%APPDATA%\icu.rqc.localid-agent\` before first launch (same pattern as `scripts/demo-native-eid.sh`).
4. **File ACL (Windows):** restrict write access on `%APPDATA%\icu.rqc.localid-agent\` to `Administrators` and `SYSTEM`; grant users read-only. This limits direct edits to `config.json` and `admin-lock.json`.
5. **PKCS#11 environment variables** (`LOCALID_PKCS11_PIN`, `LOCALID_BEID_PKCS11_MODULE`) — set via Intune device configuration; inherited by the sidecar process spawned from the Tauri shell.

Example PowerShell ACL template (run elevated, adjust if the folder does not exist yet):

```powershell
$dir = Join-Path $env:APPDATA "icu.rqc.localid-agent"
New-Item -ItemType Directory -Force -Path $dir | Out-Null
$acl = Get-Acl $dir
$acl.SetAccessRuleProtection($true, $false)
$acl.Access | ForEach-Object { $acl.RemoveAccessRule($_) | Out-Null }
$inherit = "ContainerInherit,ObjectInherit"
$acl.AddAccessRule((New-Object System.Security.AccessControl.FileSystemAccessRule(
  "SYSTEM", "FullControl", $inherit, "None", "Allow")))
$acl.AddAccessRule((New-Object System.Security.AccessControl.FileSystemAccessRule(
  "BUILTIN\Administrators", "FullControl", $inherit, "None", "Allow")))
$acl.AddAccessRule((New-Object System.Security.AccessControl.FileSystemAccessRule(
  "BUILTIN\Users", "ReadAndExecute", $inherit, "None", "Allow")))
Set-Acl $dir $acl
```

**Operational note:** whoever completes first-run setup on a machine becomes the admin gatekeeper for that device. For fleets, IT should open the app once post-install before end-user delivery.

## System tray

Closing the window hides it to the tray. Tray menu: **Open**, **Restart Agent**, **Quit**.

## Environment

Optional Vite env vars (`.env` in `apps/desktop`):

- `VITE_AGENT_URL` — default `http://127.0.0.1:17443`
- `VITE_BACKEND_URL` — default `http://localhost:8000` (demo page)

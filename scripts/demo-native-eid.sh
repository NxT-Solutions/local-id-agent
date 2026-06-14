#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

usage() {
  cat <<'EOF'
Usage: bash scripts/demo-native-eid.sh [--skip-sidecar-build]

Runs the recommended macOS flow:
- Stops Docker demo containers (frontend + agents); keeps backend
- Starts Docker backend on :8000
- Builds the desktop sidecar (unless --skip-sidecar-build)
- Launches Tauri desktop (NOT http://localhost:5173)
- Exports LOCALID_PKCS11_PIN when set in the environment
EOF
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Error: required command '$1' is not available." >&2
    exit 1
  fi
}

bold() {
  printf '\033[1m%s\033[0m\n' "$*"
}

SKIP_SIDECAR_BUILD="false"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --)
      shift
      break
      ;;
    --skip-sidecar-build)
      SKIP_SIDECAR_BUILD="true"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Error: unknown argument '$1'" >&2
      usage
      exit 1
      ;;
  esac
done

echo "============================================================"
echo " LocalID demo:native-eid"
echo " Native Tauri desktop + host sidecar + Docker backend only"
echo "============================================================"

require_command docker
require_command pnpm
require_command curl

if ! docker info >/dev/null 2>&1; then
  echo "Error: Docker is installed but not running." >&2
  exit 1
fi

if [[ -n "${LOCALID_PKCS11_PIN:-}" ]]; then
  export LOCALID_PKCS11_PIN
fi

echo
echo "1/5 Stopping Docker demo containers (frontend + agents); keeping backend..."
docker compose stop frontend agent agent-eid agent-pkcs11 2>/dev/null || true

if command -v lsof >/dev/null 2>&1; then
  frontend_listener="$(lsof -nP -iTCP:5173 -sTCP:LISTEN 2>/dev/null || true)"
  if [[ -n "${frontend_listener}" ]]; then
    echo
    bold "Note: something is still listening on :5173 (often a stale Docker frontend)."
    echo "Close http://localhost:5173 browser tabs — this demo uses the Tauri desktop window."
    echo "${frontend_listener}" >&2
  fi
fi

echo "2/5 Starting Docker backend on :8000..."
docker compose up -d backend

echo "   Waiting for backend readiness..."
backend_ready="false"
for _ in {1..30}; do
  if curl -fsS -X POST "http://127.0.0.1:8000/localid/challenge" \
    -H "Content-Type: application/json" \
    -d '{}' >/dev/null 2>&1; then
    backend_ready="true"
    break
  fi
  sleep 1
done

if [[ "${backend_ready}" != "true" ]]; then
  echo "Error: backend did not become ready on http://127.0.0.1:8000." >&2
  echo "Run 'docker compose logs backend' for details." >&2
  exit 1
fi

if command -v lsof >/dev/null 2>&1; then
  listener_output="$(lsof -nP -iTCP:17443 -sTCP:LISTEN 2>/dev/null || true)"
  if [[ -n "${listener_output}" ]]; then
    echo "Error: port 17443 is already in use. Stop the process before running this demo." >&2
    echo "${listener_output}" >&2
    exit 1
  fi
fi

echo "3/5 Backend is ready and port 17443 is free."

if [[ "${SKIP_SIDECAR_BUILD}" != "true" ]]; then
  echo "4/5 Building desktop sidecar..."
  pnpm run build:sidecar
else
  echo "4/5 Skipping sidecar build (--skip-sidecar-build)."
fi

if [[ -z "${LOCALID_PKCS11_PIN:-}" ]]; then
  if [[ -t 0 ]]; then
    echo
    read -r -s -p "Enter LOCALID_PKCS11_PIN: " LOCALID_PKCS11_PIN
    echo
  fi
fi

if [[ -z "${LOCALID_PKCS11_PIN:-}" ]]; then
  echo "Error: LOCALID_PKCS11_PIN is required for eID signing." >&2
  echo "Example: LOCALID_PKCS11_PIN=1234 pnpm demo:native-eid" >&2
  exit 1
fi

export LOCALID_PKCS11_PIN

echo "5/5 Launching Tauri desktop (foreground)..."
echo
bold "Use the Tauri desktop window, NOT http://localhost:5173"
echo "  :5173 is the Docker/browser demo (demo:docker-eid). This command opens the native app."
echo
echo "In the desktop app:"
echo "  1. Settings → Belgian eID → Save (if provider still shows mock from a prior run)"
echo "  2. Demo tab → run the full auth flow"
echo
echo "Note: pnpm run dev:desktop (browser :1420) does not auto-start the sidecar."
exec pnpm --filter desktop tauri dev

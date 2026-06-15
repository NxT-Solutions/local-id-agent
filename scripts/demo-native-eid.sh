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
- Starts React dev server on :5173 (browser demo; uses host sidecar on :17443)
- Launches Tauri desktop on :1420
- Exports LOCALID_PKCS11_PIN when set in the environment
- Auto-discovers LOCALID_BEID_PKCS11_MODULE on macOS/Linux when unset
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

REACT_DEV_PID=""

cleanup() {
  if [[ -n "${REACT_DEV_PID}" ]]; then
    kill "${REACT_DEV_PID}" 2>/dev/null || true
    wait "${REACT_DEV_PID}" 2>/dev/null || true
  fi
}

trap cleanup EXIT INT TERM

port_is_listening() {
  local port="$1"
  if command -v lsof >/dev/null 2>&1; then
    lsof -nP -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1
    return $?
  fi
  curl -fsS "http://localhost:${port}/" >/dev/null 2>&1
}

start_react_dev_server() {
  if port_is_listening 5173; then
    echo
    bold "Warning: port 5173 is already in use; skipping React dev server."
    echo "Free the port or stop the other process to use http://localhost:5173."
    if command -v lsof >/dev/null 2>&1; then
      lsof -nP -iTCP:5173 -sTCP:LISTEN 2>/dev/null || true
    fi
    return 0
  fi

  echo "Starting React dev server on http://localhost:5173 ..."
  pnpm --filter localid-react-example dev >/tmp/localid-demo-native-eid-react.log 2>&1 &
  REACT_DEV_PID=$!

  react_ready="false"
  for _ in {1..30}; do
    if curl -fsS "http://localhost:5173/" >/dev/null 2>&1; then
      react_ready="true"
      break
    fi
    if ! kill -0 "${REACT_DEV_PID}" 2>/dev/null; then
      echo "Error: React dev server exited before becoming ready." >&2
      echo "See /tmp/localid-demo-native-eid-react.log" >&2
      REACT_DEV_PID=""
      exit 1
    fi
    sleep 1
  done

  if [[ "${react_ready}" != "true" ]]; then
    echo "Error: React dev server did not become ready on http://localhost:5173." >&2
    echo "See /tmp/localid-demo-native-eid-react.log" >&2
    exit 1
  fi

  echo "React dev server is ready at http://localhost:5173"
}

discover_beid_pkcs11_module() {
  local candidates=(
    "/Library/Belgium Identity Card/Pkcs11/libbeidpkcs11.dylib"
    "/Library/Belgium Identity Card/Pkcs11/beid-pkcs11.bundle/Contents/MacOS/libbeidpkcs11.dylib"
    "/opt/homebrew/lib/libbeidpkcs11.dylib"
    "/usr/local/lib/libbeidpkcs11.dylib"
    "/usr/lib/libbeidpkcs11.so"
    "/usr/lib/x86_64-linux-gnu/libbeidpkcs11.so"
    "/usr/lib64/libbeidpkcs11.so"
  )
  local candidate
  for candidate in "${candidates[@]}"; do
    if [[ -f "${candidate}" ]]; then
      echo "${candidate}"
      return 0
    fi
  done
  return 1
}

reconcile_beid_pkcs11_module() {
  local configured="${LOCALID_BEID_PKCS11_MODULE:-}"
  if [[ -n "${configured}" ]]; then
    if [[ ! -f "${configured}" ]]; then
      echo "Error: LOCALID_BEID_PKCS11_MODULE is set but not found:" >&2
      echo "  ${configured}" >&2
      echo "Unset it to use auto-discovery, or set the real path. On macOS after installing Belgian eID middleware:" >&2
      echo '  export LOCALID_BEID_PKCS11_MODULE="/Library/Belgium Identity Card/Pkcs11/libbeidpkcs11.dylib"' >&2
      exit 1
    fi
    export LOCALID_BEID_PKCS11_MODULE="${configured}"
    echo "Using LOCALID_BEID_PKCS11_MODULE=${LOCALID_BEID_PKCS11_MODULE}"
    return 0
  fi

  local discovered
  if discovered="$(discover_beid_pkcs11_module)"; then
    export LOCALID_BEID_PKCS11_MODULE="${discovered}"
    echo "Auto-discovered Belgian eID PKCS#11 module:"
    echo "  ${LOCALID_BEID_PKCS11_MODULE}"
    return 0
  fi

  echo "Warning: Belgian eID PKCS#11 module not found on this machine." >&2
  echo "Install Belgian eID middleware, or set LOCALID_BEID_PKCS11_MODULE explicitly." >&2
}

DESKTOP_CONFIG_PATH="${HOME}/Library/Application Support/icu.rqc.localid-agent/config.json"


reconcile_desktop_native_eid_config() {
  if [[ ! -f "${DESKTOP_CONFIG_PATH}" ]]; then
    return 0
  fi

  local reconcile_status
  reconcile_status=0
  python3 - "${DESKTOP_CONFIG_PATH}" <<'PY' || reconcile_status=$?
import json
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
try:
    raw = path.read_text(encoding="utf-8")
    data = json.loads(raw)
except Exception:
    raise SystemExit(10)

if not isinstance(data, dict):
    raise SystemExit(10)

providers = data.get("providers")
if not isinstance(providers, dict):
    providers = {}
    data["providers"] = providers

changed = False

if providers.get("default") != "belgian_eid":
    providers["default"] = "belgian_eid"
    changed = True

belgian = providers.get("belgian_eid")
if not isinstance(belgian, dict):
    belgian = {}
    providers["belgian_eid"] = belgian
    changed = True

if belgian.get("enabled") is not True:
    belgian["enabled"] = True
    changed = True

if not changed:
    raise SystemExit(0)

path.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")
raise SystemExit(11)
PY

  case "${reconcile_status}" in
    0)
      return 0
      ;;
    10)
      echo
      bold "Warning: desktop config is invalid JSON:"
      echo "  ${DESKTOP_CONFIG_PATH}"
      if [[ -t 0 ]]; then
        read -r -p "Delete this invalid config and continue with defaults? [y/N] " answer
        if [[ "${answer}" =~ ^[Yy]$ ]]; then
          rm -f "${DESKTOP_CONFIG_PATH}"
          echo "Deleted invalid desktop config; Tauri will recreate it on launch."
          return 0
        fi
      fi
      echo "Aborting. Fix or remove the file, then run pnpm demo:native-eid again." >&2
      exit 1
      ;;
    11)
      echo "Adjusted desktop config for native eID (default provider + enabled flag)."
      return 0
      ;;
    *)
      echo "Error: failed to reconcile desktop config for native eID." >&2
      exit 1
      ;;
  esac
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
echo " Native Tauri desktop + host sidecar + Docker backend"
echo " Browser React demo on :5173 (optional, in parallel)"
echo "============================================================"

require_command docker
require_command pnpm
require_command curl
require_command python3

if ! docker info >/dev/null 2>&1; then
  echo "Error: Docker is installed but not running." >&2
  exit 1
fi

if [[ -n "${LOCALID_PKCS11_PIN:-}" ]]; then
  export LOCALID_PKCS11_PIN
fi

echo
echo "1/7 Reconciling desktop provider config for native eID..."
reconcile_desktop_native_eid_config
reconcile_beid_pkcs11_module

echo "2/7 Stopping Docker demo containers (frontend + agents); keeping backend..."
docker compose stop frontend agent agent-eid agent-pkcs11 2>/dev/null || true

echo "3/7 Starting Docker backend on :8000..."
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

echo "4/7 Backend is ready and port 17443 is free."

if [[ "${SKIP_SIDECAR_BUILD}" != "true" ]]; then
  echo "5/7 Building desktop sidecar..."
  pnpm run build:sidecar
else
  echo "5/7 Skipping sidecar build (--skip-sidecar-build)."
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

echo "6/7 Starting browser dev server (React on :5173)..."
start_react_dev_server

echo "7/7 Launching Tauri desktop (foreground)..."
echo
bold "Tauri desktop: primary flow (sidecar agent on :17443)"
echo "  http://localhost:1420 — desktop UI (via tauri dev)"
if [[ -n "${REACT_DEV_PID}" ]]; then
  echo "  http://localhost:5173 — React browser demo (same host agent + backend)"
fi
echo
echo "In the desktop app:"
echo "  1. Dashboard → confirm health/status update"
echo "  2. If needed, click Restart agent"
echo "  3. Demo tab → run the full auth flow"
echo
echo "Note: pnpm run dev:desktop (browser :1420) does not auto-start the sidecar."
pnpm --filter desktop tauri dev

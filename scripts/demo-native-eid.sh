#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "═══════════════════════════════════════════════════════════════"
echo " LocalID — Native desktop + Belgian eID (macOS recommended)"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "This starts:"
echo "  • Docker mock backend only (:8000)"
echo "  • Rebuilt Go agent sidecar in the Tauri app (:17443)"
echo "  • Tauri desktop window (use Demo tab — NOT browser :1420)"
echo ""
echo "Stops Docker agents on :17443 to avoid port conflicts."
echo ""

if ! command -v docker >/dev/null 2>&1; then
  echo "error: docker is required for the mock backend container" >&2
  exit 1
fi

if [[ -z "${LOCALID_PKCS11_PIN:-}" ]]; then
  echo "warning: LOCALID_PKCS11_PIN is not set."
  echo "         Signing with a real card will fail unless PIN is provided."
  echo "         Example: LOCALID_PKCS11_PIN=1234 pnpm demo:native-eid"
  echo ""
fi

echo "→ Stopping Docker agent containers (free :17443)..."
docker compose stop agent agent-eid agent-pkcs11 2>/dev/null || true

echo "→ Starting Docker backend on :8000..."
docker compose up -d backend

echo "→ Waiting for backend..."
for _ in $(seq 1 30); do
  if curl -fsS "http://localhost:8000/localid/challenge" \
    -X POST -H "Content-Type: application/json" -d '{}' >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

if [[ "${SKIP_SIDECAR_BUILD:-}" != "1" ]]; then
  echo "→ Building agent sidecar binary..."
  pnpm run build:sidecar
else
  echo "→ Skipping sidecar build (SKIP_SIDECAR_BUILD=1)"
fi

echo ""
echo "→ Launching Tauri desktop..."
echo "   In the app: Settings → Belgian eID → Save → Demo → Authenticate"
echo ""

exec pnpm --filter desktop tauri dev

#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "═══════════════════════════════════════════════════════════════"
echo " LocalID — Browser demo + Belgian eID in Docker (Linux best)"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "This starts:"
echo "  • Docker backend (:8000)"
echo "  • Docker frontend (:5173)"
echo "  • Docker agent-eid with PC/SC passthrough (:17443)"
echo ""
echo "Do NOT run the Tauri desktop app at the same time (port :17443)."
echo "Open http://localhost:5173 after containers are up."
echo ""

if [[ -z "${LOCALID_PKCS11_PIN:-}" ]]; then
  echo "warning: LOCALID_PKCS11_PIN is not set."
  echo "         Container signing needs a PIN for non-interactive use."
  echo "         Example: LOCALID_PKCS11_PIN=1234 pnpm demo:docker-eid"
  echo ""
fi

if lsof -i :17443 >/dev/null 2>&1; then
  echo "warning: something is already listening on :17443."
  echo "         Stop Tauri desktop or host agent before continuing."
  echo ""
fi

exec docker compose --profile demo-eid-container up --build backend frontend agent-eid

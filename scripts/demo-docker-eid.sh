#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

usage() {
  cat <<'EOF'
Usage: bash scripts/demo-docker-eid.sh

Runs the browser + containerized eID flow:
- Starts backend + frontend + agent-eid container profile
- Exposes frontend at http://localhost:5173
EOF
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Error: required command '$1' is not available." >&2
    exit 1
  fi
}

if [[ "${1:-}" == "--" ]]; then
  shift
fi

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ $# -gt 0 ]]; then
  echo "Error: unknown argument '$1'" >&2
  usage
  exit 1
fi

echo "============================================================"
echo " LocalID demo:docker-eid"
echo " Browser demo + Docker backend/frontend/agent-eid container"
echo "============================================================"

require_command docker

if ! docker info >/dev/null 2>&1; then
  echo "Error: Docker is installed but not running." >&2
  exit 1
fi

if command -v lsof >/dev/null 2>&1; then
  listener_output="$(lsof -nP -iTCP:17443 -sTCP:LISTEN 2>/dev/null || true)"
  if [[ -n "${listener_output}" ]]; then
    non_docker_output="$(printf "%s\n" "${listener_output}" | awk 'NR==1 || ($1 !~ /(docker|vpnkit|com\.docker)/)')"
    if [[ "$(printf "%s\n" "${non_docker_output}" | wc -l | tr -d ' ')" -gt 1 ]]; then
      echo "Warning: port 17443 appears to be used by a non-Docker process." >&2
      echo "Stop native Tauri/host agent before running demo:docker-eid." >&2
      echo "${non_docker_output}" >&2
    fi
  fi
fi

if [[ -z "${LOCALID_PKCS11_PIN:-}" ]]; then
  if [[ -t 0 ]]; then
    echo
    read -r -s -p "Enter LOCALID_PKCS11_PIN: " LOCALID_PKCS11_PIN
    echo
  fi
fi

if [[ -z "${LOCALID_PKCS11_PIN:-}" ]]; then
  echo "Error: LOCALID_PKCS11_PIN is required for containerized eID signing." >&2
  echo "Example: LOCALID_PKCS11_PIN=1234 pnpm demo:docker-eid" >&2
  exit 1
fi

echo "Starting docker compose profile: demo-eid-container..."
echo "Do not run Tauri at the same time to avoid confusion/port conflicts."
echo "Open http://localhost:5173 once services are up."

LOCALID_PKCS11_PIN="${LOCALID_PKCS11_PIN}" \
LOCALID_BEID_PKCS11_MODULE="${LOCALID_BEID_PKCS11_MODULE:-}" \
docker compose --profile demo-eid-container up --build

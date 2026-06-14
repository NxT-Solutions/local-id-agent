#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SRC="$ROOT/openapi/agent.openapi.yaml"
DEST="$ROOT/services/agent/internal/openapi/spec/agent.openapi.yaml"

cp "$SRC" "$DEST"
echo "Synced OpenAPI spec to $DEST"

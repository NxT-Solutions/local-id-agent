#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${ROOT}/apps/desktop/src-tauri/binaries"
mkdir -p "${OUT_DIR}"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "${OS}-${ARCH}" in
  darwin-arm64)
    TARGET="aarch64-apple-darwin"
    ;;
  darwin-x86_64)
    TARGET="x86_64-apple-darwin"
    ;;
  linux-x86_64|linux-amd64)
    TARGET="x86_64-unknown-linux-gnu"
    ;;
  linux-aarch64|linux-arm64)
    TARGET="aarch64-unknown-linux-gnu"
    ;;
  mingw*|msys*|cygwin*|windows_nt-*)
    TARGET="x86_64-pc-windows-msvc"
    ;;
  *)
    echo "Unsupported platform: ${OS}-${ARCH}" >&2
    exit 1
    ;;
esac

OUTPUT="${OUT_DIR}/localid-agent-${TARGET}"
if [[ "${TARGET}" == *"windows"* ]]; then
  OUTPUT="${OUTPUT}.exe"
fi

echo "Building localid-agent sidecar for ${TARGET}..."
(
  cd "${ROOT}/services/agent"
  go build -o "${OUTPUT}" ./cmd/localid-agent
)

echo "Sidecar written to ${OUTPUT}"

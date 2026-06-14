#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
AGENT_DIR="${ROOT_DIR}/services/agent"

cd "${AGENT_DIR}"
go test ./... -coverprofile=coverage.out

coverage_report="$(go tool cover -func=coverage.out)"
echo "${coverage_report}"

failed=0
while IFS= read -r line; do
  case "${line}" in
    total:*|"")
      continue
      ;;
  esac

  target="$(printf '%s' "${line}" | awk '{print $1}')"
  percent="$(printf '%s' "${line}" | awk '{print $NF}')"
  value="${percent%\%}"

  # Generated protobuf code is excluded from coverage enforcement.
  if [[ "${target}" == *"/internal/protocol/pb/"* ]]; then
    continue
  fi

  if awk "BEGIN { exit !(${value} < 100) }"; then
    echo "coverage below 100%: ${target} (${percent})"
    failed=1
  fi
done <<< "${coverage_report}"

if [[ "${failed}" -ne 0 ]]; then
  echo "Go coverage enforcement failed."
  exit 1
fi

echo "Go coverage enforcement passed."


#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(CDPATH= cd -- "${SCRIPT_DIR}/.." && pwd)"

CONFIG_PATH="${1:-${WHATOMATE_CONFIG:-${REPO_ROOT}/config.toml}}"
WORKERS="${WHATOMATE_WORKERS:-0}"
GO_BIN="${GO_BIN:-go}"
EMBEDDED_INDEX="${REPO_ROOT}/internal/frontend/dist/index.html"

cd "${REPO_ROOT}"

if [[ "${WHATOMATE_SKIP_FRONTEND_BUILD:-0}" != "1" ]]; then
  needs_embed=0
  if [[ ! -f "${EMBEDDED_INDEX}" ]]; then
    needs_embed=1
  elif find frontend/src frontend/public frontend/index.html frontend/package.json frontend/package-lock.json frontend/vite.config.ts -type f -newer "${EMBEDDED_INDEX}" -print -quit 2>/dev/null | grep -q .; then
    needs_embed=1
  fi

  if [[ "${needs_embed}" -eq 1 ]]; then
    make frontend-build embed-frontend
  fi
fi

exec env GOTOOLCHAIN="${GOTOOLCHAIN:-auto}" "${GO_BIN}" run ./cmd/whatomate server -config "${CONFIG_PATH}" -workers "${WORKERS}"

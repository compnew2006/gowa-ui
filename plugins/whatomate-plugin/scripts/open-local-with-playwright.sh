#!/usr/bin/env bash

set -euo pipefail

if ! command -v npx >/dev/null 2>&1; then
  echo "npx is required to run the Playwright wrapper." >&2
  exit 1
fi

CODEX_HOME="${CODEX_HOME:-$HOME/.codex}"
PWCLI="${CODEX_HOME}/skills/playwright/scripts/playwright_cli.sh"
URL="${1:-${WHATOMATE_FRONTEND_URL:-http://127.0.0.1:3000}}"

if [[ ! -x "${PWCLI}" ]]; then
  echo "Playwright wrapper not found at ${PWCLI}" >&2
  exit 1
fi

exec "${PWCLI}" open "${URL}" --headed

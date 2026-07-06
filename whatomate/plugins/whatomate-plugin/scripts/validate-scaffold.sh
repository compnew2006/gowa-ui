#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
PLUGIN_ROOT="${ROOT}/plugins/whatomate-plugin"
PLUGIN_JSON="${PLUGIN_ROOT}/.codex-plugin/plugin.json"
MARKETPLACE_JSON="${ROOT}/.agents/plugins/marketplace.json"
SKILL_FILE="${PLUGIN_ROOT}/skills/whatomate-browser-automation/SKILL.md"
PLAYWRIGHT_SCRIPT="${PLUGIN_ROOT}/scripts/open-local-with-playwright.sh"

for path in "${PLUGIN_JSON}" "${MARKETPLACE_JSON}" "${SKILL_FILE}" "${PLAYWRIGHT_SCRIPT}"; do
  [[ -f "${path}" ]] || {
    echo "Missing required file: ${path}" >&2
    exit 1
  }
done

python3 - <<'PY' "${PLUGIN_JSON}" "${MARKETPLACE_JSON}"
import json
import pathlib
import sys

plugin_path = pathlib.Path(sys.argv[1])
marketplace_path = pathlib.Path(sys.argv[2])

plugin = json.loads(plugin_path.read_text())
marketplace = json.loads(marketplace_path.read_text())

assert plugin["name"] == "whatomate-plugin"
assert plugin["skills"] == "./skills/"
assert plugin["mcpServers"] == "./.mcp.json"
assert plugin["apps"] == "./.app.json"
assert any("Chrome DevTools" in prompt or "Playwright" in prompt for prompt in plugin["interface"]["defaultPrompt"])

entry = marketplace["plugins"][0]
assert entry["name"] == "whatomate-plugin"
assert entry["source"]["path"] == "./plugins/whatomate-plugin"
assert entry["policy"]["installation"] == "AVAILABLE"
assert entry["policy"]["authentication"] == "ON_INSTALL"
assert entry["category"] == "Productivity"
PY

bash -n "${PLAYWRIGHT_SCRIPT}"

#!/usr/bin/env bash
set -euo pipefail

STACK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${STACK_DIR}/../../.." && pwd)"
COMPOSE_FILE="${STACK_DIR}/docker-compose.yml"
PROJECT_NAME="${CAG_SETTINGS_E2E_PROJECT:-cag-settings-e2e}"
GATEWAY_PORT="${CAG_SETTINGS_E2E_GATEWAY_PORT:-18081}"
CONSOLE_PORT="${CAG_SETTINGS_E2E_CONSOLE_PORT:-14174}"
GATEWAY_API_KEY="${CAG_GATEWAY_API_KEY:-settings-e2e-key}"
MACHINE_NAME="${CAG_MACHINE_NAME:-Settings E2E Client}"
TMP_ROOT="${STACK_DIR}/.tmp/${PROJECT_NAME}"
REPORT_PATH="${TMP_ROOT}/playwright-settings-report.json"

rm -rf "${TMP_ROOT}"
mkdir -p "${TMP_ROOT}/client-home" "${TMP_ROOT}/gateway-data"

export CAG_SETTINGS_E2E_CLIENT_HOME="${TMP_ROOT}/client-home"
export CAG_SETTINGS_E2E_GATEWAY_DATA="${TMP_ROOT}/gateway-data"
export GATEWAY_API_KEY

compose() {
  docker compose -p "${PROJECT_NAME}" -f "${COMPOSE_FILE}" "$@"
}

wait_for_http() {
  local url="$1"
  local timeout="${2:-120}"
  local started_at
  started_at="$(date +%s)"
  until curl -fsS "${url}" >/dev/null 2>&1; do
    if (( "$(date +%s)" - started_at >= timeout )); then
      echo "Timed out waiting for ${url}" >&2
      return 1
    fi
    sleep 2
  done
}

wait_for_machine() {
  local url="http://localhost:${GATEWAY_PORT}/machines"
  local authorization_header="Bearer ${GATEWAY_API_KEY}"
  local machine_name="${1}"
  local timeout="${2:-120}"
  local started_at
  started_at="$(date +%s)"
  until python3 - "${url}" "${machine_name}" "${authorization_header}" <<'PY'
import json
import sys
import urllib.request

request = urllib.request.Request(
    sys.argv[1],
    headers={"Authorization": sys.argv[3]},
)
with urllib.request.urlopen(request, timeout=3) as response:
    payload = json.load(response)

for item in payload.get("items", []):
    if item.get("name") == sys.argv[2] and item.get("status") == "online":
        raise SystemExit(0)

raise SystemExit(1)
PY
  do
    if (( "$(date +%s)" - started_at >= timeout )); then
      echo "Timed out waiting for ${MACHINE_NAME} registration" >&2
      return 1
    fi
    sleep 2
  done
}

cleanup() {
  compose down --remove-orphans >/dev/null 2>&1 || true
}

trap cleanup EXIT

compose up --build -d
wait_for_http "http://localhost:${GATEWAY_PORT}/health"
wait_for_http "http://localhost:${CONSOLE_PORT}"
wait_for_machine "${MACHINE_NAME}"

PLAYWRIGHT_BASE_URL="http://127.0.0.1:${CONSOLE_PORT}" \
SETTINGS_E2E_CLIENT_HOME="${CAG_SETTINGS_E2E_CLIENT_HOME}" \
SETTINGS_E2E_GATEWAY_URL="http://127.0.0.1:${CONSOLE_PORT}/api" \
SETTINGS_E2E_GATEWAY_API_KEY="${GATEWAY_API_KEY}" \
corepack pnpm --dir "${REPO_ROOT}/console" exec playwright test --config playwright.settings.config.ts --reporter=json > "${REPORT_PATH}"

node - "${REPORT_PATH}" <<'NODE'
const fs = require("node:fs");
const reportPath = process.argv[2];
const report = JSON.parse(fs.readFileSync(reportPath, "utf8"));

let total = 0;
let passed = 0;

function walkSuite(suite) {
  for (const spec of suite.specs ?? []) {
    for (const test of spec.tests ?? []) {
      total += 1;
      if ((test.results ?? []).some((result) => result.status === "passed")) {
        passed += 1;
      }
    }
  }
  for (const child of suite.suites ?? []) {
    walkSuite(child);
  }
}

for (const suite of report.suites ?? []) {
  walkSuite(suite);
}

if (total < 4) {
  console.error(`expected at least 4 settings e2e scenarios, got ${total}`);
  process.exit(1);
}
if (passed !== total) {
  console.error(`expected all e2e scenarios to pass, got ${passed}/${total}`);
  process.exit(1);
}

console.log(`settings e2e scenarios passed: ${passed}/${total}`);
NODE

echo "Settings E2E completed."
echo "Client home: ${CAG_SETTINGS_E2E_CLIENT_HOME}"

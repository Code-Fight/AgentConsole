#!/usr/bin/env bash
set -euo pipefail

STACK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="${STACK_DIR}/docker-compose.yml"
PROJECT_NAME="${CAG_TESTENV_PROJECT:-cag-dev-integration}"
GATEWAY_PORT="${CAG_GATEWAY_PORT:-18080}"
CONSOLE_PORT="${CAG_CONSOLE_PORT:-14173}"
MACHINE_NAME="${CAG_MACHINE_NAME:-Dev Integration Client}"
GATEWAY_API_KEY="${CAG_GATEWAY_API_KEY:-dev-integration-key}"
GATEWAY_URL="http://localhost:${GATEWAY_PORT}"
CONSOLE_URL="http://localhost:${CONSOLE_PORT}"
export CAG_CLIENT_RUNTIME_MODE="${CAG_CLIENT_RUNTIME_MODE:-appserver}"
export GATEWAY_API_KEY

compose() {
  docker compose -p "${PROJECT_NAME}" -f "${COMPOSE_FILE}" "$@"
}

wait_for_http() {
  local name="$1"
  local url="$2"
  local timeout="${3:-120}"
  local started_at
  started_at="$(date +%s)"

  until curl -fsS "${url}" >/dev/null 2>&1; do
    if (( "$(date +%s)" - started_at >= timeout )); then
      echo "Timed out waiting for ${name} at ${url}" >&2
      return 1
    fi
    sleep 2
  done
}

wait_for_machine() {
  local timeout="${1:-120}"
  local started_at
  started_at="$(date +%s)"

  until python3 - "${GATEWAY_URL}/machines" "${MACHINE_NAME}" "${GATEWAY_API_KEY}" <<'PY'
import json
import sys
import urllib.request

url, machine_name, gateway_api_key = sys.argv[1], sys.argv[2], sys.argv[3]
request = urllib.request.Request(
    url,
    headers={"Authorization": f"Bearer {gateway_api_key}"},
)
with urllib.request.urlopen(request, timeout=3) as response:
    payload = json.load(response)

for item in payload.get("items", []):
    if item.get("name") == machine_name and item.get("status") == "online":
        raise SystemExit(0)

raise SystemExit(1)
PY
  do
    if (( "$(date +%s)" - started_at >= timeout )); then
      echo "Timed out waiting for client machine ${MACHINE_NAME} to register" >&2
      return 1
    fi
    sleep 2
  done
}

fetch_machine_id() {
  python3 - "${GATEWAY_URL}/machines" "${MACHINE_NAME}" "${GATEWAY_API_KEY}" <<'PY'
import json
import sys
import urllib.request

url, machine_name, gateway_api_key = sys.argv[1], sys.argv[2], sys.argv[3]
request = urllib.request.Request(
    url,
    headers={"Authorization": f"Bearer {gateway_api_key}"},
)
with urllib.request.urlopen(request, timeout=3) as response:
    payload = json.load(response)

for item in payload.get("items", []):
    if item.get("name") == machine_name:
        print(item.get("id", ""))
        raise SystemExit(0)

raise SystemExit(1)
PY
}

cmd="${1:-up}"

case "${cmd}" in
  up)
    compose up --build -d
    wait_for_http "gateway health" "${GATEWAY_URL}/health"
    wait_for_http "console" "${CONSOLE_URL}"
    wait_for_machine
    machine_id="$(fetch_machine_id)"
    echo "Gateway: ${GATEWAY_URL}"
    echo "Console: ${CONSOLE_URL}"
    echo "Machine Name: ${MACHINE_NAME}"
    echo "Machine ID: ${machine_id}"
    echo
    echo "Quick checks:"
    echo "  curl -H \"Authorization: Bearer ${GATEWAY_API_KEY}\" ${GATEWAY_URL}/machines"
    echo "  curl -H \"Authorization: Bearer ${GATEWAY_API_KEY}\" ${GATEWAY_URL}/threads"
    ;;
  down)
    compose down --remove-orphans
    ;;
  restart)
    compose down --remove-orphans
    compose up --build -d
    wait_for_http "gateway health" "${GATEWAY_URL}/health"
    wait_for_http "console" "${CONSOLE_URL}"
    wait_for_machine
    machine_id="$(fetch_machine_id)"
    echo "Restarted stack at ${GATEWAY_URL} and ${CONSOLE_URL} (${MACHINE_NAME} / ${machine_id})"
    ;;
  logs)
    compose logs -f
    ;;
  ps|status)
    compose ps
    ;;
  *)
    echo "Usage: $0 {up|down|restart|logs|ps|status}" >&2
    exit 1
    ;;
esac

#!/usr/bin/env bash
set -euo pipefail

STACK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="${STACK_DIR}/docker-compose.yml"
PROJECT_NAME="${CAG_TESTENV_PROJECT:-cag-dev-integration}"
GATEWAY_PORT="${CAG_GATEWAY_PORT:-18080}"
CONSOLE_PORT="${CAG_CONSOLE_PORT:-14173}"
MACHINE_ID="${CAG_MACHINE_ID:-machine-01}"
GATEWAY_URL="http://localhost:${GATEWAY_PORT}"
CONSOLE_URL="http://localhost:${CONSOLE_PORT}"
export CAG_CLIENT_RUNTIME_MODE="${CAG_CLIENT_RUNTIME_MODE:-appserver}"

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

  until python3 - "${GATEWAY_URL}/machines" "${MACHINE_ID}" <<'PY'
import json
import sys
import urllib.request

url, machine_id = sys.argv[1], sys.argv[2]
with urllib.request.urlopen(url, timeout=3) as response:
    payload = json.load(response)

for item in payload.get("items", []):
    if item.get("id") == machine_id and item.get("status") == "online":
        raise SystemExit(0)

raise SystemExit(1)
PY
  do
    if (( "$(date +%s)" - started_at >= timeout )); then
      echo "Timed out waiting for client machine ${MACHINE_ID} to register" >&2
      return 1
    fi
    sleep 2
  done
}

cmd="${1:-up}"

case "${cmd}" in
  up)
    compose up --build -d
    wait_for_http "gateway health" "${GATEWAY_URL}/health"
    wait_for_http "console" "${CONSOLE_URL}"
    wait_for_machine
    echo "Gateway: ${GATEWAY_URL}"
    echo "Console: ${CONSOLE_URL}"
    echo "Machine: ${MACHINE_ID}"
    echo
    echo "Quick checks:"
    echo "  curl ${GATEWAY_URL}/machines"
    echo "  curl ${GATEWAY_URL}/threads"
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
    echo "Restarted stack at ${GATEWAY_URL} and ${CONSOLE_URL}"
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

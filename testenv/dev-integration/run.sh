#!/usr/bin/env bash
set -euo pipefail

STACK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="${STACK_DIR}/docker-compose.yml"
PROJECT_NAME="${CAG_TESTENV_PROJECT:-cag-dev-integration}"
GATEWAY_PORT="${CAG_GATEWAY_PORT:-18080}"
CONSOLE_PORT="${CAG_CONSOLE_PORT:-14173}"
DEFAULT_MACHINE_NAME="${CAG_MACHINE_NAME_DEFAULT:-Dev Integration Client}"
HOSTNAME_FALLBACK_NAME="${CAG_HOSTNAME_FALLBACK_NAME:-hostname-fallback-client}"
NOT_AGENT_MACHINE_NAME="${CAG_MACHINE_NAME_NOT_AGENT:-Not Agent}"
GATEWAY_URL="http://localhost:${GATEWAY_PORT}"
CONSOLE_URL="http://localhost:${CONSOLE_PORT}"
export CAG_CLIENT_RUNTIME_MODE="${CAG_CLIENT_RUNTIME_MODE:-appserver}"
EXPECTED_MACHINE_NAMES=(
  "${DEFAULT_MACHINE_NAME}"
  "${HOSTNAME_FALLBACK_NAME}"
  "${NOT_AGENT_MACHINE_NAME}"
)

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

wait_for_machines() {
  local timeout="${1:-120}"
  local started_at
  started_at="$(date +%s)"

  until python3 - "${GATEWAY_URL}/machines" "${EXPECTED_MACHINE_NAMES[@]}" <<'PY'
import json
import sys
import urllib.request

url = sys.argv[1]
expected_names = sys.argv[2:]
with urllib.request.urlopen(url, timeout=3) as response:
    payload = json.load(response)

online_names = {
    item.get("name")
    for item in payload.get("items", [])
    if item.get("status") == "online"
}
missing = [name for name in expected_names if name not in online_names]
if not missing:
    raise SystemExit(0)

raise SystemExit(1)
PY
  do
    if (( "$(date +%s)" - started_at >= timeout )); then
      echo "Timed out waiting for client machines: ${EXPECTED_MACHINE_NAMES[*]}" >&2
      return 1
    fi
    sleep 2
  done
}

fetch_machine_rows() {
  python3 - "${GATEWAY_URL}/machines" "${EXPECTED_MACHINE_NAMES[@]}" <<'PY'
import json
import sys
import urllib.request

url = sys.argv[1]
expected_names = sys.argv[2:]
with urllib.request.urlopen(url, timeout=3) as response:
    payload = json.load(response)

items = {
    item.get("name"): item.get("id", "")
    for item in payload.get("items", [])
}
for name in expected_names:
    print(f"{name}\t{items.get(name, '')}")
PY
}

print_machine_summary() {
  while IFS=$'\t' read -r machine_name machine_id; do
    echo "Machine Name: ${machine_name}"
    echo "Machine ID: ${machine_id}"
  done < <(fetch_machine_rows)
}

cmd="${1:-up}"

case "${cmd}" in
  up)
    compose up --build -d
    wait_for_http "gateway health" "${GATEWAY_URL}/health"
    wait_for_http "console" "${CONSOLE_URL}"
    wait_for_machines
    echo "Gateway: ${GATEWAY_URL}"
    echo "Console: ${CONSOLE_URL}"
    print_machine_summary
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
    wait_for_machines
    echo "Restarted stack at ${GATEWAY_URL} and ${CONSOLE_URL}"
    print_machine_summary
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

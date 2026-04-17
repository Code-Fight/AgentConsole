#!/usr/bin/env bash
set -euo pipefail

STACK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="${STACK_DIR}/docker-compose.yml"
PROJECT_NAME="${CAG_TESTENV_PROJECT:-cag-dev-integration}"
GATEWAY_PORT="${CAG_GATEWAY_PORT:-18080}"
CONSOLE_PORT="${CAG_CONSOLE_PORT:-14173}"
MACHINE_NAME="${CAG_MACHINE_NAME:-Dev Integration Client}"
GATEWAY_URL="http://localhost:${GATEWAY_PORT}"
CONSOLE_URL="http://localhost:${CONSOLE_PORT}"
export CAG_CLIENT_RUNTIME_MODE="${CAG_CLIENT_RUNTIME_MODE:-appserver}"

compose() {
  docker compose -p "${PROJECT_NAME}" -f "${COMPOSE_FILE}" "$@"
}

client_container_id() {
  compose ps -q client
}

wait_for_client_container() {
  local timeout="${1:-60}"
  local started_at
  started_at="$(date +%s)"

  until [ -n "$(client_container_id)" ]; do
    if (( "$(date +%s)" - started_at >= timeout )); then
      echo "Timed out waiting for client container to be created" >&2
      return 1
    fi
    sleep 1
  done
}

copy_codex_file_if_present() {
  local source_path="$1"
  local target_name="$2"

  if [ ! -f "${source_path}" ]; then
    return 0
  fi

  local container_id
  container_id="$(client_container_id)"
  if [ -z "${container_id}" ]; then
    echo "Client container is unavailable for copying ${target_name}" >&2
    return 1
  fi

  docker cp "${source_path}" "${container_id}:/home/appuser/.codex/${target_name}"
  docker exec -u 0 "${container_id}" sh -lc \
    "chown appuser:appuser /home/appuser/.codex/${target_name} && chmod 600 /home/appuser/.codex/${target_name}"
}

seed_codex_files() {
  local copied=0

  wait_for_client_container
  if [ -f "${HOME}/.codex/auth.json" ]; then
    copy_codex_file_if_present "${HOME}/.codex/auth.json" "auth.json"
    copied=1
  fi
  if [ -f "${HOME}/.codex/config.toml" ]; then
    copy_codex_file_if_present "${HOME}/.codex/config.toml" "config.toml"
    copied=1
  fi

  if (( copied == 1 )); then
    compose restart client >/dev/null
  fi
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

  until python3 - "${GATEWAY_URL}/machines" "${MACHINE_NAME}" <<'PY'
import json
import sys
import urllib.request

url, machine_name = sys.argv[1], sys.argv[2]
with urllib.request.urlopen(url, timeout=3) as response:
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
  python3 - "${GATEWAY_URL}/machines" "${MACHINE_NAME}" <<'PY'
import json
import sys
import urllib.request

url, machine_name = sys.argv[1], sys.argv[2]
with urllib.request.urlopen(url, timeout=3) as response:
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
    seed_codex_files
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
    echo "  curl ${GATEWAY_URL}/machines"
    echo "  curl ${GATEWAY_URL}/threads"
    ;;
  down)
    compose down --remove-orphans
    ;;
  restart)
    compose down --remove-orphans
    compose up --build -d
    seed_codex_files
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

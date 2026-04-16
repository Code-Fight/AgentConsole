# Dev Integration Multi-Clients Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Update `testenv/dev-integration` so one stack boot brings up three client containers with the expected names and stable isolated machine identities.

**Architecture:** Keep the feature local to `testenv/dev-integration`. Expand the compose file into three explicit client services with shared runtime defaults and isolated state volumes, then teach `run.sh` to wait for a set of expected machine names and print a multi-machine summary. Finish by documenting the new topology and verifying it end-to-end against Gateway `/machines`.

**Tech Stack:** Docker Compose, Bash, Python 3 for JSON checks, Gateway HTTP `/machines`, Codex client container.

---

### Task 1: Lock The Multi-Client Compose Topology

**Files:**
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/.worktrees/dev-integration-multi-clients/testenv/dev-integration/docker-compose.yml`

- [ ] **Step 1: Write the failing compose topology check**

```bash
set -euo pipefail

expected_services=$'client-default\nclient-hostname\nclient-not-agent\nconsole\ngateway'
actual_services="$(docker compose -f testenv/dev-integration/docker-compose.yml config --services | sort)"

if [[ "${actual_services}" != "${expected_services}" ]]; then
  printf 'unexpected services:\n%s\n' "${actual_services}" >&2
  exit 1
fi

compose_render="$(docker compose -f testenv/dev-integration/docker-compose.yml config)"

printf '%s' "${compose_render}" | grep -F 'hostname: hostname-fallback-client'
printf '%s' "${compose_render}" | grep -F 'MACHINE_NAME: Dev Integration Client'
printf '%s' "${compose_render}" | grep -F 'MACHINE_NAME: Not Agent'
printf '%s' "${compose_render}" | grep -F 'client-default-state:/home/appuser/.code-agent-gateway'
printf '%s' "${compose_render}" | grep -F 'client-hostname-state:/home/appuser/.code-agent-gateway'
printf '%s' "${compose_render}" | grep -F 'client-not-agent-state:/home/appuser/.code-agent-gateway'
```

- [ ] **Step 2: Run the compose topology check and verify it fails**

Run:

```bash
bash -lc '
set -euo pipefail
expected_services=$'"'"'client-default\nclient-hostname\nclient-not-agent\nconsole\ngateway'"'"'
actual_services="$(docker compose -f testenv/dev-integration/docker-compose.yml config --services | sort)"
if [[ "${actual_services}" != "${expected_services}" ]]; then
  printf "unexpected services:\n%s\n" "${actual_services}" >&2
  exit 1
fi
'
```

Expected: FAIL because the current compose file only defines a single `client` service and one shared `client-state` volume.

- [ ] **Step 3: Expand the compose file to three explicit client services with isolated state**

```yaml
x-client-common: &client-common
  build:
    context: ../..
    dockerfile: testenv/dev-integration/Dockerfile.client
    args:
      CODEX_NPM_VERSION: "${CAG_CODEX_NPM_VERSION:-0.118.0}"
  environment: &client-environment
    GATEWAY_URL: "ws://gateway:8080/ws/client"
    CODEX_RUNTIME_MODE: "${CAG_CLIENT_RUNTIME_MODE:-appserver}"
    CODEX_BIN: "${CAG_CODEX_BIN:-codex}"
    HOME: /home/appuser
  depends_on:
    gateway:
      condition: service_healthy

services:
  client-default:
    <<: *client-common
    environment:
      <<: *client-environment
      MACHINE_NAME: "${CAG_MACHINE_NAME_DEFAULT:-Dev Integration Client}"
    volumes:
      - client-default-state:/home/appuser/.code-agent-gateway

  client-hostname:
    <<: *client-common
    hostname: "${CAG_HOSTNAME_FALLBACK_NAME:-hostname-fallback-client}"
    volumes:
      - client-hostname-state:/home/appuser/.code-agent-gateway

  client-not-agent:
    <<: *client-common
    environment:
      <<: *client-environment
      MACHINE_NAME: "${CAG_MACHINE_NAME_NOT_AGENT:-Not Agent}"
    volumes:
      - client-not-agent-state:/home/appuser/.code-agent-gateway

volumes:
  client-default-state:
  client-hostname-state:
  client-not-agent-state:
```

- [ ] **Step 4: Run the compose topology check again**

Run:

```bash
bash -lc '
set -euo pipefail
expected_services=$'"'"'client-default\nclient-hostname\nclient-not-agent\nconsole\ngateway'"'"'
actual_services="$(docker compose -f testenv/dev-integration/docker-compose.yml config --services | sort)"
[[ "${actual_services}" == "${expected_services}" ]]
compose_render="$(docker compose -f testenv/dev-integration/docker-compose.yml config)"
printf "%s" "${compose_render}" | grep -F "hostname: hostname-fallback-client"
printf "%s" "${compose_render}" | grep -F "MACHINE_NAME: Dev Integration Client"
printf "%s" "${compose_render}" | grep -F "MACHINE_NAME: Not Agent"
printf "%s" "${compose_render}" | grep -F "client-default-state:/home/appuser/.code-agent-gateway"
printf "%s" "${compose_render}" | grep -F "client-hostname-state:/home/appuser/.code-agent-gateway"
printf "%s" "${compose_render}" | grep -F "client-not-agent-state:/home/appuser/.code-agent-gateway"
'
```

Expected: PASS

- [ ] **Step 5: Commit the compose slice**

```bash
git add testenv/dev-integration/docker-compose.yml
git commit -m "feat: add three clients to dev integration compose"
```

### Task 2: Make `run.sh` Wait For And Report All Three Machines

**Files:**
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/.worktrees/dev-integration-multi-clients/testenv/dev-integration/run.sh`

- [ ] **Step 1: Write the failing stack smoke check for multi-machine output**

```bash
set -euo pipefail

PROJECT_NAME="cag-dev-integration-red"
GATEWAY_PORT="28080"
CONSOLE_PORT="24173"
OUTPUT_FILE="$(mktemp)"

cleanup() {
  CAG_TESTENV_PROJECT="${PROJECT_NAME}" \
  CAG_GATEWAY_PORT="${GATEWAY_PORT}" \
  CAG_CONSOLE_PORT="${CONSOLE_PORT}" \
  ./testenv/dev-integration/run.sh down >/dev/null 2>&1 || true
  rm -f "${OUTPUT_FILE}"
}
trap cleanup EXIT

CAG_TESTENV_PROJECT="${PROJECT_NAME}" \
CAG_GATEWAY_PORT="${GATEWAY_PORT}" \
CAG_CONSOLE_PORT="${CONSOLE_PORT}" \
./testenv/dev-integration/run.sh up | tee "${OUTPUT_FILE}"

grep -F 'Dev Integration Client' "${OUTPUT_FILE}"
grep -F 'hostname-fallback-client' "${OUTPUT_FILE}"
grep -F 'Not Agent' "${OUTPUT_FILE}"
```

- [ ] **Step 2: Run the smoke check and verify it fails**

Run:

```bash
bash -lc '
set -euo pipefail
PROJECT_NAME="cag-dev-integration-red"
GATEWAY_PORT="28080"
CONSOLE_PORT="24173"
OUTPUT_FILE="$(mktemp)"
cleanup() {
  CAG_TESTENV_PROJECT="${PROJECT_NAME}" \
  CAG_GATEWAY_PORT="${GATEWAY_PORT}" \
  CAG_CONSOLE_PORT="${CONSOLE_PORT}" \
  ./testenv/dev-integration/run.sh down >/dev/null 2>&1 || true
  rm -f "${OUTPUT_FILE}"
}
trap cleanup EXIT
CAG_TESTENV_PROJECT="${PROJECT_NAME}" \
CAG_GATEWAY_PORT="${GATEWAY_PORT}" \
CAG_CONSOLE_PORT="${CONSOLE_PORT}" \
./testenv/dev-integration/run.sh up | tee "${OUTPUT_FILE}"
grep -F "Dev Integration Client" "${OUTPUT_FILE}"
grep -F "hostname-fallback-client" "${OUTPUT_FILE}"
grep -F "Not Agent" "${OUTPUT_FILE}"
'
```

Expected: FAIL because `run.sh` still waits for a single `MACHINE_NAME` and only prints one machine summary.

- [ ] **Step 3: Replace the single-machine helpers with expected-name set handling**

```bash
DEFAULT_MACHINE_NAME="${CAG_MACHINE_NAME_DEFAULT:-Dev Integration Client}"
HOSTNAME_FALLBACK_NAME="${CAG_HOSTNAME_FALLBACK_NAME:-hostname-fallback-client}"
NOT_AGENT_MACHINE_NAME="${CAG_MACHINE_NAME_NOT_AGENT:-Not Agent}"
EXPECTED_MACHINE_NAMES=(
  "${DEFAULT_MACHINE_NAME}"
  "${HOSTNAME_FALLBACK_NAME}"
  "${NOT_AGENT_MACHINE_NAME}"
)

wait_for_machines() {
  local timeout="${1:-120}"
  local started_at
  started_at="$(date +%s)"

  until python3 - "${GATEWAY_URL}/machines" "${EXPECTED_MACHINE_NAMES[@]}" <<'PY'
import json
import sys
import urllib.request

url = sys.argv[1]
expected = sys.argv[2:]
with urllib.request.urlopen(url, timeout=3) as response:
    payload = json.load(response)

online_names = {
    item.get("name")
    for item in payload.get("items", [])
    if item.get("status") == "online"
}
missing = [name for name in expected if name not in online_names]
if missing:
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
expected = sys.argv[2:]
with urllib.request.urlopen(url, timeout=3) as response:
    payload = json.load(response)

items = {
    item.get("name"): item.get("id", "")
    for item in payload.get("items", [])
}
for name in expected:
    print(f"{name}\t{items.get(name, '')}")
PY
}

print_machine_summary() {
  while IFS=$'\t' read -r machine_name machine_id; do
    echo "Machine Name: ${machine_name}"
    echo "Machine ID: ${machine_id}"
  done < <(fetch_machine_rows)
}
```

- [ ] **Step 4: Wire `up` and `restart` to the new helpers and rerun the smoke check**

Run:

```bash
bash -lc '
set -euo pipefail
PROJECT_NAME="cag-dev-integration-green"
GATEWAY_PORT="28080"
CONSOLE_PORT="24173"
OUTPUT_FILE="$(mktemp)"
cleanup() {
  CAG_TESTENV_PROJECT="${PROJECT_NAME}" \
  CAG_GATEWAY_PORT="${GATEWAY_PORT}" \
  CAG_CONSOLE_PORT="${CONSOLE_PORT}" \
  ./testenv/dev-integration/run.sh down >/dev/null 2>&1 || true
  rm -f "${OUTPUT_FILE}"
}
trap cleanup EXIT
CAG_TESTENV_PROJECT="${PROJECT_NAME}" \
CAG_GATEWAY_PORT="${GATEWAY_PORT}" \
CAG_CONSOLE_PORT="${CONSOLE_PORT}" \
./testenv/dev-integration/run.sh up | tee "${OUTPUT_FILE}"
grep -F "Dev Integration Client" "${OUTPUT_FILE}"
grep -F "hostname-fallback-client" "${OUTPUT_FILE}"
grep -F "Not Agent" "${OUTPUT_FILE}"
python3 - "http://localhost:${GATEWAY_PORT}/machines" <<'"'"'PY'"'"'
import json
import sys
import urllib.request

expected = {"Dev Integration Client", "hostname-fallback-client", "Not Agent"}
with urllib.request.urlopen(sys.argv[1], timeout=3) as response:
    payload = json.load(response)
online_names = {
    item.get("name")
    for item in payload.get("items", [])
    if item.get("status") == "online"
}
missing = expected - online_names
if missing:
    raise SystemExit(f"missing online machines: {sorted(missing)}")
PY
'
```

Expected: PASS

- [ ] **Step 5: Commit the runtime helper slice**

```bash
git add testenv/dev-integration/run.sh
git commit -m "feat: wait for all dev integration clients"
```

### Task 3: Document The New Topology And Capture Final Verification

**Files:**
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/.worktrees/dev-integration-multi-clients/testenv/dev-integration/README.md`

- [ ] **Step 1: Update the README to describe the three clients and the new environment variables**

```md
这个目录默认会拉起三台 client：

- `client-default`：`MACHINE_NAME=Dev Integration Client`
- `client-hostname`：不传 `MACHINE_NAME`，回退到 docker hostname `hostname-fallback-client`
- `client-not-agent`：`MACHINE_NAME=Not Agent`

可选环境变量：

- `CAG_MACHINE_NAME_DEFAULT`
- `CAG_HOSTNAME_FALLBACK_NAME`
- `CAG_MACHINE_NAME_NOT_AGENT`
- `CAG_CLIENT_RUNTIME_MODE`
```

- [ ] **Step 2: Run the final focused verification and record the exact evidence**

Run:

```bash
bash -lc '
set -euo pipefail
PROJECT_NAME="cag-dev-integration-final"
GATEWAY_PORT="28080"
CONSOLE_PORT="24173"
cleanup() {
  CAG_TESTENV_PROJECT="${PROJECT_NAME}" \
  CAG_GATEWAY_PORT="${GATEWAY_PORT}" \
  CAG_CONSOLE_PORT="${CONSOLE_PORT}" \
  ./testenv/dev-integration/run.sh down >/dev/null 2>&1 || true
}
trap cleanup EXIT
CAG_TESTENV_PROJECT="${PROJECT_NAME}" \
CAG_GATEWAY_PORT="${GATEWAY_PORT}" \
CAG_CONSOLE_PORT="${CONSOLE_PORT}" \
./testenv/dev-integration/run.sh up
python3 - "http://localhost:${GATEWAY_PORT}/machines" <<'"'"'PY'"'"'
import json
import sys
import urllib.request

with urllib.request.urlopen(sys.argv[1], timeout=3) as response:
    payload = json.load(response)
online = sorted(
    (item.get("name"), item.get("id"))
    for item in payload.get("items", [])
    if item.get("status") == "online"
)
print(json.dumps(online, indent=2))
expected_names = {"Dev Integration Client", "hostname-fallback-client", "Not Agent"}
actual_names = {name for name, _ in online}
if actual_names != expected_names:
    raise SystemExit(f"unexpected online names: {sorted(actual_names)}")
if len({machine_id for _, machine_id in online}) != 3:
    raise SystemExit("expected three distinct machine ids")
PY
'
```

Expected: PASS and print the three `(name, id)` pairs from Gateway.

- [ ] **Step 3: Commit the docs slice**

```bash
git add testenv/dev-integration/README.md
git commit -m "docs: describe multi-client dev integration stack"
```

- [ ] **Step 4: Create the final feature commit if the work stayed uncommitted across tasks**

```bash
git status --short
```

Expected: clean working tree. If any of the three files are still staged or modified, create one final commit that groups the remaining changes:

```bash
git add testenv/dev-integration/docker-compose.yml testenv/dev-integration/run.sh testenv/dev-integration/README.md
git commit -m "feat: support multiple clients in dev integration"
```

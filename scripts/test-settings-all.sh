#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

"${ROOT_DIR}/scripts/test-settings-unit.sh"
"${ROOT_DIR}/scripts/test-settings-system.sh"
"${ROOT_DIR}/testing/environments/settings-e2e/run.sh"

#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GO_BIN="${GO_BIN:-go}"

check_go_coverage() {
  local package_path="$1"
  local minimum="$2"
  local profile
  profile="$(mktemp)"

  "${GO_BIN}" test -covermode=count -coverprofile="${profile}" "${package_path}" >/dev/null
  local total
  total="$("${GO_BIN}" tool cover -func="${profile}" | awk '/total:/ {gsub("%","",$3); print $3}')"
  rm -f "${profile}"

  awk -v total="${total}" -v minimum="${minimum}" -v pkg="${package_path}" 'BEGIN {
    if (total + 0 < minimum + 0) {
      printf("coverage threshold failed for %s: %.1f < %.1f\n", pkg, total, minimum)
      exit 1
    }
    printf("coverage threshold passed for %s: %.1f >= %.1f\n", pkg, total, minimum)
  }'
}

check_go_file_functions() {
  local package_path="$1"
  local file_fragment="$2"
  local minimum="$3"
  local profile
  profile="$(mktemp)"

  "${GO_BIN}" test -covermode=count -coverprofile="${profile}" "${package_path}" >/dev/null

  local failed=0
  while read -r line; do
    local percent
    percent="$(awk '{gsub("%","",$3); print $3}' <<<"${line}")"
    awk -v total="${percent}" -v minimum="${minimum}" -v item="${line}" 'BEGIN {
      if (total + 0 < minimum + 0) {
        printf("coverage threshold failed for %s: %.1f < %.1f\n", item, total, minimum)
        exit 1
      }
    }' || failed=1
  done < <("${GO_BIN}" tool cover -func="${profile}" | grep "${file_fragment}" || true)

  rm -f "${profile}"
  if [[ "${failed}" -ne 0 ]]; then
    exit 1
  fi

  echo "coverage threshold passed for functions in ${file_fragment} >= ${minimum}"
}

check_go_coverage "./gateway/internal/settings" "90"
check_go_file_functions "./client/internal/agent/codex" "client/internal/agent/codex/config_apply.go" "90"

corepack pnpm --dir "${ROOT_DIR}/console" exec vitest run --config vitest.settings.config.ts --coverage

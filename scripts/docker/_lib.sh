#!/usr/bin/env bash
# Shared helpers for docker/ compose scripts — health URL/timeout aligned with
# scripts/pm2-reload-with-binary-rollback.sh (single source of truth for polling).
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

HEALTHCHECK_URL="${HEALTHCHECK_URL:-http://127.0.0.1:8080/api/v1/health}"
ROLLBACK_HEALTH_TIMEOUT_SEC="${ROLLBACK_HEALTH_TIMEOUT_SEC:-90}"

VALID_ENVS=(local dev staging prod)

usage_envs() {
  printf '%s\n' "${VALID_ENVS[@]}"
}

is_valid_env() {
  local name="${1:?}"
  local e
  for e in "${VALID_ENVS[@]}"; do
    [[ "$e" == "$name" ]] && return 0
  done
  return 1
}

require_env_name() {
  local name="${1:?}"
  if ! is_valid_env "$name"; then
    echo "docker: invalid environment '${name}'. Expected one of: $(usage_envs | tr '\n' ' ')" >&2
    exit 1
  fi
}

require_env_files() {
  local stage="${1:?}"
  if [[ ! -f "$REPO_ROOT/.env" ]]; then
    echo "docker: missing $REPO_ROOT/.env — copy from .env.example" >&2
    exit 1
  fi
  if [[ ! -f "$REPO_ROOT/.env.${stage}" ]]; then
    echo "docker: missing $REPO_ROOT/.env.${stage} — copy from .env.${stage}.example" >&2
    exit 1
  fi
}

compose_file() {
  echo "$REPO_ROOT/docker/compose.${1}.yml"
}

stack_file() {
  echo "$REPO_ROOT/docker/stack.${1}.yml"
}

compose_project() {
  echo "mycourse-be-${1}"
}

compose_cmd() {
  local env_name="${1:?}"
  shift
  docker compose -f "$(compose_file "$env_name")" -p "$(compose_project "$env_name")" "$@"
}

wait_for_health() {
  local url="${1:-$HEALTHCHECK_URL}"
  local timeout_sec="${2:-$ROLLBACK_HEALTH_TIMEOUT_SEC}"
  local deadline=$(( $(date +%s) + timeout_sec ))
  echo "docker: polling ${url} (timeout ${timeout_sec}s)..."
  while (( $(date +%s) < deadline )); do
    if curl -fsS --connect-timeout 2 --max-time 5 "$url" >/dev/null 2>&1; then
      echo "docker: health OK"
      return 0
    fi
    sleep 2
  done
  echo "docker: health check failed (${url}, timeout ${timeout_sec}s)" >&2
  return 1
}

#!/usr/bin/env bash
# Swarm stack deploy helper — for blue-green demo only. DO NOT run during automated tests.
#
# Prerequisites (operator, one-time):
#   docker swarm init
#
# Usage:
#   ./scripts/docker/swarm-deploy.sh <local|dev|staging|prod>
#
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=_lib.sh
source "$SCRIPT_DIR/_lib.sh"

ENV_NAME="${1:?usage: $0 <local|dev|staging|prod>}"
require_env_name "$ENV_NAME"
require_env_files "$ENV_NAME"

STACK_NAME="mycourse-be-${ENV_NAME}"
STACK_FILE="$(stack_file "$ENV_NAME")"

if ! docker info 2>/dev/null | grep -q 'Swarm: active'; then
  echo "swarm-deploy: Swarm is not active. Run 'docker swarm init' first (manual, not part of CI/tests)." >&2
  exit 1
fi

echo "swarm-deploy: deploying stack ${STACK_NAME} from ${STACK_FILE}..."
docker stack deploy -c "$STACK_FILE" "$STACK_NAME"
echo "swarm-deploy: done. Check: docker stack services ${STACK_NAME}"

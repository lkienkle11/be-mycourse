#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=_lib.sh
source "$SCRIPT_DIR/_lib.sh"

ENV_NAME="${1:?usage: $0 <local|dev|staging|prod>}"
require_env_name "$ENV_NAME"

wait_for_health "$HEALTHCHECK_URL" "$ROLLBACK_HEALTH_TIMEOUT_SEC"

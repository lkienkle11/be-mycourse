#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=_lib.sh
source "$SCRIPT_DIR/_lib.sh"

ENV_NAME="${1:?usage: $0 <local|dev|staging|prod>}"
require_env_name "$ENV_NAME"
require_env_files "$ENV_NAME"

echo "docker: building and starting mycourse-be-${ENV_NAME}..."
compose_cmd "$ENV_NAME" up --build -d
echo "docker: stack started. Run: $SCRIPT_DIR/health-check.sh ${ENV_NAME}"

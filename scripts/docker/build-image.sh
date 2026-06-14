#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=_lib.sh
source "$SCRIPT_DIR/_lib.sh"

ENV_NAME="${1:?usage: $0 <local|dev|staging|prod>}"
require_env_name "$ENV_NAME"

IMAGE_TAG="mycourse-io-be:${ENV_NAME}"
echo "docker: building image ${IMAGE_TAG} (STAGE=${ENV_NAME})..."
docker build \
  --build-arg "STAGE=${ENV_NAME}" \
  -t "$IMAGE_TAG" \
  -f "$REPO_ROOT/Dockerfile" \
  "$REPO_ROOT"
echo "docker: built ${IMAGE_TAG}"

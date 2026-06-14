#!/usr/bin/env bash
# Optional local Postgres/Redis helpers — DISABLED by default.
#
# Default Docker path uses cloud DATABASE_URL / REDIS_ADDR from .env + .env.<stage>.
# Uncomment the service blocks in docker/compose.<env>.yml AND the functions below
# when you want a fully on-box DB/cache for local experiments.
#
# When enabling:
#   1. Uncomment postgres + redis in docker/compose.local.yml (or target env file)
#   2. Point .env.local DATABASE_URL=postgres://mycourse:mycourse@postgres:5432/mycourse?sslmode=disable
#   3. Point REDIS_ADDR=redis:6379
#   4. Uncomment and run: ./scripts/docker/local-infra.sh up
#
set -euo pipefail

echo "local-infra.sh: all helpers are commented out — see file header." >&2
echo "Uncomment postgres/redis in docker/compose.local.yml first." >&2
exit 1

# --- Uncomment when local infra is enabled in compose ---
#
# SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# source "$SCRIPT_DIR/_lib.sh"
#
# local_infra_up() {
#   compose_cmd local up -d postgres redis
# }
#
# local_infra_down() {
#   compose_cmd local stop postgres redis
# }
#
# case "${1:-}" in
#   up) local_infra_up ;;
#   down) local_infra_down ;;
#   *) echo "usage: $0 up|down" >&2; exit 1 ;;
# esac

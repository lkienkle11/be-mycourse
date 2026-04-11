#!/usr/bin/env bash
# Post-rsync deploy hook: reload PM2, wait until HTTP /health responds (app finished
# setup and is listening), then git-sync. If health never succeeds, restore the
# previous binary from *.prev (created before rsync in CI) and reload again so the
# last good build keeps serving traffic; exit non-zero so the pipeline fails loudly.
set -euo pipefail

DEPLOY_PATH="${DEPLOY_PATH:?set DEPLOY_PATH to the backend root on the server}"
cd "$DEPLOY_PATH"

PM2_APP_NAME="${PM2_APP_NAME:-mycourse-api-dev}"
BINARY_REL="${BINARY_REL:-bin/mycourse-io-be-dev}"
HEALTHCHECK_URL="${HEALTHCHECK_URL:-http://127.0.0.1:8080/api/v1/health}"
ROLLBACK_HEALTH_TIMEOUT_SEC="${ROLLBACK_HEALTH_TIMEOUT_SEC:-90}"

BIN="${BINARY_REL}"
PREV="${BIN}.prev"

pm2_reload() {
  pm2 reload "$PM2_APP_NAME" || pm2 start ecosystem.config.cjs --only "$PM2_APP_NAME"
}

wait_health() {
  local deadline=$(( $(date +%s) + ROLLBACK_HEALTH_TIMEOUT_SEC ))
  while (( $(date +%s) < deadline )); do
    if curl -fsS --connect-timeout 2 --max-time 5 "$HEALTHCHECK_URL" >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done
  return 1
}

echo "pm2-reload-with-binary-rollback: reloading ${PM2_APP_NAME}..."
pm2_reload

if wait_health; then
  git stash -u
  git checkout master
  git pull
  echo "pm2-reload-with-binary-rollback: health OK, git synced."
  exit 0
fi

echo "pm2-reload-with-binary-rollback: health check failed (${HEALTHCHECK_URL}, timeout ${ROLLBACK_HEALTH_TIMEOUT_SEC}s)." >&2

if [[ -f "$PREV" ]]; then
  echo "pm2-reload-with-binary-rollback: restoring previous binary from ${PREV}" >&2
  cp "$PREV" "$BIN"
  pm2_reload
  if wait_health; then
    echo "pm2-reload-with-binary-rollback: service recovered after rollback; failing deploy (new binary rejected)." >&2
  else
    echo "pm2-reload-with-binary-rollback: CRITICAL: health still failing after rollback — inspect PM2 and logs." >&2
  fi
else
  echo "pm2-reload-with-binary-rollback: no ${PREV} — cannot rollback (first deploy?)." >&2
fi

pm2 logs "$PM2_APP_NAME" --lines 120 --nostream >&2 || true
exit 1

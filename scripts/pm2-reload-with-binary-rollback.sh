#!/usr/bin/env bash
# Post-rsync deploy hook (CI rsyncs straight to bin/mycourse-io-be-dev after copying that file to *.prev in the workflow).
# 1) Backup ecosystem.config.cjs to .prev, then pull ONLY that file from origin (default: master).
# 2) pm2 reload with the new binary + freshly pulled ecosystem; wait for HTTP health.
#    While waiting, treat PM2 "max autorestart exhausted" (errored / unstable_restarts >= max_restarts) as failure.
# 3) On success: git stash, checkout branch, full git pull.
# 4) On failure: restore previous binary + previous ecosystem, pm2 reload, health-check again; exit non-zero.
#
# PM2 does not push shell callbacks when max_restarts is hit; this script polls `pm2 jlist` (same outcome as reacting to that event).
set -euo pipefail

DEPLOY_PATH="${DEPLOY_PATH:?set DEPLOY_PATH to the backend root on the server}"
cd "$DEPLOY_PATH"

PM2_APP_NAME="${PM2_APP_NAME:-mycourse-api-dev}"
BINARY_REL="${BINARY_REL:-bin/mycourse-io-be-dev}"
ECOSYSTEM_FILE="${ECOSYSTEM_FILE:-ecosystem.config.cjs}"
HEALTHCHECK_URL="${HEALTHCHECK_URL:-http://127.0.0.1:8080/api/v1/health}"
ROLLBACK_HEALTH_TIMEOUT_SEC="${ROLLBACK_HEALTH_TIMEOUT_SEC:-90}"
GIT_REMOTE_BRANCH="${GIT_REMOTE_BRANCH:-master}"
# After reload, ignore transient PM2 statuses for this many seconds before treating stopped as fatal.
PM2_EXHAUSTED_GRACE_SEC="${PM2_EXHAUSTED_GRACE_SEC:-12}"

BIN="${BINARY_REL}"
PREV="${BIN}.prev"
ECOSYSTEM_PREV="${ECOSYSTEM_FILE}.prev"

pm2_reload() {
  pm2 reload "$ECOSYSTEM_FILE" --only "$PM2_APP_NAME" 2>/dev/null || pm2 start "$ECOSYSTEM_FILE" --only "$PM2_APP_NAME"
}

# Returns 0 if PM2 shows the app stopped retrying (aligned with ecosystem max_restarts / errored).
pm2_autorestart_exhausted() {
  local payload st unst mx elapsed
  elapsed=$(( $(date +%s) - PM2_RELOAD_EPOCH ))
  if (( elapsed < PM2_EXHAUSTED_GRACE_SEC )); then
    return 1
  fi
  payload=""
  payload="$(pm2 jlist 2>/dev/null | jq -c --arg n "$PM2_APP_NAME" '.[] | select(.name==$n) | {st:.pm2_env.status, unst:(.pm2_env.unstable_restarts//0), mx:(.pm2_env.max_restarts//3)}' 2>/dev/null | head -n1)" || payload=""
  [[ -z "$payload" || "$payload" == "null" ]] && return 1
  st="$(echo "$payload" | jq -r .st)"
  unst="$(echo "$payload" | jq -r .unst)"
  mx="$(echo "$payload" | jq -r .mx)"
  if [[ "$st" == "errored" ]]; then
    echo "pm2-reload-with-binary-rollback: PM2 status=errored (autorestart policy likely exhausted)." >&2
    return 0
  fi
  if [[ "$st" == "stopped" ]] && (( unst >= mx )); then
    echo "pm2-reload-with-binary-rollback: PM2 stopped after unstable_restarts (${unst}) >= max_restarts (${mx})." >&2
    return 0
  fi
  return 1
}

# 0 = health OK, 1 = timeout, 2 = PM2 exhausted / dead without health.
wait_health_or_pm2_exhausted() {
  local deadline=$(( $(date +%s) + ROLLBACK_HEALTH_TIMEOUT_SEC ))
  while (( $(date +%s) < deadline )); do
    if curl -fsS --connect-timeout 2 --max-time 5 "$HEALTHCHECK_URL" >/dev/null 2>&1; then
      return 0
    fi
    if pm2_autorestart_exhausted; then
      return 2
    fi
    sleep 2
  done
  return 1
}

rollback_binary_and_ecosystem() {
  if [[ -f "$PREV" ]]; then
    echo "pm2-reload-with-binary-rollback: restoring previous binary from ${PREV}" >&2
    cp "$PREV" "$BIN"
  else
    echo "pm2-reload-with-binary-rollback: no ${PREV} — binary rollback skipped." >&2
  fi
  if [[ -f "$ECOSYSTEM_PREV" ]]; then
    echo "pm2-reload-with-binary-rollback: restoring previous ecosystem from ${ECOSYSTEM_PREV}" >&2
    cp "$ECOSYSTEM_PREV" "$ECOSYSTEM_FILE"
  else
    echo "pm2-reload-with-binary-rollback: no ${ECOSYSTEM_PREV} — ecosystem rollback skipped." >&2
  fi
}

git_sync_full_after_success() {
  git stash -u
  git checkout "$GIT_REMOTE_BRANCH"
  git pull origin "$GIT_REMOTE_BRANCH"
}

pull_ecosystem_only() {
  git fetch origin "$GIT_REMOTE_BRANCH"
  git checkout "origin/${GIT_REMOTE_BRANCH}" -- "$ECOSYSTEM_FILE"
}

# Snapshot ecosystem before replacing it with origin's version (rollback target for this deploy).
if [[ -f "$ECOSYSTEM_FILE" ]]; then
  cp -a "$ECOSYSTEM_FILE" "$ECOSYSTEM_PREV"
  echo "pm2-reload-with-binary-rollback: backed up ${ECOSYSTEM_FILE} -> ${ECOSYSTEM_PREV}"
else
  echo "pm2-reload-with-binary-rollback: WARN: missing ${ECOSYSTEM_FILE}; continuing without ecosystem backup." >&2
fi

if ! pull_ecosystem_only; then
  echo "pm2-reload-with-binary-rollback: git fetch/checkout of ecosystem only failed — restoring ecosystem; restoring binary from CI backup if present." >&2
  if [[ -f "$ECOSYSTEM_PREV" ]]; then
    cp "$ECOSYSTEM_PREV" "$ECOSYSTEM_FILE"
  fi
  if [[ -f "$PREV" ]]; then
    cp "$PREV" "$BIN"
    echo "pm2-reload-with-binary-rollback: restored previous binary from ${PREV} after failed ecosystem pull." >&2
  fi
  exit 1
fi

echo "pm2-reload-with-binary-rollback: pulled latest ${ECOSYSTEM_FILE} from origin/${GIT_REMOTE_BRANCH} only."

PM2_RELOAD_EPOCH=$(date +%s)
export PM2_RELOAD_EPOCH

echo "pm2-reload-with-binary-rollback: reloading ${PM2_APP_NAME} with new binary + updated ecosystem..."
pm2_reload

wait_rc=0
wait_health_or_pm2_exhausted || wait_rc=$?

if [[ "$wait_rc" -eq 0 ]]; then
  git_sync_full_after_success
  echo "pm2-reload-with-binary-rollback: health OK, full git sync done."
  exit 0
fi

if [[ "$wait_rc" -eq 2 ]]; then
  echo "pm2-reload-with-binary-rollback: deploy aborted (PM2 autorestart exhausted while waiting for health)." >&2
else
  echo "pm2-reload-with-binary-rollback: health check failed (${HEALTHCHECK_URL}, timeout ${ROLLBACK_HEALTH_TIMEOUT_SEC}s)." >&2
fi

rollback_binary_and_ecosystem
PM2_RELOAD_EPOCH=$(date +%s)
export PM2_RELOAD_EPOCH
pm2_reload

if wait_health_or_pm2_exhausted; then
  echo "pm2-reload-with-binary-rollback: service recovered after binary+ecosystem rollback; failing deploy (new revision rejected)." >&2
else
  echo "pm2-reload-with-binary-rollback: CRITICAL: health still failing after rollback — inspect PM2 and logs." >&2
fi

pm2 logs "$PM2_APP_NAME" --lines 120 --nostream >&2 || true
exit 1

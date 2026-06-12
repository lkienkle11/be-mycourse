# Local Logs to Loki via Grafana Alloy

This guide documents the backend log file contract for `be-mycourse` and how to ship those logs to Grafana Cloud Loki using Grafana Alloy.

## Logging modes

The backend supports two file logging modes:

1. Legacy single-file tee (backward compatible)
   - Enable by setting `LOG_FILE_PATH`.
   - The logger writes NDJSON to that exact file path.
2. Dual rotated file mode (recommended for Alloy/Loki)
   - Set `LOG_FILE_ENABLED=true` and leave `LOG_FILE_PATH` empty.
   - The logger writes:
     - `app.log` for app/business logs
     - `access.log` for HTTP access logs
   - Rotation uses `lumberjack` with:
     - `LOG_MAX_SIZE_MB`
     - `LOG_MAX_BACKUPS`
     - `LOG_MAX_AGE_DAYS`
     - `LOG_COMPRESS`

Precedence:

- If `LOG_FILE_PATH` is set, legacy mode is used.
- Else if `LOG_FILE_ENABLED=true`, dual-file mode is used.
- Else logs go to stdout only (format controlled by `LOG_FORMAT`).

## Log directory resolution

When `LOG_DIR` is set, it is used directly (supports env expansion via `os.ExpandEnv`).

When `LOG_DIR` is empty and dual-file mode is enabled:

- `LOG_PATH_MODE=user` (default):
  - Windows: `%LOCALAPPDATA%/<vendor>/<app_name>/logs`
  - macOS: `~/Library/Logs/<vendor>/<app_name>`
  - Linux: `${XDG_STATE_HOME:-~/.local/state}/<app_name>/logs`
- `LOG_PATH_MODE=service`:
  - Windows: `%ProgramData%/<vendor>/<app_name>/logs`
  - macOS: `/Library/Logs/<vendor>/<app_name>`
  - Linux: `/var/log/<app_name>`

Defaults:

- `LOG_APP_NAME=be-mycourse`
- `LOG_VENDOR=mycourse`

When running multiple backend instances on the same host, set `LOG_INSTANCE_ID` to avoid writing to the same file.

## JSON field contract

Dual-file mode uses Loki-friendly JSON fields:

- Common:
  - `ts` (RFC3339Nano, UTC)
  - `level` (lowercase)
  - `msg`
  - `caller`
  - `stacktrace` (error and above)
  - `service`
  - `env`
  - `version`
  - `log_file` (`app` or `access`)
- Access log extras:
  - `kind=access`
  - `request_id`
  - `method`
  - `path`
  - `route`
  - `status`
  - `latency`
  - `latency_ms`
  - `bytes`
  - `response_bytes` (alias kept for compatibility)
  - `client_ip`
  - `user_agent`

## Recommended labels (low cardinality)

Use only low-cardinality labels in Loki:

- `service`
- `env`
- `host`
- `log_file`
- `level`
- `kind`

Do not use high-cardinality fields as labels, including:

- `request_id`
- `client_ip`
- `user_id`
- full path/query values

## Grafana Alloy pipeline example

See `config/alloy/be-mycourse.example.alloy` for a ready-to-copy template.

Pipeline shape:

1. `local.file_match` finds `*.log`.
2. `loki.source.file` tails files.
3. `loki.process` parses JSON, extracts timestamp, and promotes selected labels.
4. `loki.write` sends to Grafana Cloud Loki.

Promtail note:

- Promtail is deprecated; use Grafana Alloy for new setups.

## Example LogQL queries

Access errors:

```logql
{service="be-mycourse", kind="access"} | json | status >= 400
```

Server errors:

```logql
{service="be-mycourse", level="error"} | json
```

Slow requests (> 1s):

```logql
{service="be-mycourse", log_file="access"} | json | latency_ms > 1000
```

Filter by request ID without label explosion:

```logql
{service="be-mycourse"} | json | request_id="YOUR-REQUEST-ID"
```

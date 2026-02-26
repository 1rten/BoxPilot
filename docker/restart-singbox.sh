#!/bin/sh
set -eu

CONFIG_PATH="${SINGBOX_CONFIG:-/data/sing-box.json}"
LOG_PATH="${SINGBOX_LOG:-/data/sing-box.log}"
PID_FILE="${SINGBOX_PID_FILE:-/tmp/sing-box.pid}"

is_running() {
  if [ ! -f "$PID_FILE" ]; then
    return 1
  fi
  pid="$(cat "$PID_FILE" 2>/dev/null || true)"
  if [ -z "$pid" ]; then
    return 1
  fi
  kill -0 "$pid" 2>/dev/null
}

stop_singbox() {
  if ! is_running; then
    rm -f "$PID_FILE"
    return 0
  fi
  pid="$(cat "$PID_FILE")"
  kill "$pid" 2>/dev/null || true
  i=0
  while [ "$i" -lt 50 ]; do
    if ! kill -0 "$pid" 2>/dev/null; then
      break
    fi
    i=$((i + 1))
    sleep 0.1
  done
  if kill -0 "$pid" 2>/dev/null; then
    kill -9 "$pid" 2>/dev/null || true
  fi
  rm -f "$PID_FILE"
}

start_singbox() {
  if [ ! -f "$CONFIG_PATH" ]; then
    echo "sing-box config not found: $CONFIG_PATH" >&2
    exit 1
  fi
  mkdir -p "$(dirname "$LOG_PATH")"
  sing-box run -c "$CONFIG_PATH" >>"$LOG_PATH" 2>&1 &
  pid="$!"
  echo "$pid" >"$PID_FILE"
  sleep 0.2
  if ! kill -0 "$pid" 2>/dev/null; then
    echo "sing-box failed to start, see log: $LOG_PATH" >&2
    tail -n 40 "$LOG_PATH" 2>/dev/null || true
    exit 1
  fi
}

case "${1:-}" in
  --start-only)
    if is_running; then
      exit 0
    fi
    start_singbox
    ;;
  --stop-only)
    stop_singbox
    ;;
  *)
    stop_singbox
    start_singbox
    ;;
esac

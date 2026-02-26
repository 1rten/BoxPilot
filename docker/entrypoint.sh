#!/bin/sh
set -eu

mkdir -p /data

if [ -f "${SINGBOX_CONFIG:-/data/sing-box.json}" ]; then
  /app/docker/restart-singbox.sh --start-only || true
fi

/app/boxpilot &
BOXPILOT_PID="$!"

on_term() {
  kill "$BOXPILOT_PID" 2>/dev/null || true
  /app/docker/restart-singbox.sh --stop-only || true
  wait "$BOXPILOT_PID" 2>/dev/null || true
  exit 0
}

trap on_term INT TERM

set +e
wait "$BOXPILOT_PID"
status="$?"
set -e
/app/docker/restart-singbox.sh --stop-only || true
exit "$status"

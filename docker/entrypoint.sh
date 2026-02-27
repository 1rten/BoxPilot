#!/bin/sh
set -eu

log() {
  printf '%s %s\n' "[entrypoint]" "$*"
}

mkdir -p /data

if [ ! -x /app/boxpilot ]; then
  log "ERROR: /app/boxpilot not found or not executable"
  ls -la /app || true
  exit 127
fi

config_path="${SINGBOX_CONFIG:-/data/sing-box.json}"
if [ -f "$config_path" ]; then
  log "found sing-box config at $config_path, starting sing-box"
  if ! /app/docker/restart-singbox.sh --start-only; then
    log "WARN: sing-box start failed on boot, BoxPilot will still start"
  fi
else
  log "sing-box config not found at $config_path, skipping sing-box start"
fi

log "starting BoxPilot"
/app/boxpilot &
BOXPILOT_PID="$!"
log "BoxPilot pid=$BOXPILOT_PID"

on_term() {
  log "received shutdown signal, stopping processes"
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
log "BoxPilot exited with status=$status"
exit "$status"

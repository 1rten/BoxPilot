#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
TARGET_URL="${TARGET_URL:-https://www.gstatic.com/generate_204}"

echo "[diag] BASE_URL=${BASE_URL}"
echo "[diag] TARGET_URL=${TARGET_URL}"

fail_count=0

ok() {
  echo "[diag][ok] $*"
}

warn() {
  echo "[diag][warn] $*" >&2
}

check_get() {
  local path="$1"
  if curl --noproxy '*' -fsS "${BASE_URL}${path}" >/tmp/boxpilot-diag.out 2>/tmp/boxpilot-diag.err; then
    ok "GET ${path}"
    cat /tmp/boxpilot-diag.out
  else
    warn "GET ${path} failed"
    cat /tmp/boxpilot-diag.err >&2 || true
    fail_count=$((fail_count + 1))
  fi
}

check_post() {
  local path="$1"
  local body="$2"
  if curl --noproxy '*' -fsS -H 'Content-Type: application/json' -X POST -d "${body}" \
    "${BASE_URL}${path}" >/tmp/boxpilot-diag.out 2>/tmp/boxpilot-diag.err; then
    ok "POST ${path}"
    cat /tmp/boxpilot-diag.out
  else
    warn "POST ${path} failed"
    cat /tmp/boxpilot-diag.err >&2 || true
    fail_count=$((fail_count + 1))
  fi
}

check_get "/healthz"
check_get "/api/v1/runtime/status"
check_get "/api/v1/settings/forwarding/status"
check_post "/api/v1/runtime/proxy/check" "{\"target_url\":\"${TARGET_URL}\"}"

runtime_json="$(curl --noproxy '*' -fsS "${BASE_URL}/api/v1/runtime/status" || true)"
compact="$(printf '%s' "${runtime_json}" | tr -d '\n\r\t ')"
http_port="$(printf '%s' "${compact}" | sed -n 's/.*"http":\([0-9]\+\).*/\1/p' | head -n1)"
socks_port="$(printf '%s' "${compact}" | sed -n 's/.*"socks":\([0-9]\+\).*/\1/p' | head -n1)"

if [[ -n "${http_port}" ]]; then
  if curl --noproxy '' --proxy "http://127.0.0.1:${http_port}" -m 8 -fsSI "${TARGET_URL}" >/dev/null; then
    ok "HTTP proxy smoke via :${http_port}"
  else
    warn "HTTP proxy smoke failed via :${http_port}"
    fail_count=$((fail_count + 1))
  fi
else
  warn "Unable to parse HTTP port from runtime status"
  fail_count=$((fail_count + 1))
fi

if [[ -n "${socks_port}" ]]; then
  if curl --noproxy '' --socks5-hostname "127.0.0.1:${socks_port}" -m 8 -fsSI "${TARGET_URL}" >/dev/null; then
    ok "SOCKS proxy smoke via :${socks_port}"
  else
    warn "SOCKS proxy smoke failed via :${socks_port}"
    fail_count=$((fail_count + 1))
  fi
else
  warn "Unable to parse SOCKS port from runtime status"
  fail_count=$((fail_count + 1))
fi

if [[ ${fail_count} -gt 0 ]]; then
  echo "[diag] completed with ${fail_count} issue(s)" >&2
  exit 1
fi

echo "[diag] completed successfully"

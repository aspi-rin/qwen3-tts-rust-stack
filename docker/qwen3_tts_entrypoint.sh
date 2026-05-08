#!/bin/sh
set -eu

case "${QWEN3_TTS_DEBUG_LOGS:-0}" in
  1|true|TRUE|yes|YES|on|ON)
    exec /app/qwen3_tts_server "$@"
    ;;
esac

# Upstream currently emits verbose per-token/tensor debug lines with println! /
# eprintln! and does not expose a runtime log-level knob. Keep the upstream
# source pure and filter only those known noisy debug lines at the stack layer,
# while still returning the real server exit status.
log_pipe="/tmp/qwen3-tts-server.logpipe"
rm -f "$log_pipe"
mkfifo "$log_pipe"

sed -u \
  -e '/^\[Debug\]/d' \
  -e '/^[[:space:]]*Debug:/d' \
  < "$log_pipe" &
filter_pid="$!"

/app/qwen3_tts_server "$@" > "$log_pipe" 2>&1 &
server_pid="$!"

terminate() {
  kill "$server_pid" 2>/dev/null || true
  wait "$server_pid" 2>/dev/null || true
  wait "$filter_pid" 2>/dev/null || true
}
trap terminate INT TERM

set +e
wait "$server_pid"
status="$?"
wait "$filter_pid" 2>/dev/null
set -e
rm -f "$log_pipe"
exit "$status"

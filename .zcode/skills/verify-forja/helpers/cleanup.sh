#!/bin/sh
# Tear down a verification session: stop the stub recorded in session.env
# (only after checking the pid is still our stub), copy the full request log
# into the evidence dir, and delete the scratch run dir. Evidence survives.
#
# Usage: cleanup.sh <session.env>
set -eu

session="$1"
. "$session"

if [ -n "${STUB_PID:-}" ] && kill -0 "$STUB_PID" 2>/dev/null; then
  cmd="$(ps -p "$STUB_PID" -o command= 2>/dev/null || true)"
  case "$cmd" in
    *stub_api.py*)
      kill "$STUB_PID"
      echo "stopped stub pid $STUB_PID"
      ;;
    *)
      echo "refusing to kill pid $STUB_PID: not our stub (command: ${cmd:-unknown})" >&2
      ;;
  esac
else
  echo "stub pid ${STUB_PID:-?} already gone"
fi

if [ -n "${EVIDENCE_DIR:-}" ] && [ -n "${STUB_LOG:-}" ] && [ -f "$STUB_LOG" ]; then
  cp "$STUB_LOG" "$EVIDENCE_DIR/stub-full.log"
fi

if [ -n "${RUN_DIR:-}" ] && [ -d "$RUN_DIR" ]; then
  rm -rf "$RUN_DIR"
  echo "removed scratch $RUN_DIR"
fi

echo "evidence kept in ${EVIDENCE_DIR:-<no session>}"

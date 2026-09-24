#!/bin/sh
# Run one forja command against the session's stub API and capture proof.
#
# Usage: drive.sh <session.env> <case-name> [forja args...]
#
# case-name is a [a-z0-9-] label used for the evidence files. Writes
# $EVIDENCE_DIR/<case-name>.transcript (command, stdout, stderr, exit code)
# and, when the stub logged requests, $EVIDENCE_DIR/<case-name>.requests
# (the exact HTTP lines the CLI sent). Exits with forja's exit code.
#
# FORJA_API_KEY and FORJA_ENDPOINT are pinned to the stub and FORJA_TEAM and
# FORJA_FORMAT are unset, so ambient user config cannot leak in. Pass
# --format/--team/--config as flags on the command when the case needs them.
set -eu

session="$1"
name="$2"
shift 2

. "$session"

unset FORJA_API_KEY FORJA_ENDPOINT FORJA_TEAM FORJA_FORMAT
export FORJA_API_KEY="$STUB_API_KEY"
export FORJA_ENDPOINT="http://127.0.0.1:$STUB_PORT"

before="$(wc -l < "$STUB_LOG" 2>/dev/null || echo 0)"

set +e
"$FORJA_BIN" "$@" > "$RUN_DIR/$name.stdout" 2> "$RUN_DIR/$name.stderr"
status=$?
set -e

after="$(wc -l < "$STUB_LOG" 2>/dev/null || echo 0)"

{
  echo "CMD: forja $*"
  echo "--- stdout ---"
  cat "$RUN_DIR/$name.stdout"
  echo "--- stderr ---"
  cat "$RUN_DIR/$name.stderr"
  echo "exit=$status"
} > "$EVIDENCE_DIR/$name.transcript"

if [ "$after" -gt "$before" ]; then
  sed -n "$((before + 1)),${after}p" "$STUB_LOG" > "$EVIDENCE_DIR/$name.requests"
fi

echo "exit=$status  evidence: $EVIDENCE_DIR/$name.transcript"
exit "$status"

#!/bin/sh
# Launch a verification session: build forja fresh, start the stub API on a
# free port, and wait until it answers. Prints the session.env path to source.
#
# Usage: helpers/launch.sh
set -eu

SKILL_DIR="$(cd "$(dirname "$0")/.." && pwd)"
REPO_DIR="$(cd "$SKILL_DIR/../../.." && pwd)"
RUN_ID="$(date +%Y%m%d-%H%M%S)-$$"
RUN_DIR="/tmp/forja-verify-$RUN_ID"
EVIDENCE_DIR="$SKILL_DIR/evidence/$RUN_ID"
mkdir -p "$RUN_DIR" "$EVIDENCE_DIR"

echo "building forja..." >&2
(cd "$REPO_DIR" && go build -o "$RUN_DIR/forja" .)

STUB_LOG="$RUN_DIR/stub.log"
python3 "$SKILL_DIR/helpers/stub_api.py" \
  --port 0 \
  --api-key forja-verify-key \
  --log "$STUB_LOG" > "$RUN_DIR/stub.out" 2>&1 &
STUB_PID=$!

port=""
i=0
while [ "$i" -lt 50 ]; do
  if ! kill -0 "$STUB_PID" 2>/dev/null; then
    echo "stub died during startup:" >&2
    cat "$RUN_DIR/stub.out" >&2
    exit 1
  fi
  port="$(sed -n 's/^READY port=//p' "$RUN_DIR/stub.out" | head -1)"
  [ -n "$port" ] && break
  i=$((i + 1))
  sleep 0.2
done

if [ -z "$port" ]; then
  echo "stub never became ready:" >&2
  cat "$RUN_DIR/stub.out" >&2
  kill "$STUB_PID" 2>/dev/null || true
  exit 1
fi

SESSION="$RUN_DIR/session.env"
cat > "$SESSION" <<EOF
RUN_ID='$RUN_ID'
RUN_DIR='$RUN_DIR'
EVIDENCE_DIR='$EVIDENCE_DIR'
FORJA_BIN='$RUN_DIR/forja'
STUB_PID='$STUB_PID'
STUB_PORT='$port'
STUB_LOG='$STUB_LOG'
STUB_API_KEY='forja-verify-key'
EOF

echo "$SESSION"

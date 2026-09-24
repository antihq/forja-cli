---
name: verify-forja
description: Verify the Forja CLI end to end by driving the real forja binary against a local stub Forja API — covers every command (whoami, servers, sites, deployments), config precedence, table/json rendering, and error paths, and captures transcripts, exit codes, and the exact HTTP requests the CLI sent. Use whenever you changed CLI behavior (a command, rendering, the HTTP client, or config resolution) and must prove it works the way a user drives it.
---

# Verify Forja CLI

`forja` is a Go/cobra CLI that makes exactly one HTTP call per command against
a Forja REST API (`<endpoint>/api/v1/...`, `Authorization: Bearer <key>`,
optional `X-Forja-Team` header), then renders the response as a table or
indented JSON. The Forja server is not in this repo, so verification drives
the freshly built binary against a local stub API (`helpers/stub_api.py`)
with deterministic fixtures and a request log.

**Safety rule:** never drive the default endpoint `http://127.0.0.1:8000` or
any endpoint this run did not start. It may be a real Forja instance, and
`sites deploy` would trigger a real deployment. The stub started by
`launch.sh` is the only sanctioned target.

## Launch

One command builds the binary fresh and starts the stub on a free port (run
from the repo root):

```bash
SESSION="$(.zcode/skills/verify-forja/helpers/launch.sh)"
. "$SESSION"
```

`launch.sh` returns only after the stub answers (it waits for the stub's
`READY port=<n>` line). Sourcing the session sets `FORJA_BIN` (fresh build),
`STUB_PORT`, `STUB_PID`, `STUB_LOG` (request log, one JSON line per request),
`RUN_DIR` (scratch, deleted on cleanup), and `EVIDENCE_DIR` (proof artifacts,
survive cleanup).

Each invocation of the CLI is a short-lived process; there is no app to keep
alive beyond the stub. Sessions are fully isolated — unique port, unique
`/tmp/forja-verify-*` scratch dir, per-process in-memory stub state — so
concurrent sessions are safe.

## Doctor

Read-only; run whenever anything looks off. It answers "is this session worth
driving?":

```bash
"$FORJA_BIN" version                                  # expect "forja dev" (or built version), exit 0
curl -s "http://127.0.0.1:$STUB_PORT/api/v1/ping"     # expect {"status":"ok"}
kill -0 "$STUB_PID" 2>/dev/null && echo "stub alive"
tail -5 "$STUB_LOG"                                   # what actually arrived: method, path, query, authorization, team, body
```

`version` failing means the build is broken — rerun `launch.sh`. Ping failing
means the stub died — read `$RUN_DIR/stub.out`, then relaunch. A case
behaving oddly is usually visible in `$STUB_LOG`: it shows the exact request
the CLI sent.

## Drive

`helpers/drive.sh` runs one `forja` invocation with the environment locked to
the stub (ambient `FORJA_*` vars and `~/.forja.yaml` cannot leak in), then
writes two evidence files:

```bash
.zcode/skills/verify-forja/helpers/drive.sh "$SESSION" <case-name> [forja args...]
# example:
.zcode/skills/verify-forja/helpers/drive.sh "$SESSION" whoami-basic whoami
```

`<case-name>` is a `[a-z0-9-]` label for the evidence files. Cases needing
environment drive.sh must not control (a wrong API key, a scratch config
file, an unreachable endpoint) use the direct pattern:

```bash
before="$(wc -l < "$STUB_LOG")"
FORJA_API_KEY=wrong-key FORJA_ENDPOINT="http://127.0.0.1:$STUB_PORT" \
  "$FORJA_BIN" whoami > "$EVIDENCE_DIR/<case>.transcript" 2>&1
echo "exit=$?" >> "$EVIDENCE_DIR/<case>.transcript"
tail -n +"$((before + 1))" "$STUB_LOG" > "$EVIDENCE_DIR/<case>.requests"
```

When a case needs a specific format, team, or config file, pass them as
flags (`--format json`, `--team 2`, `--config <file>`): flags win
precedence, so the request log tells the whole story.

Stub fixtures to drive against: servers `srv-01` (web-1), `srv-02` (db-1);
sites `site-01` (acme.com, srv-01), `site-02` (staging.acme.com, srv-01),
`site-03` (intranet.corp, srv-02); seeded deployments `dep-100` (site-01,
success, commit a1b2c3) and `dep-105` (site-03, pending, null commit). The
first `sites deploy` in a session creates `dep-200`, the next `dep-201`,
and so on; they are visible to later reads.

## Evidence

Every case produces, in `$EVIDENCE_DIR`:

- `<case>.transcript` — `CMD:` line, stdout, stderr, and `exit=N`.
- `<case>.requests` — the JSON lines the stub logged while the case ran: the
  exact method, path, query, `authorization`, `team`, and body. Absent or
  empty when the CLI correctly made no request.

Proof standards:

- Rendered output alone never proves driving. A server table proves the CLI
  renders; the `.requests` file proves it called `GET /api/v1/servers` with
  the right headers. Cite both.
- Mutations (`sites deploy`, the only POST in the CLI) need three views: the
  transcript (exit 0, new deployment with status `pending`), the `.requests`
  file (`POST /api/v1/sites/<id>/deploy`), and a second read showing the
  side effect (`deployments list --site <id>` now contains the new id).
- Error cases must show the message, `exit=1`, and no cobra usage dump
  (the root command sets `SilenceUsage`).
- Empty results print `No results.` — drive at least one empty state.
- Table output caps at the first 6 keys of the first row; use
  `--format json` when a case depends on anything beyond that.
- The CLI unwraps a sole top-level `{"data": ...}` envelope; the stub wraps
  `/me` and list endpoints that way on purpose. An object with `data` plus
  sibling keys must come through verbatim.

Before driving a mapped feature, read its file in `features/` — a proof
through one entry point is incomplete when the map lists others.

## Cleanup

```bash
.zcode/skills/verify-forja/helpers/cleanup.sh "$SESSION"
```

Kills only the stub PID recorded in the session (and only after checking the
process is still `stub_api.py`), copies the full request log to
`$EVIDENCE_DIR/stub-full.log`, then deletes `$RUN_DIR`. Evidence in
`$EVIDENCE_DIR` survives cleanup — never delete it during teardown. Never
kill by process name. To rescue a crashed run, find its `session.env` under
`/tmp/forja-verify-*` and run cleanup the same way, or kill a pid only after
confirming its command line contains `stub_api.py`.

## Helpers

- `helpers/launch.sh` — build + start stub + wait for ready; prints the
  session.env path: `SESSION="$(.zcode/skills/verify-forja/helpers/launch.sh)"`.
- `helpers/drive.sh <session.env> <case-name> <forja args...>` — run one
  case, capture transcript, exit code, and request slice; exits with forja's
  exit code.
- `helpers/cleanup.sh <session.env>` — stop the recorded stub, archive the
  full request log into evidence, delete scratch.
- `helpers/stub_api.py` — the fake Forja API. Started by `launch.sh`; for
  error-injection cases start a second one and parse `READY port=<n>` from
  stdout:
  `python3 helpers/stub_api.py --port 0 --api-key k --log /tmp/fail.log --fail-status 422 --fail-body '{"message":"Ya hay un deploy en curso."}'`

`go test ./...` is the fast gate, but it exercises the command tree
in-process; the stub harness is what proves the real binary end to end.

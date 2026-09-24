# Config and errors

The CLI resolves `api_key`, `endpoint`, `team`, and `format` from a config
file, then `FORJA_*` environment variables, then flags — later sources win.
Errors must name the exact missing source and never dump cobra usage.

## Sub-features

- `config-precedence` file < env < flag, observable in the Authorization header the stub logs.
- `config-missing-key` fails naming all three sources before making any request.
- `config-malformed-file` fails reading a broken YAML file.
- `config-bad-format` rejects a format that is not `table` or `json`.
- `endpoint-unreachable` reports the connection failure with the full URL.
- `errors-http-messages` maps a JSON `message`, a raw body, and an empty body for 4xx/5xx.

## How to get to it (user POV)

- Put settings in `~/.forja.yaml`, export `FORJA_*` variables, or pass flags.
- Run any command with a missing or invalid key and read the error.

## Driving it with the stub-API harness

Preconditions:

- Session sourced; doctor passes. These cases use the direct pattern —
  drive.sh pins the key/endpoint, which would hide the very thing under
  test. Write scratch configs into `$RUN_DIR`, never the user's home.

- **File provides the key.** Write `$RUN_DIR/file.yaml` containing
  `api_key: from-file`. Direct pattern with all four `FORJA_*` unset
  (`env -u FORJA_API_KEY -u FORJA_ENDPOINT -u FORJA_TEAM -u FORJA_FORMAT ...`)
  and `--config "$RUN_DIR/file.yaml" whoami`: exit `0`; `.requests` shows
  `authorization: "Bearer from-file"`.
- **Env beats file.** Same config plus `FORJA_API_KEY=from-env`: exit `0`;
  `.requests` shows `Bearer from-env`.
- **Flag beats env.** Same plus `--api-key from-flag`: exit `0`; `.requests`
  shows `Bearer from-flag`.
- **Missing key.** Direct pattern with a scratch config lacking `api_key`
  and all `FORJA_*` unset: `exit=1`, stderr
  `forja: api_key is required. Set api_key in ~/.forja.yaml, or FORJA_API_KEY, or --api-key`,
  `.requests` empty or absent.
- **Malformed file.** Scratch file containing `api_key: [unclosed`:
  `exit=1`, stderr starting `forja: reading config:`, no request made.
- **Bad format.** Valid scratch config plus `--format yaml whoami`:
  `exit=1`, stderr `forja: format must be "table" or "json"`, no request.
- **Unreachable endpoint.** `FORJA_ENDPOINT=http://127.0.0.1:1` with a valid
  key: `exit=1`, stderr matching
  `forja: request to http://127.0.0.1:1/api/v1/me failed:`.
- **HTTP error mapping.** Start a second stub with fault injection and parse
  its port from `READY port=`:
  `python3 helpers/stub_api.py --port 0 --api-key forja-verify-key --log "$RUN_DIR/fail.log" --fail-status 422 --fail-body '{"message":"Ya hay un deploy en curso."}' > "$RUN_DIR/fail.out" 2>&1 &`
  Then drive `sites deploy site-01` against it (direct pattern,
  `FORJA_ENDPOINT=http://127.0.0.1:<port>`): `exit=1`, stderr
  `forja: Ya hay un deploy en curso. (HTTP 422)`. Repeat with
  `--fail-status 500 --fail-body boom` → `boom (HTTP 500)`, and
  `--fail-status 501 --fail-body ''` → `unknown error (HTTP 501)`. Kill the
  second stub by its recorded PID only, after checking its command line.

## Gotchas

- A user's real `~/.forja.yaml` and shell `FORJA_*` vars poison these cases.
  The direct pattern must unset all four env vars and pass `--config`
  explicitly. If `.requests` shows a key that is neither from-file, from-env,
  nor from-flag, discard the run and tighten the invocation.
- `--config /nonexistent` is not an error (treated as "no file"); a malformed
  file is. Do not conflate the two.
- The 422 fixture message is Spanish on purpose: it proves the CLI passes
  the API's message through verbatim instead of translating or wrapping it.
- Endpoint precedence resolves the same way as the key; the key cases cover
  the mechanism — spot-check one endpoint case via the request path if in doubt.

# whoami

`whoami` verifies the configured credentials against the API and shows the
signed-in user's identity and teams. It is the CLI's smoke test: if it fails,
nothing else will work.

## Sub-features

- `whoami-identity` renders the user's id, name, and email as a table.
- `whoami-teams` includes each team's id and name, personal team first.
- `whoami-json` emits indented JSON with document-order keys under `--format json`.
- `whoami-team-header` sends the `--team` value as the `X-Forja-Team` header.
- `whoami-unauthenticated` fails with the API's 401 message, exit 1, no usage dump.

## How to get to it (user POV)

- Run `forja whoami` in a terminal (with api_key and endpoint configured).

## Driving it with the stub-API harness

Preconditions:

- A launch.sh session is sourced; doctor passes (`"$FORJA_BIN" version`,
  ping answers `{"status":"ok"}`).
- The stub accepts only `Bearer forja-verify-key`.

- **Identity table.** Run
  `helpers/drive.sh "$SESSION" whoami-basic whoami`. Exit `0`; stdout has
  the header `ID  NAME  EMAIL  TEAMS` and values `1`, `Verify Agent`,
  `verify@example.test`. `.requests` shows `GET /api/v1/me` with
  `authorization: "Bearer forja-verify-key"` and `accept: "application/json"`.
- **Teams.** The same transcript's TEAMS cell contains both teams,
  `{"id":1,"name":"Personal","personal_team":true}` first and
  `{"id":2,"name":"Acme","personal_team":false}`.
- **JSON format.** Run
  `helpers/drive.sh "$SESSION" whoami-json --format json whoami`. Exit `0`;
  stdout is the fixture indented with 4 spaces, keys in document order
  (`id`, `name`, `email`, `teams`) — the `{"data": ...}` envelope unwrapped.
- **Team header.** Run
  `helpers/drive.sh "$SESSION" whoami-team --team 2 whoami`. Exit `0`;
  `.requests` shows `"team": "2"`. A run without `--team` shows `"team": ""`.
- **Bad key.** Direct pattern with `FORJA_API_KEY=wrong-key`: stderr is
  `forja: Unauthenticated. (HTTP 401)`, `exit=1`, and no cobra usage dump;
  `.requests` proves the wrong key was the one actually sent.

## Gotchas

- The stub wraps `/me` in a sole `{"data": ...}` envelope on purpose; the
  CLI must unwrap it. Raw `{"data":...}` text in the transcript means the
  unwrap broke.
- drive.sh always sends the stub key; testing auth failure requires the
  direct pattern, not drive.sh.
- A missing key never reaches the API: `api_key is required. ...` with exit
  1 and an empty `.requests`. That case lives in
  [Config and errors](./config-and-errors.md).

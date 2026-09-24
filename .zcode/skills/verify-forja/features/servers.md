# Servers

Servers are the machines Forja manages. The CLI lists them and shows one by
id.

## Sub-features

- `servers-list` lists all servers as a table (six columns: id, name, ip, status, region, created_at).
- `servers-list-json` lists all servers as JSON with the `data` envelope unwrapped.
- `servers-get` shows one server by id.
- `servers-get-missing` fails with the API's 404 message, exit 1, no usage dump.

## How to get to it (user POV)

- Run `forja servers list` in a terminal.
- Run `forja servers get <id>` in a terminal with an id from the list.

## Driving it with the stub-API harness

Preconditions:

- Session sourced; doctor passes.

- **List.** Run `helpers/drive.sh "$SESSION" servers-list servers list`.
  Exit `0`; header row `ID  NAME  IP  STATUS  REGION  CREATED_AT`, rows
  `srv-01 web-1` and `srv-02 db-1`. `.requests` shows
  `GET /api/v1/servers` with an empty query.
- **List JSON.** Run
  `helpers/drive.sh "$SESSION" servers-list-json --format json servers list`.
  Exit `0`; both server objects, keys in document order, no `data` wrapper.
- **Get.** Run `helpers/drive.sh "$SESSION" servers-get servers get srv-01`.
  Exit `0`; the row contains `web-1` and `203.0.113.10`. `.requests` shows
  `GET /api/v1/servers/srv-01`.
- **Missing.** Run
  `helpers/drive.sh "$SESSION" servers-get-missing servers get nope`.
  `exit=1`, stderr `forja: Server not found. (HTTP 404)`, no usage dump.
  `.requests` shows the 404 request — the CLI surfaces the API's message
  verbatim.

## Gotchas

- The table caps at the first six keys of the first row. The fixtures have
  exactly six; if a fixture grows a seventh field, only `--format json`
  shows it.
- Ids are passed verbatim into the path — `servers get SRV-01` is a 404 from
  the stub, not a CLI normalization bug.

# Command reference

## Commands

| Command | Action |
| --- | --- |
| `forja version` | Print the version |
| `forja whoami` | Show the signed-in user and teams |
| `forja servers list` | List servers |
| `forja servers get <id>` | Show one server |
| `forja sites list [--server <id>]` | List sites, optionally for one server |
| `forja sites get <id>` | Show one site |
| `forja sites create [flags]` | Create a site on a server |
| `forja sites deploy <id>` | Trigger a deployment |
| `forja deployments list --site <id>` | List deployments for a site |
| `forja deployments get <id>` | Show one deployment |
| `forja completion <shell>` | Generate the autocompletion script for the specified shell |
| `forja help [command]` | Help about any command |

## Global flags

Every command accepts these flags:

| Flag | Meaning | Default |
| --- | --- | --- |
| `--api-key` | Forja personal API key | |
| `--config` | Config file | `~/.forja.yaml` |
| `--endpoint` | Forja base URL | `https://forja.antihq.com` |
| `--format` | Output format, table or json | `table` |
| `--team` | Team id | your personal team |
| `-h`, `--help` | Help for the command | |

## Settings

The config file is `~/.forja.yaml`, or the file that `--config` names. Settings resolve file first, then env var, then flag. A missing config file is not an error. A malformed one is.

| Setting | Config key | Env var | Flag | Default |
| --- | --- | --- | --- | --- |
| API key | `api_key` | `FORJA_API_KEY` | `--api-key` | |
| Endpoint | `endpoint` | `FORJA_ENDPOINT` | `--endpoint` | `https://forja.antihq.com` |
| Team | `team` | `FORJA_TEAM` | `--team` | your personal team |
| Format | `format` | `FORJA_FORMAT` | `--format` | `table` |

## Command groups

**whoami.** `forja whoami` takes no arguments and shows the signed-in user and teams.

**servers.** `forja servers list` takes no arguments. `forja servers get <id>` takes exactly one server id.

**sites.** `forja sites list` takes no arguments and accepts `--server <id>` for one server's sites. `forja sites get <id>` and `forja sites deploy <id>` each take exactly one site id. `forja sites create` takes no positional arguments and requires `--server <id>`, `--address <domain>`, `--php-version <version>`, and `--type <type>`. It also accepts `--web-folder` (default `public`), `--zero-downtime` (default true), `--repository`, `--branch`, and `--deploy-key`. Missing required flags fail before any HTTP call, naming all of them at once:

```console
$ forja sites create --address example.com
forja: sites create requires --server <id>, --php-version <version>, --type <type>. Example: forja sites create --server 01abc --address example.com --php-version php85 --type laravel
```

**deployments.** `forja deployments list` requires `--site <id>`. `forja deployments get <id>` takes exactly one deployment id.

**version.** `forja version` takes no arguments, prints `forja dev` on a source build, and makes no network call. The root command also accepts `-v` and `--version`.

**completion.** `forja completion <shell>` generates the autocompletion script for the specified shell.

**help.** `forja help [command]` prints help about any command.

## Output rules

- Table columns come from the keys of the first row, in document order, capped at the first 6.
- `null` prints as `-`.
- An empty result prints `No results.`.
- JSON output is indented with 4 spaces.
- A scalar response prints as one plain line.

## HTTP shape

Each API command makes exactly one call to `<endpoint>/api/v1/...`. Every call sends `Authorization: Bearer <key>` and `Accept: application/json`. The `X-Forja-Team` header goes out only when a team is set. The HTTP client times out after 60 seconds. `sites create` and `sites deploy` are the only commands that POST. Every other command uses GET.

## Errors

Errors print on stderr as `forja: <message>` and exit 1. No usage dump follows. Examples:

- `forja: api_key is required. Set api_key in ~/.forja.yaml, or FORJA_API_KEY, or --api-key`
- `forja: Unauthenticated. (HTTP 401)`, for a rejected key
- `forja: deployments list requires --site <id>. Example: forja deployments list --site 01abc`

HTTP errors take the server's `message` field, or the raw body, and append `(HTTP <code>)`. A validation body that also carries an `errors` object adds each `field: first message` pair to the line, fields sorted alphabetically. An invalid `--format` fails every API command with `forja: format must be "table" or "json"`.

Next: [Recipes and pitfalls](./06-recipes-and-pitfalls.md).

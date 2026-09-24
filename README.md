# forja

forja is the command line client for Forja. it manages servers, sites, and deployments from your terminal. you type a command, forja makes one api call, and you get a table or json back. the same commands work for you at a prompt and for an agent in a script.

## why a cli

**if you have to click it, you can't automate it.** the Forja web ui shows you state. a cli acts on it: from a script, from cron, from an agent you hand a task to. forja covers the loop that matters: list what exists, get one thing, deploy, read the result.

## install

```bash
curl -fsSL https://raw.githubusercontent.com/antihq/forja-cli/main/install.sh | sh
```

the script detects your operating system and architecture, downloads the latest release from GitHub, and installs a single binary to `~/.local/bin`. run as root, it uses `/usr/local/bin` instead. set `FORJA_INSTALL_DIR` to pick any other directory. the binary has no runtime dependencies.

on windows, download `forja-windows-amd64.exe` from the [releases page](https://github.com/antihq/forja-cli/releases) and put it on your `PATH`. with go installed, `go install github.com/antihq/forja-cli/cmd/forja@latest` works too.

## get started

1. generate a personal api key in Forja under **Settings > API**. the key is shown once, so copy it right away. regenerating invalidates the old key.
2. write your settings to `~/.forja.yaml`:

   ```yaml
   api_key: your-personal-api-key
   endpoint: https://forja.example.com
   ```

   `team` and `format` have defaults. that is the whole file.
3. run `forja whoami`. you should see your name, email, and teams.

## usage

the core loop:

```console
$ forja servers list
ID      NAME   IP            STATUS   REGION  CREATED_AT
srv-01  web-1  203.0.113.10  running  nyc1    2026-01-15T10:00:00Z
srv-02  db-1   203.0.113.20  running  nyc1    2026-02-01T09:30:00Z

$ forja sites deploy site-01
ID       SITE_ID  STATUS   COMMIT  CREATED_AT
dep-200  site-01  pending  -       2026-09-24T01:34:42Z

$ forja deployments list --site site-03 --format json
[
    {
        "id": "dep-105",
        "site_id": "site-03",
        "status": "pending",
        "commit": null,
        "created_at": "2026-04-02T11:00:00Z"
    }
]
```

<details>
<summary>every command</summary>

| command | what it does |
| --- | --- |
| `forja version` | print the version |
| `forja whoami` | show the signed-in user and teams |
| `forja servers list` | list servers |
| `forja servers get <id>` | show one server |
| `forja sites list [--server <id>]` | list sites, optionally for one server |
| `forja sites get <id>` | show one site |
| `forja sites deploy <id>` | trigger a deployment |
| `forja deployments list --site <id>` | list deployments for a site |
| `forja deployments get <id>` | show one deployment |

</details>

`--format json` prints indented json on every api command. `sites deploy` is the only command that changes anything. it tells Forja to start a deployment and prints the new one with status `pending`. `--team` targets another team you belong to. without it, forja acts on your personal team.

## configuration

forja reads four settings. the config file is the base, environment variables override it, and flags override both.

| setting | environment variable | flag | default |
| --- | --- | --- | --- |
| `api_key` | `FORJA_API_KEY` | `--api-key` | none |
| `endpoint` | `FORJA_ENDPOINT` | `--endpoint` | `http://127.0.0.1:8000` |
| `team` | `FORJA_TEAM` | `--team` | your personal team |
| `format` | `FORJA_FORMAT` | `--format` | `table` |

each setting name is its key in `~/.forja.yaml`. a missing config file is fine. a malformed one stops the command with an error. `--config <path>` reads another file. `api_key` is the only setting with no default, and every command that talks to the api needs it.

## for agents

failures print one line to stderr and exit 1. the line names the exact file, variable, or flag to fix:

```console
$ forja whoami
forja: api_key is required. Set api_key in ~/.forja.yaml, or FORJA_API_KEY, or --api-key
```

each api command is one http call and one render. there is nothing to log into and no state between commands.

## why so small?

**the server owns the hard part.** forja triggers a deployment and reads the result. the deploy itself happens on the server. that leaves the cli with one command, one api call, and one render. there is no daemon and no plugin system.

the table renderer reads its columns from the first row of the response and stops at six. a response that is a bare `{"data": ...}` envelope is unwrapped first. anything past six columns is a job for `--format json`, which prints everything the api sent.

## development

```bash
go test ./...
```

for end-to-end proof, [`.zcode/skills/verify-forja/`](.zcode/skills/verify-forja/SKILL.md) drives the real binary against a stub Forja api and records transcripts, exit codes, and the exact http requests.

pushing a `v*` tag builds darwin, linux, and windows binaries on GitHub Actions and publishes them to the [releases page](https://github.com/antihq/forja-cli/releases).

## fork it

the whole client lives in [`cmd/`](cmd/). fork it, rename it, make it yours. PRs are welcome.

## license

[MIT](LICENSE)

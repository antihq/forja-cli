# Forja CLI

A command line interface for managing Forja: servers, sites, and deployments. Built for humans and for AI agents.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/antihq/forja-cli/main/install.sh | sh
```

The script installs a static binary to `~/.local/bin` (or `/usr/local/bin` when run as root). Set `FORJA_INSTALL_DIR` to choose another location. There are no runtime dependencies.

Windows: download `forja-windows-amd64.exe` from the [releases page](https://github.com/antihq/forja-cli/releases) and put it on your `PATH`.

With Go installed you can also run `go install github.com/antihq/forja-cli@latest`.

## Getting an API key

Generate a personal API key in Forja under **Settings > API**. The key is shown once, so copy it immediately. Regenerating invalidates the old key.

## Configuration

The CLI reads a config file, then environment variables, then command line flags. Later sources win.

`~/.forja.yaml`:

```yaml
api_key: your-personal-api-key
endpoint: https://forja.example.com
team: 1
format: json
```

| Config     | Env var          | Flag         | Default              |
| ---------- | ---------------- | ------------ | -------------------- |
| `api_key`  | `FORJA_API_KEY`  | `--api-key`  |                      |
| `endpoint` | `FORJA_ENDPOINT` | `--endpoint` | `http://127.0.0.1:8000` |
| `team`     | `FORJA_TEAM`     | `--team`     | your personal team   |
| `format`   | `FORJA_FORMAT`   | `--format`   | `table`              |

## Usage

```bash
forja whoami                          # verify credentials, list your teams
forja servers list
forja servers get <id>
forja sites list [--server <id>]
forja sites get <id>
forja sites deploy <id>               # trigger a zero-downtime deployment
forja deployments list --site <id>
forja deployments get <id>
```

Every command accepts `--format json` for machine-readable output. A missing or invalid key fails with a message telling you exactly which config file, environment variable, or flag to set. `--team` targets another team you belong to; without it the CLI uses your personal team.

## Development

```bash
go test ./...
```

Releases are built by GitHub Actions when a `v*` tag is pushed.

## License

[MIT](LICENSE)

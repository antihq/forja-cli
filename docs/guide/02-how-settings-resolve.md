# How settings resolve

forja takes each setting from one of three sources, always in the same order. The config file is the base. The `FORJA_*` environment variables override the file. Flags override both. A flag counts only when you pass it.

The order serves override-in-place workflows. Keep the durable settings in the file, so every call starts from them. Export per-machine values in the shell, such as the endpoint of a test instance. Win a single call with a flag, without editing any file.

## A worked example

The three runs below share one scratch file. `~/.forja.yaml` holds an `api_key` and `format: table`, and nothing else changes between the runs.

```bash
forja whoami
```

```text
ID  NAME          EMAIL                TEAMS
1   Verify Agent  verify@example.test  [{"id":1,"name":"Personal","personal_team":true},{"id":2,"name":"Acme","personal_team":false}]
```

The file sets the format, so the output is a table. Now override the file from the shell:

```bash
export FORJA_FORMAT=json
forja whoami
```

```text
{
    "id": 1,
    "name": "Verify Agent",
    "email": "verify@example.test",
    "teams": [
        {
            "id": 1,
            "name": "Personal",
            "personal_team": true
        },
        {
            "id": 2,
            "name": "Acme",
            "personal_team": false
        }
    ]
}
```

The environment variable beats the file, and the same command prints JSON. Now win the call back with a flag, with `FORJA_FORMAT` still exported:

```bash
forja whoami --format table
```

```text
ID  NAME          EMAIL                TEAMS
1   Verify Agent  verify@example.test  [{"id":1,"name":"Personal","personal_team":true},{"id":2,"name":"Acme","personal_team":false}]
```

The flag beats the environment variable, and the table returns.

## Defaults, other files, and the team

Two settings have defaults that apply when no source sets them. The endpoint is `http://127.0.0.1:8000` and the format is `table`. A trailing slash on the endpoint is trimmed.

`--config` names a file for one call. The named file replaces `~/.forja.yaml` as the base.

Team is the fourth setting. When the file holds `team: 2`, every API call carries the `X-Forja-Team` header with the value `2`. The block below is the request the CLI sent for `forja sites list` under that file:

```bash
forja sites list
```

```text
{"time": "2026-09-24T01:37:01.732+00:00", "method": "GET", "path": "/api/v1/sites", "query": {}, "authorization": "Bearer forja-verify-key", "team": "2", "accept": "application/json", "body": null}
```

The `team` field in the log is the header value. Without a team, the CLI sends no header at all.

A missing config file is fine. A malformed one fails the command. The [settings table](./05-command-reference.md) lists every setting with its config key, env var, flag, and default.

Next: [Your first deployment](./03-first-deployment.md).

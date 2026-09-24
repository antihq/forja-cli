# Automate with JSON

This page is for scripts and AI agents that drive forja.

## Add `--format json`

Every API command accepts `--format json`. The output is one JSON document indented with 4 spaces:

```bash
forja whoami --format json
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

Prefer JSON in scripts. The table format prints at most the first 6 keys of the first row and flattens the rest. The `whoami` table shows the difference, with the teams array crushed into one cell:

```bash
forja whoami
```

```text
ID  NAME          EMAIL                TEAMS
1   Verify Agent  verify@example.test  [{"id":1,"name":"Personal","personal_team":true},{"id":2,"name":"Acme","personal_team":false}]
```

The JSON above keeps the real shape, with `teams` as an array of objects.

## Trust the exit codes

forja exits 0 on success and 1 on any failure. Errors go to stderr as `forja: <message>`, and stdout stays empty, so a script can capture one stream without the other. Both errors below exit 1.

A rejected credential fails like this:

```bash
forja whoami --api-key wrong-key
```

```text
forja: Unauthenticated. (HTTP 401)
```

A missing argument fails before any HTTP call:

```bash
forja deployments list
```

```text
forja: deployments list requires --site <id>. Example: forja deployments list --site 01abc
```

## Expect an empty result

An empty result is not a failure. The command still exits 0 and prints `No results.`:

```bash
forja sites list --server srv-99
```

```text
No results.
```

A script that checks only the exit code reads an empty list as success.

Next: [Command reference](./05-command-reference.md).

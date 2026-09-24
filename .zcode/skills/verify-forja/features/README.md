# Forja CLI verification map

This directory is the maintained source for verifying the user-facing
behavior of the `forja` CLI. Read this index before driving, then use the
matching feature file as the recipe. A proof that drives one convenient
entry point is incomplete when the feature file lists others.

## Baseline preconditions

- Start a session with `.zcode/skills/verify-forja/helpers/launch.sh` and
  source the printed `session.env`. Drive only `$FORJA_BIN` (built fresh by
  the launcher) — never the stale binaries sitting in the repo root
  (`./forja`, `./forja-cli`).
- Never point the CLI at the default endpoint `http://127.0.0.1:8000` or any
  instance this run did not start; `sites deploy` triggers a real deployment.
- The stub requires `Authorization: Bearer forja-verify-key` and logs every
  request (method, path, query, authorization, team, body) to `$STUB_LOG`.
- Fixtures (per session, in-memory): servers `srv-01` (web-1), `srv-02`
  (db-1); sites `site-01` (acme.com, srv-01), `site-02` (staging.acme.com,
  srv-01), `site-03` (intranet.corp, srv-02); deployments `dep-100`
  (site-01, success, commit a1b2c3) and `dep-105` (site-03, pending, null
  commit). First deploy of a session creates `dep-200`, then `dep-201`, ...
  visible to later reads.

## Driving conventions

- Default to `helpers/drive.sh "$SESSION" <case> <args...>`; it pins the
  environment to the stub and captures proof automatically.
- Cases needing other env (wrong key, scratch config, dead endpoint) use the
  direct pattern from SKILL.md; never leave ambient `FORJA_*` vars set — the
  user's real `~/.forja.yaml` and shell env poison results otherwise.
- Pass `--format`, `--team`, `--config` as flags when a case needs them;
  flags win precedence, so the request log tells the whole story.
- Treat every command as literal. Keep ids and flags unchanged.

## Proof and skip reporting

- Every case needs the transcript (command, stdout, stderr, exit code) and
  the requests file (the HTTP the CLI actually sent). Output alone is not
  proof of driving.
- Mutations need a second read proving the side effect.
- Report an unreachable path with the attempted command and the unmet
  precondition. Do not report a skipped entry point as verified through a
  different path.

## Features

- [whoami](./whoami.md) — credential check, identity, teams, team header, auth failure.
- [Servers](./servers.md) — list, get, missing server, JSON format.
- [Sites](./sites.md) — list, server filter, get, deploy (the only mutation), empty state.
- [Deployments](./deployments.md) — list per site, `--site` requirement, get, empty and null states.
- [Config and errors](./config-and-errors.md) — file/env/flag precedence, key errors, format validation, HTTP error mapping.

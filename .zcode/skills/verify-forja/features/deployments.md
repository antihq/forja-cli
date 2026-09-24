# Deployments

Deployments are the history of site deploys. The CLI lists them per site and
shows one by id.

## Sub-features

- `deployments-list` lists a site's deployments with `--site <id>`.
- `deployments-list-empty` prints `No results.` when the site has none.
- `deployments-list-requires-site` refuses to run without `--site`, making no request.
- `deployments-get` shows one deployment by id.
- `deployments-get-missing` fails with 404.
- `deployments-null-commit` renders a null commit as `-`.

## How to get to it (user POV)

- Run `forja deployments list --site <id>`.
- Run `forja deployments get <id>`.

## Driving it with the stub-API harness

Preconditions:

- Session sourced; doctor passes. `site-02` starts with zero deployments in
  a fresh session.

- **List.** `helpers/drive.sh "$SESSION" deployments-list deployments list --site site-01`
  — exit `0`; `dep-100` with `success` and commit `a1b2c3`. `.requests`
  shows `GET /api/v1/sites/site-01/deployments` with an empty query.
- **Empty.** `helpers/drive.sh "$SESSION" deployments-empty deployments list --site site-02`
  — stdout `No results.`, exit `0`.
- **Requires --site.** `helpers/drive.sh "$SESSION" deployments-no-site deployments list`
  — `exit=1`, stderr `forja: deployments list requires --site <id>. Example:
  forja deployments list --site 01abc`, and `.requests` is empty or absent:
  no HTTP call happened.
- **Get.** `helpers/drive.sh "$SESSION" deployments-get deployments get dep-105`
  — exit `0`; row shows `pending` and commit `-` (null renders as a dash);
  `.requests` shows `GET /api/v1/deployments/dep-105`.
- **Missing.** `helpers/drive.sh "$SESSION" deployments-get-missing deployments get nope`
  — `exit=1`, stderr `forja: Deployment not found. (HTTP 404)`.

## Gotchas

- The `--site` guard fires before any HTTP: an error transcript plus a
  non-empty `.requests` means the guard broke.
- `deployments get` looks up the id across all sites in the stub; it does
  not need `--site`.
- After a `sites deploy` in the same session the new deployment appears in
  that site's list — expected session state, not pollution.
- Null and missing fields render as `-` in tables; assert on `--format json`
  when the difference between null and absent matters.

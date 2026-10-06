# Sites

Sites are the web properties deployed to servers. The CLI lists them
(optionally per server), shows one, creates one on a server, and triggers
zero-downtime deployments.

## Sub-features

- `sites-list` lists all sites.
- `sites-list-filter` filters by `--server <id>` via the `server_id` query param.
- `sites-list-empty` prints `No results.` for a server id with no sites.
- `sites-get` shows one site.
- `sites-create` POSTs the payload to the server's sites path and renders the created site with its pending deployment.
- `sites-create-deploy-key` sends `use_deploy_key` and gets `deploy_key_public` instead of a deployment.
- `sites-create-missing-flags` refuses to run without the required flags, making no request.
- `sites-create-duplicate` fails with the 422 message plus the flattened `errors`.
- `sites-deploy` POSTs to the deploy path and renders the new pending deployment.
- `sites-deploy-missing` fails with 404 for an unknown site id and creates nothing.
- Deployment settings (`settings get`, `settings set`) live in [sites-settings.md](./sites-settings.md).

## How to get to it (user POV)

- Run `forja sites list [--server <server id>]`.
- Run `forja sites get <id>`.
- Run `forja sites create --server <id> --address <domain> --php-version <version> --type <type>` to create a site.
- Run `forja sites deploy <id>` to trigger a deployment.

## Driving it with the stub-API harness

Preconditions:

- Session sourced; doctor passes. Each session starts a fresh in-memory
  stub, so the site and deploy counters reset.

- **List.** `helpers/drive.sh "$SESSION" sites-list sites list` — exit `0`;
  rows `site-01 acme.com`, `site-02 staging.acme.com`, `site-03
  intranet.corp`; `.requests` shows `GET /api/v1/sites` with an empty query.
- **Filter.** `helpers/drive.sh "$SESSION" sites-filter sites list --server srv-01`
  — exactly `site-01` and `site-02`. `.requests` shows `"query":
  {"server_id": "srv-01"}`. The stub filters for real, so the rows prove the
  param flowed through end to end, not just that a query was attached.
- **Empty.** `helpers/drive.sh "$SESSION" sites-filter-empty sites list --server nope`
  — stdout `No results.`, exit `0`; the query is still sent.
- **Get.** `helpers/drive.sh "$SESSION" sites-get sites get site-01` — exit
  `0`; row contains `acme.com`; `.requests` shows `GET /api/v1/sites/site-01`.
- **Create.** `helpers/drive.sh "$SESSION" sites-create sites create --server srv-01 --address new.acme.com --php-version php85 --type laravel`
  — exit `0`; the table shows the created `site-04` with `new.acme.com`.
  `.requests` shows `POST /api/v1/servers/srv-01/sites` and the logged body:
  the five always-sent fields, no repository keys, no `use_deploy_key`.
  Then prove the side effect with a second read:
  `helpers/drive.sh "$SESSION" sites-create-after deployments list --site site-04`
  — now contains `dep-200` with status `pending`. Rendering alone is not a
  create proof.
- **Create with deploy key.** `helpers/drive.sh "$SESSION" sites-create-key --format json sites create --server srv-01 --address key.acme.com --php-version php85 --type laravel --repository git@github.com:acme/site.git --branch main --deploy-key`
  — exit `0`; the JSON carries `deploy_key_public` and no `deployment` key.
  The body in `.requests` carries `repository_url`, `repository_branch`, and
  `"use_deploy_key": true`. Second read:
  `helpers/drive.sh "$SESSION" sites-create-key-after deployments list --site site-05`
  — prints `No results.`; with `--deploy-key` the first deploy is deferred
  to `sites deploy site-05`.
- **Create missing flags.** `helpers/drive.sh "$SESSION" sites-create-missing sites create --address x.test`
  — `exit=1`, stderr `forja: sites create requires --server <id>,
  --php-version <version>, --type <type>. Example: forja sites create
  --server 01abc --address example.com --php-version php85 --type laravel`,
  and `.requests` is empty or absent: no HTTP call happened.
- **Create duplicate.** `helpers/drive.sh "$SESSION" sites-create-duplicate sites create --server srv-01 --address acme.com --php-version php85 --type laravel`
  — `exit=1`, stderr `forja: El campo address ya ha sido tomado.; address:
  El campo address ya ha sido tomado. (HTTP 422)`. The POST appears in
  `.requests`, and a follow-up `sites list` shows nothing created.
- **Deploy.** `helpers/drive.sh "$SESSION" sites-deploy sites deploy site-02`
  — exit `0`; the table shows a new deployment with `site-02` and
  `pending`. `.requests` shows `POST /api/v1/sites/site-02/deploy`. Then
  prove the side effect with a second read:
  `helpers/drive.sh "$SESSION" sites-deploy-after deployments list --site site-02`
  — now contains that deployment id. Rendering alone is not a deploy proof.
- **Deploy missing.** `helpers/drive.sh "$SESSION" sites-deploy-missing sites deploy nope`
  — `exit=1`, stderr `forja: Site not found. (HTTP 404)`; the POST appears
  in `.requests`, and a follow-up `deployments list --site site-02` is
  unchanged.

## Gotchas

- `sites create` and `sites deploy` are the POSTs and `sites settings set`
  is the PATCH. A mutation proof without the matching line in `.requests`
  is not a proof.
- Created ids depend on session state. The first create in a session makes
  `site-04`, the second `site-05`; created deployments and deploys draw
  from one counter, so the first of either in a session is `dep-200`, the
  next `dep-201`. Assert the id you observed in the transcript; do not
  assume a fixed id across cases.
- The filter matches the stub's `server_id` exactly — `--server 1` (a team
  id, not a server id) legitimately prints `No results.`.
- A create table shows the first six keys of the contract body (`id`
  through `php_version`). The `deployment` and `deploy_key_public` keys sit
  past the cap — assert them with `--format json`.

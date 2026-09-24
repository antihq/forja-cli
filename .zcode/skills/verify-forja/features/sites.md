# Sites

Sites are the web properties deployed to servers. The CLI lists them
(optionally per server), shows one, and triggers zero-downtime deployments.

## Sub-features

- `sites-list` lists all sites.
- `sites-list-filter` filters by `--server <id>` via the `server_id` query param.
- `sites-list-empty` prints `No results.` for a server id with no sites.
- `sites-get` shows one site.
- `sites-deploy` POSTs to the deploy path and renders the new pending deployment.
- `sites-deploy-missing` fails with 404 for an unknown site id and creates nothing.

## How to get to it (user POV)

- Run `forja sites list [--server <server id>]`.
- Run `forja sites get <id>`.
- Run `forja sites deploy <id>` to trigger a deployment.

## Driving it with the stub-API harness

Preconditions:

- Session sourced; doctor passes. Each session starts a fresh in-memory
  stub, so deploy counters reset.

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
- **Deploy.** `helpers/drive.sh "$SESSION" sites-deploy sites deploy site-02`
  — exit `0`; the table shows the new deployment `dep-200` with `site-02`
  and `pending`. `.requests` shows `POST /api/v1/sites/site-02/deploy`.
  Then prove the side effect with a second read:
  `helpers/drive.sh "$SESSION" sites-deploy-after deployments list --site site-02`
  — now contains `dep-200`. Rendering alone is not a deploy proof.
- **Deploy missing.** `helpers/drive.sh "$SESSION" sites-deploy-missing sites deploy nope`
  — `exit=1`, stderr `forja: Site not found. (HTTP 404)`; the POST appears
  in `.requests`, and a follow-up `deployments list --site site-02` is
  unchanged.

## Gotchas

- `sites deploy` is the only mutation and the only POST in the CLI. A deploy
  proof without the POST line in `.requests` is not a proof.
- Deploy ids depend on session state: the first deploy in a session is
  `dep-200`, the second `dep-201`. Assert the id you observed in the deploy
  transcript; do not assume a fixed id across cases.
- The filter matches the stub's `server_id` exactly — `--server 1` (a team
  id, not a server id) legitimately prints `No results.`.

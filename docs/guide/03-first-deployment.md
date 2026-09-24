# Your first deployment

In this tutorial you deploy a site and watch the deployment. The ids below come from one example instance. Yours differ.

## Find a server

```bash
forja servers list
```

```text
ID      NAME   IP            STATUS   REGION  CREATED_AT
srv-01  web-1  203.0.113.10  running  nyc1    2026-01-15T10:00:00Z
srv-02  db-1   203.0.113.20  running  nyc1    2026-02-01T09:30:00Z
```

We deploy a site on `srv-01`.

## Find the site

```bash
forja sites list
```

```text
ID       NAME              SERVER_ID  STATUS  REPOSITORY     CREATED_AT
site-01  acme.com          srv-01     live    acme/site      2026-01-16T08:00:00Z
site-02  staging.acme.com  srv-01     live    acme/site      2026-01-20T12:00:00Z
site-03  intranet.corp     srv-02     paused  corp/intranet  2026-03-01T15:30:00Z
```

Pass `--server` to narrow the list to one server:

```bash
forja sites list --server srv-01
```

```text
ID       NAME              SERVER_ID  STATUS  REPOSITORY  CREATED_AT
site-01  acme.com          srv-01     live    acme/site   2026-01-16T08:00:00Z
site-02  staging.acme.com  srv-01     live    acme/site   2026-01-20T12:00:00Z
```

We deploy `site-02`.

## Deploy the site

```bash
forja sites deploy site-02
```

```text
ID       SITE_ID  STATUS   COMMIT  CREATED_AT
dep-200  site-02  pending  -       2026-09-24T01:36:29Z
```

The command returns at once. The new deployment sits at status `pending` with no commit, printed as `-`. There is no confirmation prompt. `sites deploy` is the only forja command that changes anything.

## Watch the deployment

```bash
forja deployments list --site site-02
```

```text
ID       SITE_ID  STATUS   COMMIT  CREATED_AT
dep-200  site-02  pending  -       2026-09-24T01:36:29Z
```

`--site` is required, and the command fails without it. To inspect one deployment, use `get`. The run below looks at `dep-100`, an older deployment that the server has finished. Its status and commit come from the server:

```bash
forja deployments get dep-100
```

```text
ID       SITE_ID  STATUS   COMMIT  CREATED_AT
dep-100  site-01  success  a1b2c3  2026-04-01T09:00:00Z
```

`get` shows the current status and commit. Run `forja deployments get dep-200` to see where the deployment you just triggered stands.

Next: [Automate with JSON](./04-automate-with-json.md).

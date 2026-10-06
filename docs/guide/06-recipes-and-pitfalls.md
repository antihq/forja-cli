# Recipes and pitfalls

## Recipes

Each recipe has one goal. Copy the block and swap in your ids.

### List site ids for a script

`jq` reads the JSON and prints one id per line:

```bash
forja sites list --format json | jq -r '.[].id'
```

### Deploy and confirm in one CI step

Deploy, then list the site's deployments to see the new row. Each command exits 1 on failure, so a failed deploy fails the step.

```bash
forja sites deploy site-02
forja deployments list --site site-02
```

### Point one command at another instance

`--endpoint` overrides the endpoint for this call only. The config file stays as it is:

```bash
forja servers list --endpoint https://forja.example.com
```

### Prove a machine's credentials

`whoami` in JSON shows the account behind the key and its teams. `jq` reads the teams array:

```bash
forja whoami --format json | jq -r '.teams[].name'
```

### Create a site and ship it

Create the site with a deploy key, register the key on GitHub as read-only, then start the first deploy. The key first: a clone before GitHub knows it fails, and the failed deployment emails you about it.

```bash
forja servers get srv-01 --format json        # read php_versions
forja sites create --server srv-01 --address soplon.antihq.com \
  --php-version php85 --type laravel \
  --repository git@github.com:antihq/soplon.git --branch main \
  --deploy-key --format json                  # deploy_key_public, no deployment
# register deploy_key_public on the repository as a deploy key
forja sites deploy <site-id>                  # pending deployment
forja deployments get <dep-id>                # poll until finished
curl -fsS https://soplon.antihq.com           # the actual proof
```

### Change deployment settings without SSH

The settings document is reachable from a script, so a hook or a retention change needs no browser and no SSH. `@scripts/build.sh` reads the hook body from the file in your checkout. A set never deploys, so the deploy comes after it, and `&&` keeps the deploy from firing if the set failed:

```bash
forja sites settings set site-02 --hook-before-making-current @scripts/build.sh && forja sites deploy site-02
```

Read the result first with `forja sites settings get site-02` — the full document prints as json. Check what the deploy did with `forja deployments get <dep-id>`.

## The pitfalls

- **Settings changes apply on the NEXT deploy.** `sites settings set` only writes the document; it never triggers a deployment, and a running deploy does not pick the change up. The loop is `set`, then `sites deploy`.
- **Retention is never null.** The server keeps an integer release count, so clearing it is not a thing: `--stdin '{"deployment_releases_retention":null}'` fails with `forja: El campo deployment releases retention debe ser un entero.; deployment_releases_retention: El campo deployment releases retention debe ser un entero. (HTTP 422)`. Omit `--retention` to leave it alone, or pass a number from 1 to 50.
- **A pending-uninstall site refuses settings changes.** A site requested for uninstall fails the manage gate on both settings commands: `forja: Esta acción no está autorizada. (HTTP 403)`.
- **`sites create` and `sites deploy` fire with no confirmation.** Both POST at once, `sites settings set` PATCHes at once, against whatever endpoint the file, the environment, and the flags resolve to. Check `--endpoint` or your config before you script them.
- **The API key is shown once.** Forja displays it under Settings > API when you generate it. Regenerating invalidates the old key.
- **Tables hide columns.** A table prints only the first 6 keys of the first row and drops the rest. Use `--format json` when a response has more keys.
- **Environment variables leak into one-off commands.** An exported `FORJA_FORMAT` or `FORJA_ENDPOINT` shapes every call in that shell. A flag wins, but only for the call you pass it to. `--config` swaps the whole file.
- **A wrong `--team` looks like missing resources.** The API scopes the list to the team in the `X-Forja-Team` header, so a mistyped id hides the resources you expect.
- **`forja version` proves the binary, not the credentials.** It never touches the network. `forja whoami` makes a real call, so success there means the key, the endpoint, and the team resolve.

Back to the [guide index](./README.md).

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

## The pitfalls

- **`sites deploy` fires with no confirmation.** The command POSTs at once, against whatever endpoint the file, the environment, and the flags resolve to. Check `--endpoint` or your config before you script it.
- **The API key is shown once.** Forja displays it under Settings > API when you generate it. Regenerating invalidates the old key.
- **Tables hide columns.** A table prints only the first 6 keys of the first row and drops the rest. Use `--format json` when a response has more keys.
- **Environment variables leak into one-off commands.** An exported `FORJA_FORMAT` or `FORJA_ENDPOINT` shapes every call in that shell. A flag wins, but only for the call you pass it to. `--config` swaps the whole file.
- **A wrong `--team` looks like missing resources.** The API scopes the list to the team in the `X-Forja-Team` header, so a mistyped id hides the resources you expect.
- **`forja version` proves the binary, not the credentials.** It never touches the network. `forja whoami` makes a real call, so success there means the key, the endpoint, and the team resolve.

Back to the [guide index](./README.md).

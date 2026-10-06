# Sites settings

Every site carries a deployment settings document: the notification email,
the release retention, the shared and writeable paths, and the four deploy
hooks. The CLI reads it with `sites settings get` and patches it with
`sites settings set` (PATCH). `set` never deploys; the changed settings
apply on the site's NEXT deploy.

## Sub-features

- `settings-get` prints the settings document. It is json by default; an
  explicit `--format table` renders the generic six-column table.
- `settings-set-flags` PATCHes only what the passed flags name; a second get
  proves the other fields survived.
- `settings-set-hook-file` expands `@<path>` into the file contents before
  any request.
- `settings-set-stdin` carries a whole json body on stdin.
- `settings-clear-empty` proves the documented clears: empty email sends
  null, an empty list flag sends `[]`.
- `settings-set-no-flags` refuses to run without `--stdin` or a field flag,
  making no request.
- `settings-set-422` fails with the server's Spanish retention message.
- `settings-set-missing` fails with Laravel's model-not-found 404.

## How to get to it (user POV)

- Run `forja sites settings get <site-id>`.
- Run `forja sites settings set <site-id> --retention 20` (or any of
  `--shared-directory`, `--shared-file`, `--writeable-directory`,
  `--hook-before-updating-repository`, `--hook-after-updating-repository`,
  `--hook-before-making-current`, `--hook-after-making-current`,
  `--notification-email`; or `--stdin` for a whole json body).

## Driving it with the stub-API harness

Preconditions:

- Session sourced; doctor passes. Each session starts a fresh in-memory
  stub, so the settings documents reset.
- Settings state is per site and mutations persist for the session. The
  cases below run in order against distinct sites so the expectations stay
  independent: site-01 (creation defaults), site-02 (seeded retention 30,
  email ops@acme.test, two shared directories), site-03 (creation defaults).

- **Get.** `helpers/drive.sh "$SESSION" settings-get sites settings get site-01`
  — exit `0`; indented json, the eleven keys in contract order,
  `"deployment_releases_retention": 10`, `"deploy_notification_email": null`.
  `.requests` shows `GET /api/v1/sites/site-01/settings`. Then the table
  path: `helpers/drive.sh "$SESSION" settings-get-table sites settings get site-01 --format table`
  — exit `0`; one row, `site-01 true - 10 ["storage"] [".env"]`, the first
  six columns only.
- **Set flags.** `helpers/drive.sh "$SESSION" settings-set-flags sites settings set site-01 --retention 20 --shared-directory storage --shared-directory public`
  — exit `0`; the output is the merged json document. `.requests` shows
  `PATCH /api/v1/sites/site-01/settings` and the logged body carries ONLY
  `"deployment_releases_retention": 20` and
  `"shared_directories": ["storage", "public"]` — no email, no hooks, no
  other keys. Then prove the merge with a second read:
  `helpers/drive.sh "$SESSION" settings-set-flags-after sites settings get site-01`
  — retention `20` and the two shared directories, while
  `deploy_notification_email` is still `null` and the hooks are still `""`.
  Rendering alone is not a patch proof.
- **Set hook from a file.** Write the fixture first, then pass it with `@`:

  ```bash
  cat > "$EVIDENCE_DIR/hook-fixture.sh" <<'EOF'
  php artisan config:clear
  php artisan migrate --force
  EOF
  helpers/drive.sh "$SESSION" settings-set-hook-file sites settings set site-02 --hook-before-making-current @"$EVIDENCE_DIR/hook-fixture.sh"
  ```

  — exit `0`. The body in `.requests` carries
  `"hook_before_making_current"` with the two lines verbatim, including the
  trailing newline. Only that one key is in the body.
- **Set from stdin.** Pipe the body through jq and into the case:

  ```bash
  echo '{"deployment_releases_retention": 15, "hook_before_making_current": "php artisan down"}' \
    | jq -c . \
    | helpers/drive.sh "$SESSION" settings-set-stdin sites settings set site-03 --stdin
  ```

  — exit `0`; `.requests` shows the PATCH with the minified object as the
  body. A second read
  (`helpers/drive.sh "$SESSION" settings-set-stdin-after sites settings get site-03`)
  shows retention `15` and the down hook.
- **Clear with empty values.** `helpers/drive.sh "$SESSION" settings-clear-empty sites settings set site-02 --notification-email "" --shared-file ""`
  — exit `0`. The body carries `"deploy_notification_email": null` and
  `"shared_files": []` — the documented clears, not empty strings. A second
  read proves site-02 lost `ops@acme.test`.
- **No flags.** `helpers/drive.sh "$SESSION" settings-set-no-flags sites settings set site-01`
  — `exit=1`, stderr `forja: sites settings set requires --stdin or at
  least one settings flag. Example: forja sites settings set 01abc
  --retention 20`, and `.requests` is empty or absent: no HTTP call
  happened.
- **Retention null is a 422.** The server rejects a null retention with the
  standard Spanish integer message:

  ```bash
  echo '{"deployment_releases_retention": null}' | jq -c . \
    | helpers/drive.sh "$SESSION" settings-set-422 sites settings set site-01 --stdin
  ```

  — `exit=1`, stderr `forja: El campo deployment releases retention debe
  ser un entero.; deployment_releases_retention: El campo deployment
  releases retention debe ser un entero. (HTTP 422)`. The PATCH appears in
  `.requests`, and a follow-up `sites settings get site-01` shows retention
  still `20` from the flags case: nothing was saved.
- **Unknown site.** `helpers/drive.sh "$SESSION" settings-set-missing sites settings set nope --retention 20`
  — `exit=1`, stderr `forja: No query results for model
  [App\Models\Site] nope. (HTTP 404)`. The PATCH appears in `.requests`.
- **Created sites.** A `sites create` in the same session gets a full
  default settings document too:
  `helpers/drive.sh "$SESSION" settings-created-site sites settings get site-04`
  — retention `10`, and `zero_downtime_deployment` mirrors the created
  site's flag.

## Gotchas

- Both settings commands print json unless `--format table` is passed — the
  opposite default of every other command. A settings table caps at the
  first six columns and crushes the hooks into unreadable cells, so the
  json default is deliberate.
- `--stdin` rejects keys outside the editable nine before any request. The
  error lists the allowed keys. `zero_downtime_deployment` and `site_id`
  are read-only and are rejected, not ignored.
- `sites settings set` is the only PATCH in the CLI, and `sites create`,
  `sites deploy` are the POSTs. A mutation proof without the matching line
  in `.requests` is not a proof.
- `@file` values are read before any HTTP: a missing fixture fails with the
  flag named in the error and no request. There is no `@-` and no `@@`
  escape; a hook that must start with `@` cannot be passed literally.
- Settings changes apply on the NEXT deploy. `sites settings set` never
  triggers one; the loop is `set`, then `sites deploy`.

# Install and sign in

In this tutorial you install the forja binary, write a config file, and prove that your key works. You finish with the setup every later chapter assumes.

## Install the binary

Run the install script.

```bash
curl -fsSL https://raw.githubusercontent.com/antihq/forja-cli/main/install.sh | sh
```

The script puts a static binary in `~/.local/bin`, or in `/usr/local/bin` when you run it as root.

Two alternates, if you prefer them:

- With Go installed, run `go install github.com/antihq/forja-cli/cmd/forja@latest`.
- On Windows, download `forja-windows-amd64.exe` from the [releases page](https://github.com/antihq/forja-cli/releases) and put it on your `PATH`.

## Confirm the binary

Confirm the build with `forja version`:

```bash
forja version
```

```text
forja dev
```

A release binary prints its version tag, and a build from source prints `dev`. The command never touches the network.

## Generate an API key

Open Forja and go to **Settings > API**. Generate a personal API key and copy it now. The key is shown once, and regenerating invalidates the old one.

## Write the config file

Create `~/.forja.yaml` and paste your key into it:

```yaml
api_key: your-personal-api-key
endpoint: https://forja.example.com
team: 1
format: json
```

Set `endpoint` to the base URL of your Forja instance. Chapter 2 covers what this file may hold.

## Prove the key

Prove the key with `forja whoami`:

```bash
forja whoami
```

```text
ID  NAME          EMAIL                TEAMS
1   Verify Agent  verify@example.test  [{"id":1,"name":"Personal","personal_team":true},{"id":2,"name":"Acme","personal_team":false}]
```

The table shows the account your key belongs to, with your teams in the last column. Your values differ from these.

Next: [How settings resolve](./02-how-settings-resolve.md).

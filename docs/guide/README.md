# The forja guide

forja is a command line tool that manages Forja servers, sites, and deployments. This guide serves the person typing the commands and the scripts or agents that drive them.

Here's what you'll learn:

1. [Install and sign in](./01-install-and-sign-in.md). Put a working binary on the machine and prove the key.
2. [How settings resolve](./02-how-settings-resolve.md). Know which source wins when the file, the environment, and the flags disagree.
3. [Your first deployment](./03-first-deployment.md). Run the loop from listing servers to watching a deployment.
4. [Automate with JSON](./04-automate-with-json.md). Point scripts and agents at JSON, exit codes, and stderr.
5. [Command reference](./05-command-reference.md). Look up every command, flag, default, and error message.
6. [Recipes and pitfalls](./06-recipes-and-pitfalls.md). Copy working recipes and skip the traps.

Read the pages in order the first time. After that, each one stands alone.

## If you only remember one thing

When anything fails, run `forja whoami` first. It makes one real call, so it proves the key and the endpoint.

```bash
forja whoami
```

```text
ID  NAME          EMAIL                TEAMS
1   Verify Agent  verify@example.test  [{"id":1,"name":"Personal","personal_team":true},{"id":2,"name":"Acme","personal_team":false}]
```

Next: [Install and sign in](./01-install-and-sign-in.md).

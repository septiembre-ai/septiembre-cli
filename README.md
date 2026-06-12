# septiembre CLI

Command-line interface for the [Septiembre cloud platform](https://cloud.septiembre.ai).

## Agent-first contract

JSON is the **default** output format. All commands print structured JSON to stdout and JSON error envelopes to stderr. This makes every command trivially composable with `jq` and usable from LLM agents without any flag changes.

```bash
# List apps — stdout is valid JSON
septiembre apps list

# Get the machine-readable command tree (for agents and scripts)
septiembre --help --json | jq '.commands[].name'
```

Error envelopes on stderr always follow this shape:
```json
{ "error": "app not found", "code": "not_found", "http_status": 404 }
```

Exit codes:

| Code | Meaning |
|------|---------|
| 0 | success |
| 1 | general / API error |
| 2 | auth error (401 / 403 / missing token) |
| 3 | not found (404) |
| 4 | validation / bad input (400 / 422) |
| 5 | network error (no HTTP response) |

## Install

> Installation packages are available from v1.0.0 release. Until then, build from source.

```bash
go install github.com/septiembre-ai/septiembre-cli/cmd/septiembre@latest
```

Homebrew (coming in v1.0.0):
```bash
brew install septiembre-ai/tap/septiembre
```

## Authentication

### CI and agents (recommended)

Set the `SEPTIEMBRE_TOKEN` environment variable to a personal access token:

```bash
export SEPTIEMBRE_TOKEN=sapi_<hex>
septiembre apps list
```

Create a token via the API or CLI:
```bash
septiembre auth token create --name "ci-deploy"
# stdout: {"id":"...","name":"ci-deploy","token":"sapi_...","last_four":"...","warning":"Token shown once..."}
```

Tokens are created at `POST https://api.septiembre.ai/api/v1/auth/tokens`.

### Config file (dev/local)

Tokens can also be stored in the config file at:
- **Linux/macOS**: `~/.config/septiembre/config.yaml`
- **Windows**: `%APPDATA%\septiembre\config.yaml`

```yaml
token: sapi_<hex>
org: my-org          # default organization slug (used when --org is omitted)
api_url: https://api.septiembre.ai   # override for local dev
```

The file must be `0600` (owner read/write only). CI **must** use `SEPTIEMBRE_TOKEN`.

## Command reference

### Global flags

| Flag | Default | Description |
|------|---------|-------------|
| `--output json\|table` | `json` | Output format (JSON for agents, table for humans) |
| `--org <slug>` | — | Organization slug (overrides config default) |
| `--config <path>` | `~/.config/septiembre/config.yaml` | Config file path |
| `--json` | — | With `--help`, emit machine-readable JSON command tree |

### Version and help

```bash
septiembre --version                       # JSON: {"version":"...","commit":"...","built_at":"..."}
septiembre --help --json                   # machine-readable command tree for agents
```

### Auth

```bash
septiembre auth whoami                     # show current user identity
septiembre auth login                      # browser login (coming soon — use SEPTIEMBRE_TOKEN)

septiembre auth token create               # create a PAT (raw token shown once in JSON)
septiembre auth token create --name ci-deploy --expires-at 2026-12-31T00:00:00Z
septiembre auth token list                 # list your PATs (raw value never shown)
septiembre auth token revoke <token-id>    # revoke a PAT
```

### Organizations

```bash
septiembre orgs list                       # list orgs you belong to
```

### Applications

```bash
septiembre apps list                       # list all visible apps (cross-org)
septiembre apps list --org <slug>          # list apps in a specific org
septiembre apps get <app-id> --org <slug>  # get app details
```

### Environment variables

```bash
septiembre env get <app-id> --org <slug>          # list env vars (values masked as ***)
septiembre env get <app-id> --org <slug> --reveal # show plaintext values
septiembre env set <app-id> --org <slug> KEY=value OTHER=value2
# env set is a full replacement (PUT) — omitted keys are deleted
```

### Deployments

```bash
septiembre deploys trigger <app-id> --org <slug> --tag v1.2.3
septiembre deploys trigger <app-id> --org <slug> --tag v1.2.3 --env-id <uuid>
septiembre deploys list <app-id> --org <slug>
septiembre deploys status <app-id> <deploy-id> --org <slug>
```

### Logs

```bash
septiembre logs <app-id> --org <slug>              # fetch log snapshot (non-streaming)
septiembre logs <app-id> --org <slug> --env-id <uuid>
```

## Output formats

All commands default to JSON stdout. Use `--output table` for human-readable output:

```bash
septiembre apps list --output table
```

## Development

```bash
go build ./...
go vet ./...
go test ./...
```

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
septiembre auth token create --description "ci-deploy"
# stdout: {"token":"sapi_...","id":"...","last_four":"..."}
# WARNING (stderr): Token shown once — store it securely.
```

Tokens are created at `POST https://api.septiembre.ai/api/v1/auth/tokens`.

### Config file (dev/local)

Tokens can also be stored in the config file at:
- **Linux/macOS**: `~/.config/septiembre/config.yaml`
- **Windows**: `%APPDATA%\septiembre\config.yaml`

```yaml
token: sapi_<hex>
org: my-org          # default organization slug
api_url: https://api.septiembre.ai   # override for local dev
```

The file must be `0600` (owner read/write only). CI **must** use `SEPTIEMBRE_TOKEN`.

## Quick reference

```bash
septiembre version                         # print version JSON
septiembre auth token create               # create a PAT (shown once)
septiembre auth token list                 # list your PATs
septiembre auth token revoke <id>          # revoke a PAT
septiembre apps list                       # list apps
septiembre apps get <id>                   # get app details
septiembre deploys trigger <app-id> --tag v1.0.0
septiembre env get <app-id>                # env values masked by default
septiembre env get <app-id> --reveal       # show plaintext values
septiembre logs <app-id>                   # fetch log snapshot
```

## Development

```bash
go build ./...
go vet ./...
go test ./...
```

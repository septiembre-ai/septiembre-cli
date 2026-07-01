# septiembre CLI

Command-line interface for the [Septiembre cloud platform](https://cloud.septiembre.ai).

## Quickstart (agents and CI)

```bash
# 1. Install
go install github.com/septiembre-ai/septiembre-cli/cmd/septiembre@latest

# 2. Set your personal access token (create one at POST /api/v1/auth/tokens)
export SEPTIEMBRE_TOKEN=sapi_<hex>

# 3. Run
septiembre apps list                         # JSON array of your apps
septiembre --help --json | jq '.commands'    # full command tree for LLM agents
```

## Agent-first contract

JSON is the **default** output format for automation. Exception: `septiembre changes` opens a local visual graph unless `--output json` or `--output table` is explicit. Commands print JSON error envelopes to stderr, so agent/script paths should pass `--output json` when they need guaranteed JSON stdout.

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

### Install with Homebrew

```bash
brew install --cask septiembre-ai/tap/septiembre
```

Upgrade or uninstall with standard Homebrew commands:

```bash
brew upgrade --cask septiembre
brew uninstall --cask septiembre
```

### Download a release binary (recommended)

Pre-built binaries for Linux, macOS, and Windows are available on the
[Releases page](https://github.com/septiembre-ai/septiembre-cli/releases).

**macOS / Linux:**
```bash
# Set PLATFORM to one of: darwin_arm64, darwin_amd64, linux_amd64, linux_arm64
PLATFORM=darwin_arm64
VERSION=$(curl -fsSL https://api.github.com/repos/septiembre-ai/septiembre-cli/releases/latest \
  | sed -n 's/.*"tag_name": "v\{0,1\}\([^"]*\)".*/\1/p')

curl -L "https://github.com/septiembre-ai/septiembre-cli/releases/latest/download/septiembre_${VERSION}_${PLATFORM}.tar.gz" \
  | tar -xz
sudo mv septiembre /usr/local/bin/
```

**Windows (PowerShell):**
```powershell
$Repo = "septiembre-ai/septiembre-cli"
$Release = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest"
$Version = $Release.tag_name.TrimStart("v")
$Arch = "amd64" # Use "arm64" for Windows on ARM.
$Archive = "septiembre_${Version}_windows_${Arch}.zip"
$InstallDir = "$env:USERPROFILE\bin"

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
Invoke-WebRequest "https://github.com/$Repo/releases/latest/download/$Archive" -OutFile $Archive
Expand-Archive $Archive -DestinationPath $InstallDir -Force
```

Checksums are published alongside each release in `checksums.txt`.

Once installed as a standalone binary, you can update in place with:

```bash
septiembre upgrade          # download and install the latest release
septiembre upgrade --check  # report current vs latest without changing anything
```

`upgrade` downloads the release asset for your OS/arch, verifies its SHA-256
checksum against `checksums.txt`, and replaces the running binary atomically.
If the CLI was installed via Homebrew it will not self-replace — it points you
at `brew upgrade --cask septiembre` so Homebrew keeps managing the version.

### Install with Go

```bash
go install github.com/septiembre-ai/septiembre-cli/cmd/septiembre@latest
```

Requires Go 1.23 or later. The binary is placed in `$GOBIN` (default `~/go/bin`).

### Package managers

Scoop bucket distribution is planned but not available yet. Follow the releases
page for announcements.

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

For local development, verify and save a PAT without putting it in shell
history:

```bash
printf '%s' "$SEPTIEMBRE_TOKEN" | septiembre auth login --token-stdin
septiembre auth whoami
```

Tokens can also be stored manually in the config file at:
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
| `--output json\|table` | `json` | Output format (JSON for agents, table for humans; overrides `changes` visual default) |
| `--org <slug>` | — | Organization slug (overrides config default) |
| `--config <path>` | `~/.config/septiembre/config.yaml` | Config file path |
| `--json` | — | With `--help`, emit machine-readable JSON command tree |

### Version and help

```bash
septiembre --version                       # JSON: {"version":"...","commit":"...","built_at":"..."}
septiembre --help --json                   # machine-readable command tree for agents
septiembre upgrade                         # update a standalone binary to the latest release
septiembre upgrade --check                 # report current vs latest without changing anything
```

`upgrade` is a no-op for Homebrew installs: it returns
`{"status":"managed_by_homebrew"}` and tells you to run `brew upgrade --cask septiembre`.

### Auth

```bash
septiembre auth whoami                     # show current user identity
printf '%s' "$SEPTIEMBRE_TOKEN" | septiembre auth login --token-stdin

septiembre auth token create               # create a PAT (raw token shown once in JSON)
septiembre auth token create --name ci-deploy --expires-at 2026-12-31T00:00:00Z
septiembre auth token list                 # list your PATs (raw value never shown)
septiembre auth token revoke <token-id>    # revoke a PAT
```

`auth login` verifies the PAT with `/api/v1/auth/me` before saving it. Browser
or device-flow login is not available until cloud-api exposes that auth flow.

### Organizations

```bash
septiembre orgs list                       # list orgs you belong to
```

### Teams

```bash
septiembre teams list --org <slug>         # list teams in an org
```

### Applications

```bash
septiembre apps list                       # list all visible apps (cross-org)
septiembre apps list --org <slug>          # list apps in a specific org
septiembre apps get <app-id> --org <slug>  # get app details (includes composed url field)

# Create an app (team auto-selected when org has exactly one team)
septiembre apps create --name my-app --type web --region us-east-1 --org <slug>
septiembre apps create --name my-api --type api --runtime nodejs24 --region us-east-1 --team <slug> --org <slug>

# Create and wait for domain to become active
septiembre apps create --name my-app --type web --region us-east-1 --org <slug> --wait

# Delete an app (async teardown — --yes required)
septiembre apps delete <app-id> --org <slug> --yes
```

**App types**: `web` | `web-ssr` | `api` | `sse`
**Runtimes** (required for non-web types): `nodejs24` | `python314` | `go126`
**Regions**: `us-east-1` | `us-east-2` | `sa-east-1`

The `url` field in `apps get` output is composed as `https://{subdomain}.septiembre.co`.
Override the domain suffix with `SEPTIEMBRE_DOMAIN_SUFFIX` or the `domain_suffix` config key.

### Environment variables

```bash
septiembre env get <app-id> --org <slug>          # list env vars (values masked as ***)
septiembre env get <app-id> --org <slug> --reveal # show plaintext values
septiembre env set <app-id> --org <slug> KEY=value OTHER=value2
printf 'API_KEY=secret\n' | septiembre env set <app-id> --org <slug> --from-stdin
septiembre env set <app-id> --org <slug> --from-file .env
# env set is a full replacement (PUT) — omitted keys are deleted
```

Prefer `--from-stdin` or `--from-file` for secret values. Passing `KEY=value`
directly can expose secrets in shell history or process listings.

### Services

```bash
# Managed KVS (DynamoDB-backed data plane) — tables only; creating the first table activates KVS.
septiembre services kvs tables list <app-id> --org <slug>
septiembre services kvs tables create <app-id> --org <slug> --name sessions --minute-limit 300
septiembre services kvs tables rotate <app-id> sessions --org <slug>
septiembre services kvs tables delete <app-id> sessions --org <slug> --yes
```

KVS is available for `api`, `web-ssr`, and `sse` apps. Plaintext KVS tokens are
shown only on enable/create/rotate responses; store them immediately.

### Deployments

```bash
septiembre deploys trigger <app-id> --org <slug> --tag v1.2.3
septiembre deploys trigger <app-id> --org <slug> --tag v1.2.3 --env-id <uuid>

# Trigger and wait for terminal state (success|failed|cancelled)
septiembre deploys trigger <app-id> --org <slug> --tag v1.2.3 --wait

septiembre deploys list <app-id> --org <slug>
septiembre deploys status <app-id> <deploy-id> --org <slug>
```

**`--wait` flags** (available on `apps create` and `deploys trigger`):

| Flag | Default | Description |
|------|---------|-------------|
| `--wait` | false | Block until terminal state |
| `--wait-interval` | 5s | Polling interval |
| `--wait-timeout` | 10m (create) / 15m (deploys) | Max wait time |

Timeout exits 1 with `code: "wait_timeout"`. Non-success terminal exits 1 with `code: "domain_failed"` or `"deploy_failed"`.

### Logs

```bash
septiembre logs <app-id> --org <slug>              # fetch log snapshot (non-streaming)
septiembre logs <app-id> --org <slug> --env-id <uuid>
```

### Changes

Visualize what changed on the current branch (including uncommitted work) as an
interactive graph of files and internal import edges. Works in any repository:
import edges are resolved for Go, JavaScript/TypeScript (including Astro, Vue,
and Svelte), and Python; other languages still show as files without edges.
Click a node to see its diff with syntax highlighting and `+/-` counts. A
readiness **checklist** flags whether the change set includes source code,
tests, documentation, and a changelog entry.

```bash
septiembre changes                    # open the visual graph (branch vs main)
septiembre changes --base develop     # compare against another branch/ref
septiembre changes --release v0.5.0   # graph + changelog for a release, vs the previous tag
septiembre changes --output json      # full payload for automation
septiembre changes --output table     # deterministic text
```

With `--release`, the graph covers the range from the previous tag to the given
tag and a **Changelog** tab groups the commits by conventional-commit type
(features, fixes, breaking changes, …). Import edges are resolved from the
working tree, so they are exact when the tag equals `HEAD` and approximate for
older tags.

## Output formats

Commands default to JSON stdout for automation, except `septiembre changes`, which opens a visual graph unless `--output` is explicit. Use `--output table` for human-readable text:

```bash
septiembre apps list --output table
septiembre changes --output json
```

## Development

```bash
go build ./...
go vet ./...
go test ./...
```
